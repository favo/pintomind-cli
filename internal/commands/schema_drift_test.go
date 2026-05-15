package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"favo/pintomind-cli/internal/api"
	"favo/pintomind-cli/internal/config"
)

// TestSchemaDriftPosts fetches every post schema from the live API and compares
// the schema's "properties" against the fields the typed `posts create *`
// subcommands send. Drift falls into two buckets:
//
//   - CLI sends a field the schema doesn't list → t.Errorf (the API would 422).
//   - Schema lists a field the CLI doesn't know about → t.Logf (informational —
//     possible new API feature to expose as a flag).
//
// The test is skipped unless credentials are available. Two ways to supply them:
//
//   - PINTOMIND_TEST_API_URL + PINTOMIND_TEST_API_KEY env vars, or
//   - the user's existing `pintomind connection add` config (optionally select
//     a non-default connection with PINTOMIND_TEST_CONNECTION).
//
// Run with:
//
//	go test ./internal/commands -run TestSchemaDrift -v
//
// Add new post types to postCLIFields below as typed subcommands are added.

// postCLIFields maps schema id → the exhaustive set of fields a typed
// `posts create <type>` subcommand can put inside the `post` object. Schemas
// not listed here have no typed subcommand and only the raw `--data` path —
// nothing to compare against, so they're excluded.
var postCLIFields = map[string][]string{
	"post_image": {"name", "duration_per_item", "images", "template", "background_id"},
	"post_plain": {"name", "heading", "body", "heading_options", "body_options", "media_box_ids", "background_id"},
	"post_feed":     {"name", "resource_ids", "background_id"},
	"post_calendar": {"name", "resource_ids", "background_id"},
	"post_iframe":   {"name", "resource_ids", "background_id"},
	"post_poster":   {"name", "background_id"},
	"post_counter": {
		"name", "title", "duration", "show_title", "background_id",
		"template",
		"countdown_type",
		"countdown_target_date",
		"countdown_target_time",
		"countdown_from_date",
		"countdown_from_time",
		"countdown_format",
		"countdown_time_zone",
		"description",
		"small_label",
		"finished_title",
		"finished_description",
		"primary_color",
		"repeatable",
		"repeat_delay",
		"repeat_delay_unit",
		"countdown_finished_media_box_id",
	},
}

// rawOnlyPostSchemas have no typed subcommand. The drift test still fetches
// them so schema_only logs surface candidates for typed-flag coverage, but
// reports no cli_only errors.
var rawOnlyPostSchemas = []string{
	"post_video",
	"post_youtube",
	"post_clock",
	"post_world_clock",
	"post_forecast",
	"post_entur",
	"post_power_price",
}

func TestSchemaDriftPosts(t *testing.T) {
	client, err := liveAPIClient()
	if err != nil {
		t.Skipf("schema drift test skipped: %v", err)
	}

	keys := make([]string, 0, len(postCLIFields)+len(rawOnlyPostSchemas))
	for k := range postCLIFields {
		keys = append(keys, k)
	}
	keys = append(keys, rawOnlyPostSchemas...)
	sort.Strings(keys)

	for _, schemaKey := range keys {
		t.Run(schemaKey, func(t *testing.T) {
			var schema map[string]any
			if err := client.Get("/schemas/"+schemaKey, nil, &schema); err != nil {
				t.Fatalf("fetch %s: %v", schemaKey, err)
			}

			schemaProps := extractSchemaProperties(schema)
			if len(schemaProps) == 0 {
				t.Fatalf("%s schema returned no properties — response shape unexpected: %v", schemaKey, schema)
			}

			cliFields, typed := postCLIFields[schemaKey]
			cliOnly := setDifference(cliFields, schemaProps)
			schemaOnly := setDifference(schemaProps, cliFields)

			if typed && len(cliOnly) > 0 {
				t.Errorf("CLI sends fields not in %s schema (would 422): %s", schemaKey, strings.Join(cliOnly, ", "))
			}
			if len(schemaOnly) > 0 {
				prefix := schemaKey
				if !typed {
					prefix += " (raw-only)"
				}
				t.Logf("%s: schema has fields the CLI doesn't expose: %s", prefix, strings.Join(schemaOnly, ", "))
			}
		})
	}
}

// liveAPIClient builds an api.Client from env vars first, then falls back to
// the user's config. Returns an error (causing t.Skip) when neither is set.
func liveAPIClient() (*api.Client, error) {
	if base, key := os.Getenv("PINTOMIND_TEST_API_URL"), os.Getenv("PINTOMIND_TEST_API_KEY"); base != "" && key != "" {
		return api.New(base, key), nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	_, d, err := cfg.ActiveDomain(os.Getenv("PINTOMIND_TEST_CONNECTION"))
	if err != nil {
		return nil, fmt.Errorf("set PINTOMIND_TEST_API_URL + PINTOMIND_TEST_API_KEY or run `pintomind connection add` first: %w", err)
	}
	return api.New(d.BaseURL, d.APIKey), nil
}

// extractSchemaProperties walks a JSON-schema-shaped map and returns the union
// of property names found at the top level and inside any allOf/oneOf/anyOf
// branches one level deep. Sufficient for current Pintomind schemas.
func extractSchemaProperties(schema map[string]any) []string {
	seen := make(map[string]struct{})
	collect := func(m map[string]any) {
		props, _ := m["properties"].(map[string]any)
		for k := range props {
			seen[k] = struct{}{}
		}
	}
	collect(schema)
	for _, key := range []string{"allOf", "oneOf", "anyOf"} {
		arr, _ := schema[key].([]any)
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				collect(m)
			}
		}
	}

	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// setDifference returns sorted elements of a that are not in b.
func setDifference(a, b []string) []string {
	in := make(map[string]struct{}, len(b))
	for _, s := range b {
		in[s] = struct{}{}
	}
	var out []string
	for _, s := range a {
		if _, ok := in[s]; !ok {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
