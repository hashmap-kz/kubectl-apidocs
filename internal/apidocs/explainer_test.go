package apidocs

import (
	"reflect"
	"testing"
)

func TestExplainFields_ResourcePath(t *testing.T) {
	got := explainFields("Binding.metadata.name")
	want := []string{"metadata", "name"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestExplainFields_ResourceOnly(t *testing.T) {
	got := explainFields("Binding")

	if len(got) != 0 {
		t.Fatalf("expected no fields, got %v", got)
	}
}

func TestExplainCacheKey(t *testing.T) {
	if got := explainCacheKey("Binding", false); got != "Binding" {
		t.Fatalf("expected Binding, got %q", got)
	}

	if got := explainCacheKey("Binding", true); got != "Binding#recursive" {
		t.Fatalf("expected Binding#recursive, got %q", got)
	}
}
