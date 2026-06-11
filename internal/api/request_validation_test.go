package api

import (
	"testing"
)

func TestParseUUIDListRejectsInvalidInput(t *testing.T) {
	_, err := parseUUIDList([]string{"550e8400-e29b-41d4-a716-446655440000", "not-a-uuid"})
	if err == nil {
		t.Fatal("expected invalid UUID to be rejected")
	}
}

func TestNormalizeStoryTitleTrimsAndRejectsBlankInput(t *testing.T) {
	normalized, err := normalizeStoryTitle("  The Red Path  ")
	if err != nil {
		t.Fatalf("expected title to be accepted: %v", err)
	}
	if normalized != "The Red Path" {
		t.Fatalf("expected normalized title to be trimmed, got %q", normalized)
	}

	if _, err := normalizeStoryTitle("   "); err == nil {
		t.Fatal("expected blank title to be rejected")
	}
}
