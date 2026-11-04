package story

import (
	"context"
	"fmt"
	"strings"
)

// TemplateGenerator is the local, deterministic Slice 2 generator. It keeps
// development and tests free of paid provider calls while preserving the same
// streaming port a real provider will implement.
type TemplateGenerator struct{}

func (TemplateGenerator) Version() string { return "template-v2" }

func (TemplateGenerator) Generate(ctx context.Context, input GenerationContext, send func(string) error) error {
	scene := input.Scene
	brief := strings.TrimSpace(scene.Brief)
	if brief == "" {
		brief = "A choice changes the direction of the story."
	}
	for _, chunk := range []string{fmt.Sprintf("%s\n\n", scene.Title), fmt.Sprintf("%s\n\n", brief), "The moment settles into focus. Someone acts, and the story cannot return to exactly where it began."} {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := send(chunk); err != nil {
			return err
		}
	}
	return nil
}
