package mask

import (
	"testing"
)

func TestMasker_MaskField(t *testing.T) {
	cfg := DefaultMaskConfig()
	masker := NewMasker(cfg)

	tests := []struct {
		name     string
		value    string
		maskType string
		want     string
	}{
		{
			name:     "phone number",
			value:    "13812345678",
			maskType: "phone",
			want:     "138****5678",
		},
		{
			name:     "id card",
			value:    "110101199001011234",
			maskType: "idcard",
			want:     "110101********1234",
		},
		{
			name:     "bank card",
			value:    "6222021234567890123",
			maskType: "bankcard",
			want:     "6222****0123",
		},
		{
			name:     "email",
			value:    "test@example.com",
			maskType: "email",
			want:     "te***@example.com",
		},
		{
			name:     "unknown mask type",
			value:    "sensitive",
			maskType: "unknown",
			want:     "sensitive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := masker.MaskField(tt.value, tt.maskType)
			if got != tt.want {
				t.Errorf("MaskField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMasker_Mask(t *testing.T) {
	cfg := DefaultMaskConfig()
	masker := NewMasker(cfg)

	deepMap := map[string]any{
		"user": map[string]any{
			"name":  "John",
			"phone": "13812345678",
		},
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": map[string]any{
					"value": "deep",
				},
			},
		},
	}

	result := masker.Mask(deepMap)

	user := result["user"].(map[string]any)
	if user["name"] != "John" {
		t.Errorf("Mask() user.name = %v, want John", user["name"])
	}

	level1 := result["level1"].(map[string]any)
	level2 := level1["level2"].(map[string]any)
	level3 := level2["level3"].(map[string]any)
	if level3["value"] != "deep" {
		t.Errorf("Mask() nested value = %v, want deep", level3["value"])
	}
}

func TestMasker_DepthLimit(t *testing.T) {
	cfg := DefaultMaskConfig()
	masker := NewMasker(cfg)

	deepMap := make(map[string]any)
	current := deepMap
	for i := 0; i < 10; i++ {
		current["level"] = make(map[string]any)
		current = current["level"].(map[string]any)
	}
	current["value"] = "deep"

	result := masker.Mask(deepMap)

	t.Logf("Result: %v", result)
}

type TestStruct struct {
	Name  string `mask:"phone"`
	Email string `mask:"email"`
}

func TestMaskStruct(t *testing.T) {
	cfg := DefaultMaskConfig()
	masker := NewMasker(cfg)

	testData := TestStruct{
		Name:  "13812345678",
		Email: "test@example.com",
	}

	masked := MaskStruct(testData, masker)
	if masked.Name != "138****5678" {
		t.Errorf("MaskStruct() Name = %v, want %v", masked.Name, "138****5678")
	}
	if masked.Email != "te***@example.com" {
		t.Errorf("MaskStruct() Email = %v, want %v", masked.Email, "te***@example.com")
	}
}

func TestMasker_MaskField_IDCard(t *testing.T) {
	cfg := DefaultMaskConfig()
	masker := NewMasker(cfg)

	result := masker.MaskField("110101199001011234", "idcard")
	if result != "110101********1234" {
		t.Errorf("MaskField idcard = %v, want 110101********1234", result)
	}
}
