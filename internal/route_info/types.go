package route_info

// RouteInfo holds complete information about one HTTP API route.
type RouteInfo struct {
	Method      string       `json:"method"`                // GET, POST, PUT, DELETE, PATCH
	FullPath    string       `json:"full_path"`             // full URL path with group prefix, e.g. /v1/user/login
	Path        string       `json:"path"`                  // relative path within group, e.g. /login
	Handler     string       `json:"handler"`               // handler function name
	Group       string       `json:"group"`                 // public, auth, admin
	Middlewares []string     `json:"middlewares,omitempty"` // route-level middleware expressions
	File        string       `json:"file"`                  // source file (relative to project root)
	Line        int          `json:"line,omitempty"`        // registration source line
	Params      []ParamInfo  `json:"params,omitempty"`      // extracted parameters
	Returns     []ReturnInfo `json:"returns,omitempty"`     // extracted return info
}

// ParamInfo describes a parameter extracted from a handler.
type ParamInfo struct {
	Source     string      `json:"source"`                // body, query, uri, context
	StructType string      `json:"struct_type,omitempty"` // struct type name for body/query structs
	Fields     []FieldInfo `json:"fields,omitempty"`      // struct fields when resolvable
	Key        string      `json:"key,omitempty"`         // param key for Query/Param/Get
	Type       string      `json:"type,omitempty"`        // scalar type when it is known
	Default    string      `json:"default,omitempty"`     // default value for DefaultQuery
	Package    string      `json:"package,omitempty"`     // full package path
}

// FieldInfo describes a single struct field.
type FieldInfo struct {
	Name     string      `json:"name"`               // field name
	Type     string      `json:"type"`               // field type as string
	Tag      string      `json:"tag,omitempty"`      // struct tag (json/form/binding)
	Required bool        `json:"required,omitempty"` // whether binding:"required" is set
	Fields   []FieldInfo `json:"fields,omitempty"`   // nested struct fields
}

// ReturnInfo describes a return value from a handler.
type ReturnInfo struct {
	Type        string      `json:"type"`                  // Go type of returned data
	Description string      `json:"description,omitempty"` // contextual info (e.g. "error", "list response", "success")
	Fields      []FieldInfo `json:"fields,omitempty"`      // struct fields when resolvable
}

// ProjectRoutes holds all routes extracted from a project.
type ProjectRoutes struct {
	Module string      `json:"module"`
	Routes []RouteInfo `json:"routes"`
}
