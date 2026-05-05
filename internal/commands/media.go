package commands

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

type MediaCollection struct {
	ID                int    `json:"id"`
	Title             string `json:"title"`
	Category          string `json:"category"`
	DefaultCollection bool   `json:"default_collection"`
	MediaType         string `json:"media_type"`
	Sort              int    `json:"sort"`
}

type MediaCollectionsResponse struct {
	Total int               `json:"total"`
	Items []MediaCollection `json:"items"`
}

type Media struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Filename          string `json:"filename"`
	ContentType       string `json:"content_type"`
	ByteSize          int64  `json:"byte_size"`
	MediaType         string `json:"media_type"`
	Status            string `json:"status"`
	MediaCollectionID int    `json:"media_collection_id"`
}

type MediaResponse struct {
	Total int     `json:"total"`
	Items []Media `json:"items"`
}

type directUploadResponse struct {
	SignedID     string `json:"signed_id"`
	DirectUpload struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	} `json:"direct_upload"`
}

func NewMediaCollectionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "media-collections",
		Aliases: []string{"collections"},
		Short:   "Manage media collections",
	}
	cmd.AddCommand(newMediaCollectionsListCmd())
	cmd.AddCommand(newMediaCollectionsShowCmd())
	cmd.AddCommand(newMediaCollectionsCreateCmd())
	cmd.AddCommand(newMediaCollectionsUpdateCmd())
	cmd.AddCommand(newMediaCollectionsDeleteCmd())
	return cmd
}

func NewMediaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media",
		Short: "Manage media and upload files",
	}
	cmd.AddCommand(newMediaListCmd())
	cmd.AddCommand(newMediaUploadCmd())
	cmd.AddCommand(newMediaCreateCmd())
	cmd.AddCommand(newMediaShowCmd())
	cmd.AddCommand(newMediaUpdateCmd())
	cmd.AddCommand(newMediaDeleteCmd())
	return cmd
}

func newMediaCollectionsListCmd() *cobra.Command {
	var fields string
	var sortBy string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List media collections",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			q := url.Values{}
			if fields != "" {
				q.Set("fields", fields)
			}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			applyPagination(cmd, q)

			var resp MediaCollectionsResponse
			if err := a.Client.Get("/media_collections", q, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, c := range resp.Items {
				defaultCollection := "no"
				if c.DefaultCollection {
					defaultCollection = "yes"
				}
				rows[i] = []string{
					strconv.Itoa(c.ID),
					c.Title,
					c.Category,
					c.MediaType,
					defaultCollection,
				}
			}
			printTable(cmd, []string{"ID", "TITLE", "CATEGORY", "TYPE", "DEFAULT"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to include")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field (e.g. title, title:desc)")
	addPaginationFlags(cmd)
	return cmd
}

func newMediaCollectionsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a media collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/media_collections/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newMediaCollectionsCreateCmd() *cobra.Command {
	var title, category, icon string
	var sort int

	cmd := &cobra.Command{
		Use:   "create --title <title> --category <category>",
		Short: "Create a media collection",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			mediaCollection := map[string]any{
				"title":    title,
				"category": category,
			}
			if icon != "" {
				mediaCollection["icon"] = icon
			}
			if cmd.Flags().Changed("sort") {
				mediaCollection["sort"] = sort
			}

			var resp map[string]any
			if err := a.Client.Post("/media_collections", map[string]any{"media_collection": mediaCollection}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Collection title (required)")
	cmd.Flags().StringVar(&category, "category", "", "Collection category: background, logo, image, video, or document (required)")
	cmd.Flags().StringVar(&icon, "icon", "", "Collection icon")
	cmd.Flags().IntVar(&sort, "sort", 0, "Sort position")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("category")
	return cmd
}

func newMediaCollectionsUpdateCmd() *cobra.Command {
	var title, category, icon string
	var sort int

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a media collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			mediaCollection := map[string]any{}
			if cmd.Flags().Changed("title") {
				mediaCollection["title"] = title
			}
			if cmd.Flags().Changed("category") {
				mediaCollection["category"] = category
			}
			if cmd.Flags().Changed("icon") {
				mediaCollection["icon"] = icon
			}
			if cmd.Flags().Changed("sort") {
				mediaCollection["sort"] = sort
			}
			if len(mediaCollection) == 0 {
				return fmt.Errorf("provide at least one field to update")
			}

			var resp map[string]any
			if err := a.Client.Patch("/media_collections/"+args[0], map[string]any{"media_collection": mediaCollection}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Collection title")
	cmd.Flags().StringVar(&category, "category", "", "Collection category: background, logo, image, video, or document")
	cmd.Flags().StringVar(&icon, "icon", "", "Collection icon")
	cmd.Flags().IntVar(&sort, "sort", 0, "Sort position")
	return cmd
}

func newMediaCollectionsDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a media collection (soft-delete; second call hard-deletes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Delete media collection %s? This soft-deletes (run again to hard-delete). Pass --force to skip this prompt.\n", args[0])
				var confirm string
				fmt.Print("Type 'yes' to confirm: ")
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/media_collections/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted media collection %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newMediaListCmd() *cobra.Command {
	var fields string
	var sortBy string

	cmd := &cobra.Command{
		Use:   "list <collection-id>",
		Short: "List media in a collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			q := url.Values{}
			if fields != "" {
				q.Set("fields", fields)
			}
			if sortBy != "" {
				q.Set("sort_by", sortBy)
			}
			applyPagination(cmd, q)

			var resp MediaResponse
			if err := a.Client.Get("/media_collections/"+args[0]+"/media", q, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, m := range resp.Items {
				rows[i] = []string{
					strconv.Itoa(m.ID),
					firstNonEmpty(m.Name, m.Filename),
					m.MediaType,
					m.Status,
					formatBytes(m.ByteSize),
				}
			}
			printTable(cmd, []string{"ID", "NAME", "TYPE", "STATUS", "SIZE"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to include")
	cmd.Flags().StringVar(&sortBy, "sort-by", "", "Sort field (e.g. name, created_at:desc)")
	addPaginationFlags(cmd)
	return cmd
}

func newMediaUploadCmd() *cobra.Command {
	var name, description, contentType string
	var extractPages, wait bool

	cmd := &cobra.Command{
		Use:   "upload <collection-id> <file-or-url>",
		Short: "Upload a file or URL to a media collection",
		Example: `  pintomind media upload 42 ./cat.jpg --name "Cat photo"
  pintomind media upload 42 https://example.com/cat.jpg --name "Cat photo"
  pintomind media upload 42 ./deck.pdf --extract-pages --wait`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			collectionID := args[0]
			input := args[1]

			if !a.JSONOutput && isHTTPURL(input) {
				fmt.Fprintf(os.Stderr, "Downloading %s...\n", input)
			}
			source, err := prepareUploadSource(input, contentType, a.Client.HTTPClient)
			if err != nil {
				return err
			}
			defer source.Close()

			if !a.JSONOutput {
				fmt.Fprintf(os.Stderr, "Creating upload for %s (%s)...\n", source.metadata.filename, formatBytes(source.metadata.byteSize))
			}

			var upload directUploadResponse
			body := map[string]any{
				"blob": map[string]any{
					"filename":     source.metadata.filename,
					"content_type": source.metadata.contentType,
					"byte_size":    source.metadata.byteSize,
					"checksum":     source.metadata.checksum,
				},
			}
			if err := a.Client.Post("/direct_uploads", body, &upload); err != nil {
				return err
			}
			if upload.SignedID == "" || upload.DirectUpload.URL == "" {
				return fmt.Errorf("direct upload response did not include signed_id and upload URL")
			}

			file, err := os.Open(source.path)
			if err != nil {
				return err
			}
			defer file.Close()

			if !a.JSONOutput {
				fmt.Fprintf(os.Stderr, "Uploading bytes...\n")
			}
			if err := a.Client.PutDirectUpload(upload.DirectUpload.URL, upload.DirectUpload.Headers, file, source.metadata.byteSize); err != nil {
				return err
			}

			mediaAttrs := map[string]any{"source": upload.SignedID}
			if name != "" {
				mediaAttrs["name"] = name
			}
			if description != "" {
				mediaAttrs["description"] = description
			}
			claimBody := map[string]any{"media": mediaAttrs}
			if extractPages {
				claimBody["extract_pages"] = true
			}

			var taskResp TaskResponse
			if err := a.Client.Post("/media_collections/"+collectionID+"/media", claimBody, &taskResp); err != nil {
				return err
			}

			if !wait {
				printJSON(taskResp)
				return nil
			}

			if !a.JSONOutput {
				fmt.Fprintf(os.Stderr, "Processing task %d...\n", taskResp.Task.ID)
			}
			mediaIDs, err := waitForTask(a, taskResp.Task.ID)
			if err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(map[string]any{"media_ids": mediaIDs})
			} else {
				fmt.Printf("Uploaded. Media IDs: %v\n", mediaIDs)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Media name")
	cmd.Flags().StringVar(&description, "description", "", "Media description")
	cmd.Flags().StringVar(&contentType, "content-type", "", "Override detected MIME content type")
	cmd.Flags().BoolVar(&extractPages, "extract-pages", false, "Extract pages from an uploaded PDF")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for processing to complete and print media IDs")
	return cmd
}

func newMediaCreateCmd() *cobra.Command {
	var source, name, description string
	var extractPages, wait bool

	cmd := &cobra.Command{
		Use:   "create <collection-id> --source <signed-id>",
		Short: "Create a media record from an existing direct-upload signed ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			media := map[string]any{"source": source}
			if name != "" {
				media["name"] = name
			}
			if description != "" {
				media["description"] = description
			}
			body := map[string]any{"media": media}
			if extractPages {
				body["extract_pages"] = true
			}

			var taskResp TaskResponse
			if err := a.Client.Post("/media_collections/"+args[0]+"/media", body, &taskResp); err != nil {
				return err
			}

			if !wait {
				printJSON(taskResp)
				return nil
			}

			if !a.JSONOutput {
				fmt.Fprintf(os.Stderr, "Processing task %d...\n", taskResp.Task.ID)
			}
			mediaIDs, err := waitForTask(a, taskResp.Task.ID)
			if err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(map[string]any{"media_ids": mediaIDs})
			} else {
				fmt.Printf("Created. Media IDs: %v\n", mediaIDs)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "Active Storage signed_id from POST /direct_uploads (required)")
	cmd.Flags().StringVar(&name, "name", "", "Media name")
	cmd.Flags().StringVar(&description, "description", "", "Media description")
	cmd.Flags().BoolVar(&extractPages, "extract-pages", false, "Extract pages from an uploaded PDF")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for processing to complete and print media IDs")
	_ = cmd.MarkFlagRequired("source")
	return cmd
}

func newMediaShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a media item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp map[string]any
			if err := a.Client.Get("/media/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newMediaUpdateCmd() *cobra.Command {
	var name, description string
	var mediaCollectionID int

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a media item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			media := map[string]any{}
			if cmd.Flags().Changed("name") {
				media["name"] = name
			}
			if cmd.Flags().Changed("description") {
				media["description"] = description
			}
			if cmd.Flags().Changed("collection-id") {
				media["media_collection_id"] = mediaCollectionID
			}
			if len(media) == 0 {
				return fmt.Errorf("provide at least one field to update")
			}

			var resp map[string]any
			if err := a.Client.Patch("/media/"+args[0], map[string]any{"media": media}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Media name")
	cmd.Flags().StringVar(&description, "description", "", "Media description")
	cmd.Flags().IntVar(&mediaCollectionID, "collection-id", 0, "Move media to another collection ID")
	return cmd
}

func newMediaDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a media item (soft-delete; second call hard-deletes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Delete media %s? This soft-deletes (run again to hard-delete). Pass --force to skip this prompt.\n", args[0])
				var confirm string
				fmt.Print("Type 'yes' to confirm: ")
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/media/" + args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted media %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

type uploadMetadata struct {
	filename    string
	contentType string
	byteSize    int64
	checksum    string
}

type uploadSource struct {
	path     string
	metadata uploadMetadata
	cleanup  func() error
}

func (s uploadSource) Close() error {
	if s.cleanup == nil {
		return nil
	}
	return s.cleanup()
}

func prepareUploadSource(input, contentTypeOverride string, httpClient *http.Client) (uploadSource, error) {
	if isHTTPURL(input) {
		return downloadUploadSource(input, contentTypeOverride, httpClient)
	}

	metadata, err := fileUploadMetadataFromPath(input, filepath.Base(input), "", contentTypeOverride)
	if err != nil {
		return uploadSource{}, err
	}
	return uploadSource{path: input, metadata: metadata}, nil
}

func downloadUploadSource(rawURL, contentTypeOverride string, httpClient *http.Client) (uploadSource, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return uploadSource{}, err
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return uploadSource{}, err
	}

	downloadClient := http.DefaultClient
	if httpClient != nil {
		clientCopy := *httpClient
		clientCopy.Timeout = 0
		downloadClient = &clientCopy
	}

	resp, err := downloadClient.Do(req)
	if err != nil {
		return uploadSource{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return uploadSource{}, fmt.Errorf("downloading %s failed: HTTP %d", rawURL, resp.StatusCode)
	}

	tempFile, err := os.CreateTemp("", "pintomind-upload-*")
	if err != nil {
		return uploadSource{}, err
	}
	tempPath := tempFile.Name()
	cleanup := func() error {
		return os.Remove(tempPath)
	}

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		tempFile.Close()
		cleanup()
		return uploadSource{}, fmt.Errorf("downloading %s: %w", rawURL, err)
	}
	if err := tempFile.Close(); err != nil {
		cleanup()
		return uploadSource{}, err
	}

	filename := filenameFromResponse(resp, parsed)
	metadata, err := fileUploadMetadataFromPath(tempPath, filename, resp.Header.Get("Content-Type"), contentTypeOverride)
	if err != nil {
		cleanup()
		return uploadSource{}, err
	}

	return uploadSource{
		path:     tempPath,
		metadata: metadata,
		cleanup:  cleanup,
	}, nil
}

func isHTTPURL(input string) bool {
	u, err := url.Parse(input)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func filenameFromResponse(resp *http.Response, parsed *url.URL) string {
	if disposition := resp.Header.Get("Content-Disposition"); disposition != "" {
		if _, params, err := mime.ParseMediaType(disposition); err == nil {
			if filename := filepath.Base(params["filename"]); filename != "." && filename != string(filepath.Separator) && filename != "" {
				return filename
			}
			if filename := filepath.Base(params["filename*"]); filename != "." && filename != string(filepath.Separator) && filename != "" {
				return filename
			}
		}
	}

	if parsed != nil {
		if filename := filepath.Base(parsed.EscapedPath()); filename != "." && filename != string(filepath.Separator) && filename != "" {
			if unescaped, err := url.PathUnescape(filename); err == nil && unescaped != "" {
				return unescaped
			}
			return filename
		}
	}

	return "download"
}

func fileUploadMetadata(path, contentTypeOverride string) (uploadMetadata, error) {
	return fileUploadMetadataFromPath(path, filepath.Base(path), "", contentTypeOverride)
}

func fileUploadMetadataFromPath(path, filename, contentTypeHint, contentTypeOverride string) (uploadMetadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return uploadMetadata{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return uploadMetadata{}, err
	}
	if info.IsDir() {
		return uploadMetadata{}, fmt.Errorf("%s is a directory", path)
	}

	hash := md5.New()
	header := make([]byte, 512)
	n, readErr := io.ReadFull(file, header)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		return uploadMetadata{}, readErr
	}
	if n > 0 {
		if _, err := hash.Write(header[:n]); err != nil {
			return uploadMetadata{}, err
		}
	}
	if _, err := io.Copy(hash, file); err != nil {
		return uploadMetadata{}, err
	}

	contentType := contentTypeOverride
	if contentType == "" {
		contentType = cleanContentType(contentTypeHint)
	}
	if contentType == "" {
		contentType = cleanContentType(mime.TypeByExtension(filepath.Ext(filename)))
	}
	if contentType == "" {
		contentType = http.DetectContentType(header[:n])
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return uploadMetadata{
		filename:    filename,
		contentType: contentType,
		byteSize:    info.Size(),
		checksum:    base64.StdEncoding.EncodeToString(hash.Sum(nil)),
	}, nil
}

func cleanContentType(contentType string) string {
	if contentType == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return mediaType
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}
