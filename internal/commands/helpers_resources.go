package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// newResourcesCreateURLCmd is a factory for URL-based resource types (feed, calendar, external_webpage, external_image, qr_code).
// interactiveFn, when non-nil, is called in interactive mode instead of the default URL+title form.
func newResourcesCreateURLCmd(
	use, short, resourceType string,
	extraFlags func(cmd *cobra.Command, data map[string]any),
	interactiveFn func(urlStr, title *string) error,
) *cobra.Command {
	var title, urlStr string
	var interactive bool

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			if interactive {
				fn := interactiveFn
				if fn == nil {
					fn = func(u, t *string) error { return interactiveResourceURL(u, t, nil) }
				}
				if err := fn(&urlStr, &title); err != nil {
					if isAborted(err) {
						return nil
					}
					return err
				}
			}

			if urlStr == "" {
				return fmt.Errorf("--url is required (or use --interactive / -i)")
			}

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
	cmd.Flags().StringVar(&urlStr, "url", "", "URL")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Fill fields interactively")
	return cmd
}

func newResourcesCreateTextCmd() *cobra.Command {
	var label, text string
	var interactive bool

	cmd := &cobra.Command{
		Use:     "text",
		Short:   "Create a text resource",
		Example: `  pintomind resources create text --label "Greeting" --text "Hello world"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			if interactive {
				if err := interactiveResourceText(&label, &text); err != nil {
					if isAborted(err) {
						return nil
					}
					return err
				}
			}

			if label == "" {
				return fmt.Errorf("--label is required (or use --interactive / -i)")
			}

			data := map[string]any{"label": label}
			if cmd.Flags().Changed("text") || text != "" {
				data["text"] = text
			}
			return createResourceAndPrint(a, "text", data)
		},
	}
	cmd.Flags().StringVar(&label, "label", "", "Resource label")
	cmd.Flags().StringVar(&text, "text", "", "Text content")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Fill fields interactively")
	return cmd
}

func newResourcesCreateFeedCmd() *cobra.Command {
	return newResourcesCreateURLCmd(
		"feed",
		"Create a feed resource from an RSS/Atom URL",
		"feed",
		nil,
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
			if color != "" {
				data["color"] = color
			}
		},
		func(urlStr, title *string) error {
			return interactiveResourceCalendar(urlStr, title, &color)
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
		nil,
	)
}

func newResourcesCreateExternalImageCmd() *cobra.Command {
	return newResourcesCreateURLCmd(
		"external-image",
		"Create an external image resource from a URL",
		"external_image",
		nil,
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
			if labelText != "" {
				data["text"] = labelText
			}
		},
		func(urlStr, title *string) error {
			return interactiveResourceQRCode(urlStr, title, &labelText)
		},
	)
	cmd.Flags().StringVar(&labelText, "label-text", "", "Optional label shown below the QR code")
	return cmd
}

func newResourcesCreateHTMLCmd() *cobra.Command {
	var title, htmlCode string
	var interactive bool

	cmd := &cobra.Command{
		Use:   "html",
		Short: "Create an HTML resource",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			if interactive {
				if err := interactiveResourceHTML(&title, &htmlCode); err != nil {
					if isAborted(err) {
						return nil
					}
					return err
				}
			}

			if title == "" && htmlCode == "" {
				return fmt.Errorf("provide at least --html-code or --title (or use --interactive / -i)")
			}
			data := map[string]any{}
			if title != "" {
				data["title"] = title
			}
			if htmlCode != "" {
				data["html_code"] = htmlCode
			}
			return createResourceAndPrint(a, "html", data)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Resource title")
	cmd.Flags().StringVar(&htmlCode, "html-code", "", "HTML markup")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Fill fields interactively")
	return cmd
}

func newResourcesCreateLocationCmd() *cobra.Command {
	var name, countryCode, timezone, country, adminArea1, adminArea2 string
	var latitude, longitude, altitude float64
	var interactive bool

	cmd := &cobra.Command{
		Use:     "location",
		Short:   "Create a location resource (used for weather/forecast posts)",
		Example: `  pintomind resources create location --name "Oslo" --lat 59.9139 --lon 10.7522 --country-code NO --timezone Europe/Oslo`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			if interactive {
				if err := interactiveResourceLocation(&name, &countryCode, &timezone, &country, &adminArea1, &adminArea2, &latitude, &longitude, &altitude); err != nil {
					if isAborted(err) {
						return nil
					}
					return err
				}
			}

			if name == "" || countryCode == "" || timezone == "" {
				return fmt.Errorf("--name, --country-code, and --timezone are required (or use --interactive / -i)")
			}

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
			if cmd.Flags().Changed("altitude") || altitude != 0 {
				data["altitude"] = altitude
			}
			return createResourceAndPrint(a, "location", data)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Location name")
	cmd.Flags().Float64Var(&latitude, "lat", 0, "Latitude -90 to 90")
	cmd.Flags().Float64Var(&longitude, "lon", 0, "Longitude -180 to 180")
	cmd.Flags().StringVar(&countryCode, "country-code", "", "ISO 3166-1 alpha-2 country code, e.g. NO")
	cmd.Flags().StringVar(&timezone, "timezone", "", "IANA timezone, e.g. Europe/Oslo")
	cmd.Flags().StringVar(&country, "country", "", "Country name")
	cmd.Flags().StringVar(&adminArea1, "admin-area1", "", "Administrative area level 1 (e.g. state/county)")
	cmd.Flags().StringVar(&adminArea2, "admin-area2", "", "Administrative area level 2 (e.g. municipality)")
	cmd.Flags().Float64Var(&altitude, "altitude", 0, "Altitude in metres (-1000 to 10000)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Fill fields interactively")
	return cmd
}

func newResourcesCreateYouTubeCmd() *cobra.Command {
	var title, youtubeURL string
	var interactive bool

	cmd := &cobra.Command{
		Use:     "youtube",
		Short:   "Create a YouTube resource from a video URL",
		Example: `  pintomind resources create youtube --url "https://www.youtube.com/watch?v=dQw4w9WgXcQ"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			if interactive {
				if err := interactiveResourceYouTube(&youtubeURL, &title); err != nil {
					if isAborted(err) {
						return nil
					}
					return err
				}
			}

			if youtubeURL == "" {
				return fmt.Errorf("--url is required (or use --interactive / -i)")
			}

			data := map[string]any{"youtube_url": youtubeURL}
			if title != "" {
				data["title"] = title
			}
			return createResourceAndPrint(a, "youtube", data)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Title override (defaults to YouTube video title)")
	cmd.Flags().StringVar(&youtubeURL, "url", "", "YouTube video URL")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Fill fields interactively")
	return cmd
}

func newResourcesCreateCalendarEventsCmd() *cobra.Command {
	var title, itemsJSON string
	var interactive bool

	cmd := &cobra.Command{
		Use:   "calendar-events",
		Short: "Create a calendar events resource with manually provided items",
		Example: `  pintomind resources create calendar-events --items '[{"start_at":"2026-06-01T09:00:00Z","end_at":"2026-06-01T10:00:00Z","description":"Meeting"}]'
  pintomind resources create calendar-events --interactive`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			if interactive && cmd.Flags().Changed("items") {
				return fmt.Errorf("--interactive and --items are mutually exclusive")
			}

			var items []any
			if interactive {
				collected, err := interactiveResourceCalendarEvents(&title)
				if err != nil {
					if isAborted(err) {
						return nil
					}
					return err
				}
				for _, item := range collected {
					items = append(items, item)
				}
			} else {
				if itemsJSON == "" {
					return fmt.Errorf("--items is required (or use --interactive / -i)")
				}
				if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
					return fmt.Errorf("invalid JSON for --items: %w", err)
				}
			}

			data := map[string]any{"items": items}
			if title != "" {
				data["title"] = title
			}
			return createResourceAndPrint(a, "calendar_events", data)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Resource title")
	cmd.Flags().StringVar(&itemsJSON, "items", "", "Calendar event items as JSON array.\n"+
		"Required per item: start_at (date-time), end_at (date-time), summary (string).\n"+
		"Optional per item: description, location, all_day (bool), time_zone, recurrence_rule.")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Fill fields interactively")
	return cmd
}

func newResourcesCreateFeedItemsCmd() *cobra.Command {
	var title, itemsJSON string
	var interactive bool

	cmd := &cobra.Command{
		Use:   "feed-items",
		Short: "Create a feed items resource with manually provided items",
		Example: `  pintomind resources create feed-items --items '[{"title":"Breaking news","content":"Details here","url":"https://example.com"}]'
  pintomind resources create feed-items --interactive`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)

			if interactive && cmd.Flags().Changed("items") {
				return fmt.Errorf("--interactive and --items are mutually exclusive")
			}

			var items []any
			if interactive {
				collected, err := interactiveResourceFeedItems(&title)
				if err != nil {
					if isAborted(err) {
						return nil
					}
					return err
				}
				for _, item := range collected {
					items = append(items, item)
				}
			} else {
				if itemsJSON == "" {
					return fmt.Errorf("--items is required (or use --interactive / -i)")
				}
				if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
					return fmt.Errorf("invalid JSON for --items: %w", err)
				}
			}

			data := map[string]any{"items": items}
			if title != "" {
				data["title"] = title
			}
			return createResourceAndPrint(a, "feed_items", data)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Resource title")
	cmd.Flags().StringVar(&itemsJSON, "items", "", "Feed items as JSON array.\n"+
		"Required per item: title (string).\n"+
		"Optional per item: content, url (https), image_url (https), published_at (date-time).")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Fill fields interactively")
	return cmd
}
