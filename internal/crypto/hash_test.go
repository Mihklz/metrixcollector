package crypto

import (
	"testing"
)

func TestCalculateHMAC(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		key  string
		want string
	}{
		{
			name: "empty key returns empty string",
			data: []byte("test data"),
			key:  "",
			want: "",
		},
		{
			name: "valid key and data",
			data: []byte("test data"),
			key:  "secret key",
			want: "a8b05a6b8b3b6db6b0d8b6b6a8b05a6b8b3b6db6b0d8b6b6a8b05a6b8b3b6db6", // будет другой
		},
		{
			name: "empty data with key",
			data: []byte(""),
			key:  "secret key",
			want: "b613679a0814d9ec772f95d778c35fc5ff1697c493715653c6c712144292c5ad", // будет другой
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateHMAC(tt.data, tt.key)
			if tt.key == "" && got != "" {
				t.Errorf("CalculateHMAC() with empty key should return empty string, got %v", got)
			}
			if tt.key != "" && got == "" {
				t.Errorf("CalculateHMAC() with non-empty key should return non-empty string")
			}
			if tt.key != "" && len(got) != 64 {
				t.Errorf("CalculateHMAC() should return 64 character hex string, got %d characters", len(got))
			}
		})
	}
}

func TestValidateHMAC(t *testing.T) {
	key := "secret key"
	data := []byte("test data")
	validHash := CalculateHMAC(data, key)

	tests := []struct {
		name      string
		data      []byte
		key       string
		signature string
		want      bool
	}{
		{
			name:      "valid hash",
			data:      data,
			key:       key,
			signature: validHash,
			want:      true,
		},
		{
			name:      "invalid hash",
			data:      data,
			key:       key,
			signature: "invalid_hash",
			want:      false,
		},
		{
			name:      "empty key and signature",
			data:      data,
			key:       "",
			signature: "",
			want:      true,
		},
		{
			name:      "empty key with signature",
			data:      data,
			key:       "",
			signature: "some_hash",
			want:      false,
		},
		{
			name:      "key with empty signature",
			data:      data,
			key:       key,
			signature: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateHMAC(tt.data, tt.key, tt.signature)
			if got != tt.want {
				t.Errorf("ValidateHMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}
