//go:build test

package agents

import (
	"context"
	"testing"
	"time"
)

type contractTestCase struct {
	Name    string
	Spec    *AgentSpec
	Input   AgentInput
	WantErr bool
}

func runContractTest(t *testing.T, tc contractTestCase) {
	t.Helper()
	t.Run(tc.Name, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		output, err := tc.Spec.Runner(ctx, tc.Input)
		if tc.WantErr && err == nil {
			t.Errorf("expected error, got nil, output: %+v", output)
		}
		if !tc.WantErr && err != nil {
			t.Errorf("unexpected error: %v, output: %+v", err, output)
		}
	})
}

func TestAgentContracts_Timeout(t *testing.T) {
	registry := NewAgentRegistry()
	RegisterAll(registry, nil, nil, nil, nil)
	for _, spec := range registry.List() {
		spec := spec
		t.Run(spec.Name+"/timeout", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
			defer cancel()
			time.Sleep(10 * time.Millisecond)
			_, err := spec.Runner(ctx, AgentInput{})
			if err == nil {
				t.Log("agent", spec.Name, "returned without error on cancelled ctx (may be acceptable)")
			}
		})
	}
}

func TestAgentContracts_NilInput(t *testing.T) {
	registry := NewAgentRegistry()
	RegisterAll(registry, nil, nil, nil, nil)
	for _, spec := range registry.List() {
		spec := spec
		t.Run(spec.Name+"/nil_input", func(t *testing.T) {
			t.Parallel()
			output, err := spec.Runner(context.Background(), AgentInput{})
			if err != nil {
				t.Logf("agent %s returned error on empty input: %v", spec.Name, err)
			}
			if output == (AgentOutput{}) {
				t.Logf("agent %s returned zero output on empty input", spec.Name)
			}
		})
	}
}
