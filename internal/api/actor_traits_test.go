package api

import "testing"

func TestActorTraitsRoundTrip(t *testing.T) {
	traits := map[string]interface{}{"role": "lead", "score": 3, "active": true}

	encoded, err := actorTraitValueToJSON(traits["role"])
	if err != nil {
		t.Fatalf("expected role to encode: %v", err)
	}
	decoded, err := actorTraitValueFromJSON(encoded)
	if err != nil {
		t.Fatalf("expected role to decode: %v", err)
	}
	if decoded != "lead" {
		t.Fatalf("expected lead, got %v", decoded)
	}

	numberEncoded, err := actorTraitValueToJSON(traits["score"])
	if err != nil {
		t.Fatalf("expected score to encode: %v", err)
	}
	numberDecoded, err := actorTraitValueFromJSON(numberEncoded)
	if err != nil {
		t.Fatalf("expected score to decode: %v", err)
	}
	if numberDecoded != float64(3) {
		t.Fatalf("expected score 3, got %v", numberDecoded)
	}
}
