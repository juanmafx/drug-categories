package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestParseNewCategoryTrimsInput(t *testing.T) {
	got, err := ParseNewCategory("  Antibiotics ", "  Treat bacterial infections  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := NewCategory{Name: "Antibiotics", Description: "Treat bacterial infections"}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseNewCategoryRejectsInvalidInput(t *testing.T) {
	cases := map[string]struct{ name, description string }{
		"empty name":             {"", ""},
		"blank name":             {"   ", ""},
		"long name":              {strings.Repeat("a", MaxNameLength+1), ""},
		"long description":       {"Valid", strings.Repeat("a", MaxDescriptionLength+1)},
		"control in name":        {"Bad\x00Name", ""},
		"newline in description": {"Valid", "line one\nline two"},
		"invalid utf8":           {"Bad\xffName", ""},
	}
	for label, input := range cases {
		t.Run(label, func(t *testing.T) {
			_, err := ParseNewCategory(input.name, input.description)
			if !errors.Is(err, ErrInvalidCategory) {
				t.Fatalf("got %v, want ErrInvalidCategory", err)
			}
		})
	}
}

func TestParseNewCategoryAcceptsLimitsInCharacters(t *testing.T) {
	if _, err := ParseNewCategory(strings.Repeat("é", MaxNameLength), strings.Repeat("é", MaxDescriptionLength)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNameKeyFoldsOnlyASCII(t *testing.T) {
	if got := NameKey("AbC"); got != "abc" {
		t.Fatalf("got %q", got)
	}
	if got := NameKey("É"); got != "É" {
		t.Fatalf("got %q", got)
	}
}
