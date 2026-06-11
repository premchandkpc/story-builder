package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func parseUUID(s string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil {
		return uuid.UUID{}, errors.New("invalid uuid")
	}
	return parsed, nil
}

func parseUUIDList(values []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		parsed, err := uuid.Parse(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("invalid uuid %q", value)
		}
		result = append(result, parsed)
	}
	return result, nil
}

func normalizeStoryTitle(title string) (string, error) {
	normalized := strings.TrimSpace(title)
	if normalized == "" {
		return "", fmt.Errorf("title is required")
	}
	return normalized, nil
}

func parseOptionalUUID(value *string) (*uuid.UUID, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(*value))
	if err != nil {
		return nil, fmt.Errorf("invalid location_ref")
	}
	return &parsed, nil
}
