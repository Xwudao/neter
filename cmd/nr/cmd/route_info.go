package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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

type routeInfoConfig struct {
	Dir    string
	Output string
	Format string
	Filter string
	Server string
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
		Dir:    dir,
		Output: output,
		Format: format,
		Filter: filter,
		Server: server,
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
	if cfg.Filter != "" {
		projectRoutes = route_info.ApplyFilter(projectRoutes, &route_info.FilterOption{Keyword: cfg.Filter})
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

	if cfg.Filter != "" {
		projectRoutes = route_info.ApplyFilter(projectRoutes, &route_info.FilterOption{Keyword: cfg.Filter})
	}

	if err := writeRouteInfo(projectRoutes, cfg); err != nil {
		return err
	}
	fmt.Printf("route info written to %s (%d routes)\n", cfg.Output, len(projectRoutes.Routes))

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
	for _, line := range strings.Split(content, "\n") {
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

func init() {
	routeInfoCmd.Flags().StringP("dir", "d", "", "project directory (default: current dir)")
	routeInfoCmd.Flags().StringP("output", "o", "", "output file path")
	routeInfoCmd.Flags().String("format", "", "output format: json, md, curl (default: curl for stdout, auto-detected from --output extension)")
	routeInfoCmd.Flags().StringP("filter", "f", "", "filter routes by handler name or path (substring match)")
	routeInfoCmd.Flags().StringP("server", "s", "http://localhost:8080", "server URL for curl examples")

	routeInfoExportCmd.Flags().StringP("dir", "d", "", "project directory (default: current dir)")
	routeInfoExportCmd.Flags().StringP("output", "o", "", "output file path")
	routeInfoExportCmd.Flags().String("format", "", "output format: json, md, curl (auto-detected from --output extension)")
	routeInfoExportCmd.Flags().StringP("filter", "f", "", "filter routes by handler name or path")
	routeInfoExportCmd.Flags().StringP("server", "s", "http://localhost:8080", "server URL for curl examples")
	_ = routeInfoExportCmd.MarkFlagRequired("output")

	routeInfoCmd.AddCommand(routeInfoExportCmd)
	rootCmd.AddCommand(routeInfoCmd)
}
