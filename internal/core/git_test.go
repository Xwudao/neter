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

func TestGetGitTag(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Get Git Tag",
			wantErr: false, // may fail if no tags exist, but normally should succeed
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, err := GetGitTag()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGitTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && tag != "" {
				t.Logf("latest tag: %s", tag)
			}
		})
	}
}
