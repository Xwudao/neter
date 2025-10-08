package core

import (
	"testing"
)

func TestGetGitHash(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			name:    "Get Git Hash",
			want:    "", // We cannot predict the git hash, so we just check for no error
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetGitHash()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGitHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
