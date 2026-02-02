package models

import "testing"

func TestIsValidChemistry(t *testing.T) {
	tests := []struct {
		name      string
		chemistry BatteryChemistry
		want      bool
	}{
		{"LIPO valid", ChemistryLIPO, true},
		{"LIPO_HV valid", ChemistryLIPOHV, true},
		{"LIION valid", ChemistryLIION, true},
		{"empty invalid", "", false},
		{"NIMH invalid", "NIMH", false},
		{"lowercase lipo invalid", "lipo", false},
		{"random string invalid", "something", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidChemistry(tt.chemistry)
			if got != tt.want {
				t.Errorf("IsValidChemistry(%q) = %v, want %v", tt.chemistry, got, tt.want)
			}
		})
	}
}

func TestValidChemistries(t *testing.T) {
	chemistries := ValidChemistries()

	if len(chemistries) != 3 {
		t.Errorf("ValidChemistries() returned %d items, want 3", len(chemistries))
	}

	// Check all expected chemistries are present
	expected := map[BatteryChemistry]bool{
		ChemistryLIPO:   false,
		ChemistryLIPOHV: false,
		ChemistryLIION:  false,
	}

	for _, c := range chemistries {
		if _, ok := expected[c]; ok {
			expected[c] = true
		} else {
			t.Errorf("ValidChemistries() contains unexpected value %q", c)
		}
	}

	for c, found := range expected {
		if !found {
			t.Errorf("ValidChemistries() missing expected value %q", c)
		}
	}
}
