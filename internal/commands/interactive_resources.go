package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

// isAborted reports whether the form was cancelled by the user (Ctrl+C / Esc).
func isAborted(err error) bool {
	return errors.Is(err, huh.ErrUserAborted)
}

func interactiveResourceText(label, text *string) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Label (required)").Value(label).Validate(huh.ValidateNotEmpty()),
			huh.NewText().Title("Text content (optional)").Value(text),
		),
	).Run()
}

// interactiveResourceURL prompts for URL + title and optionally extra fields in a second group.
func interactiveResourceURL(urlStr, title *string, extraFields []huh.Field) error {
	group1 := huh.NewGroup(
		huh.NewInput().Title("URL (required)").Value(urlStr).Validate(huh.ValidateNotEmpty()),
		huh.NewInput().Title("Title (optional)").Value(title),
	)
	groups := []*huh.Group{group1}
	if len(extraFields) > 0 {
		groups = append(groups, huh.NewGroup(extraFields...))
	}
	return huh.NewForm(groups...).Run()
}

func interactiveResourceCalendar(urlStr, title, color *string) error {
	return interactiveResourceURL(urlStr, title, []huh.Field{
		huh.NewInput().Title("Color override (optional, hex e.g. #FF5733)").Value(color),
	})
}

func interactiveResourceQRCode(urlStr, title, labelText *string) error {
	return interactiveResourceURL(urlStr, title, []huh.Field{
		huh.NewInput().Title("Label text below QR code (optional)").Value(labelText),
	})
}

func interactiveResourceHTML(title, htmlCode *string) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Title (optional)").Value(title),
			huh.NewText().Title("HTML code (optional, but at least one of title/html-code required)").Value(htmlCode),
		),
	).Run()
}

func interactiveResourceYouTube(urlStr, title *string) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("YouTube URL (required)").Value(urlStr).Validate(huh.ValidateNotEmpty()),
			huh.NewInput().Title("Title override (optional)").Value(title),
		),
	).Run()
}

func interactiveResourceLocation(
	name, countryCode, timezone, country, adminArea1, adminArea2 *string,
	lat, lon, altitude *float64,
) error {
	var latStr, lonStr, altStr string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Location name (required)").Value(name).Validate(huh.ValidateNotEmpty()),
			huh.NewInput().Title("Latitude -90 to 90 (required)").Value(&latStr).Validate(validateFloat64Range(-90, 90)),
			huh.NewInput().Title("Longitude -180 to 180 (required)").Value(&lonStr).Validate(validateFloat64Range(-180, 180)),
			huh.NewInput().Title("Country code ISO 3166-1 alpha-2, e.g. NO (required)").Value(countryCode).Validate(huh.ValidateNotEmpty()),
			huh.NewInput().Title("IANA timezone, e.g. Europe/Oslo (required)").Value(timezone).Validate(huh.ValidateNotEmpty()),
		),
		huh.NewGroup(
			huh.NewInput().Title("Country name (optional)").Value(country),
			huh.NewInput().Title("Administrative area level 1 (optional)").Value(adminArea1),
			huh.NewInput().Title("Administrative area level 2 (optional)").Value(adminArea2),
			huh.NewInput().Title("Altitude in metres, -1000 to 10000 (optional)").Value(&altStr).Validate(validateOptionalFloat64Range(-1000, 10000)),
		),
	).Run()
	if err != nil {
		return err
	}

	if f, err := strconv.ParseFloat(strings.TrimSpace(latStr), 64); err == nil {
		*lat = f
	}
	if f, err := strconv.ParseFloat(strings.TrimSpace(lonStr), 64); err == nil {
		*lon = f
	}
	if altStr != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(altStr), 64); err == nil {
			*altitude = f
		}
	}
	return nil
}

func interactiveResourceCalendarEvents(title *string) ([]map[string]any, error) {
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Resource title (optional)").Value(title),
		),
	).Run(); err != nil {
		return nil, err
	}

	var items []map[string]any
	for {
		var startAt, endAt, description, summary, location, timeZone string
		var allDay bool

		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("start_at (ISO 8601, e.g. 2026-06-01T09:00:00Z) (required)").Value(&startAt).Validate(validateISO8601),
				huh.NewInput().Title("end_at (ISO 8601, e.g. 2026-06-01T10:00:00Z) (required)").Value(&endAt).Validate(validateISO8601),
				huh.NewInput().Title("Summary (required)").Value(&summary).Validate(huh.ValidateNotEmpty()),
			),
			huh.NewGroup(
				huh.NewInput().Title("Description (optional)").Value(&description),
				huh.NewInput().Title("Location (optional)").Value(&location),
				huh.NewInput().Title("Timezone IANA, e.g. Europe/Oslo (optional)").Value(&timeZone),
				huh.NewConfirm().Title("All day?").Value(&allDay),
			),
		).Run(); err != nil {
			return nil, err
		}

		item := map[string]any{
			"start_at": startAt,
			"end_at":   endAt,
			"summary":  summary,
		}
		if description != "" {
			item["description"] = description
		}
		if location != "" {
			item["location"] = location
		}
		if timeZone != "" {
			item["time_zone"] = timeZone
		}
		if allDay {
			item["all_day"] = true
		}
		items = append(items, item)

		var more bool
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().Title("Add another calendar event?").Value(&more),
			),
		).Run(); err != nil {
			return nil, err
		}
		if !more {
			break
		}
	}
	return items, nil
}

func interactiveResourceFeedItems(title *string) ([]map[string]any, error) {
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Resource title (optional)").Value(title),
		),
	).Run(); err != nil {
		return nil, err
	}

	var items []map[string]any
	for {
		var itemTitle, content, urlStr, imageURL, publishedAt string

		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Title (required)").Value(&itemTitle).Validate(huh.ValidateNotEmpty()),
				huh.NewText().Title("Content (optional)").Value(&content),
			),
			huh.NewGroup(
				huh.NewInput().Title("URL, must start with https:// (optional)").Value(&urlStr).Validate(validateOptionalHTTPS),
				huh.NewInput().Title("Image URL, must start with https:// (optional)").Value(&imageURL).Validate(validateOptionalHTTPS),
				huh.NewInput().Title("Published at ISO 8601 (optional)").Value(&publishedAt).Validate(validateOptionalISO8601),
			),
		).Run(); err != nil {
			return nil, err
		}

		item := map[string]any{"title": itemTitle}
		if content != "" {
			item["content"] = content
		}
		if urlStr != "" {
			item["url"] = urlStr
		}
		if imageURL != "" {
			item["image_url"] = imageURL
		}
		if publishedAt != "" {
			item["published_at"] = publishedAt
		}
		items = append(items, item)

		var more bool
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().Title("Add another feed item?").Value(&more),
			),
		).Run(); err != nil {
			return nil, err
		}
		if !more {
			break
		}
	}
	return items, nil
}

// --- validators ---

func validateISO8601(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("required")
	}
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		return fmt.Errorf("must be ISO 8601 date-time (e.g. 2026-06-01T09:00:00Z)")
	}
	return nil
}

func validateOptionalISO8601(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		return fmt.Errorf("must be ISO 8601 date-time (e.g. 2026-06-01T09:00:00Z)")
	}
	return nil
}

func validateOptionalHTTPS(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	if !strings.HasPrefix(s, "https://") {
		return fmt.Errorf("must start with https://")
	}
	return nil
}

func validateFloat64Range(min, max float64) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("required")
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return fmt.Errorf("must be a number")
		}
		if f < min || f > max {
			return fmt.Errorf("must be between %v and %v", min, max)
		}
		return nil
	}
}

func validateOptionalFloat64Range(min, max float64) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return fmt.Errorf("must be a number")
		}
		if f < min || f > max {
			return fmt.Errorf("must be between %v and %v", min, max)
		}
		return nil
	}
}
