package route_info

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ─── JSON output ─────────────────────────────────────────────────────────────

func WriteJSON(routes *ProjectRoutes, outputPath string) error {
	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}

	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func WriteJSONStdout(routes *ProjectRoutes) error {
	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// ─── Markdown output ─────────────────────────────────────────────────────────

func WriteMarkdown(routes *ProjectRoutes, outputPath string) error {
	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}

	content := generateMarkdown(routes)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func WriteMarkdownStdout(routes *ProjectRoutes) {
	fmt.Print(generateMarkdown(routes))
}

func generateMarkdown(routes *ProjectRoutes) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# API Routes\n\n**Module:** %s  \n**Total Routes:** %d\n\n", routes.Module, len(routes.Routes)))

	// Group routes by file for cleaner output
	for _, r := range routes.Routes {
		// Title line: method + full path
		b.WriteString(fmt.Sprintf("## %s **%s**\n\n", r.Method, r.FullPath))

		// Meta info on one line
		source := r.File
		if r.Line > 0 {
			source = fmt.Sprintf("%s:%d", r.File, r.Line)
		}
		b.WriteString(fmt.Sprintf("> `%s` · %s · %s\n\n", r.Handler, r.Group, source))
		if len(r.Middlewares) > 0 {
			b.WriteString(fmt.Sprintf("**Route middleware:** `%s`\n\n", strings.Join(r.Middlewares, "`, `")))
		}

		// ── Request ──
		b.WriteString("### Request\n\n")

		if len(r.Params) == 0 {
			b.WriteString("_No parameters._\n\n")
		} else {
			// Group params by source
			hasBody := false
			hasQuery := false
			hasURI := false
			hasCtx := false
			var bodyParams []ParamInfo
			var queryParams []ParamInfo
			var uriParams []ParamInfo
			var ctxParams []ParamInfo
			var extraParams []ParamInfo

			for _, p := range r.Params {
				switch p.Source {
				case "body":
					hasBody = true
					bodyParams = append(bodyParams, p)
				case "query":
					hasQuery = true
					queryParams = append(queryParams, p)
				case "uri":
					hasURI = true
					uriParams = append(uriParams, p)
				case "context":
					hasCtx = true
					ctxParams = append(ctxParams, p)
				default:
					extraParams = append(extraParams, p)
				}
			}

			// URI params
			if hasURI {
				b.WriteString("**URI**\n\n")
				b.WriteString("| Field | Type |\n")
				b.WriteString("|-------|------|\n")
				for _, p := range uriParams {
					if p.StructType != "" && len(p.Fields) > 0 {
						for _, f := range p.Fields {
							b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", fieldName(f, "uri"), f.Type))
						}
						continue
					}
					typeName := p.Type
					if typeName == "" {
						typeName = "string"
					}
					b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", p.Key, typeName))
				}
				b.WriteString("\n")
			}

			// Query params (struct or simple key-value)
			if hasQuery {
				b.WriteString("**Query**\n\n")
				hasStruct := hasStructQueryParams(queryParams)
				if hasStruct {
					for _, p := range queryParams {
						if p.StructType != "" && len(p.Fields) > 0 {
							b.WriteString(fmt.Sprintf("_Type: `%s`_\n\n", p.StructType))
							b.WriteString("| Field | Type | Required | Tag |\n")
							b.WriteString("|-------|------|----------|-----|\n")
							for _, f := range p.Fields {
								req := ""
								if f.Required {
									req = "✓"
								}
								b.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | `%s` |\n", fieldName(f, "query"), f.Type, req, f.Tag))
							}
							b.WriteString("\n")
						} else {
							// Individual simple keys alongside struct
							b.WriteString("| Field | Type | Default |\n")
							b.WriteString("|-------|------|---------|\n")
							typeName := p.Type
							if typeName == "" {
								typeName = "string"
							}
							b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` |\n", p.Key, typeName, strDefault(p.Default)))
						}
					}
					b.WriteString("\n")
				} else {
					b.WriteString("| Field | Type | Default |\n")
					b.WriteString("|-------|------|---------|\n")
					for _, p := range queryParams {
						typeName := p.Type
						if typeName == "" {
							typeName = "string"
						}
						b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` |\n", p.Key, typeName, strDefault(p.Default)))
					}
					b.WriteString("\n")
				}
			}

			// Body
			if hasBody {
				for _, p := range bodyParams {
					if p.StructType != "" {
						b.WriteString(fmt.Sprintf("**Body** (`%s`)\n\n", p.StructType))
					} else {
						b.WriteString("**Body**\n\n")
					}
					if len(p.Fields) > 0 {
						b.WriteString("| Field | Type | Required | Tag |\n")
						b.WriteString("|-------|------|----------|-----|\n")
						for _, f := range p.Fields {
							req := ""
							if f.Required {
								req = "✓"
							}
							b.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | `%s` |\n", fieldName(f, "query"), f.Type, req, f.Tag))
						}
					} else if p.StructType != "" {
						b.WriteString("_(struct fields not resolved)_\n\n")
					}
					b.WriteString("\n")
				}
			}

			// Context
			if hasCtx {
				b.WriteString("**Context**\n\n")
				b.WriteString("| Key | Type |\n")
				b.WriteString("|-----|------|\n")
				for _, p := range ctxParams {
					b.WriteString(fmt.Sprintf("| `%s` | `-` |\n", p.Key))
				}
				b.WriteString("\n")
			}

			if len(extraParams) > 0 {
				b.WriteString("**Other inputs**\n\n")
				b.WriteString("| Source | Field | Type | Default |\n")
				b.WriteString("|--------|-------|------|---------|\n")
				for _, p := range extraParams {
					if p.StructType != "" {
						b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | |\n", p.Source, p.StructType, p.StructType))
						continue
					}
					typeName := p.Type
					if typeName == "" {
						typeName = "string"
					}
					b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` |\n", p.Source, p.Key, typeName, strDefault(p.Default)))
				}
				b.WriteString("\n")
			}
		}

		// ── Response ──
		b.WriteString("### Response\n\n")

		if len(r.Returns) == 0 {
			b.WriteString("_No return info._\n\n")
		} else {
			for _, ret := range r.Returns {
				label := ret.Description
				switch label {
				case "error":
					label = "Error"
				case "success":
					label = "Success"
				case "custom status":
					label = "Custom Status"
				case "list response":
					label = "Paginated List"
				case "custom error status":
					label = "Error (Custom Code)"
				}

				// Check if it's an error (skip field expansion for errors)
				if ret.Type == "error" || ret.Description == "error" {
					b.WriteString(fmt.Sprintf("- **%s:** `error`\n\n", label))
					continue
				}

				b.WriteString(fmt.Sprintf("**%s** — `%s`\n\n", label, ret.Type))

				if len(ret.Fields) > 0 {
					b.WriteString("| Field | Type | Required | Tag |\n")
					b.WriteString("|-------|------|----------|-----|\n")
					for _, f := range ret.Fields {
						req := ""
						if f.Required {
							req = "✓"
						}
						b.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | `%s` |\n", fieldName(f, "body"), f.Type, req, f.Tag))
					}
					b.WriteString("\n")
				}
			}
		}

		b.WriteString("---\n\n")
	}

	return b.String()
}

// ─── Terminal / curl-friendly output ─────────────────────────────────────────

// TerminalConfig controls the terminal output format.
type TerminalConfig struct {
	ServerURL string // e.g. http://localhost:4677
}

// WriteTerminal writes terminal/curl-friendly output to a file.
func WriteTerminal(routes *ProjectRoutes, outputPath string, cfg *TerminalConfig) error {
	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}

	var b strings.Builder
	if cfg == nil {
		cfg = &TerminalConfig{ServerURL: "http://localhost:8080"}
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "http://localhost:8080"
	}
	server := strings.TrimRight(cfg.ServerURL, "/")

	for _, r := range routes.Routes {
		b.WriteString(generateTerminalRoute(r, server))
		b.WriteString("\n")
	}

	if err := os.WriteFile(outputPath, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func WriteTerminalStdout(routes *ProjectRoutes, cfg *TerminalConfig) {
	if cfg == nil {
		cfg = &TerminalConfig{ServerURL: "http://localhost:8080"}
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "http://localhost:8080"
	}
	server := strings.TrimRight(cfg.ServerURL, "/")

	for _, r := range routes.Routes {
		fmt.Println(generateTerminalRoute(r, server))
	}
}

func generateTerminalRoute(r RouteInfo, server string) string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("### %s %s\n\n", r.Method, r.FullPath))

	// Group params
	hasBody := false
	hasQuery := false
	hasURI := false
	var bodyParams []ParamInfo
	var queryParams []ParamInfo
	var uriParams []ParamInfo

	for _, p := range r.Params {
		switch p.Source {
		case "body":
			hasBody = true
			bodyParams = append(bodyParams, p)
		case "query":
			hasQuery = true
			queryParams = append(queryParams, p)
		case "uri":
			hasURI = true
			uriParams = append(uriParams, p)
		}
	}

	// Build full URL with URI params filled
	fullURL := server + r.FullPath
	for _, p := range uriParams {
		if p.StructType != "" {
			for _, f := range p.Fields {
				key := fieldName(f, "uri")
				fullURL = strings.ReplaceAll(fullURL, ":"+key, "{"+key+"}")
			}
			continue
		}
		fullURL = strings.ReplaceAll(fullURL, ":"+p.Key, "{"+p.Key+"}")
	}

	// Build query string
	var queryParts []string
	for _, p := range queryParams {
		if p.StructType != "" {
			for _, f := range p.Fields {
				key := fieldName(f, "query")
				queryParts = append(queryParts, key+"={"+key+"}")
			}
			continue
		}
		if p.Key != "" {
			val := p.Default
			if val == "" {
				val = "{" + p.Key + "}"
			}
			queryParts = append(queryParts, p.Key+"="+val)
		}
	}
	if len(queryParts) > 0 {
		fullURL += "?" + strings.Join(queryParts, "&")
	}

	// Build curl
	b.WriteString("```sh\n")

	if r.Method == "GET" {
		b.WriteString(fmt.Sprintf("curl '%s'\n", fullURL))
	} else {
		// POST etc.
		b.WriteString(fmt.Sprintf("curl -X %s '%s'", r.Method, fullURL))

		if hasBody {
			b.WriteString(" \\\n  -H 'Content-Type: application/json'")
		}

		// Build JSON body example
		if hasBody {
			jsonStr := buildBodyJSONExample(bodyParams)
			if jsonStr != "" {
				b.WriteString(fmt.Sprintf(" \\\n  -d '%s'", jsonStr))
			}
		}

		b.WriteString("\n")
	}

	b.WriteString("```\n\n")

	// Print body JSON schema
	if hasBody {
		b.WriteString("**Body:**\n\n")
		b.WriteString("```json\n")
		b.WriteString(buildBodyJSONSchema(bodyParams))
		b.WriteString("```\n\n")
	}

	// Print query table (simple text)
	if hasQuery {
		b.WriteString("**Query:**\n\n")
		hasStruct := hasStructQueryParams(queryParams)
		if hasStruct {
			for _, p := range queryParams {
				if p.StructType != "" && len(p.Fields) > 0 {
					b.WriteString(fmt.Sprintf("Type: `%s`\n\n", p.StructType))
					b.WriteString("| Field | Type | Required |\n")
					b.WriteString("|-------|------|----------|\n")
					for _, f := range p.Fields {
						req := ""
						if f.Required {
							req = "✓"
						}
						b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", fieldName(f, "query"), f.Type, req))
					}
				} else if p.Key != "" {
					b.WriteString("| Field | Type | Default |\n")
					b.WriteString("|-------|------|---------|\n")
					b.WriteString(fmt.Sprintf("| %s | string | %s |\n", p.Key, strDefault(p.Default)))
				}
			}
		} else {
			b.WriteString("| Field | Type | Default |\n")
			b.WriteString("|-------|------|---------|\n")
			for _, p := range queryParams {
				b.WriteString(fmt.Sprintf("| %s | string | %s |\n", p.Key, strDefault(p.Default)))
			}
		}
		b.WriteString("\n")
	}

	// Print URI params
	if hasURI {
		b.WriteString("**URI:**\n\n")
		for _, p := range uriParams {
			b.WriteString(fmt.Sprintf("- `%s`: replace `:%s` in URL\n", p.Key, p.Key))
		}
		b.WriteString("\n")
	}

	// ── Response ──
	b.WriteString("**Response:**\n\n")
	for _, ret := range r.Returns {
		if ret.Type == "error" || ret.Description == "error" {
			b.WriteString("```\n{\n  \"code\": 0 | 400 | 405 | ...,\n  \"msg\": \"error message\",\n  \"data\": null\n}\n```\n\n")
			continue
		}
		if ret.Type == "paginated_list" {
			b.WriteString("```\n{\n  \"code\": 200,\n  \"msg\": \"ok\",\n  \"data\": {\n    \"list\": [...],\n    \"total\": 0\n  }\n}\n```\n\n")
			continue
		}

		if len(ret.Fields) > 0 {
			b.WriteString("```json\n{\n")
			b.WriteString("  \"code\": 200,\n")
			b.WriteString("  \"msg\": \"ok\",\n")
			b.WriteString("  \"data\": {\n")
			for i, f := range ret.Fields {
				comma := ","
				if i == len(ret.Fields)-1 {
					comma = ""
				}
				example := jsonExampleValue(f.Type)
				b.WriteString(fmt.Sprintf("    \"%s\": %s%s\n", fieldName(f, "body"), example, comma))
			}
			b.WriteString("  }\n")
			b.WriteString("}\n")
			b.WriteString("```\n\n")
		} else {
			b.WriteString("```json\n")
			b.WriteString("{\n  \"code\": 200,\n  \"msg\": \"ok\",\n  \"data\": {}\n}\n")
			b.WriteString("```\n\n")
		}
	}
	b.WriteString("\n")

	return b.String()
}

// buildBodyJSONExample creates a compact JSON example from body params.
func buildBodyJSONExample(bodyParams []ParamInfo) string {
	return buildBodyJSONSchema(bodyParams)
}

// buildBodyJSONSchema creates a valid, copyable JSON example from body params.
func buildBodyJSONSchema(bodyParams []ParamInfo) string {
	if len(bodyParams) == 0 {
		return "{}\n"
	}

	fields := make([]FieldInfo, 0)
	for _, p := range bodyParams {
		if len(p.Fields) > 0 {
			fields = append(fields, p.Fields...)
		} else if p.StructType != "" {
			fields = append(fields, FieldInfo{Name: p.StructType, Type: "string"})
		}
	}
	if len(fields) == 0 {
		return "{}\n"
	}

	var b strings.Builder
	writeJSONObject(&b, fields, 0)
	b.WriteByte('\n')
	return b.String()
}

func writeJSONObject(b *strings.Builder, fields []FieldInfo, indent int) {
	pad := strings.Repeat("  ", indent)
	b.WriteString("{")
	for i, f := range fields {
		b.WriteByte('\n')
		writeJSONField(b, f, indent+1)
		if i < len(fields)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteByte('\n')
	b.WriteString(pad)
	b.WriteByte('}')
}

// writeJSONField writes a JSON field with recursively expanded nested structs.
func writeJSONField(b *strings.Builder, f FieldInfo, indent int) {
	pad := strings.Repeat("  ", indent)
	jsonName := jsonTagName(f.Tag, f.Name)
	b.WriteString(fmt.Sprintf("%s%q: ", pad, jsonName))
	if len(f.Fields) > 0 {
		if strings.HasPrefix(strings.TrimLeft(f.Type, "*"), "[]") {
			b.WriteByte('[')
			writeJSONObject(b, f.Fields, indent+1)
			b.WriteByte(']')
			return
		}
		writeJSONObject(b, f.Fields, indent)
	} else {
		value := jsonExampleValue(f.Type)
		if value == "string" {
			b.WriteString(fmt.Sprintf("%q", jsonName))
			return
		}
		b.WriteString(value)
	}
}

// jsonTagName extracts the JSON field name from a struct tag.
func jsonTagName(tag string, fallback string) string {
	return tagName(tag, "json", fallback)
}

// fieldName returns the name Gin reads for a field in the specified input
// source. Query binding uses form tags; URI binding uses uri tags; JSON bodies
// use json tags. It falls back through the other common tags for incomplete
// request structs, then to the Go field name.
func fieldName(f FieldInfo, source string) string {
	keys := []string{"json", "form", "uri"}
	switch source {
	case "query", "form":
		keys = []string{"form", "json", "uri"}
	case "uri":
		keys = []string{"uri", "form", "json"}
	}
	for _, key := range keys {
		if name := tagName(f.Tag, key, ""); name != "" {
			return name
		}
	}
	return f.Name
}

func tagName(tag, key, fallback string) string {
	if tag == "" {
		return fallback
	}
	needle := key + `:"`
	idx := strings.Index(tag, needle)
	if idx < 0 {
		return fallback
	}
	rest := tag[idx+len(needle):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return fallback
	}
	name := rest[:end]
	if comma := strings.Index(name, ","); comma >= 0 {
		name = name[:comma]
	}
	if name == "-" || name == "" {
		return fallback
	}
	return name
}

// jsonExampleValue returns an example JSON value for a Go type.
func jsonExampleValue(goType string) string {
	switch {
	case goType == "string" || strings.HasPrefix(goType, "string"):
		return "string"
	case goType == "int" || goType == "int64" || goType == "int32" ||
		goType == "float64" || goType == "float32":
		return "0"
	case goType == "bool":
		return "false"
	case strings.HasPrefix(goType, "[]"):
		return "[]"
	case strings.HasPrefix(goType, "map"):
		return "{}"
	case strings.HasPrefix(goType, "*"):
		return jsonExampleValue(goType[1:])
	default:
		return "\"\""
	}
}

func hasStructQueryParams(params []ParamInfo) bool {
	for _, p := range params {
		if p.StructType != "" && len(p.Fields) > 0 {
			return true
		}
	}
	return false
}

func strDefault(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
