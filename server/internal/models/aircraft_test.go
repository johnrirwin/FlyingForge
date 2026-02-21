package models

import (
	"testing"
)

func TestAircraftType_Values(t *testing.T) {
	tests := []struct {
		name     string
		value    AircraftType
		expected string
	}{
		{"quad", AircraftTypeQuad, "quad"},
		{"fixed_wing", AircraftTypeFixedWing, "fixed_wing"},
		{"whoop", AircraftTypeWhoop, "whoop"},
		{"cine_lift", AircraftTypeCineLift, "cine_lift"},
		{"long_range", AircraftTypeLongRange, "long_range"},
		{"other", AircraftTypeOther, "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("AircraftType %s = %s, want %s", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestComponentCategory_Values(t *testing.T) {
	tests := []struct {
		name     string
		value    ComponentCategory
		expected string
	}{
		{"fc", ComponentCategoryFC, "fc"},
		{"esc", ComponentCategoryESC, "esc"},
		{"aio", ComponentCategoryAIO, "aio"},
		{"stack", ComponentCategoryStack, "stack"},
		{"receiver", ComponentCategoryReceiver, "receiver"},
		{"vtx", ComponentCategoryVTX, "vtx"},
		{"motors", ComponentCategoryMotors, "motors"},
		{"camera", ComponentCategoryCamera, "camera"},
		{"frame", ComponentCategoryFrame, "frame"},
		{"propellers", ComponentCategoryProps, "propellers"},
		{"antenna", ComponentCategoryAntenna, "antenna"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("ComponentCategory %s = %s, want %s", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestIsValidAircraftType(t *testing.T) {
	validTypes := []AircraftType{
		AircraftTypeQuad,
		AircraftTypeFixedWing,
		AircraftTypeWhoop,
		AircraftTypeCineLift,
		AircraftTypeLongRange,
		AircraftTypeOther,
	}

	for _, typ := range validTypes {
		if typ == "" {
			t.Errorf("Valid AircraftType should not be empty")
		}
	}
}

func TestNormalizeComponentCategory(t *testing.T) {
	tests := []struct {
		name     string
		input    ComponentCategory
		expected ComponentCategory
	}{
		{name: "canonical fc", input: ComponentCategoryFC, expected: ComponentCategoryFC},
		{name: "canonical propellers", input: ComponentCategoryProps, expected: ComponentCategoryProps},
		{name: "legacy props alias", input: ComponentCategory("props"), expected: ComponentCategoryProps},
		{name: "legacy stacks alias", input: ComponentCategory("stacks"), expected: ComponentCategoryStack},
		{name: "trim and lowercase", input: ComponentCategory("  ProPs  "), expected: ComponentCategoryProps},
		{name: "unknown passes through normalized", input: ComponentCategory("  custom  "), expected: ComponentCategory("custom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeComponentCategory(tt.input); got != tt.expected {
				t.Fatalf("NormalizeComponentCategory(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
