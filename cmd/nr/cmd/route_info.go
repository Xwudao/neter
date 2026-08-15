package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Xwudao/neter/internal/route_info"
	"github.com/Xwudao/neter/pkg/utils"
)

var routeInfoCmd = &cobra.Command{
	Use:   "route-info",
	Short: "analyze and export route information from Gin projects",
	Long: `Scans a Go project directory for Gin route registrations and extracts:
- HTTP method and path
- Handler function name, registration location, and route middleware
- Route group (public, auth, admin), including nested Gin groups
- Parameters (body/query/URI bindings, headers, forms, files, and context values)
- Return types

Output formats: json, md, curl (terminal-friendly with curl examples)
Filter with -f to match handler name or full path.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		utils.CheckErrWithStatus(runRouteInfo(cmd))
	},
}

var routeInfoExportCmd = &cobra.Command{
	Use:   "export",
	Short: "export route info to a file",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		utils.CheckErrWithStatus(runRouteInfoExport(cmd))
	},
}

var routeInfoGenTSCmd = &cobra.Command{
	Use:   "gen-ts",
	Short: "generate TypeScript API contracts from route definitions",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		utils.CheckErrWithStatus(runRouteInfoGenTS(cmd))
	},
}

type routeInfoConfig struct {
	Dir     string
	Output  string
	Format  string
	Filter  string
	Package string
	Server  string
}

func getRouteInfoConfig(cmd *cobra.Command) (*routeInfoConfig, error) {
	dir, _ := cmd.Flags().GetString("dir")
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get current dir: %w", err)
		}
	}

	output, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	filter, _ := cmd.Flags().GetString("filter")
	pkg, _ := cmd.Flags().GetString("package")

	server, _ := cmd.Flags().GetString("server")
	// Auto-detect from config.yml if server not explicitly set.
	if !cmd.Flags().Changed("server") {
		if detected := detectServerFromConfig(dir); detected != "" {
			server = detected
		}
	}

	if format == "" {
		if output != "" {
			switch {
			case strings.HasSuffix(output, ".json"):
				format = "json"
			case strings.HasSuffix(output, ".md"):
				format = "md"
			default:
				format = "json"
			}
		} else {
			format = "curl"
		}
	}
	format = strings.ToLower(format)

	return &routeInfoConfig{
		Dir:     dir,
		Output:  output,
		Format:  format,
		Filter:  filter,
		Package: pkg,
		Server:  server,
	}, nil
}

func runRouteInfo(cmd *cobra.Command) error {
	cfg, err := getRouteInfoConfig(cmd)
	if err != nil {
		return err
	}

	projectRoutes, err := route_info.AnalyzeRoutes(cfg.Dir)
	if err != nil {
		return fmt.Errorf("analyze routes: %w", err)
	}

	// Apply filter
	if cfg.Filter != "" || cfg.Package != "" {
		projectRoutes = route_info.ApplyFilter(projectRoutes, &route_info.FilterOption{
			Keyword: cfg.Filter,
			Package: cfg.Package,
		})
	}

	if cfg.Output != "" {
		if err := writeRouteInfo(projectRoutes, cfg); err != nil {
			return err
		}
		fmt.Printf("route info written to %s (%d routes)\n", cfg.Output, len(projectRoutes.Routes))
	} else {
		// Print to stdout
		switch cfg.Format {
		case "json":
			if err := route_info.WriteJSONStdout(projectRoutes); err != nil {
				return err
			}
		case "md", "markdown":
			route_info.WriteMarkdownStdout(projectRoutes)
		case "curl":
			route_info.WriteTerminalStdout(projectRoutes, &route_info.TerminalConfig{
				ServerURL: cfg.Server,
			})
		default:
			return fmt.Errorf("unsupported format: %s (use json, md, or curl)", cfg.Format)
		}
	}

	return nil
}

func runRouteInfoExport(cmd *cobra.Command) error {
	cfg, err := getRouteInfoConfig(cmd)
	if err != nil {
		return err
	}

	if cfg.Output == "" {
		return fmt.Errorf("--output is required for export command")
	}

	projectRoutes, err := route_info.AnalyzeRoutes(cfg.Dir)
	if err != nil {
		return fmt.Errorf("analyze routes: %w", err)
	}

	if cfg.Filter != "" || cfg.Package != "" {
		projectRoutes = route_info.ApplyFilter(projectRoutes, &route_info.FilterOption{
			Keyword: cfg.Filter,
			Package: cfg.Package,
		})
	}

	if err := writeRouteInfo(projectRoutes, cfg); err != nil {
		return err
	}
	fmt.Printf("route info written to %s (%d routes)\n", cfg.Output, len(projectRoutes.Routes))

	return nil
}

func runRouteInfoGenTS(cmd *cobra.Command) error {
	dir, _ := cmd.Flags().GetString("dir")
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get current dir: %w", err)
		}
	}
	output, _ := cmd.Flags().GetString("output")
	if !cmd.Flags().Changed("output") {
		// Default output follows the configured frontend directory
		// (neter.yml dev.frontend.dir, default "web").
		webDir, _ := resolveFrontendOptions("", false)
		output = filepath.Join(webDir, "src", "api", "generated")
	}
	if !filepath.IsAbs(output) {
		output = filepath.Join(dir, output)
	}
	filter, _ := cmd.Flags().GetString("filter")
	pkg, _ := cmd.Flags().GetString("package")
	check, _ := cmd.Flags().GetBool("check")

	routes, err := route_info.AnalyzeRoutes(dir)
	if err != nil {
		return fmt.Errorf("analyze routes: %w", err)
	}
	if filter != "" || pkg != "" {
		routes = route_info.ApplyFilter(routes, &route_info.FilterOption{Keyword: filter, Package: pkg})
	}

	// --output ending in .ts keeps the legacy single-file mode; anything else
	// is treated as a directory of per-route-file *.gen.ts contracts.
	singleFile := strings.HasSuffix(output, ".ts")
	if singleFile {
		generated := route_info.GenerateTypeScript(routes)
		if check {
			existing, err := os.ReadFile(output)
			if err != nil {
				return fmt.Errorf("read generated TypeScript file: %w", err)
			}
			if string(existing) != generated {
				return fmt.Errorf("generated TypeScript contract %s is stale: run nr route-info gen-ts --dir %s --output %s", output, dir, output)
			}
			fmt.Printf("TypeScript contracts are current: %s (%d routes)\n", output, len(routes.Routes))
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
		if err := os.WriteFile(output, []byte(generated), 0o644); err != nil {
			return fmt.Errorf("write generated TypeScript file: %w", err)
		}
		fmt.Printf("TypeScript contracts written to %s (%d routes)\n", output, len(routes.Routes))
		checkOxfmtIgnore(output)
		return nil
	}

	return runRouteInfoGenTSDir(dir, output, routes, check)
}

// runRouteInfoGenTSDir writes one .gen.ts file per route source file into the
// output directory, plus a shared _common.gen.ts.
func runRouteInfoGenTSDir(projectDir, outputDir string, routes *route_info.ProjectRoutes, check bool) error {
	files := route_info.GenerateTypeScriptFiles(routes)

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	if check {
		var stale []string
		for _, name := range names {
			existing, err := os.ReadFile(filepath.Join(outputDir, name))
			if err != nil {
				return fmt.Errorf("read generated TypeScript file %s: %w", name, err)
			}
			if string(existing) != files[name] {
				stale = append(stale, name)
			}
		}
		if len(stale) > 0 {
			return fmt.Errorf("generated TypeScript contracts are stale: %s (run nr route-info gen-ts --dir %s --output %s)", strings.Join(stale, ", "), projectDir, outputDir)
		}
		fmt.Printf("TypeScript contracts are current: %s (%d routes, %d files)\n", outputDir, len(routes.Routes), len(files))
		return nil
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(outputDir, name), []byte(files[name]), 0o644); err != nil {
			return fmt.Errorf("write generated TypeScript file %s: %w", name, err)
		}
	}
	fmt.Printf("TypeScript contracts written to %s (%d routes, %d files)\n", outputDir, len(routes.Routes), len(files))
	checkOxfmtIgnore(outputDir)
	return nil
}

// writeRouteInfo writes routes in the requested format.  Keeping this in one
// place ensures `route-info` and `route-info export` always behave identically.
func writeRouteInfo(projectRoutes *route_info.ProjectRoutes, cfg *routeInfoConfig) error {
	switch cfg.Format {
	case "json":
		return route_info.WriteJSON(projectRoutes, cfg.Output)
	case "md", "markdown":
		return route_info.WriteMarkdown(projectRoutes, cfg.Output)
	case "curl":
		return route_info.WriteTerminal(projectRoutes, cfg.Output, &route_info.TerminalConfig{
			ServerURL: cfg.Server,
		})
	default:
		return fmt.Errorf("unsupported format: %s (use json, md, or curl)", cfg.Format)
	}
}

// detectServerFromConfig reads the project's config.yml and extracts
// the app.port to build a default server URL like http://localhost:4677.
func detectServerFromConfig(projectDir string) string {
	cfgPath := filepath.Join(projectDir, "config.yml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return ""
	}

	content := string(data)

	// Simple line-by-line parser for the YAML subset used by go-reman.
	const appKey = "app:"
	const portKey = "port:"

	inApp := false
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if trimmed == appKey {
			inApp = true
			continue
		}

		if inApp {
			// If we hit another top-level key (no leading indent), stop.
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				break
			}

			if strings.HasPrefix(trimmed, portKey) {
				portStr := strings.TrimSpace(trimmed[len(portKey):])
				portStr = strings.TrimSuffix(portStr, "\r")
				return fmt.Sprintf("http://localhost:%s", portStr)
			}
		}
	}

	return ""
}

// checkOxfmtIgnore looks for .oxfmtrc.json in ancestor directories of the
// output path and warns the user if the generated files are not in the
// ignorePatterns array.
func checkOxfmtIgnore(outputPath string) {
	// Walk up from outputPath looking for .oxfmtrc.json
	dir := outputPath
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}

	var oxfmtPath string
	for {
		candidate := filepath.Join(dir, ".oxfmtrc.json")
		if _, err := os.Stat(candidate); err == nil {
			oxfmtPath = candidate
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if oxfmtPath == "" {
		return // no .oxfmtrc.json found, nothing to check
	}

	data, err := os.ReadFile(oxfmtPath)
	if err != nil {
		return
	}

	var config struct {
		IgnorePatterns []string `json:"ignorePatterns"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return
	}

	oxfmtDir := filepath.Dir(oxfmtPath)

	// Compute relative path from oxfmtDir to outputPath
	rel, err := filepath.Rel(oxfmtDir, outputPath)
	if err != nil {
		return
	}

	// Use forward slashes for pattern matching (oxfmt uses forward slashes)
	rel = filepath.ToSlash(rel)

	// Check if any pattern covers the output path
	// For directories, try matching rel/*.gen.ts
	testPath := rel
	if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
		testPath = rel + "/*.gen.ts"
	}

	for _, pattern := range config.IgnorePatterns {
		if matched, _ := filepath.Match(pattern, testPath); matched {
			return // covered
		}
	}

	// Not covered, print warning
	fmt.Printf("\n⚠️  Warning: output path %q is not in %s ignorePatterns.\n", rel, oxfmtPath)
	fmt.Printf("   Consider adding %q (or similar) to ignorePatterns to avoid formatting generated files.\n", rel+"/*")
}

func init() {
	routeInfoCmd.Flags().StringP("dir", "d", "", "project directory (default: current dir)")
	routeInfoCmd.Flags().StringP("output", "o", "", "output file path")
	routeInfoCmd.Flags().String("format", "", "output format: json, md, curl (default: curl for stdout, auto-detected from --output extension)")
	routeInfoCmd.Flags().StringP("filter", "f", "", "filter routes by handler name or path (substring match)")
	routeInfoCmd.Flags().StringP("package", "p", "", "filter routes by route package dir, e.g. v1, app, open")
	routeInfoCmd.Flags().StringP("server", "s", "http://localhost:8080", "server URL for curl examples")

	routeInfoExportCmd.Flags().StringP("dir", "d", "", "project directory (default: current dir)")
	routeInfoExportCmd.Flags().StringP("output", "o", "", "output file path")
	routeInfoExportCmd.Flags().String("format", "", "output format: json, md, curl (auto-detected from --output extension)")
	routeInfoExportCmd.Flags().StringP("filter", "f", "", "filter routes by handler name or path")
	routeInfoExportCmd.Flags().StringP("package", "p", "", "filter routes by route package dir, e.g. v1, app, open")
	routeInfoExportCmd.Flags().StringP("server", "s", "http://localhost:8080", "server URL for curl examples")
	_ = routeInfoExportCmd.MarkFlagRequired("output")

	routeInfoGenTSCmd.Flags().StringP("dir", "d", "", "project directory (default: current dir)")
	routeInfoGenTSCmd.Flags().StringP("output", "o", "web/src/api/generated", "output path: a directory writes per-route-file *.gen.ts contracts; a path ending in .ts writes a single file")
	routeInfoGenTSCmd.Flags().StringP("filter", "f", "", "filter routes by handler name or path")
	routeInfoGenTSCmd.Flags().StringP("package", "p", "", "filter routes by route package dir, e.g. v1, app, open")
	routeInfoGenTSCmd.Flags().Bool("check", false, "fail if the generated files are missing or stale")

	routeInfoCmd.AddCommand(routeInfoExportCmd, routeInfoGenTSCmd)
	rootCmd.AddCommand(routeInfoCmd)
}
