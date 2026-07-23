package cmd

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var migrateRoutesCmd = &cobra.Command{
	Use:   "routes",
	Short: "preview or migrate legacy Route.Reg registration to RouteRegistry",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("dir")
		if dir == "" {
			var err error
			dir, err = os.Getwd()
			if err != nil {
				return err
			}
		}
		apply, _ := cmd.Flags().GetBool("apply-registry")
		injectRouter, _ := cmd.Flags().GetBool("inject-router")
		skipWire, _ := cmd.Flags().GetBool("skip-wire")
		if injectRouter {
			return migrateRouterInjection(dir, apply, skipWire)
		}
		return migrateRoutes(dir, apply, skipWire)
	},
}

var migrateCmd = &cobra.Command{Use: "migrate", Short: "migrate neter project structures"}

type legacyRoute struct{ field, typ string }

func init() {
	migrateRoutesCmd.Flags().StringP("dir", "d", "", "project root (default: current directory)")
	migrateRoutesCmd.Flags().Bool("apply-registry", false, "write the RouteRegistry migration after a preview")
	migrateRoutesCmd.Flags().Bool("inject-router", false, "migrate registered routes to Register(gin.IRouter)")
	migrateRoutesCmd.Flags().Bool("skip-wire", false, "do not regenerate Wire after applying (advanced)")
	migrateCmd.AddCommand(migrateRoutesCmd)
	rootCmd.AddCommand(migrateCmd)
}

func migrateRouterInjection(root string, apply, skipWire bool) error {
	registry := filepath.Join(root, "internal", "routes", "registry.go")
	if _, err := os.Stat(registry); err != nil {
		return fmt.Errorf("RouteRegistry is required before --inject-router: %w", err)
	}
	count := 0
	err := filepath.WalkDir(filepath.Join(root, "internal", "routes"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		b, _ := os.ReadFile(path)
		count += strings.Count(string(b), "r.g.")
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("Router injection migration preview: %d r.g registrations\n", count)
	if !apply {
		fmt.Println("Preview only. Re-run with --apply-registry --inject-router to write changes.")
		return nil
	}
	err = filepath.WalkDir(filepath.Join(root, "internal", "routes"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || filepath.Base(path) == "registry.go" {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		s := string(b)
		s = regexp.MustCompile(`(?m)^\s*g\s+\*gin\.Engine\s*$\n?`).ReplaceAllString(s, "")
		s = regexp.MustCompile(`(?m)^\s*g\s+\*gin\.Engine,\s*$\n?`).ReplaceAllString(s, "")
		s = regexp.MustCompile(`\s*g:\s*g,\s*`).ReplaceAllString(s, "")
		s = regexp.MustCompile(`func \(r \*([^)]*Route)\) Register\(\)`).ReplaceAllString(s, "func (r *$1) Register(router gin.IRouter)")
		s = strings.ReplaceAll(s, "r.g.", "router.")
		return writeFormatted(path, []byte(s))
	})
	if err != nil {
		return err
	}
	b, err := os.ReadFile(registry)
	if err != nil {
		return err
	}
	s := string(b)
	if !strings.Contains(s, "github.com/gin-gonic/gin") {
		s = strings.Replace(s, "import (", "import (\n\t\"github.com/gin-gonic/gin\"", 1)
	}
	s = strings.ReplaceAll(s, "Register()", "Register(gin.IRouter)")
	s = strings.ReplaceAll(s, "RegisterAll()", "RegisterAll(router gin.IRouter)")
	s = strings.ReplaceAll(s, "route.Register(gin.IRouter)", "route.Register(router)")
	if err := writeFormatted(registry, []byte(s)); err != nil {
		return err
	}
	rootFile := filepath.Join(root, "internal", "routes", "root.go")
	b, err = os.ReadFile(rootFile)
	if err != nil {
		return err
	}
	s = strings.ReplaceAll(string(b), "r.routes.RegisterAll()", "r.routes.RegisterAll(r.router)")
	if err := writeFormatted(rootFile, []byte(s)); err != nil {
		return err
	}
	if !skipWire {
		wire := exec.Command("wire", "./cmd/app")
		wire.Dir = root
		wire.Stdout, wire.Stderr = os.Stdout, os.Stderr
		if err := wire.Run(); err != nil {
			return err
		}
	}
	fmt.Println("Router injection migration written.")
	return nil
}

func migrateRoutes(root string, apply, skipWire bool) error {
	rootFile := filepath.Join(root, "internal", "routes", "root.go")
	registryFile := filepath.Join(root, "internal", "routes", "registry.go")
	if _, err := os.Stat(registryFile); err == nil {
		if !apply {
			return fmt.Errorf("RouteRegistry already exists: %s", registryFile)
		}
		// A previous migration may have stopped after creating the registry
		// (for example because a non-source reference file was encountered).
		// Resume only the remaining route-method rewrite and Wire generation.
		if err := renameRouteRegMethods(root); err != nil {
			return err
		}
		if !skipWire {
			return regenerateWire(root)
		}
		fmt.Println("RouteRegistry migration resumed.")
		return nil
	}
	src, err := os.ReadFile(rootFile)
	if err != nil {
		return err
	}
	routes, err := legacyRoutesFromRoot(string(src))
	if err != nil {
		return err
	}
	if err := verifyOnlyRouteRegCalls(root); err != nil {
		return err
	}
	fmt.Printf("RouteRegistry migration preview: %d routes\n", len(routes))
	for _, r := range routes {
		fmt.Printf("  %s %s\n", r.field, r.typ)
	}
	if !apply {
		fmt.Println("Preview only. Re-run with --apply-registry to write changes.")
		return nil
	}

	updated, err := rewriteLegacyRoot(string(src), routes)
	if err != nil {
		return err
	}
	registry, err := buildRegistry(rootFile, routes)
	if err != nil {
		return err
	}
	provider := filepath.Join(root, "internal", "routes", "provider.go")
	providerSrc, err := os.ReadFile(provider)
	if err != nil {
		return err
	}
	if !strings.Contains(string(providerSrc), "NewRouteRegistry") {
		providerSrc = []byte(strings.Replace(string(providerSrc), "NewHttpEngine,", "NewHttpEngine,\n\tNewRouteRegistry,", 1))
	}
	updated = stripMigratedRouteImports(updated, routes)
	if err := writeFormatted(rootFile, []byte(updated)); err != nil {
		return err
	}
	if err := writeFormatted(registryFile, []byte(registry)); err != nil {
		return err
	}
	if err := writeFormatted(provider, providerSrc); err != nil {
		return err
	}
	if err := renameRouteRegMethods(root); err != nil {
		return err
	}
	if !skipWire {
		if err := regenerateWire(root); err != nil {
			return err
		}
	}
	fmt.Println("RouteRegistry migration written.")
	return nil
}

func legacyRoutesFromRoot(src string) ([]legacyRoute, error) {
	re := regexp.MustCompile(`r\.([A-Za-z0-9_]+)\.Reg\(\)`)
	matches := re.FindAllStringSubmatch(src, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no r.<route>.Reg() calls found in HttpEngine.Register")
	}
	fieldTypes := map[string]string{}
	fieldRe := regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_]+)\s+(\*[A-Za-z0-9_]+\.[A-Za-z0-9_]+)\s*$`)
	for _, m := range fieldRe.FindAllStringSubmatch(src, -1) {
		fieldTypes[m[1]] = m[2]
	}
	seen := map[string]bool{}
	var out []legacyRoute
	for _, m := range matches {
		if !seen[m[1]] {
			typ := fieldTypes[m[1]]
			if typ == "" {
				return nil, fmt.Errorf("can't find type for Route field %s", m[1])
			}
			out = append(out, legacyRoute{m[1], typ})
			seen[m[1]] = true
		}
	}
	return out, nil
}

func rewriteLegacyRoot(src string, routes []legacyRoute) (string, error) {
	for _, r := range routes {
		fieldLine := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(r.field) + `\s+` + regexp.QuoteMeta(r.typ) + `\s*$\n?`)
		src = fieldLine.ReplaceAllString(src, "")
		paramLine := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(r.field) + `\s+` + regexp.QuoteMeta(r.typ) + `,\s*$\n?`)
		src = paramLine.ReplaceAllString(src, "")
		valueLine := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(r.field) + `:\s*` + regexp.QuoteMeta(r.field) + `,\s*$\n?`)
		src = valueLine.ReplaceAllString(src, "")
		src = strings.ReplaceAll(src, "\tr."+r.field+".Reg()\n", "")
	}
	if !strings.Contains(src, "\troutes RouteRegistry") {
		engineStruct := regexp.MustCompile(`(?s)(type HttpEngine struct \{.*?)(\n\})`)
		if !engineStruct.MatchString(src) {
			return "", fmt.Errorf("unsupported root.go: can't find HttpEngine struct")
		}
		src = engineStruct.ReplaceAllString(src, "${1}\n\troutes RouteRegistry${2}")
	}
	needle := "\tctx *system.AppContext,\n"
	if !strings.Contains(src, needle) {
		return "", fmt.Errorf("unsupported NewHttpEngine signature: missing ctx parameter")
	}
	src = strings.Replace(src, needle, needle+"\troutes RouteRegistry,\n", 1)
	if !strings.Contains(src, "\t\troutes: routes,") {
		src = strings.Replace(src, "\the := &HttpEngine{", "\the := &HttpEngine{\n\t\troutes: routes,", 1)
	}
	if !strings.Contains(src, "r.routes.RegisterAll()") {
		src = strings.Replace(src, "func (r *HttpEngine) Register() {", "func (r *HttpEngine) Register() {\n\tr.routes.RegisterAll()", 1)
	}
	return src, nil
}

func buildRegistry(rootFile string, routes []legacyRoute) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rootFile, nil, parser.ParseComments)
	if err != nil {
		return "", err
	}
	imports := map[string]string{}
	for _, im := range f.Imports {
		p := strings.Trim(im.Path.Value, `"`)
		name := filepath.Base(p)
		if im.Name != nil {
			name = im.Name.Name
		}
		imports[name] = p
	}
	used := map[string]bool{}
	for _, r := range routes {
		used[strings.TrimPrefix(strings.Split(r.typ, ".")[0], "*")] = true
	}
	var aliases []string
	for a := range used {
		aliases = append(aliases, a)
	}
	sort.Strings(aliases)
	var b strings.Builder
	b.WriteString("package routes\n\nimport (\n")
	for _, a := range aliases {
		p := imports[a]
		if p == "" {
			return "", fmt.Errorf("missing import for route package %s", a)
		}
		b.WriteString(fmt.Sprintf("\t%s %q\n", a, p))
	}
	b.WriteString(")\n\n// Registrar is a migrated legacy route.\ntype Registrar interface { Register() }\n\ntype RouteRegistry []Registrar\n\nfunc NewRouteRegistry(\n")
	for _, r := range routes {
		b.WriteString(fmt.Sprintf("\t%s %s,\n", r.field, r.typ))
	}
	b.WriteString(") RouteRegistry {\n\treturn RouteRegistry{\n")
	for _, r := range routes {
		b.WriteString("\t\t" + r.field + ",\n")
	}
	b.WriteString("\t}\n}\n\nfunc (r RouteRegistry) RegisterAll() {\n\tfor _, route := range r {\n\t\troute.Register()\n\t}\n}\n")
	return b.String(), nil
}

func stripMigratedRouteImports(source string, routes []legacyRoute) string {
	aliases := map[string]bool{}
	for _, route := range routes {
		aliases[strings.TrimPrefix(strings.Split(route.typ, ".")[0], "*")] = true
	}
	for alias := range aliases {
		if strings.Contains(source, alias+".") {
			continue
		}
		// The root no longer references these route sub-packages after their
		// fields move into registry.go. Match both named and default imports.
		source = regexp.MustCompile(`(?m)^\s*(?:`+regexp.QuoteMeta(alias)+`\s+)?"[^"]+/internal/routes/`+regexp.QuoteMeta(alias)+`"\s*\n`).ReplaceAllString(source, "")
	}
	return source
}

func verifyOnlyRouteRegCalls(root string) error {
	var bad []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".agents" || d.Name() == ".git" || d.Name() == "vendor" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		b, _ := os.ReadFile(path)
		for line := range strings.SplitSeq(string(b), "\n") {
			if strings.Contains(line, ".Reg()") && !strings.Contains(line, "Route") && !strings.Contains(line, "r.") {
				bad = append(bad, path+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if len(bad) > 0 {
		return fmt.Errorf("refusing migration: non-Route .Reg() calls found:\n%s", strings.Join(bad, "\n"))
	}
	return nil
}

func renameRouteRegMethods(root string) error {
	return filepath.WalkDir(filepath.Join(root, "internal", "routes"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.Contains(path, "/internal/data/ent/") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		s := regexp.MustCompile(`func \(r \*([^)]*Route)\) Reg\(\)`).ReplaceAllString(string(b), "func (r *$1) Register()")
		// verifyOnlyRouteRegCalls makes this replacement safe for the supported
		// legacy layouts: the remaining calls are only route registrations/tests.
		s = strings.ReplaceAll(s, ".Reg()", ".Register()")
		return writeFormatted(path, []byte(s))
	})
}

func regenerateWire(root string) error {
	wire := exec.Command("wire", "./cmd/app")
	wire.Dir = root
	wire.Stdout, wire.Stderr = os.Stdout, os.Stderr
	if err := wire.Run(); err != nil {
		return fmt.Errorf("migration written but Wire regeneration failed: %w", err)
	}
	return nil
}

func writeFormatted(path string, source []byte) error {
	formatted, err := format.Source(source)
	if err != nil {
		return fmt.Errorf("format %s: %w", path, err)
	}
	return os.WriteFile(path, formatted, 0o644)
}
