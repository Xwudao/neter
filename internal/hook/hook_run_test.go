package hook

import "testing"

func TestShouldRunOnPlatform(t *testing.T) {
	testCases := []struct {
		name   string
		goos   string
		action string
		want   bool
	}{
		{
			name:   "windows runs bat",
			goos:   "windows",
			action: "scripts\\update_random_factor.bat",
			want:   true,
		},
		{
			name:   "windows skips sh",
			goos:   "windows",
			action: "scripts/update_random_factor.sh",
			want:   false,
		},
		{
			name:   "unix skips bat",
			goos:   "linux",
			action: "scripts\\update_random_factor.bat",
			want:   false,
		},
		{
			name:   "unix skips cmd",
			goos:   "darwin",
			action: "scripts\\update_random_factor.cmd --flag",
			want:   false,
		},
		{
			name:   "unix runs sh",
			goos:   "linux",
			action: "scripts/update_random_factor.sh --flag",
			want:   true,
		},
		{
			name:   "empty action skipped",
			goos:   "linux",
			action: "",
			want:   false,
		},
		{
			name:   "binary command allowed everywhere",
			goos:   "linux",
			action: "go version",
			want:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRunOnPlatform(tc.goos, tc.action)
			if got != tc.want {
				t.Fatalf("shouldRunOnPlatform(%q, %q) = %v, want %v", tc.goos, tc.action, got, tc.want)
			}
		})
	}
}
