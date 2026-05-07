package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
)

// plainMediaBoxInputs bundles all media-box input slices for a plain post.
type plainMediaBoxInputs struct {
	images      []string
	mediaIDs    []int
	gifs        []string
	unsplashIDs []string
	emojis      []string
	icons       []string
}

// interactivePublishOpts prompts for channel IDs (comma-separated), area, and full-screen.
func interactivePublishOpts(p *publishOpts) error {
	var channelIDsStr string

	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Channel IDs to publish to, comma-separated (optional)").
				Value(&channelIDsStr).
				Validate(validateOptionalIntList),
			huh.NewInput().Title("Area (optional)").Value(&p.area),
			huh.NewConfirm().Title("Publish fullscreen?").Value(&p.fullScreen),
		),
	).Run(); err != nil {
		return err
	}

	for _, part := range strings.Split(channelIDsStr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, _ := strconv.Atoi(part)
		p.channelIDs = append(p.channelIDs, id)
	}
	return nil
}

// collectImagesLoop prompts the user to add image file paths/URLs or existing media IDs in a loop.
func collectImagesLoop(images *[]string, mediaIDs *[]int) error {
	for {
		var choice string
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Add media").
					Value(&choice).
					Options(
						huh.NewOption("File path or URL", "file"),
						huh.NewOption("Existing media ID", "id"),
						huh.NewOption("Done", "done"),
					),
			),
		).Run(); err != nil {
			return err
		}
		if choice == "done" {
			break
		}
		if choice == "file" {
			var path string
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().Title("File path or URL (required)").Value(&path).Validate(huh.ValidateNotEmpty()),
				),
			).Run(); err != nil {
				return err
			}
			*images = append(*images, path)
		} else {
			var idStr string
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().Title("Media ID (required)").Value(&idStr).Validate(func(s string) error {
						if _, err := strconv.Atoi(strings.TrimSpace(s)); err != nil {
							return fmt.Errorf("must be an integer")
						}
						return nil
					}),
				),
			).Run(); err != nil {
				return err
			}
			id, _ := strconv.Atoi(strings.TrimSpace(idStr))
			*mediaIDs = append(*mediaIDs, id)
		}
	}
	return nil
}

// collectMediaBoxesLoop prompts the user to add media boxes (up to 4) for a plain post.
func collectMediaBoxesLoop(mbi *plainMediaBoxInputs) error {
	for {
		total := len(mbi.images) + len(mbi.mediaIDs) + len(mbi.gifs) + len(mbi.unsplashIDs) + len(mbi.emojis) + len(mbi.icons)
		if total >= 4 {
			break
		}

		var boxType string
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Add a media box (max 4 total)").
					Value(&boxType).
					Options(
						huh.NewOption("Image file or URL", "image"),
						huh.NewOption("Existing media ID", "media"),
						huh.NewOption("GIF (Giphy ID)", "gif"),
						huh.NewOption("Unsplash photo ID", "unsplash"),
						huh.NewOption("Emoji character", "emoji"),
						huh.NewOption("Icon (name or name:type)", "icon"),
						huh.NewOption("Done — no more boxes", "done"),
					),
			),
		).Run(); err != nil {
			return err
		}
		if boxType == "done" {
			break
		}

		titles := map[string]string{
			"image":   "File path or URL",
			"media":   "Media ID (integer)",
			"gif":     "Giphy GIF ID",
			"unsplash": "Unsplash photo ID",
			"emoji":   "Emoji character",
			"icon":    "Icon name (or name:type, e.g. rocket-launch:solid)",
		}

		var val string
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title(titles[boxType]).Value(&val).Validate(huh.ValidateNotEmpty()),
			),
		).Run(); err != nil {
			return err
		}

		switch boxType {
		case "image":
			mbi.images = append(mbi.images, val)
		case "media":
			id, _ := strconv.Atoi(strings.TrimSpace(val))
			mbi.mediaIDs = append(mbi.mediaIDs, id)
		case "gif":
			mbi.gifs = append(mbi.gifs, val)
		case "unsplash":
			mbi.unsplashIDs = append(mbi.unsplashIDs, val)
		case "emoji":
			mbi.emojis = append(mbi.emojis, val)
		case "icon":
			mbi.icons = append(mbi.icons, val)
		}
	}
	return nil
}

func interactivePostFeed(name, urlStr *string, p *publishOpts) error {
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Post name (optional)").Value(name),
			huh.NewInput().Title("Feed URL RSS/Atom (required)").Value(urlStr).Validate(huh.ValidateNotEmpty()),
		),
	).Run(); err != nil {
		return err
	}
	return interactivePublishOpts(p)
}

func interactivePostCalendar(name, urlStr *string, p *publishOpts) error {
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Post name (optional)").Value(name),
			huh.NewInput().Title("Calendar URL iCal/WebCal (required)").Value(urlStr).Validate(huh.ValidateNotEmpty()),
		),
	).Run(); err != nil {
		return err
	}
	return interactivePublishOpts(p)
}

func interactivePostHTML(name, htmlContent *string, p *publishOpts) error {
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Post name (optional)").Value(name),
			huh.NewText().Title("HTML content (required)").Value(htmlContent).Validate(huh.ValidateNotEmpty()),
		),
	).Run(); err != nil {
		return err
	}
	return interactivePublishOpts(p)
}

func interactivePostPoster(name, sourceID *string, colorPaletteID *int, p *publishOpts) error {
	var colorPaletteIDStr string
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Post name (optional)").Value(name),
			huh.NewInput().Title("Source template ID (optional)").Value(sourceID),
			huh.NewInput().
				Title("Color palette ID (optional)").
				Value(&colorPaletteIDStr).
				Validate(validateOptionalInt),
		),
	).Run(); err != nil {
		return err
	}
	if colorPaletteIDStr != "" {
		id, _ := strconv.Atoi(strings.TrimSpace(colorPaletteIDStr))
		*colorPaletteID = id
	}
	return interactivePublishOpts(p)
}

// interactivePostIframe runs a two-pass form: first selects the source type, then
// prompts for the matching field. iframeSource is set to "url", "html", or "image".
func interactivePostIframe(name, urlStr, htmlContent, imageURL, iframeSource *string, p *publishOpts) error {
	*iframeSource = "url"
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Post name (optional)").Value(name),
			huh.NewSelect[string]().
				Title("Source type").
				Value(iframeSource).
				Options(
					huh.NewOption("Webpage URL", "url"),
					huh.NewOption("HTML content", "html"),
					huh.NewOption("Image URL", "image"),
				),
		),
	).Run(); err != nil {
		return err
	}

	var field huh.Field
	switch *iframeSource {
	case "url":
		field = huh.NewInput().Title("Webpage URL (https, required)").Value(urlStr).Validate(huh.ValidateNotEmpty())
	case "html":
		field = huh.NewText().Title("HTML content (required)").Value(htmlContent).Validate(huh.ValidateNotEmpty())
	case "image":
		field = huh.NewInput().Title("Image URL (https, required)").Value(imageURL).Validate(huh.ValidateNotEmpty())
	}
	if err := huh.NewForm(huh.NewGroup(field)).Run(); err != nil {
		return err
	}

	return interactivePublishOpts(p)
}

func interactivePostImage(name *string, images *[]string, mediaIDs *[]int, duration *int, p *publishOpts) error {
	var durationStr string = strconv.Itoa(*duration)
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Post name (optional)").Value(name),
			huh.NewInput().
				Title("Seconds per slide (default 7)").
				Value(&durationStr).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return nil
					}
					if _, err := strconv.Atoi(strings.TrimSpace(s)); err != nil {
						return fmt.Errorf("must be an integer")
					}
					return nil
				}),
		),
	).Run(); err != nil {
		return err
	}
	if v, err := strconv.Atoi(strings.TrimSpace(durationStr)); err == nil {
		*duration = v
	}

	if err := collectImagesLoop(images, mediaIDs); err != nil {
		return err
	}
	return interactivePublishOpts(p)
}

func interactivePostPlain(
	name, heading, body, headingAlignment, bodyAlignment *string,
	headingFontsize, bodyFontsize *int,
	mbi *plainMediaBoxInputs,
	p *publishOpts,
) error {
	var headingFontsizeStr = strconv.Itoa(*headingFontsize)
	var bodyFontsizeStr = strconv.Itoa(*bodyFontsize)

	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Post name (optional)").Value(name),
			huh.NewText().Title("Heading HTML, at least one of heading/body required (optional)").Value(heading),
			huh.NewText().Title("Body HTML, at least one of heading/body required (optional)").Value(body),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Heading alignment (optional)").
				Value(headingAlignment).
				Options(
					huh.NewOption("(none)", ""),
					huh.NewOption("Left", "left"),
					huh.NewOption("Center", "center"),
					huh.NewOption("Right", "right"),
				),
			huh.NewInput().
				Title("Heading font size % 50–400 (optional, default 100)").
				Value(&headingFontsizeStr).
				Validate(validateOptionalFontsize),
			huh.NewSelect[string]().
				Title("Body alignment (optional)").
				Value(bodyAlignment).
				Options(
					huh.NewOption("(none)", ""),
					huh.NewOption("Left", "left"),
					huh.NewOption("Center", "center"),
					huh.NewOption("Right", "right"),
				),
			huh.NewInput().
				Title("Body font size % 50–400 (optional, default 100)").
				Value(&bodyFontsizeStr).
				Validate(validateOptionalFontsize),
		),
	).Run(); err != nil {
		return err
	}

	if v, err := strconv.Atoi(strings.TrimSpace(headingFontsizeStr)); err == nil {
		*headingFontsize = v
	}
	if v, err := strconv.Atoi(strings.TrimSpace(bodyFontsizeStr)); err == nil {
		*bodyFontsize = v
	}

	var addBoxes bool
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title("Add media boxes?").Value(&addBoxes),
		),
	).Run(); err != nil {
		return err
	}
	if addBoxes {
		if err := collectMediaBoxesLoop(mbi); err != nil {
			return err
		}
	}

	return interactivePublishOpts(p)
}

// --- validators used only by posts ---

func validateOptionalIntList(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, err := strconv.Atoi(part); err != nil {
			return fmt.Errorf("%q is not a valid integer", part)
		}
	}
	return nil
}

func validateOptionalInt(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	if _, err := strconv.Atoi(strings.TrimSpace(s)); err != nil {
		return fmt.Errorf("must be an integer")
	}
	return nil
}

func validateOptionalFontsize(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fmt.Errorf("must be an integer")
	}
	if v < 50 || v > 400 {
		return fmt.Errorf("must be between 50 and 400")
	}
	return nil
}
