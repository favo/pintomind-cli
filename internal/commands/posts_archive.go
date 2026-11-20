package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func newPostsArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "archive <id>",
		Short: "Archive a post (unpublishes it from every channel)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return postStateChange(cmd, args[0], "archive", "Archived")
		},
	}
}

func newPostsRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <id>",
		Short: "Restore an archived or soft-deleted post (it is not republished)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return postStateChange(cmd, args[0], "restore", "Restored")
		},
	}
}

func postStateChange(cmd *cobra.Command, id, action, done string) error {
	a := app(cmd)
	var resp map[string]any
	if err := a.Client.Post("/posts/"+id+"/"+action, nil, &resp); err != nil {
		return err
	}
	if a.JSONOutput {
		printJSON(resp)
		return nil
	}
	fmt.Printf("%s post %s\n", done, id)
	return nil
}

func newPostsPreviewCmd() *cobra.Command {
	var postType, data, area string
	var channelID, sourceID int

	cmd := &cobra.Command{
		Use:   "preview [id]",
		Short: "Preview a new post, or changes to a post, without saving (signed link valid for an hour)",
		Long: `With a post ID, previews that post with the --data changes applied. Without one, previews a new
post of --type built from --data (or, for a poster, the template --source-id). The link shows the
post in --channel-id (or one of the account's channels): as the Go app shows it on a Go channel,
as a screen plays it on others.`,
		Example: `  pintomind posts preview 123
  pintomind posts preview 123 --data '{"title":"New title"}' --channel-id 7 --area F11
  pintomind posts preview --type plain --data '{"heading":"<p>Hi</p>","body":"<p>There</p>"}'
  pintomind posts preview --type poster --source-id 42`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			post := map[string]any{}
			if data != "" {
				parsed, err := parseJSONFlag("data", data)
				if err != nil {
					return err
				}
				fields, ok := parsed.(map[string]any)
				if !ok {
					return errors.New("--data must be a JSON object of post fields")
				}
				post = fields
			}
			setIfChanged(cmd, body, "channel-id", "channel_id", channelID)
			setIfChanged(cmd, body, "area", "area", area)

			if len(args) == 1 {
				if postType != "" || sourceID != 0 {
					return errors.New("--type and --source-id are for new posts; leave out the post ID")
				}
				body["post"] = post
				return printPreview(cmd, "/posts/"+args[0]+"/preview", body)
			}
			if postType == "" {
				return errors.New("give a post ID, or --type for a new post")
			}
			body["type"] = postType
			body["post"] = post
			setIfChanged(cmd, body, "source-id", "source_id", sourceID)
			return printPreview(cmd, "/posts/preview", body)
		},
	}
	cmd.Flags().StringVar(&postType, "type", "", "Type of a new post: plain, image, video, youtube, iframe, calendar, clock, world_clock, counter, forecast, feed, entur, power_price, poster, scores_and_fixtures")
	cmd.Flags().StringVar(&data, "data", "", "Post fields as JSON, @file or -")
	cmd.Flags().IntVar(&channelID, "channel-id", 0, "Channel to show the post in")
	cmd.Flags().StringVar(&area, "area", "", "Area of a screen channel, or F11 for full screen (default: the post's own)")
	cmd.Flags().IntVar(&sourceID, "source-id", 0, "Poster template from 'poster-templates list' (new posters)")
	return cmd
}
