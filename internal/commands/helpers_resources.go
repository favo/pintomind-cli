package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newResourcesCreateURLCmd is a factory for URL-based resource types (feed, calendar, external_webpage, external_image, qr_code).
func newResourcesCreateURLCmd(use, short, resourceType string, extraFlags func(cmd *cobra.Command, data map[string]any)) *cobra.Command {
	var title, urlStr string

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			data := map[string]any{"url": urlStr}
			if title != "" {
				data["title"] = title
			}
			if extraFlags != nil {
				extraFlags(cmd, data)
			}
			var resp map[string]any
			if err := a.Client.Post("/resources", map[string]any{
				"type":     resourceType,
				"resource": data,
			}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Resource title")
	cmd.Flags().StringVar(&urlStr, "url", "", "URL (required)")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func newResourcesCreateTextCmd() *cobra.Command {
	var label, text string

	cmd := &cobra.Command{
		Use:   "text",
		Short: "Create a text resource",
		Example: `  pintomind resources create text --label "Greeting" --text "Hello world"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			data := map[string]any{"label": label}
			if cmd.Flags().Changed("text") {
				data["text"] = text
			}
			var resp map[string]any
			if err := a.Client.Post("/resources", map[string]any{
				"type":     "text",
				"resource": data,
			}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&label, "label", "", "Resource label (required)")
	cmd.Flags().StringVar(&text, "text", "", "Text content")
	_ = cmd.MarkFlagRequired("label")
	return cmd
}

func newResourcesCreateFeedCmd() *cobra.Command {
	return newResourcesCreateURLCmd(
		"feed",
		"Create a feed resource from an RSS/Atom URL",
		"feed",
		nil,
	)
}

func newResourcesCreateCalendarCmd() *cobra.Command {
	var color string

	cmd := newResourcesCreateURLCmd(
		"calendar",
		"Create a calendar resource from an iCal/WebCal URL",
		"calendar",
		func(c *cobra.Command, data map[string]any) {
			if c.Flags().Changed("color") {
				data["color"] = color
			}
		},
	)
	cmd.Flags().StringVar(&color, "color", "", "Event color override (hex, e.g. #FF5733)")
	return cmd
}

func newResourcesCreateExternalWebpageCmd() *cobra.Command {
	return newResourcesCreateURLCmd(
		"external-webpage",
		"Create an external webpage resource (https only)",
		"external_webpage",
		nil,
	)
}

func newResourcesCreateExternalImageCmd() *cobra.Command {
	return newResourcesCreateURLCmd(
		"external-image",
		"Create an external image resource from a URL",
		"external_image",
		nil,
	)
}

func newResourcesCreateQRCodeCmd() *cobra.Command {
	var labelText string

	cmd := newResourcesCreateURLCmd(
		"qr-code",
		"Create a QR code resource from a URL",
		"qr_code",
		func(c *cobra.Command, data map[string]any) {
			if c.Flags().Changed("label-text") {
				data["text"] = labelText
			}
		},
	)
	cmd.Flags().StringVar(&labelText, "label-text", "", "Optional label shown below the QR code")
	return cmd
}

func newResourcesCreateHTMLCmd() *cobra.Command {
	var title, htmlCode string

	cmd := &cobra.Command{
		Use:   "html",
		Short: "Create an HTML resource",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			if !cmd.Flags().Changed("html-code") && !cmd.Flags().Changed("title") {
				return fmt.Errorf("provide at least --html-code or --title")
			}
			data := map[string]any{}
			if title != "" {
				data["title"] = title
			}
			if cmd.Flags().Changed("html-code") {
				data["html_code"] = htmlCode
			}
			var resp map[string]any
			if err := a.Client.Post("/resources", map[string]any{
				"type":     "html",
				"resource": data,
			}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Resource title")
	cmd.Flags().StringVar(&htmlCode, "html-code", "", "HTML markup")
	return cmd
}

func newResourcesCreateYouTubeCmd() *cobra.Command {
	var title, youtubeURL string

	cmd := &cobra.Command{
		Use:   "youtube",
		Short: "Create a YouTube resource from a video URL",
		Example: `  pintomind resources create youtube --url "https://www.youtube.com/watch?v=dQw4w9WgXcQ"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			data := map[string]any{"youtube_url": youtubeURL}
			if title != "" {
				data["title"] = title
			}
			var resp map[string]any
			if err := a.Client.Post("/resources", map[string]any{
				"type":     "youtube",
				"resource": data,
			}, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Title override (defaults to YouTube video title)")
	cmd.Flags().StringVar(&youtubeURL, "url", "", "YouTube video URL (required)")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}
