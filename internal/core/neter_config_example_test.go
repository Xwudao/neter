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
	}

	for _, want := range wants {
		if !strings.Contains(example, want) {
			t.Fatalf("example config missing %q\n%s", want, example)
		}
	}
}
