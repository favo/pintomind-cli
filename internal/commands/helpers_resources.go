package commands

import (
	"encoding/json"
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
			return createResourceAndPrint(a, resourceType, data)
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
			return createResourceAndPrint(a, "text", data)
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
			return createResourceAndPrint(a, "html", data)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Resource title")
	cmd.Flags().StringVar(&htmlCode, "html-code", "", "HTML markup")
	return cmd
}

func newResourcesCreateLocationCmd() *cobra.Command {
	var name, countryCode, timezone, country, adminArea1, adminArea2 string
	var latitude, longitude, altitude float64

	cmd := &cobra.Command{
		Use:   "location",
		Short: "Create a location resource (used for weather/forecast posts)",
		Example: `  pintomind resources create location --name "Oslo" --lat 59.9139 --lon 10.7522 --country-code NO --timezone Europe/Oslo`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			data := map[string]any{
				"name":         name,
				"latitude":     latitude,
				"longitude":    longitude,
				"country_code": countryCode,
				"timezone":     timezone,
			}
			if country != "" {
				data["country"] = country
			}
			if adminArea1 != "" {
				data["admin_area1"] = adminArea1
			}
			if adminArea2 != "" {
				data["admin_area2"] = adminArea2
			}
			if cmd.Flags().Changed("altitude") {
				data["altitude"] = altitude
			}
			return createResourceAndPrint(a, "location", data)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Location name (required)")
	cmd.Flags().Float64Var(&latitude, "lat", 0, "Latitude -90 to 90 (required)")
	cmd.Flags().Float64Var(&longitude, "lon", 0, "Longitude -180 to 180 (required)")
	cmd.Flags().StringVar(&countryCode, "country-code", "", "ISO 3166-1 alpha-2 country code, e.g. NO (required)")
	cmd.Flags().StringVar(&timezone, "timezone", "", "IANA timezone, e.g. Europe/Oslo (required)")
	cmd.Flags().StringVar(&country, "country", "", "Country name")
	cmd.Flags().StringVar(&adminArea1, "admin-area1", "", "Administrative area level 1 (e.g. state/county)")
	cmd.Flags().StringVar(&adminArea2, "admin-area2", "", "Administrative area level 2 (e.g. municipality)")
	cmd.Flags().Float64Var(&altitude, "altitude", 0, "Altitude in metres (-1000 to 10000)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("lat")
	_ = cmd.MarkFlagRequired("lon")
	_ = cmd.MarkFlagRequired("country-code")
	_ = cmd.MarkFlagRequired("timezone")
	return cmd
}

func newResourcesCreateCalendarEventsCmd() *cobra.Command {
	var title, itemsJSON string

	cmd := &cobra.Command{
		Use:   "calendar-events",
		Short: "Create a calendar events resource with manually provided items",
		Example: `  pintomind resources create calendar-events --items '[{"start_at":"2026-06-01T09:00:00Z","end_at":"2026-06-01T10:00:00Z","description":"Meeting"}]'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var items []any
			if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
				return fmt.Errorf("invalid JSON for --items: %w", err)
			}
			data := map[string]any{"items": items}
			if title != "" {
				data["title"] = title
			}
			return createResourceAndPrint(a, "calendar_events", data)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Resource title")
	cmd.Flags().StringVar(&itemsJSON, "items", "", "Calendar event items as JSON array (required).\n"+
		"Required per item: start_at (date-time), end_at (date-time), description (string).\n"+
		"Optional per item: summary, location, all_day (bool), time_zone, recurrence_rule.")
	_ = cmd.MarkFlagRequired("items")
	return cmd
}

func newResourcesCreateFeedItemsCmd() *cobra.Command {
	var title, itemsJSON string

	cmd := &cobra.Command{
		Use:   "feed-items",
		Short: "Create a feed items resource with manually provided items",
		Example: `  pintomind resources create feed-items --items '[{"title":"Breaking news","content":"Details here","url":"https://example.com"}]'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			var items []any
			if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
				return fmt.Errorf("invalid JSON for --items: %w", err)
			}
			data := map[string]any{"items": items}
			if title != "" {
				data["title"] = title
			}
			return createResourceAndPrint(a, "feed_items", data)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Resource title")
	cmd.Flags().StringVar(&itemsJSON, "items", "", "Feed items as JSON array (required).\n"+
		"Required per item: title (string).\n"+
		"Optional per item: content, url (https), image_url (https), published_at (date-time).")
	_ = cmd.MarkFlagRequired("items")
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
			return createResourceAndPrint(a, "youtube", data)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Title override (defaults to YouTube video title)")
	cmd.Flags().StringVar(&youtubeURL, "url", "", "YouTube video URL (required)")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}
