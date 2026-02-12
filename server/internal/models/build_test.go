package models

import "testing"

func TestBuildPartInputsFromParts_NormalizesAndFilters(t *testing.T) {
	inputs := BuildPartInputsFromParts([]BuildPart{
		{
			GearType:      GearType(" frame "),
			CatalogItemID: " frame-1 ",
			Position:      -4,
			Notes:         "  note  ",
		},
		{
			GearType:      "",
			CatalogItemID: "missing-gear-type",
		},
		{
			GearType:      GearTypeMotor,
			CatalogItemID: "   ",
		},
	})

	if len(inputs) != 1 {
		t.Fatalf("len(inputs)=%d want 1", len(inputs))
	}

	if inputs[0].GearType != GearTypeFrame {
		t.Fatalf("gearType=%q want %q", inputs[0].GearType, GearTypeFrame)
	}
	if inputs[0].CatalogItemID != "frame-1" {
		t.Fatalf("catalogItemId=%q want frame-1", inputs[0].CatalogItemID)
	}
	if inputs[0].Position != 0 {
		t.Fatalf("position=%d want 0", inputs[0].Position)
	}
	if inputs[0].Notes != "note" {
		t.Fatalf("notes=%q want note", inputs[0].Notes)
	}
}

func TestBuildPartInputsFromParts_EmptyInput(t *testing.T) {
	if out := BuildPartInputsFromParts(nil); out != nil {
		t.Fatalf("expected nil for nil input, got %+v", out)
	}
	if out := BuildPartInputsFromParts([]BuildPart{}); out != nil {
		t.Fatalf("expected nil for empty input, got %+v", out)
	}
}
