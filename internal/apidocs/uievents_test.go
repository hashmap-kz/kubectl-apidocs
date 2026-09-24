package apidocs

import (
	"reflect"
	"strings"
	"testing"
)

func TestFindDetailsMatches_CaseInsensitive(t *testing.T) {
	got := findDetailsMatches("Binding binding BINDING", "binding")

	want := []detailsMatch{
		{start: 0, end: 7, regionID: "details-0"},
		{start: 8, end: 15, regionID: "details-1"},
		{start: 16, end: 23, regionID: "details-2"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFindDetailsMatches_EmptyTerm(t *testing.T) {
	got := findDetailsMatches("Binding", "   ")
	if got != nil {
		t.Fatalf("expected nil matches, got %v", got)
	}
}

func TestBuildDetailsSearchText_HighlightsCurrentDifferently(t *testing.T) {
	text := "Binding binding"
	matches := findDetailsMatches(text, "binding")

	got := buildDetailsSearchText(text, matches, 1)

	if !strings.Contains(got, `["details-0"]`+detailsSearchMatchStyle+`Binding`+detailsSearchResetStyle+`[""]`) {
		t.Fatalf("expected non-current match styling in %q", got)
	}

	if !strings.Contains(got, `["details-1"]`+detailsSearchCurrentMatchStyle+`binding`+detailsSearchResetStyle+`[""]`) {
		t.Fatalf("expected current match styling in %q", got)
	}
}

func TestDetailsSearchScrollRow(t *testing.T) {
	text := "events\n\nKIND:\nEvent\nVERSION:\nv1"
	match := detailsMatch{start: strings.Index(text, "KIND"), end: strings.Index(text, "KIND") + len("KIND")}

	if got := detailsSearchScrollRow(text, match); got != 1 {
		t.Fatalf("expected scroll row 1, got %d", got)
	}
}
