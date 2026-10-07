package commands

import (
	"reflect"
	"testing"
)

func TestTotalLinePrefersPaginationTotalCount(t *testing.T) {
	if got := totalLine(3, nil); got != "Total: 3" {
		t.Fatalf("got %q", got)
	}
	single := &Pagination{CurrentPage: 1, TotalPages: 1, TotalCount: 3}
	if got := totalLine(3, single); got != "Total: 3" {
		t.Fatalf("got %q", got)
	}
	paged := &Pagination{CurrentPage: 2, TotalPages: 5, TotalCount: 230}
	want := "Total: 230 (page 2/5, 50 shown; use --page/--per-page for more)"
	if got := totalLine(50, paged); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSelectColumnsKeepsOnlyRequestedFields(t *testing.T) {
	headers := []string{"ID", "NAME", "ONLINE"}
	columnFields := [][]string{{"id"}, {"name", "title"}, {"online_screens"}}
	rows := [][]string{{"1", "Lobby", "0"}}

	gotHeaders, gotRows := selectColumns("id, title", headers, columnFields, rows)
	if !reflect.DeepEqual(gotHeaders, []string{"ID", "NAME"}) {
		t.Fatalf("headers = %v", gotHeaders)
	}
	if !reflect.DeepEqual(gotRows, [][]string{{"1", "Lobby"}}) {
		t.Fatalf("rows = %v", gotRows)
	}

	gotHeaders, _ = selectColumns("", headers, columnFields, rows)
	if !reflect.DeepEqual(gotHeaders, headers) {
		t.Fatalf("without --fields headers = %v", gotHeaders)
	}
}

func TestScreenFailureDescribesErrors(t *testing.T) {
	got := screenFailure(map[string]any{"id": float64(4), "name": "Lobby", "success": false, "errors": []any{"Another action is already in progress on this screen"}})
	want := "screen 4 (Lobby): Another action is already in progress on this screen"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := screenFailure(map[string]any{"id": float64(4), "success": false, "errors": nil}); got != "screen 4: update failed" {
		t.Fatalf("got %q", got)
	}
}
