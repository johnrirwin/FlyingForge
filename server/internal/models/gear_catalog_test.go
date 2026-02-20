package models

import (
	"testing"
)

func TestBuildCanonicalKey(t *testing.T) {
	tests := []struct {
		name     string
		gearType GearType
		brand    string
		model    string
		variant  string
		expected string
	}{
		{
			name:     "basic lowercase",
			gearType: GearTypeMotor,
			brand:    "TMotor",
			model:    "F80 Pro",
			variant:  "",
			expected: "motor|tmotor|f80 pro",
		},
		{
			name:     "with variant",
			gearType: GearTypeESC,
			brand:    "Hobbywing",
			model:    "XRotor",
			variant:  "60A",
			expected: "esc|hobbywing|xrotor|60a",
		},
		{
			name:     "punctuation normalization",
			gearType: GearTypeFC,
			brand:    "BetaFlight",
			model:    "F7-HD",
			variant:  "V2.0",
			expected: "fc|betaflight|f7 hd|v2 0",
		},
		{
			name:     "already lowercase",
			gearType: GearTypeReceiver,
			brand:    "tbs",
			model:    "crossfire nano",
			variant:  "",
			expected: "receiver|tbs|crossfire nano",
		},
		{
			name:     "mixed case brand collision",
			gearType: GearTypeVTX,
			brand:    "TBS",
			model:    "Unify Pro",
			variant:  "5G8 HV",
			expected: "vtx|tbs|unify pro|5g8 hv",
		},
		{
			name:     "special characters removed",
			gearType: GearTypeFrame,
			brand:    "ImpulseRC",
			model:    "Apex™",
			variant:  "5\"",
			expected: "frame|impulserc|apex|5",
		},
		{
			name:     "multiple spaces collapsed",
			gearType: GearTypeProp,
			brand:    "  HQ  ",
			model:    "  Prop  6x4.5  ",
			variant:  "",
			expected: "prop|hq|prop 6x4 5",
		},
		{
			name:     "unicode normalization",
			gearType: GearTypeMotor,
			brand:    "Émax",
			model:    "RS2205",
			variant:  "",
			expected: "motor|emax|rs2205",
		},
		{
			name:     "accented characters",
			gearType: GearTypeCamera,
			brand:    "Caddx",
			model:    "Nébula",
			variant:  "Pro",
			expected: "camera|caddx|nebula|pro",
		},
		{
			name:     "dashes and underscores",
			gearType: GearTypeAIO,
			brand:    "Speed_Bee",
			model:    "F7-AIO",
			variant:  "BL32-55A",
			expected: "aio|speed bee|f7 aio|bl32 55a",
		},
		{
			name:     "numbers and letters mixed",
			gearType: GearTypeESC,
			brand:    "BLHeli32",
			model:    "4in1 55A",
			variant:  "Rev_D",
			expected: "esc|blheli32|4in1 55a|rev d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCanonicalKey(tt.gearType, tt.brand, tt.model, tt.variant)
			if got != tt.expected {
				t.Errorf("BuildCanonicalKey(%q, %q, %q, %q) = %q, want %q",
					tt.gearType, tt.brand, tt.model, tt.variant, got, tt.expected)
			}
		})
	}
}

func TestBuildCanonicalKey_Deduplication(t *testing.T) {
	// Test that different user inputs that should match the same product
	// produce the same canonical key
	tests := []struct {
		name   string
		inputs []struct {
			gearType GearType
			brand    string
			model    string
			variant  string
		}
	}{
		{
			name: "TBS Crossfire variations",
			inputs: []struct {
				gearType GearType
				brand    string
				model    string
				variant  string
			}{
				{GearTypeReceiver, "TBS", "Crossfire Nano", ""},
				{GearTypeReceiver, "tbs", "crossfire nano", ""},
				{GearTypeReceiver, "TBS", "Crossfire-Nano", ""},
				{GearTypeReceiver, "  TBS  ", "  Crossfire Nano  ", ""},
			},
		},
		{
			name: "Motor KV variations",
			inputs: []struct {
				gearType GearType
				brand    string
				model    string
				variant  string
			}{
				{GearTypeMotor, "TMotor", "F80 Pro", "1900KV"},
				{GearTypeMotor, "tmotor", "f80 pro", "1900kv"},
				{GearTypeMotor, "TMOTOR", "F80-Pro", "1900KV"},
				{GearTypeMotor, "TMotor", "F80 Pro", "1900kv"},
			},
		},
		{
			name: "FC with version",
			inputs: []struct {
				gearType GearType
				brand    string
				model    string
				variant  string
			}{
				{GearTypeFC, "Matek", "F722-SE", "V3"},
				{GearTypeFC, "MATEK", "F722 SE", "v3"},
				{GearTypeFC, "matek", "f722-se", "V3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var keys []string
			for _, input := range tt.inputs {
				key := BuildCanonicalKey(input.gearType, input.brand, input.model, input.variant)
				keys = append(keys, key)
			}

			// All keys should be identical
			for i := 1; i < len(keys); i++ {
				if keys[i] != keys[0] {
					t.Errorf("Key mismatch: %q != %q (inputs %d vs 0)", keys[i], keys[0], i)
				}
			}
		})
	}
}

func TestGearTypeFromEquipmentCategory(t *testing.T) {
	tests := []struct {
		category EquipmentCategory
		expected GearType
	}{
		{CategoryMotors, GearTypeMotor},
		{CategoryESC, GearTypeESC},
		{CategoryFC, GearTypeFC},
		{CategoryAIO, GearTypeAIO},
		{CategoryStacks, GearTypeStack},
		{CategoryFrames, GearTypeFrame},
		{CategoryVTX, GearTypeVTX},
		{CategoryReceivers, GearTypeReceiver},
		{CategoryAntennas, GearTypeAntenna},
		{CategoryBatteries, GearTypeBattery},
		{CategoryPropellers, GearTypeProp},
		{CategoryCameras, GearTypeCamera},
		{CategoryAccessories, GearTypeOther},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			got := GearTypeFromEquipmentCategory(tt.category)
			if got != tt.expected {
				t.Errorf("GearTypeFromEquipmentCategory(%q) = %q, want %q", tt.category, got, tt.expected)
			}
		})
	}
}

func TestGearType_ToEquipmentCategory(t *testing.T) {
	tests := []struct {
		gearType GearType
		expected EquipmentCategory
	}{
		{GearTypeMotor, CategoryMotors},
		{GearTypeESC, CategoryESC},
		{GearTypeFC, CategoryFC},
		{GearTypeAIO, CategoryAIO},
		{GearTypeStack, CategoryStacks},
		{GearTypeFrame, CategoryFrames},
		{GearTypeVTX, CategoryVTX},
		{GearTypeReceiver, CategoryReceivers},
		{GearTypeAntenna, CategoryAntennas},
		{GearTypeBattery, CategoryBatteries},
		{GearTypeProp, CategoryPropellers},
		{GearTypeCamera, CategoryCameras},
		{GearTypeRadio, CategoryAccessories},
		{GearTypeOther, CategoryAccessories},
	}

	for _, tt := range tests {
		t.Run(string(tt.gearType), func(t *testing.T) {
			got := tt.gearType.ToEquipmentCategory()
			if got != tt.expected {
				t.Errorf("GearType(%q).ToEquipmentCategory() = %q, want %q", tt.gearType, got, tt.expected)
			}
		})
	}
}

func TestGearCatalogItem_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		item     GearCatalogItem
		expected string
	}{
		{
			name: "brand and model only",
			item: GearCatalogItem{
				Brand: "TBS",
				Model: "Crossfire Nano",
			},
			expected: "TBS Crossfire Nano",
		},
		{
			name: "with variant",
			item: GearCatalogItem{
				Brand:   "TMotor",
				Model:   "F80 Pro",
				Variant: "1900KV",
			},
			expected: "TMotor F80 Pro 1900KV",
		},
		{
			name: "empty variant",
			item: GearCatalogItem{
				Brand:   "Hobbywing",
				Model:   "XRotor",
				Variant: "",
			},
			expected: "Hobbywing XRotor",
		},
		{
			name: "whitespace handling",
			item: GearCatalogItem{
				Brand:   "EMAX",
				Model:   "RS2205",
				Variant: "",
			},
			expected: "EMAX RS2205",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.DisplayName()
			if got != tt.expected {
				t.Errorf("DisplayName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAllGearTypes(t *testing.T) {
	gearTypes := AllGearTypes()

	// Should include all defined gear types
	expectedCount := 14 // motor, esc, fc, aio, stack, frame, vtx, receiver, antenna, battery, prop, radio, camera, other
	if len(gearTypes) != expectedCount {
		t.Errorf("AllGearTypes() returned %d types, want %d", len(gearTypes), expectedCount)
	}

	// Check that specific gear types are included
	expectedTypes := []GearType{
		GearTypeMotor, GearTypeESC, GearTypeFC, GearTypeAIO, GearTypeStack,
		GearTypeFrame, GearTypeVTX, GearTypeReceiver, GearTypeAntenna,
		GearTypeBattery, GearTypeProp, GearTypeRadio, GearTypeCamera, GearTypeOther,
	}

	for _, expected := range expectedTypes {
		found := false
		for _, gt := range gearTypes {
			if gt == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllGearTypes() missing %q", expected)
		}
	}
}

func TestExtractBrandModelFromName(t *testing.T) {
	tests := []struct {
		name            string
		productName     string
		manufacturer    string
		expectedBrand   string
		expectedModel   string
		expectedVariant string
	}{
		{
			name:            "manufacturer provided",
			productName:     "F80 Pro 1900KV",
			manufacturer:    "TMotor",
			expectedBrand:   "TMotor",
			expectedModel:   "F80 Pro",
			expectedVariant: "1900KV",
		},
		{
			name:            "brand prefix in name with manufacturer",
			productName:     "TMotor F80 Pro",
			manufacturer:    "TMotor",
			expectedBrand:   "TMotor",
			expectedModel:   "F80",
			expectedVariant: "Pro",
		},
		{
			name:            "no manufacturer - extract from name",
			productName:     "TBS Crossfire Nano",
			manufacturer:    "",
			expectedBrand:   "TBS",
			expectedModel:   "Crossfire Nano",
			expectedVariant: "",
		},
		{
			name:            "single word",
			productName:     "Crossfire",
			manufacturer:    "",
			expectedBrand:   "Crossfire",
			expectedModel:   "",
			expectedVariant: "",
		},
		{
			name:            "empty string",
			productName:     "",
			manufacturer:    "",
			expectedBrand:   "",
			expectedModel:   "",
			expectedVariant: "",
		},
		{
			name:            "with version variant",
			productName:     "F722-SE V3",
			manufacturer:    "Matek",
			expectedBrand:   "Matek",
			expectedModel:   "F722-SE",
			expectedVariant: "V3",
		},
		{
			name:            "with Pro variant",
			productName:     "Ratel 2 Pro",
			manufacturer:    "Caddx",
			expectedBrand:   "Caddx",
			expectedModel:   "Ratel 2",
			expectedVariant: "Pro",
		},
		{
			name:            "motor with KV",
			productName:     "RS2205 2300KV",
			manufacturer:    "EMAX",
			expectedBrand:   "EMAX",
			expectedModel:   "RS2205",
			expectedVariant: "2300KV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			brand, model, variant := ExtractBrandModelFromName(tt.productName, tt.manufacturer)
			if brand != tt.expectedBrand {
				t.Errorf("ExtractBrandModelFromName(%q, %q) brand = %q, want %q", tt.productName, tt.manufacturer, brand, tt.expectedBrand)
			}
			if model != tt.expectedModel {
				t.Errorf("ExtractBrandModelFromName(%q, %q) model = %q, want %q", tt.productName, tt.manufacturer, model, tt.expectedModel)
			}
			if variant != tt.expectedVariant {
				t.Errorf("ExtractBrandModelFromName(%q, %q) variant = %q, want %q", tt.productName, tt.manufacturer, variant, tt.expectedVariant)
			}
		})
	}
}

func TestNormalizeString_EdgeCases(t *testing.T) {
	// Test edge cases for the normalize function via BuildCanonicalKey
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "motor||",
		},
		{
			name:     "only punctuation",
			input:    "!!!---...",
			expected: "motor||",
		},
		{
			name:     "unicode combining characters",
			input:    "café",
			expected: "motor|cafe|",
		},
		{
			name:     "numbers",
			input:    "12345",
			expected: "motor|12345|",
		},
		{
			name:     "mixed unicode",
			input:    "日本語",
			expected: "motor|日本語|",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCanonicalKey(GearTypeMotor, tt.input, "", "")
			if got != tt.expected {
				t.Errorf("BuildCanonicalKey with input %q = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
