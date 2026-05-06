package commands

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/appctx"
)

// publishOpts holds the common publish-target flags shared by all post-creation commands.
type publishOpts struct {
	channelIDs []int
	area       string
	fullScreen bool
}

func addPublishFlags(cmd *cobra.Command, p *publishOpts) {
	cmd.Flags().IntSliceVar(&p.channelIDs, "channel-id", nil, "Channel to publish to (repeatable)")
	cmd.Flags().StringVar(&p.area, "area", "", "Channel area for publication")
	cmd.Flags().BoolVar(&p.fullScreen, "full-screen", false, "Publish as fullscreen")
}

// findDefaultCollection returns the ID of the first default collection matching category.
func findDefaultCollection(a *appctx.App, category string) (int, error) {
	var resp MediaCollectionsResponse
	if err := a.Client.Get("/media_collections", url.Values{"per_page": {"200"}}, &resp); err != nil {
		return 0, err
	}
	for _, c := range resp.Items {
		if c.DefaultCollection && c.Category == category {
			return c.ID, nil
		}
	}
	return 0, fmt.Errorf("no default %s media collection found; use --media-collection to specify one", category)
}

// uploadFileToCollection uploads a local file or URL, waits for the processing task,
// and returns the resulting media IDs (multiple when a PDF is extracted into pages).
func uploadFileToCollection(a *appctx.App, input, collectionID string) ([]int, error) {
	if !a.JSONOutput && isHTTPURL(input) {
		fmt.Fprintf(os.Stderr, "Downloading %s...\n", input)
	}
	source, err := prepareUploadSource(input, "", a.Client.HTTPClient)
	if err != nil {
		return nil, err
	}
	defer source.Close()

	if !a.JSONOutput {
		fmt.Fprintf(os.Stderr, "Creating upload for %s (%s)...\n", source.metadata.filename, formatBytes(source.metadata.byteSize))
	}

	var upload directUploadResponse
	if err := a.Client.Post("/direct_uploads", map[string]any{
		"blob": map[string]any{
			"filename":     source.metadata.filename,
			"content_type": source.metadata.contentType,
			"byte_size":    source.metadata.byteSize,
			"checksum":     source.metadata.checksum,
		},
	}, &upload); err != nil {
		return nil, err
	}
	if upload.SignedID == "" || upload.DirectUpload.URL == "" {
		return nil, fmt.Errorf("direct upload response missing signed_id or upload URL")
	}

	file, err := os.Open(source.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if !a.JSONOutput {
		fmt.Fprintf(os.Stderr, "Uploading %s...\n", source.metadata.filename)
	}
	if err := a.Client.PutDirectUpload(upload.DirectUpload.URL, upload.DirectUpload.Headers, file, source.metadata.byteSize); err != nil {
		return nil, err
	}

	var taskResp TaskResponse
	if err := a.Client.Post("/media_collections/"+collectionID+"/media", map[string]any{
		"media": map[string]any{"source": upload.SignedID},
	}, &taskResp); err != nil {
		return nil, err
	}
	if !a.JSONOutput {
		fmt.Fprintf(os.Stderr, "Processing task %d...\n", taskResp.Task.ID)
	}
	return waitForTask(a, taskResp.Task.ID)
}

// createResource creates a resource and returns its ID.
func createResource(a *appctx.App, resourceType string, data map[string]any) (int, error) {
	var resp map[string]any
	if err := a.Client.Post("/resources", map[string]any{
		"type":     resourceType,
		"resource": data,
	}, &resp); err != nil {
		return 0, err
	}
	return intFromNestedResp(resp, "resource", "id")
}

// createResourceAndPrint creates a resource and prints the full response as JSON.
func createResourceAndPrint(a *appctx.App, resourceType string, data map[string]any) error {
	var resp map[string]any
	if err := a.Client.Post("/resources", map[string]any{
		"type":     resourceType,
		"resource": data,
	}, &resp); err != nil {
		return err
	}
	printJSON(resp)
	return nil
}

// createAndPublishImagePost creates an image post from mediaIDs, then publishes to channels.
func createAndPublishImagePost(a *appctx.App, name string, mediaIDs []int, duration int, p publishOpts) error {
	images := make([]map[string]any, len(mediaIDs))
	for i, id := range mediaIDs {
		images[i] = map[string]any{"media_id": id}
	}
	post := map[string]any{
		"duration_per_item": duration,
		"images":            images,
	}
	if name != "" {
		post["name"] = name
	}

	var postResp map[string]any
	if err := a.Client.Post("/posts", map[string]any{"type": "image", "post": post}, &postResp); err != nil {
		return err
	}

	postID, err := intFromNestedResp(postResp, "post", "id")
	if err != nil {
		return err
	}
	if !a.JSONOutput {
		fmt.Printf("Created image post %d\n", postID)
	}

	if err := publishToChannels(a, postID, p.channelIDs, p.area, p.fullScreen); err != nil {
		return err
	}

	if a.JSONOutput {
		printJSON(postResp)
	}
	return nil
}

// createResourceBackedPost creates a resource, then a post wired to it, then publishes.
func createResourceBackedPost(a *appctx.App, postType, resourceType string, resourceData map[string]any, name string, p publishOpts) error {
	resourceID, err := createResource(a, resourceType, resourceData)
	if err != nil {
		return fmt.Errorf("creating %s resource: %w", resourceType, err)
	}
	if !a.JSONOutput {
		fmt.Printf("Created %s resource %d\n", resourceType, resourceID)
	}

	post := map[string]any{"resource_ids": []int{resourceID}}
	if name != "" {
		post["name"] = name
	}

	var postResp map[string]any
	if err := a.Client.Post("/posts", map[string]any{"type": postType, "post": post}, &postResp); err != nil {
		return err
	}

	postID, err := intFromNestedResp(postResp, "post", "id")
	if err != nil {
		return err
	}
	if !a.JSONOutput {
		fmt.Printf("Created %s post %d\n", postType, postID)
	}

	if err := publishToChannels(a, postID, p.channelIDs, p.area, p.fullScreen); err != nil {
		return err
	}

	if a.JSONOutput {
		printJSON(postResp)
	}
	return nil
}

// publishToChannels publishes postID to each channel, applying area and fullScreen to all.
func publishToChannels(a *appctx.App, postID int, channelIDs []int, area string, fullScreen bool) error {
	postIDStr := strconv.Itoa(postID)
	for _, chID := range channelIDs {
		pub := map[string]any{"channel_id": chID}
		if area != "" {
			pub["area"] = area
		}
		if fullScreen {
			pub["full_screen"] = true
		}
		var resp map[string]any
		if err := a.Client.Post("/posts/"+postIDStr+"/publications", map[string]any{"publication": pub}, &resp); err != nil {
			return fmt.Errorf("publishing to channel %d: %w", chID, err)
		}
		if !a.JSONOutput {
			fmt.Printf("Published post %d to channel %d\n", postID, chID)
		}
	}
	return nil
}

// intFromNestedResp extracts an int from resp[outer][inner] where JSON numbers decode as float64.
func intFromNestedResp(resp map[string]any, outer, inner string) (int, error) {
	obj, ok := resp[outer].(map[string]any)
	if !ok {
		return 0, fmt.Errorf("unexpected response: missing %q object", outer)
	}
	f, ok := obj[inner].(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected response: %s.%s not numeric", outer, inner)
	}
	return int(f), nil
}

// NewPublishCmd is a top-level shorthand: upload a file and publish it as an image post.
func NewPublishCmd() *cobra.Command {
	var name string
	var collectionID int
	var duration int
	var pf publishOpts

	cmd := &cobra.Command{
		Use:   "publish <file-or-url>",
		Short: "Upload a file and publish it as an image post to one or more channels",
		Example: `  pintomind publish ./photo.jpg --channel-id 7 --name "Lobby photo"
  pintomind publish https://example.com/photo.jpg --channel-id 7 --channel-id 8 --name "Remote image"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			input := args[0]

			colID := collectionID
			if colID == 0 {
				id, err := findDefaultCollection(a, "image")
				if err != nil {
					return err
				}
				colID = id
			}

			uploadedIDs, err := uploadFileToCollection(a, input, strconv.Itoa(colID))
			if err != nil {
				return err
			}

			postName := name
			if postName == "" {
				postName = filepath.Base(input)
			}

			return createAndPublishImagePost(a, postName, uploadedIDs, duration, pf)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name (defaults to filename)")
	cmd.Flags().IntVar(&collectionID, "media-collection", 0, "Media collection ID for upload (defaults to default image collection)")
	cmd.Flags().IntVar(&duration, "duration", 7, "Seconds per slide")
	addPublishFlags(cmd, &pf)
	return cmd
}

// newPostsCreateImageCmd creates an image post, optionally uploading files first.
func newPostsCreateImageCmd() *cobra.Command {
	var name string
	var images []string
	var mediaIDs []int
	var mediaCollection int
	var duration int
	var pf publishOpts

	cmd := &cobra.Command{
		Use:   "image",
		Short: "Create an image post from uploaded files or existing media IDs",
		Example: `  pintomind posts create image --name "Gallery" --image ./a.jpg --image ./b.jpg --channel-id 7
  pintomind posts create image --name "Promo" --media 42 --media 43 --duration 10
  pintomind posts create image --name "Mix" --image ./photo.jpg --media 55 --channel-id 7`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			if len(images) == 0 && len(mediaIDs) == 0 {
				return fmt.Errorf("provide at least one --image or --media flag")
			}

			allMediaIDs := make([]int, 0, len(images)+len(mediaIDs))

			if len(images) > 0 {
				colID := mediaCollection
				if colID == 0 {
					id, err := findDefaultCollection(a, "image")
					if err != nil {
						return err
					}
					colID = id
				}
				colIDStr := strconv.Itoa(colID)
				for _, img := range images {
					ids, err := uploadFileToCollection(a, img, colIDStr)
					if err != nil {
						return err
					}
					allMediaIDs = append(allMediaIDs, ids...)
				}
			}
			allMediaIDs = append(allMediaIDs, mediaIDs...)

			return createAndPublishImagePost(a, name, allMediaIDs, duration, pf)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name")
	cmd.Flags().StringArrayVar(&images, "image", nil, "File path or URL to upload (repeatable)")
	cmd.Flags().IntSliceVar(&mediaIDs, "media", nil, "Existing media ID to include (repeatable)")
	cmd.Flags().IntVar(&mediaCollection, "media-collection", 0, "Media collection ID for uploads (defaults to default image collection)")
	cmd.Flags().IntVar(&duration, "duration", 7, "Seconds per slide")
	addPublishFlags(cmd, &pf)
	return cmd
}

// createMediaBoxID creates a media box and returns its ID.
func createMediaBoxID(a *appctx.App, boxType string, data map[string]any) (int, error) {
	var resp map[string]any
	if err := a.Client.Post("/media_boxes", map[string]any{
		"type":      boxType,
		"media_box": data,
	}, &resp); err != nil {
		return 0, err
	}
	return intFromNestedResp(resp, "media_box", "id")
}

// newPostsCreatePlainCmd creates a plain text post.
func newPostsCreatePlainCmd() *cobra.Command {
	var name, heading, body string
	var headingAlignment, bodyAlignment string
	var headingFontsize, bodyFontsize int
	var images, emojis, icons, gifs, unsplashIDs []string
	var mediaIDs []int
	var pf publishOpts

	cmd := &cobra.Command{
		Use:   "plain",
		Short: "Create a plain text post",
		Example: `  pintomind posts create plain --name "Welcome" --heading "<p>Hello</p>" --channel-id 7
  pintomind posts create plain --name "Msg" --heading "<p>Hi</p>" --heading-alignment center --heading-fontsize 150 --body "<p>Content</p>" --body-alignment left --body-fontsize 80
  pintomind posts create plain --name "Photo" --heading "<p>Look</p>" --image ./photo.jpg --unsplash abc123 --channel-id 7`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			if !cmd.Flags().Changed("heading") && !cmd.Flags().Changed("body") {
				return fmt.Errorf("provide at least one of --heading or --body")
			}

			for _, f := range []struct {
				name string
				val  int
			}{
				{"heading-fontsize", headingFontsize},
				{"body-fontsize", bodyFontsize},
			} {
				if cmd.Flags().Changed(f.name) && (f.val < 50 || f.val > 400) {
					return fmt.Errorf("--%s must be between 50 and 400", f.name)
				}
			}

			totalBoxes := len(images) + len(mediaIDs) + len(gifs) + len(unsplashIDs) + len(emojis) + len(icons)
			if totalBoxes > 4 {
				return fmt.Errorf("maximum 4 media boxes allowed, got %d", totalBoxes)
			}

			// Collect media box IDs, creating each in turn.
			var mediaBoxIDs []int

			for _, imgPath := range images {
				colID, err := findDefaultCollection(a, "image")
				if err != nil {
					return err
				}
				uploaded, err := uploadFileToCollection(a, imgPath, strconv.Itoa(colID))
				if err != nil {
					return err
				}
				boxID, err := createMediaBoxID(a, "media", map[string]any{"media_id": uploaded[0]})
				if err != nil {
					return fmt.Errorf("creating media box for %s: %w", imgPath, err)
				}
				if !a.JSONOutput {
					fmt.Printf("Created media box %d\n", boxID)
				}
				mediaBoxIDs = append(mediaBoxIDs, boxID)
			}

			for _, mID := range mediaIDs {
				boxID, err := createMediaBoxID(a, "media", map[string]any{"media_id": mID})
				if err != nil {
					return fmt.Errorf("creating media box for media %d: %w", mID, err)
				}
				if !a.JSONOutput {
					fmt.Printf("Created media box %d\n", boxID)
				}
				mediaBoxIDs = append(mediaBoxIDs, boxID)
			}

			for _, gifID := range gifs {
				boxID, err := createMediaBoxID(a, "gif", map[string]any{"gif_id": gifID})
				if err != nil {
					return fmt.Errorf("creating media box for gif %s: %w", gifID, err)
				}
				if !a.JSONOutput {
					fmt.Printf("Created media box %d\n", boxID)
				}
				mediaBoxIDs = append(mediaBoxIDs, boxID)
			}

			for _, photoID := range unsplashIDs {
				boxID, err := createMediaBoxID(a, "unsplash", map[string]any{"photo_id": photoID})
				if err != nil {
					return fmt.Errorf("creating media box for unsplash %s: %w", photoID, err)
				}
				if !a.JSONOutput {
					fmt.Printf("Created media box %d\n", boxID)
				}
				mediaBoxIDs = append(mediaBoxIDs, boxID)
			}

			for _, emoji := range emojis {
				boxID, err := createMediaBoxID(a, "emoji", map[string]any{"emoji": emoji})
				if err != nil {
					return fmt.Errorf("creating media box for emoji %s: %w", emoji, err)
				}
				if !a.JSONOutput {
					fmt.Printf("Created media box %d\n", boxID)
				}
				mediaBoxIDs = append(mediaBoxIDs, boxID)
			}

			for _, icon := range icons {
				iconName, iconType, _ := strings.Cut(icon, ":")
				if iconType == "" {
					iconType = "regular"
				}
				boxID, err := createMediaBoxID(a, "icon", map[string]any{"icon_name": iconName, "icon_type": iconType})
				if err != nil {
					return fmt.Errorf("creating media box for icon %s: %w", icon, err)
				}
				if !a.JSONOutput {
					fmt.Printf("Created media box %d\n", boxID)
				}
				mediaBoxIDs = append(mediaBoxIDs, boxID)
			}

			post := map[string]any{}
			if name != "" {
				post["name"] = name
			}
			if cmd.Flags().Changed("heading") {
				post["heading"] = heading
			}
			if cmd.Flags().Changed("body") {
				post["body"] = body
			}
			if len(mediaBoxIDs) > 0 {
				post["media_box_ids"] = mediaBoxIDs
			}

			if cmd.Flags().Changed("heading-alignment") || cmd.Flags().Changed("heading-fontsize") {
				opts := map[string]any{}
				if cmd.Flags().Changed("heading-alignment") {
					opts["alignment"] = headingAlignment
				}
				if cmd.Flags().Changed("heading-fontsize") {
					opts["fontsize"] = headingFontsize
				}
				post["heading_options"] = opts
			}

			if cmd.Flags().Changed("body-alignment") || cmd.Flags().Changed("body-fontsize") {
				opts := map[string]any{}
				if cmd.Flags().Changed("body-alignment") {
					opts["alignment"] = bodyAlignment
				}
				if cmd.Flags().Changed("body-fontsize") {
					opts["fontsize"] = bodyFontsize
				}
				post["body_options"] = opts
			}

			var postResp map[string]any
			if err := a.Client.Post("/posts", map[string]any{"type": "plain", "post": post}, &postResp); err != nil {
				return err
			}

			postID, err := intFromNestedResp(postResp, "post", "id")
			if err != nil {
				return err
			}
			if !a.JSONOutput {
				fmt.Printf("Created plain post %d\n", postID)
			}

			if err := publishToChannels(a, postID, pf.channelIDs, pf.area, pf.fullScreen); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(postResp)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name")
	cmd.Flags().StringVar(&heading, "heading", "", "Heading HTML (e.g. '<p>Hello</p>')")
	cmd.Flags().StringVar(&body, "body", "", "Body HTML")
	cmd.Flags().StringVar(&headingAlignment, "heading-alignment", "", "Heading text alignment: left, center, or right")
	cmd.Flags().IntVar(&headingFontsize, "heading-fontsize", 100, "Heading font size as percentage (50–400, default 100)")
	cmd.Flags().StringVar(&bodyAlignment, "body-alignment", "", "Body text alignment: left, center, or right")
	cmd.Flags().IntVar(&bodyFontsize, "body-fontsize", 100, "Body font size as percentage (50–400, default 100)")
	cmd.Flags().StringArrayVar(&images, "image", nil, "Image file or URL to upload as media box (repeatable, max 4 total)")
	cmd.Flags().IntSliceVar(&mediaIDs, "media", nil, "Existing media ID to use as media box (repeatable)")
	cmd.Flags().StringArrayVar(&gifs, "gif", nil, "GIF ID for a gif media box (repeatable)")
	cmd.Flags().StringArrayVar(&unsplashIDs, "unsplash", nil, "Unsplash photo ID for an unsplash media box (repeatable)")
	cmd.Flags().StringArrayVar(&emojis, "emoji", nil, "Emoji character for an emoji media box (repeatable)")
	cmd.Flags().StringArrayVar(&icons, "icon", nil, "Icon name (or name:type) for an icon media box (repeatable)")
	addPublishFlags(cmd, &pf)
	return cmd
}

// newPostsCreateURLResourcePostCmd is a factory for post types backed by a single URL-based resource.
func newPostsCreateURLResourcePostCmd(use, short, postType, resourceType, urlField string) *cobra.Command {
	var name, urlStr string
	var pf publishOpts

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			resourceData := map[string]any{urlField: urlStr}
			if name != "" {
				resourceData["title"] = name
			}
			return createResourceBackedPost(a, postType, resourceType, resourceData, name, pf)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name")
	cmd.Flags().StringVar(&urlStr, "url", "", "URL (required)")
	addPublishFlags(cmd, &pf)
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func newPostsCreateFeedCmd() *cobra.Command {
	return newPostsCreateURLResourcePostCmd(
		"feed",
		"Create a feed post from an RSS/Atom URL",
		"feed", "feed", "url",
	)
}

func newPostsCreateCalendarCmd() *cobra.Command {
	return newPostsCreateURLResourcePostCmd(
		"calendar",
		"Create a calendar post from an iCal/WebCal URL",
		"calendar", "calendar", "url",
	)
}

func newPostsCreateIframeCmd() *cobra.Command {
	var name, urlStr, htmlContent, imageURL string
	var pf publishOpts

	cmd := &cobra.Command{
		Use:   "iframe",
		Short: "Create an iframe post from a URL, HTML content, or image URL",
		Example: `  pintomind posts create iframe --name "Dashboard" --url https://dashboard.example.com --channel-id 7
  pintomind posts create iframe --name "Announcement" --html "<h1>Hello</h1>" --channel-id 7
  pintomind posts create iframe --name "Banner" --image https://example.com/banner.png --channel-id 7`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			changed := 0
			for _, f := range []string{"url", "html", "image"} {
				if cmd.Flags().Changed(f) {
					changed++
				}
			}
			if changed == 0 {
				return fmt.Errorf("provide one of --url, --html, or --image")
			}
			if changed > 1 {
				return fmt.Errorf("only one of --url, --html, or --image may be specified")
			}

			resourceData := map[string]any{}
			if name != "" {
				resourceData["title"] = name
			}
			var resourceType string
			switch {
			case cmd.Flags().Changed("url"):
				resourceType = "external_webpage"
				resourceData["url"] = urlStr
			case cmd.Flags().Changed("html"):
				resourceType = "html"
				resourceData["html_code"] = htmlContent
			case cmd.Flags().Changed("image"):
				resourceType = "external_image"
				resourceData["url"] = imageURL
			}

			return createResourceBackedPost(a, "iframe", resourceType, resourceData, name, pf)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name")
	cmd.Flags().StringVar(&urlStr, "url", "", "Webpage URL to embed (https only)")
	cmd.Flags().StringVar(&htmlContent, "html", "", "HTML content to display")
	cmd.Flags().StringVar(&imageURL, "image", "", "Image URL to display (https only)")
	addPublishFlags(cmd, &pf)
	return cmd
}

func newPostsCreateHTMLCmd() *cobra.Command {
	var name, htmlContent string
	var pf publishOpts

	cmd := &cobra.Command{
		Use:     "html",
		Short:   "Create an iframe post displaying HTML content",
		Example: `  pintomind posts create html --name "Announcement" --html "<h1>Hello world</h1>" --channel-id 7`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			resourceData := map[string]any{"html_code": htmlContent}
			if name != "" {
				resourceData["title"] = name
			}
			return createResourceBackedPost(a, "iframe", "html", resourceData, name, pf)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name")
	cmd.Flags().StringVar(&htmlContent, "html", "", "HTML content to display (required)")
	addPublishFlags(cmd, &pf)
	_ = cmd.MarkFlagRequired("html")
	return cmd
}

func newPostsCreatePosterCmd() *cobra.Command {
	var name, sourceID string
	var colorPaletteID int
	var pf publishOpts

	cmd := &cobra.Command{
		Use:   "poster",
		Short: "Create a poster post, optionally from a network template",
		Example: `  pintomind posts create poster --name "Spring campaign" --source-id 42 --channel-id 7
  pintomind posts create poster --name "Brand poster" --source-id 42 --color-palette-id 5 --channel-id 7`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			post := map[string]any{}
			if name != "" {
				post["name"] = name
			}

			body := map[string]any{"type": "poster", "post": post}
			if sourceID != "" {
				body["source_id"] = sourceID
			}

			var postResp map[string]any
			if err := a.Client.Post("/posts", body, &postResp); err != nil {
				return err
			}

			postID, err := intFromNestedResp(postResp, "post", "id")
			if err != nil {
				return err
			}
			if !a.JSONOutput {
				fmt.Printf("Created poster post %d\n", postID)
			}

			if cmd.Flags().Changed("color-palette-id") {
				resourceID, err := resourceIDFromPostResp(postResp)
				if err != nil {
					return fmt.Errorf("applying color palette: %w", err)
				}
				var resourceResp map[string]any
				if err := a.Client.Patch("/resources/"+strconv.Itoa(resourceID), map[string]any{
					"resource": map[string]any{"color_palette_id": colorPaletteID},
				}, &resourceResp); err != nil {
					return fmt.Errorf("applying color palette to poster resource %d: %w", resourceID, err)
				}
				if !a.JSONOutput {
					fmt.Printf("Applied color palette %d to poster resource %d\n", colorPaletteID, resourceID)
				}
			}

			if err := publishToChannels(a, postID, pf.channelIDs, pf.area, pf.fullScreen); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(postResp)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name")
	cmd.Flags().StringVar(&sourceID, "source-id", "", "Network poster template ID to duplicate")
	cmd.Flags().IntVar(&colorPaletteID, "color-palette-id", 0, "Color palette ID to apply to the poster")
	addPublishFlags(cmd, &pf)
	return cmd
}

func resourceIDFromPostResp(resp map[string]any) (int, error) {
	post, ok := resp["post"].(map[string]any)
	if !ok {
		return 0, fmt.Errorf("unexpected response: missing post object")
	}
	ids, ok := post["resource_ids"].([]any)
	if !ok || len(ids) == 0 {
		return 0, fmt.Errorf("poster post has no resource_ids in response")
	}
	f, ok := ids[0].(float64)
	if !ok {
		return 0, fmt.Errorf("resource_ids[0] is not numeric")
	}
	return int(f), nil
}
