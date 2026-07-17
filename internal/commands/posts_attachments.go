package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/appctx"
)

type PostAttachment struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Text      string `json:"text"`
	Alignment string `json:"alignment"`
	URL       string `json:"url"`
	Label     string `json:"label"`
	LinkType  string `json:"link_type"`
	MediaID   *int   `json:"media_id"`
}

type PostAttachmentsResponse struct {
	Total int              `json:"total"`
	Items []PostAttachment `json:"items"`
}

var linkTypes = []string{"link", "mail", "sms", "tel", "go_channel"}

func newPostsAttachmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachments",
		Short: "Manage a post's Go attachments (text, link, document)",
		Long: `Manage attachments shown with a post in the branded Go mobile app.

Three types: text (one per post), link, and document (references media in a
document media collection). See 'pintomind schemas show post_attachments'.`,
	}
	cmd.AddCommand(newPostsAttachmentsListCmd())
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add an attachment to a post (use a type subcommand)",
	}
	addCmd.AddCommand(newPostsAttachmentsAddTextCmd())
	addCmd.AddCommand(newPostsAttachmentsAddLinkCmd())
	addCmd.AddCommand(newPostsAttachmentsAddDocumentCmd())
	cmd.AddCommand(addCmd)
	cmd.AddCommand(newPostsAttachmentsUpdateCmd())
	cmd.AddCommand(newPostsAttachmentsDeleteCmd())
	return cmd
}

func newPostsAttachmentsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <post-id>",
		Short: "List a post's attachments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp PostAttachmentsResponse
			if err := a.Client.Get("/posts/"+args[0]+"/attachments", nil, &resp); err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}

			fmt.Printf("Total: %d\n\n", resp.Total)
			rows := make([][]string, len(resp.Items))
			for i, at := range resp.Items {
				content := ""
				switch at.Type {
				case "text":
					content = at.Text
				case "link":
					content = at.URL
					if at.Label != "" {
						content = at.Label + " → " + at.URL
					}
					content += " (" + at.LinkType + ")"
				case "document":
					if at.MediaID != nil {
						content = "media " + strconv.Itoa(*at.MediaID)
					}
				}
				rows[i] = []string{strconv.Itoa(at.ID), at.Type, content}
			}
			printTable(cmd, []string{"ID", "TYPE", "CONTENT"}, rows)
			return nil
		},
	}
}

// createAttachment posts an attachment and prints the JSON response.
func createAttachment(a *appctx.App, postID, attachmentType string, attachment map[string]any) error {
	var resp map[string]any
	if err := a.Client.Post("/posts/"+postID+"/attachments", map[string]any{
		"type":       attachmentType,
		"attachment": attachment,
	}, &resp); err != nil {
		return err
	}
	printJSON(resp)
	return nil
}

func newPostsAttachmentsAddTextCmd() *cobra.Command {
	var text, alignment string
	cmd := &cobra.Command{
		Use:   "text <post-id> --text <html>",
		Short: "Add a text attachment (one per post; edit with 'update' after that)",
		Example: `  pintomind posts attachments add text 123 --text "<p>Read more in the PDF below</p>"
  pintomind posts attachments add text 123 --text "<p>Hi</p>" --alignment center`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			attachment := map[string]any{"text": text}
			if alignment != "" {
				attachment["alignment"] = alignment
			}
			return createAttachment(a, args[0], "text", attachment)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "Text content, HTML allowed (required)")
	cmd.Flags().StringVar(&alignment, "alignment", "", "Text alignment: left, center, or right")
	_ = cmd.MarkFlagRequired("text")
	return cmd
}

func newPostsAttachmentsAddLinkCmd() *cobra.Command {
	var urlStr, label, linkType string
	cmd := &cobra.Command{
		Use:   "link <post-id> --url <url>",
		Short: "Add a link attachment",
		Example: `  pintomind posts attachments add link 123 --url https://example.com --label "Sign up"
  pintomind posts attachments add link 123 --url hi@example.com --link-type mail
  pintomind posts attachments add link 123 --url +4712345678 --link-type tel`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			attachment := map[string]any{"url": urlStr, "link_type": linkType}
			if label != "" {
				attachment["label"] = label
			}
			return createAttachment(a, args[0], "link", attachment)
		},
	}
	cmd.Flags().StringVar(&urlStr, "url", "", "Link target (required): a URL for link, an email address for mail, a phone number for sms/tel")
	cmd.Flags().StringVar(&label, "label", "", "Button label shown in the Go app")
	cmd.Flags().StringVar(&linkType, "link-type", "link", "Link type: "+strings.Join(linkTypes, ", "))
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func newPostsAttachmentsAddDocumentCmd() *cobra.Command {
	var mediaID, mediaCollection int
	var file string
	cmd := &cobra.Command{
		Use:   "document <post-id> (--media <id> | --file <path-or-url>)",
		Short: "Add a document attachment from existing media or by uploading a file",
		Long: `Add a document attachment from existing media or by uploading a file.

--media must reference media in a document media collection. --file uploads to
the default document collection first (override with --media-collection).`,
		Example: `  pintomind posts attachments add document 123 --media 42
  pintomind posts attachments add document 123 --file ./menu.pdf`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if (mediaID > 0) == (file != "") {
				return fmt.Errorf("provide exactly one of --media or --file")
			}
			if mediaID > 0 {
				return createAttachment(a, args[0], "document", map[string]any{"media_id": mediaID})
			}

			colID := mediaCollection
			if colID == 0 {
				id, err := findDefaultCollection(a, "document")
				if err != nil {
					return err
				}
				colID = id
			}
			mediaIDs, err := uploadFileToCollection(a, file, strconv.Itoa(colID))
			if err != nil {
				return err
			}
			for _, id := range mediaIDs {
				if err := createAttachment(a, args[0], "document", map[string]any{"media_id": id}); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&mediaID, "media", 0, "Existing media ID from a document media collection")
	cmd.Flags().StringVar(&file, "file", "", "File path or URL to upload as document media")
	cmd.Flags().IntVar(&mediaCollection, "media-collection", 0, "Media collection ID for --file uploads (defaults to default document collection)")
	return cmd
}

func newPostsAttachmentsUpdateCmd() *cobra.Command {
	var text, alignment, urlStr, label, linkType string
	var mediaID int
	cmd := &cobra.Command{
		Use:   "update <post-id> <attachment-id> [flags]",
		Short: "Update an attachment (pass only the flags to change; must match its type)",
		Example: `  pintomind posts attachments update 123 7 --text "<p>Updated</p>"
  pintomind posts attachments update 123 8 --url https://example.com/new --label "New link"
  pintomind posts attachments update 123 9 --media 55`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			attachment := map[string]any{}
			for flag, val := range map[string]any{
				"text": text, "alignment": alignment, "url": urlStr,
				"label": label, "link-type": linkType, "media": mediaID,
			} {
				if cmd.Flags().Changed(flag) {
					key := strings.ReplaceAll(flag, "-", "_")
					if flag == "media" {
						key = "media_id"
					}
					attachment[key] = val
				}
			}
			if len(attachment) == 0 {
				return fmt.Errorf("pass at least one field flag to change (e.g. --text, --url, --media)")
			}
			var resp map[string]any
			if err := a.Client.Patch("/posts/"+args[0]+"/attachments/"+args[1], map[string]any{
				"attachment": attachment,
			}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "Text content (text attachments)")
	cmd.Flags().StringVar(&alignment, "alignment", "", "Text alignment: left, center, or right (text attachments)")
	cmd.Flags().StringVar(&urlStr, "url", "", "Link target: URL, email address, or phone number (link attachments)")
	cmd.Flags().StringVar(&label, "label", "", "Button label (link attachments)")
	cmd.Flags().StringVar(&linkType, "link-type", "", "Link type: "+strings.Join(linkTypes, ", ")+" (link attachments)")
	cmd.Flags().IntVar(&mediaID, "media", 0, "Media ID from a document media collection (document attachments)")
	return cmd
}

func newPostsAttachmentsDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <post-id> <attachment-id>",
		Short: "Remove an attachment from a post",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			if !force {
				fmt.Printf("Delete attachment %s from post %s? Type 'yes' to confirm: ", args[1], args[0])
				var confirm string
				fmt.Scanln(&confirm)
				if strings.TrimSpace(confirm) != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}
			if err := a.Client.Delete("/posts/" + args[0] + "/attachments/" + args[1]); err != nil {
				return err
			}
			fmt.Printf("Deleted attachment %s from post %s\n", args[1], args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
