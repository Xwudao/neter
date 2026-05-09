package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindPreferredInterfaceName(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    string
		wantErr string
	}{
		{
			name: "prefers repository interface",
			source: `package biz

type helper interface {
	Do()
}

type UserRepository interface {
	List()
}
`,
			want: "UserRepository",
		},
		{
			name: "accepts repo suffix",
			source: `package biz

type RedisRepo interface {
	Get()
}
`,
			want: "RedisRepo",
		},
		{
			name: "falls back to first interface",
			source: `package biz

type diskExportWriter interface {
	Write()
}
`,
			want: "diskExportWriter",
		},
		{
			name: "errors without interface",
			source: `package biz

type DiskExportBiz struct{}
`,
			wantErr: "no interface definition found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sample_biz.go")
			require.NoError(t, os.WriteFile(path, []byte(tt.source), 0o644))

			got, err := findPreferredInterfaceName(path)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
