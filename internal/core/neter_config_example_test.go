package core

import (
	"strings"
	"testing"
)

func TestExampleNeterConfigYAMLContainsAllSupportedSections(t *testing.T) {
	example := ExampleNeterConfigYAML()

	wants := []string{
		"ldflags:",
		"package: main",
		"vars:",
		`gitTag: "${git_tag}"`,
		"dev:",
		"backend:",
		`cmd: "nr run -dr"`,
		"frontend:",
		`dir: "web"`,
		`pm: "pnpm"`,
		`cmd: "run dev"`,
		"hooks:",
		"items:",
		`event: "on_start"`,
		`action: "scripts/pre_build.sh"`,
		"deploy:",
		`alias: "prod-app"`,
		`remote_upload_dir: "/srv/myapp"`,
		`remote_script: "/srv/myapp/deploy.sh"`,
	}

	for _, want := range wants {
		if !strings.Contains(example, want) {
			t.Fatalf("example config missing %q\n%s", want, example)
		}
	}
}
