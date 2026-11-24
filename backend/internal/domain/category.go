package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaxNameLength        = 120
	MaxDescriptionLength = 1000
)

var (
	ErrInvalidCategory   = errors.New("invalid category")
	ErrDuplicateCategory = errors.New("a category with this name already exists")
)

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type NewCategory struct {
	Name        string
	Description string
}

func ParseNewCategory(name, description string) (NewCategory, error) {
	cleanName := strings.TrimSpace(name)
	cleanDescription := strings.TrimSpace(description)

	if cleanName == "" {
		return NewCategory{}, fmt.Errorf("%w: name is required", ErrInvalidCategory)
	}
	if err := validateText("name", cleanName, MaxNameLength); err != nil {
		return NewCategory{}, err
	}
	if err := validateText("description", cleanDescription, MaxDescriptionLength); err != nil {
		return NewCategory{}, err
	}
	return NewCategory{Name: cleanName, Description: cleanDescription}, nil
}

func NameKey(name string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return r
	}, name)
}

func validateText(field, value string, maxLength int) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: %s must be valid UTF-8", ErrInvalidCategory, field)
	}
	if utf8.RuneCountInString(value) > maxLength {
		return fmt.Errorf("%w: %s must be at most %d characters", ErrInvalidCategory, field, maxLength)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("%w: %s must not contain control characters", ErrInvalidCategory, field)
		}
	}
	return nil
}
