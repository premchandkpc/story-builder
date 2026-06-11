package narrative

import "testing"

func TestBlueprintValidate(t *testing.T) {
	bp := &Blueprint{
		Premise:  "A thief must steal a moon",
		Theme:    "sacrifice",
		Conflict: "The city hunts the thief",
		Acts: []Act{{
			Title: "Act I",
			Goal:  "Introduce the thief and the city",
		}},
	}
	if err := bp.Validate(); err != nil {
		t.Fatalf("expected valid blueprint, got %v", err)
	}
}

func TestBlueprintValidateRequiresCoreFields(t *testing.T) {
	bp := &Blueprint{}
	if err := bp.Validate(); err == nil {
		t.Fatal("expected validation error for missing core fields")
	}
}
