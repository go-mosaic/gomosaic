package envconfig

import (
	"testing"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

func TestParseValidation(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantRequired bool
		wantMaxLen   int
		wantMinLen   int
		wantErr      bool
	}{
		{"empty", "", false, 0, 0, false},
		{"required", "required", true, 0, 0, false},
		{"max-len", "max-len=100", false, 100, 0, false},
		{"min-len", "min-len=3", false, 0, 3, false},
		{"combined", "required max-len=100 min-len=3", true, 100, 3, false},
		{"invalid max-len", "max-len=abc", false, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			required, maxLen, minLen, err := ParseValidation(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseValidation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if required != tt.wantRequired {
				t.Errorf("required = %v, want %v", required, tt.wantRequired)
			}
			if maxLen != tt.wantMaxLen {
				t.Errorf("maxLen = %d, want %d", maxLen, tt.wantMaxLen)
			}
			if minLen != tt.wantMinLen {
				t.Errorf("minLen = %d, want %d", minLen, tt.wantMinLen)
			}
		})
	}
}

func TestIsExported(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Exported", true},
		{"unexported", false},
		{"", false},
		{"A", true},
		{"z", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExported(tt.name)
			if got != tt.want {
				t.Errorf("isExported(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsStructType(t *testing.T) {
	tests := []struct {
		name     string
		typeInfo *gomosaic.TypeInfo
		want     bool
	}{
		{
			name:     "nil",
			typeInfo: nil,
			want:     false,
		},
		{
			name: "basic string",
			typeInfo: &gomosaic.TypeInfo{
				Name:    "string",
				IsBasic: true,
			},
			want: false,
		},
		{
			name: "struct type",
			typeInfo: &gomosaic.TypeInfo{
				IsNamed: true,
				ElemType: &gomosaic.TypeInfo{
					Struct: &gomosaic.StructInfo{
						Fields: []*gomosaic.VarInfo{
							{Name: "Field1"},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "ptr to struct",
			typeInfo: &gomosaic.TypeInfo{
				IsPtr: true,
				ElemType: &gomosaic.TypeInfo{
					IsNamed: true,
					ElemType: &gomosaic.TypeInfo{
						Struct: &gomosaic.StructInfo{
							Fields: []*gomosaic.VarInfo{{Name: "F"}},
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStructType(tt.typeInfo)
			if got != tt.want {
				t.Errorf("isStructType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlugin_Name(t *testing.T) {
	p := NewPlugin()
	if p.Name() != "env-config" {
		t.Errorf("Name() = %s, want env-config", p.Name())
	}
}

func TestPlugin_ImplementsGenerator(t *testing.T) {
	// Проверка времени компиляции, что Plugin реализует gomosaic.Generator
	var _ gomosaic.Generator = NewPlugin()
}
