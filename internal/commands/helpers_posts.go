package commands

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/appctx"
)

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

// uploadFileToCollection uploads a local file or URL and returns the created media ID.
func uploadFileToCollection(a *appctx.App, input, collectionID string) (int, error) {
	if !a.JSONOutput && isHTTPURL(input) {
		fmt.Fprintf(os.Stderr, "Downloading %s...\n", input)
	}
	source, err := prepareUploadSource(input, "", a.Client.HTTPClient)
	if err != nil {
		return 0, err
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
		return 0, err
	}
	if upload.SignedID == "" || upload.DirectUpload.URL == "" {
		return 0, fmt.Errorf("direct upload response missing signed_id or upload URL")
	}

	file, err := os.Open(source.path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	if !a.JSONOutput {
		fmt.Fprintf(os.Stderr, "Uploading %s...\n", source.metadata.filename)
	}
	if err := a.Client.PutDirectUpload(upload.DirectUpload.URL, upload.DirectUpload.Headers, file, source.metadata.byteSize); err != nil {
		return 0, err
	}

	var mediaResp map[string]any
	if err := a.Client.Post("/media_collections/"+collectionID+"/media", map[string]any{
		"media": map[string]any{"source": upload.SignedID},
	}, &mediaResp); err != nil {
		return 0, err
	}
	return intFromNestedResp(mediaResp, "media", "id")
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
	var channelIDs []int
	var area string
	var fullScreen bool

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

			mediaID, err := uploadFileToCollection(a, input, strconv.Itoa(colID))
			if err != nil {
				return err
			}

			postName := name
			if postName == "" {
				postName = filepath.Base(input)
			}

			var postResp map[string]any
			if err := a.Client.Post("/posts", map[string]any{
				"type": "image",
				"post": map[string]any{
					"name":              postName,
					"duration_per_item": duration,
					"media_resources":   []map[string]any{{"media_id": mediaID}},
				},
			}, &postResp); err != nil {
				return err
			}

			postID, err := intFromNestedResp(postResp, "post", "id")
			if err != nil {
				return err
			}
			if !a.JSONOutput {
				fmt.Printf("Created image post %d %q\n", postID, postName)
			}

			if err := publishToChannels(a, postID, channelIDs, area, fullScreen); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(postResp)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name (defaults to filename)")
	cmd.Flags().IntVar(&collectionID, "media-collection", 0, "Media collection ID for upload (defaults to default image collection)")
	cmd.Flags().IntVar(&duration, "duration", 7, "Seconds per slide")
	cmd.Flags().IntSliceVar(&channelIDs, "channel-id", nil, "Channel to publish to (repeatable)")
	cmd.Flags().StringVar(&area, "area", "", "Channel area for publication")
	cmd.Flags().BoolVar(&fullScreen, "full-screen", false, "Publish as fullscreen")
	return cmd
}

// newPostsCreateImageCmd creates an image post, optionally uploading files first.
func newPostsCreateImageCmd() *cobra.Command {
	var name string
	var images []string
	var mediaIDs []int
	var mediaCollection int
	var duration int
	var channelIDs []int
	var area string
	var fullScreen bool

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
					mID, err := uploadFileToCollection(a, img, colIDStr)
					if err != nil {
						return err
					}
					allMediaIDs = append(allMediaIDs, mID)
				}
			}
			allMediaIDs = append(allMediaIDs, mediaIDs...)

			mediaResources := make([]map[string]any, len(allMediaIDs))
			for i, id := range allMediaIDs {
				mediaResources[i] = map[string]any{"media_id": id}
			}

			post := map[string]any{
				"duration_per_item": duration,
				"media_resources":   mediaResources,
			}
			if name != "" {
				post["name"] = name
			}

			var postResp map[string]any
			if err := a.Client.Post("/posts", map[string]any{
				"type": "image",
				"post": post,
			}, &postResp); err != nil {
				return err
			}

			postID, err := intFromNestedResp(postResp, "post", "id")
			if err != nil {
				return err
			}
			if !a.JSONOutput {
				fmt.Printf("Created image post %d\n", postID)
			}

			if err := publishToChannels(a, postID, channelIDs, area, fullScreen); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(postResp)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name")
	cmd.Flags().StringArrayVar(&images, "image", nil, "File path or URL to upload (repeatable)")
	cmd.Flags().IntSliceVar(&mediaIDs, "media", nil, "Existing media ID to include (repeatable)")
	cmd.Flags().IntVar(&mediaCollection, "media-collection", 0, "Media collection ID for uploads (defaults to default image collection)")
	cmd.Flags().IntVar(&duration, "duration", 7, "Seconds per slide")
	cmd.Flags().IntSliceVar(&channelIDs, "channel-id", nil, "Channel to publish to (repeatable)")
	cmd.Flags().StringVar(&area, "area", "", "Channel area for publication")
	cmd.Flags().BoolVar(&fullScreen, "full-screen", false, "Publish as fullscreen")
	return cmd
}

// newPostsCreatePlainCmd creates a plain text post.
func newPostsCreatePlainCmd() *cobra.Command {
	var name, heading, body, justification, fontsize string
	var channelIDs []int
	var area string
	var fullScreen bool

	cmd := &cobra.Command{
		Use:   "plain",
		Short: "Create a plain text post",
		Example: `  pintomind posts create plain --name "Welcome" --heading "<p>Hello</p>" --channel-id 7
  pintomind posts create plain --name "Msg" --heading "<p>Hi</p>" --body "<p>Content</p>" --justification center --fontsize large`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			if !cmd.Flags().Changed("heading") && !cmd.Flags().Changed("body") {
				return fmt.Errorf("provide at least one of --heading or --body")
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
			if cmd.Flags().Changed("justification") {
				post["justification"] = justification
			}
			if cmd.Flags().Changed("fontsize") {
				post["fontsize"] = fontsize
			}

			var postResp map[string]any
			if err := a.Client.Post("/posts", map[string]any{
				"type": "plain",
				"post": post,
			}, &postResp); err != nil {
				return err
			}

			postID, err := intFromNestedResp(postResp, "post", "id")
			if err != nil {
				return err
			}
			if !a.JSONOutput {
				fmt.Printf("Created plain post %d\n", postID)
			}

			if err := publishToChannels(a, postID, channelIDs, area, fullScreen); err != nil {
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
	cmd.Flags().StringVar(&justification, "justification", "", "Text alignment: left, center, or right")
	cmd.Flags().StringVar(&fontsize, "fontsize", "", "Font size: small, medium, large, or xlarge")
	cmd.Flags().IntSliceVar(&channelIDs, "channel-id", nil, "Channel to publish to (repeatable)")
	cmd.Flags().StringVar(&area, "area", "", "Channel area for publication")
	cmd.Flags().BoolVar(&fullScreen, "full-screen", false, "Publish as fullscreen")
	return cmd
}

// newPostsCreateURLResourcePostCmd is a factory for post types backed by a single URL-based resource.
func newPostsCreateURLResourcePostCmd(use, short, postType, resourceType, urlField string) *cobra.Command {
	var name, urlStr string
	var channelIDs []int
	var area string
	var fullScreen bool

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			resourceData := map[string]any{urlField: urlStr}
			if name != "" {
				resourceData["title"] = name
			}

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
			if err := a.Client.Post("/posts", map[string]any{
				"type": postType,
				"post": post,
			}, &postResp); err != nil {
				return err
			}

			postID, err := intFromNestedResp(postResp, "post", "id")
			if err != nil {
				return err
			}
			if !a.JSONOutput {
				fmt.Printf("Created %s post %d\n", postType, postID)
			}

			if err := publishToChannels(a, postID, channelIDs, area, fullScreen); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(postResp)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name")
	cmd.Flags().StringVar(&urlStr, "url", "", "URL (required)")
	cmd.Flags().IntSliceVar(&channelIDs, "channel-id", nil, "Channel to publish to (repeatable)")
	cmd.Flags().StringVar(&area, "area", "", "Channel area for publication")
	cmd.Flags().BoolVar(&fullScreen, "full-screen", false, "Publish as fullscreen")
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
	var channelIDs []int
	var area string
	var fullScreen bool

	cmd := &cobra.Command{
		Use:   "iframe",
		Short: "Create an iframe post from a URL, HTML content, or image URL",
		Example: `  pintomind posts create iframe --name "Dashboard" --url https://dashboard.example.com --channel-id 7
  pintomind posts create iframe --name "Announcement" --html "<h1>Hello</h1>" --channel-id 7
  pintomind posts create iframe --name "Banner" --image https://example.com/banner.png --channel-id 7`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			urlChanged := cmd.Flags().Changed("url")
			htmlChanged := cmd.Flags().Changed("html")
			imageChanged := cmd.Flags().Changed("image")
			count := 0
			for _, b := range []bool{urlChanged, htmlChanged, imageChanged} {
				if b {
					count++
				}
			}
			if count == 0 {
				return fmt.Errorf("provide one of --url, --html, or --image")
			}
			if count > 1 {
				return fmt.Errorf("only one of --url, --html, or --image may be specified")
			}

			var resourceType string
			resourceData := map[string]any{}
			if name != "" {
				resourceData["title"] = name
			}
			switch {
			case urlChanged:
				resourceType = "external_webpage"
				resourceData["url"] = urlStr
			case htmlChanged:
				resourceType = "html"
				resourceData["html_code"] = htmlContent
			case imageChanged:
				resourceType = "external_image"
				resourceData["url"] = imageURL
			}

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
			if err := a.Client.Post("/posts", map[string]any{
				"type": "iframe",
				"post": post,
			}, &postResp); err != nil {
				return err
			}

			postID, err := intFromNestedResp(postResp, "post", "id")
			if err != nil {
				return err
			}
			if !a.JSONOutput {
				fmt.Printf("Created iframe post %d\n", postID)
			}

			if err := publishToChannels(a, postID, channelIDs, area, fullScreen); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(postResp)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name")
	cmd.Flags().StringVar(&urlStr, "url", "", "Webpage URL to embed (https only)")
	cmd.Flags().StringVar(&htmlContent, "html", "", "HTML content to display")
	cmd.Flags().StringVar(&imageURL, "image", "", "Image URL to display (https only)")
	cmd.Flags().IntSliceVar(&channelIDs, "channel-id", nil, "Channel to publish to (repeatable)")
	cmd.Flags().StringVar(&area, "area", "", "Channel area for publication")
	cmd.Flags().BoolVar(&fullScreen, "full-screen", false, "Publish as fullscreen")
	return cmd
}

func newPostsCreateHTMLCmd() *cobra.Command {
	var name, htmlContent string
	var channelIDs []int
	var area string
	var fullScreen bool

	cmd := &cobra.Command{
		Use:   "html",
		Short: "Create an iframe post displaying HTML content",
		Example: `  pintomind posts create html --name "Announcement" --html "<h1>Hello world</h1>" --channel-id 7`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			resourceData := map[string]any{"html_code": htmlContent}
			if name != "" {
				resourceData["title"] = name
			}

			resourceID, err := createResource(a, "html", resourceData)
			if err != nil {
				return err
			}
			if !a.JSONOutput {
				fmt.Printf("Created html resource %d\n", resourceID)
			}

			post := map[string]any{"resource_ids": []int{resourceID}}
			if name != "" {
				post["name"] = name
			}

			var postResp map[string]any
			if err := a.Client.Post("/posts", map[string]any{
				"type": "iframe",
				"post": post,
			}, &postResp); err != nil {
				return err
			}

			postID, err := intFromNestedResp(postResp, "post", "id")
			if err != nil {
				return err
			}
			if !a.JSONOutput {
				fmt.Printf("Created html post %d\n", postID)
			}

			if err := publishToChannels(a, postID, channelIDs, area, fullScreen); err != nil {
				return err
			}

			if a.JSONOutput {
				printJSON(postResp)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Post name")
	cmd.Flags().StringVar(&htmlContent, "html", "", "HTML content to display (required)")
	cmd.Flags().IntSliceVar(&channelIDs, "channel-id", nil, "Channel to publish to (repeatable)")
	cmd.Flags().StringVar(&area, "area", "", "Channel area for publication")
	cmd.Flags().BoolVar(&fullScreen, "full-screen", false, "Publish as fullscreen")
	_ = cmd.MarkFlagRequired("html")
	return cmd
}

func newPostsCreatePosterCmd() *cobra.Command {
	var name, sourceID string
	var colorPaletteID int
	var channelIDs []int
	var area string
	var fullScreen bool

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
			if cmd.Flags().Changed("color-palette-id") {
				post["resources"] = []map[string]any{{"color_palette_id": colorPaletteID}}
			}

			body := map[string]any{
				"type": "poster",
				"post": post,
			}
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

			if err := publishToChannels(a, postID, channelIDs, area, fullScreen); err != nil {
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
	cmd.Flags().IntSliceVar(&channelIDs, "channel-id", nil, "Channel to publish to (repeatable)")
	cmd.Flags().StringVar(&area, "area", "", "Channel area for publication")
	cmd.Flags().BoolVar(&fullScreen, "full-screen", false, "Publish as fullscreen")
	return cmd
}
