package wire2loom

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/imports"
)

// entry is one constructor listed in a provider set.
type entry struct {
	text string
	// as holds the interface text when the constructor is bound to one,
	// which turns loom.Provide into loom.As[as].
	as string
	// start and end are byte offsets of the argument in its file.
	start, end int
}

// dropped is an argument removed from the output, such as a wire.Bind.
type dropped struct {
	start, end int
}

// setModel is a `var X = wire.NewSet(...)` declaration.
type setModel struct {
	src   *source
	name  string
	start int
	end   int
	// funStart and funEnd bound the wire.NewSet callee, which becomes loom.Module.
	funStart, funEnd int
	entries          []entry
	dropped          []dropped
}

// option is one argument of a wire.Build call.
type option struct {
	text string
	as   string
}

// injectorModel is one Wire injector stub.
type injectorModel struct {
	src     *source
	name    string
	target  string
	start   int
	end     int
	options []option
	binds   []bindRef
}

// bindRef is a parsed wire.Bind(new(iface), new(*concrete)).
type bindRef struct {
	iface string
	// impl is the implementation type name, such as "LogBroadcaster".
	impl string
	// raw is the implementation reference as written, such as
	// "*biz.PayBiz", whose qualifier the graph file already imports.
	raw string
}

func buildPlan(root string) (*plan, error) {
	sources, err := parseSources(root)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, errors.New("no files reference wire; nothing to convert")
	}

	var sets []*setModel
	var injectors []*injectorModel
	for _, src := range sources {
		fileSets, err := parseSets(src)
		if err != nil {
			return nil, err
		}
		sets = append(sets, fileSets...)
		fileInjectors, err := parseInjectors(src)
		if err != nil {
			return nil, err
		}
		injectors = append(injectors, fileInjectors...)
	}
	if len(sets) == 0 && len(injectors) == 0 {
		return nil, errors.New("no wire.NewSet or wire.Build declarations found")
	}

	registry := make(map[string]*setModel, len(sets))
	for _, set := range sets {
		if previous, clash := registry[set.name]; clash {
			return nil, fmt.Errorf("provider sets %s and %s share the name %s; rename one first",
				relPath(root, previous.src.path), relPath(root, set.src.path), set.name)
		}
		registry[set.name] = set
	}
	for _, injector := range injectors {
		if err := resolveGraphBinds(injector, registry); err != nil {
			return nil, err
		}
	}

	setsByFile := map[string][]*setModel{}
	for _, set := range sets {
		setsByFile[set.src.path] = append(setsByFile[set.src.path], set)
	}
	injectorsByFile := map[string][]*injectorModel{}
	for _, injector := range injectors {
		injectorsByFile[injector.src.path] = append(injectorsByFile[injector.src.path], injector)
	}
	hasInjectors := make(map[string]bool, len(injectorsByFile))
	for path := range injectorsByFile {
		hasInjectors[path] = true
	}

	p := &plan{root: root, changes: map[string][]byte{}, graphs: map[string][]string{}}
	if err := p.addProviderRewrites(setsByFile, hasInjectors); err != nil {
		return nil, err
	}
	if err := p.addGraphRewrites(injectorsByFile, setsByFile, registry); err != nil {
		return nil, err
	}
	if err := p.addCallSiteRewrites(injectors); err != nil {
		return nil, err
	}
	if err := p.addRemoveRewrites(); err != nil {
		return nil, err
	}
	if err := p.addMakefileRewrite(); err != nil {
		return nil, err
	}
	if err := p.noteLeftovers(); err != nil {
		return nil, err
	}
	return p, nil
}

// leftoverPattern finds prose that still refers to Wire after the conversion.
var leftoverPattern = regexp.MustCompile(`(?i)\bwire\b`)

// noteLeftovers lists files that still mention Wire, so a stale comment or a
// hand-written helper is never silently left behind.
func (p *plan) noteLeftovers() error {
	planned := p.changes
	var found []string
	err := filepath.WalkDir(p.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == p.root {
				return nil
			}
			if skipDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".md":
		default:
			return nil
		}
		if filepath.Base(path) == "wire_gen.go" {
			return nil
		}
		content, ok := planned[path]
		if !ok {
			if content, err = os.ReadFile(path); err != nil {
				return err
			}
		}
		if content == nil {
			return nil
		}
		if leftoverPattern.Match(content) {
			found = append(found, relPath(p.root, path))
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(found) > 0 {
		sort.Strings(found)
		p.notes = append(p.notes, "still mention Wire, by hand or in a comment: "+strings.Join(found, ", "))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Parsing
// ---------------------------------------------------------------------------

func parseSets(src *source) ([]*setModel, error) {
	var sets []*setModel
	for _, decl := range src.file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
				continue
			}
			call, ok := valueSpec.Values[0].(*ast.CallExpr)
			if !ok || !isWireCall(call.Fun, "NewSet") {
				continue
			}
			set, err := parseSet(src, valueSpec.Names[0].Name, call)
			if err != nil {
				return nil, err
			}
			sets = append(sets, set)
		}
	}
	return sets, nil
}

func parseSet(src *source, name string, call *ast.CallExpr) (*setModel, error) {
	set := &setModel{
		src: src, name: name,
		start: src.offset(call.Pos()), end: src.offset(call.End()),
		funStart: src.offset(call.Fun.Pos()), funEnd: src.offset(call.Fun.End()),
	}
	var pending []bindRef
	for _, arg := range call.Args {
		if bind, ok := asWireBind(src, arg); ok {
			pending = append(pending, bind)
			set.dropped = append(set.dropped, dropped{
				start: src.offset(arg.Pos()), end: src.offset(arg.End()),
			})
			continue
		}
		set.entries = append(set.entries, entry{
			text:  src.text(arg),
			start: src.offset(arg.Pos()), end: src.offset(arg.End()),
		})
	}
	for _, bind := range pending {
		index := indexOfConstructor(set.entries, bind.impl)
		if index < 0 {
			return nil, fmt.Errorf(
				"%s: %s binds %s, but no constructor for it is in the same provider set; "+
					"add it to %s first, or bind it in the graph that provides it",
				src.path, name, bind.impl, name)
		}
		if set.entries[index].as != "" {
			return nil, fmt.Errorf("%s: %s is bound to more than one interface", src.path, bind.impl)
		}
		set.entries[index].as = bind.iface
	}
	return set, nil
}

func parseInjectors(src *source) ([]*injectorModel, error) {
	var injectors []*injectorModel
	for _, decl := range src.file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil || funcDecl.Recv != nil {
			continue
		}
		build := findWireBuild(funcDecl)
		if build == nil {
			continue
		}
		target, err := injectorTarget(src, funcDecl)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", src.path, err)
		}
		injector := &injectorModel{
			src: src, name: funcDecl.Name.Name, target: target,
			start: src.offset(funcDecl.Pos()), end: src.offset(funcDecl.End()),
		}
		for _, arg := range build.Args {
			if bind, ok := asWireBind(src, arg); ok {
				injector.binds = append(injector.binds, bind)
				continue
			}
			injector.options = append(injector.options, option{text: src.text(arg)})
		}
		injectors = append(injectors, injector)
	}
	return injectors, nil
}

// injectorTarget reads the first result type of an injector stub.
func injectorTarget(src *source, funcDecl *ast.FuncDecl) (string, error) {
	results := funcDecl.Type.Results
	if results == nil || len(results.List) == 0 {
		return "", fmt.Errorf("injector %s has no results", funcDecl.Name.Name)
	}
	if len(results.List) != 3 {
		return "", fmt.Errorf("injector %s must return (value, func(), error) to be converted", funcDecl.Name.Name)
	}
	return src.text(results.List[0].Type), nil
}

func findWireBuild(funcDecl *ast.FuncDecl) *ast.CallExpr {
	var found *ast.CallExpr
	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		if found != nil {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok || !isWireCall(call.Fun, "Build") {
			return true
		}
		found = call
		return false
	})
	return found
}

// asWireBind parses wire.Bind(new(Iface), new(*Impl)).
func asWireBind(src *source, arg ast.Expr) (bindRef, bool) {
	call, ok := arg.(*ast.CallExpr)
	if !ok || !isWireCall(call.Fun, "Bind") || len(call.Args) != 2 {
		return bindRef{}, false
	}
	iface, ok := newArgument(src, call.Args[0])
	if !ok {
		return bindRef{}, false
	}
	concrete, ok := newArgument(src, call.Args[1])
	if !ok {
		return bindRef{}, false
	}
	impl := typeName(concrete)
	if impl == "" {
		return bindRef{}, false
	}
	return bindRef{iface: iface, impl: impl, raw: concrete}, true
}

// newArgument unwraps new(T) into the text of T.
func newArgument(src *source, arg ast.Expr) (string, bool) {
	call, ok := arg.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	if ident, ok := call.Fun.(*ast.Ident); !ok || ident.Name != "new" || len(call.Args) != 1 {
		return "", false
	}
	return src.text(call.Args[0]), true
}

// typeName turns "*logger.LogBroadcaster" into "LogBroadcaster".
func typeName(concrete string) string {
	name := strings.TrimPrefix(strings.TrimSpace(concrete), "*")
	if index := strings.LastIndexByte(name, '.'); index >= 0 {
		name = name[index+1:]
	}
	return name
}

// matchesConstructor reports whether a provider entry constructs the named type.
//
// Wire binds a type, not a constructor, so the entry has to be found by name:
// NewLogBroadcaster constructs LogBroadcaster, and newStore constructs store.
// Matching case-insensitively after the "New" prefix covers both spellings
// without inventing a constructor name that may not exist.
func matchesConstructor(text, typeName string) bool {
	name := bareName(text)
	if len(name) <= 3 || !strings.EqualFold(name[:3], "new") {
		return false
	}
	return strings.EqualFold(name[3:], typeName)
}

func indexOfConstructor(entries []entry, typeName string) int {
	for i, e := range entries {
		if matchesConstructor(e.text, typeName) {
			return i
		}
	}
	return -1
}

// resolveGraphBinds turns a graph-level wire.Bind into a loom.As option.
//
// The binding stays in the graph rather than moving next to the constructor,
// because the interface package frequently imports the constructor's package:
// moving it would make that package import the interface back, which Go rejects
// as a cycle. Loom allows a graph to expose a module's constructor this way, so
// the Wire shape converts one to one.
func resolveGraphBinds(injector *injectorModel, registry map[string]*setModel) error {
	for _, bind := range injector.binds {
		// A constructor listed directly in the graph is folded into its entry.
		matched := false
		for i, opt := range injector.options {
			if !matchesConstructor(opt.text, bind.impl) {
				continue
			}
			injector.options[i].as = bind.iface
			matched = true
			break
		}
		if matched {
			continue
		}
		ref, err := boundConstructorRef(bind, injector, registry)
		if err != nil {
			return err
		}
		injector.options = append(injector.options, option{text: ref, as: bind.iface})
	}
	return nil
}

// boundConstructorRef writes the reference to a bound implementation as it must
// appear in the graph file.
//
// The qualifier comes from the wire.Bind argument, which already names the
// implementation's package the way the graph imports it. The constructor name
// comes from the provider set that declares it when the converter knows one, so
// an unconventional name still resolves; otherwise it falls back to the New<Type>
// convention and lets the verification build reject a wrong guess.
func boundConstructorRef(bind bindRef, injector *injectorModel, registry map[string]*setModel) (string, error) {
	qualifier, typeName := splitQualified(bind.raw)
	if name := declaringConstructor(registry, injector, bind.impl); name != "" {
		return qualify(qualifier, bareName(name)), nil
	}
	if qualifier == "" {
		return "", fmt.Errorf(
			"%s: %s binds %s, but the converter cannot tell which constructor builds it; "+
				"add loom.As[%s](<constructor>) by hand",
			injector.src.path, injector.name, bind.impl, bind.iface)
	}
	return qualifier + ".New" + typeName, nil
}

// declaringConstructor finds the constructor a provider set in this graph
// declares for the bound type.
func declaringConstructor(registry map[string]*setModel, injector *injectorModel, impl string) string {
	for _, opt := range injector.options {
		set, ok := registry[bareName(opt.text)]
		if !ok {
			continue
		}
		if index := indexOfConstructor(set.entries, impl); index >= 0 {
			return set.entries[index].text
		}
	}
	return ""
}

// splitQualified splits "*biz.PayBiz" into its package qualifier ("biz") and
// type name ("PayBiz"). The pointer belongs to the bound type, not to the
// constructor reference being built.
func splitQualified(ref string) (string, string) {
	name := strings.TrimPrefix(strings.TrimSpace(ref), "*")
	index := strings.LastIndexByte(name, '.')
	if index < 0 {
		return "", name
	}
	return name[:index], name[index+1:]
}

func qualify(qualifier, name string) string {
	if qualifier == "" {
		return name
	}
	return qualifier + "." + name
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// addProviderRewrites converts the sets in files that hold nothing else.
//
// A file that also declares injectors is handled by addGraphRewrites, which has
// to produce the graph file from the same edit pass; writing it here as well
// would plan two changes for one path and lose one of them.
func (p *plan) addProviderRewrites(setsByFile map[string][]*setModel, hasInjectors map[string]bool) error {
	paths := make([]string, 0, len(setsByFile))
	for path := range setsByFile {
		if hasInjectors[path] {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		sets := setsByFile[path]
		content, err := renderProviderFile(sets[0].src, sets)
		if err != nil {
			return err
		}
		if err := p.set(path, content); err != nil {
			return err
		}
	}
	return nil
}

// providerEdits rewrites every set declared in src: the callee becomes
// loom.Module and each constructor is wrapped in loom.Provide, or loom.As when
// the set binds it to an interface.
func providerEdits(src *source, sets []*setModel) []edit {
	var edits []edit
	for _, set := range sets {
		edits = append(edits, edit{start: set.funStart, end: set.funEnd, text: "loom.Module"})
		for _, e := range set.entries {
			prefix := "loom.Provide("
			if e.as != "" {
				prefix = "loom.As[" + e.as + "]("
			}
			edits = append(edits,
				edit{start: e.start, end: e.start, text: prefix},
				edit{start: e.end, end: e.end, text: ")"},
			)
		}
		for _, dropped := range set.dropped {
			start, end := dropRange(src.src, dropped.start, dropped.end)
			edits = append(edits, edit{start: start, end: end})
		}
	}
	return edits
}

// edit is one textual splice into a file.
type edit struct {
	start, end int
	text       string
}

// renderProviderFile rewrites every set in a file in place, wrapping each
// constructor and dropping the bind entries.
//
// Editing arguments individually rather than replacing the whole call keeps the
// order of the entries and any comments written between them, which is exactly
// the part of a provider set a reader cares about.
func renderProviderFile(src *source, sets []*setModel) ([]byte, error) {
	return processImports(src.path, applyEdits(src.src, providerEdits(src, sets)))
}

// dropRange widens a removed argument to swallow one comma, so the argument
// list stays syntactically valid.
func dropRange(src []byte, start, end int) (int, int) {
	// Prefer the comma that precedes the argument; every removed entry in a
	// converted set follows at least one kept entry in practice, and removing
	// the comma after the previous entry leaves the list well formed.
	i := start - 1
	for i >= 0 && isSpaceByte(src[i]) {
		i--
	}
	if i >= 0 && src[i] == ',' {
		return i, end
	}
	// Fall back to the comma that follows, for a leading argument.
	j := end
	for j < len(src) && isSpaceByte(src[j]) {
		j++
	}
	if j < len(src) && src[j] == ',' {
		return start, j + 1
	}
	return start, end
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// applyEdits splices edits into src. Later positions are applied first so the
// offsets of the remaining edits stay valid.
func applyEdits(src []byte, edits []edit) []byte {
	sort.Slice(edits, func(i, j int) bool {
		if edits[i].start != edits[j].start {
			return edits[i].start > edits[j].start
		}
		return edits[i].end > edits[j].end
	})
	out := append([]byte(nil), src...)
	for _, e := range edits {
		out = append(out[:e.start], append([]byte(e.text), out[e.end:]...)...)
	}
	return out
}

func (p *plan) addGraphRewrites(injectorsByFile map[string][]*injectorModel, setsByFile map[string][]*setModel, registry map[string]*setModel) error {
	paths := make([]string, 0, len(injectorsByFile))
	for path := range injectorsByFile {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		fileInjectors := injectorsByFile[path]
		src := fileInjectors[0].src
		graphPath := filepath.Join(filepath.Dir(path), "graph.go")
		if _, err := os.Stat(graphPath); err == nil {
			return fmt.Errorf("%s already exists; convert that package by hand", graphPath)
		}
		content, err := renderGraphFile(src, fileInjectors, setsByFile[path], registry)
		if err != nil {
			return err
		}
		if err := p.set(graphPath, content); err != nil {
			return err
		}
		if err := p.remove(path); err != nil {
			return err
		}
		names := make([]string, 0, len(fileInjectors))
		for _, injector := range fileInjectors {
			names = append(names, injector.name)
			if graphVarName(injector.name) == "" {
				return fmt.Errorf("%s: cannot derive a graph variable from %s", path, injector.name)
			}
		}
		sort.Strings(names)
		p.graphs[filepath.Dir(path)] = names
	}
	return nil
}

// renderGraphFile rewrites an injector file into a graph file. The original
// imports are kept, so every package the graph references stays available;
// goimports then drops whatever the rewrite made unused.
func renderGraphFile(src *source, injectors []*injectorModel, sets []*setModel, registry map[string]*setModel) ([]byte, error) {
	ordered := append([]*injectorModel(nil), injectors...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].start > ordered[j].start })

	var block strings.Builder
	block.WriteString("//go:generate go tool loom generate .\n\n")
	block.WriteString("var (\n")
	names := append([]*injectorModel(nil), injectors...)
	sort.Slice(names, func(i, j int) bool { return names[i].name < names[j].name })
	for i, injector := range names {
		if i > 0 {
			block.WriteString("\n")
		}
		fmt.Fprintf(&block, "\t%s = loom.Graph[%s](\n\t\tloom.Name(%q),\n",
			graphVarName(injector.name), injector.target, injector.name)
		for _, opt := range injector.options {
			if opt.as != "" {
				fmt.Fprintf(&block, "\t\tloom.As[%s](%s),\n", opt.as, opt.text)
				continue
			}
			if isModuleReference(opt.text, registry) {
				fmt.Fprintf(&block, "\t\t%s,\n", opt.text)
				continue
			}
			fmt.Fprintf(&block, "\t\tloom.Provide(%s),\n", opt.text)
		}
		block.WriteString("\t)\n")
	}
	block.WriteString(")")

	// Injectors and provider sets are rewritten in one pass over the original
	// offsets. A wireinject file may declare both, and a set it declares is
	// referenced by the graphs that move into the new file, so it has to be
	// converted rather than dropped with the stub.
	edits := providerEdits(src, sets)
	for i, injector := range ordered {
		replacement := ""
		if i == len(ordered)-1 {
			replacement = block.String()
		}
		edits = append(edits, edit{start: injector.start, end: injector.end, text: replacement})
	}
	out := applyEdits(src.src, edits)
	out = stripBuildConstraint(out)
	out = bytesReplaceImport(out, wireImport, loomImport)
	return processImports(src.path, out)
}

// stripBuildConstraint removes the //go:build wireinject lines that kept the
// stub out of the normal build.
func stripBuildConstraint(src []byte) []byte {
	var out []string
	for _, line := range strings.Split(string(src), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//go:build") && strings.Contains(trimmed, "wireinject") {
			continue
		}
		if strings.HasPrefix(trimmed, "// +build") && strings.Contains(trimmed, "wireinject") {
			continue
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n"))
}

func bytesReplaceImport(src []byte, from, to string) []byte {
	return []byte(strings.Replace(string(src), `"`+from+`"`, `"`+to+`"`, 1))
}

// processImports formats a rewritten file and repairs its import block.
func processImports(path string, src []byte) ([]byte, error) {
	out, err := imports.Process(path, src, nil)
	if err != nil {
		return nil, fmt.Errorf("format %s: %w", path, err)
	}
	return out, nil
}

// graphVarName derives the package-level variable for an injector.
func graphVarName(injector string) string {
	if injector == "" {
		return ""
	}
	first := strings.ToLower(injector[:1])
	return first + injector[1:] + "Graph"
}

// isModuleReference reports whether a graph option is a provider module rather
// than a constructor.
//
// The registry is authoritative: it holds every wire.NewSet variable found in
// the project. Naming alone cannot decide, because this family of projects has
// both ProviderXSet modules and ProvideX constructors. The suffix check only
// covers a set imported from another module, which keep their conventional
// name; anything else is treated as a constructor.
func isModuleReference(text string, registry map[string]*setModel) bool {
	name := bareName(text)
	if strings.ContainsAny(name, "[]()") {
		return false
	}
	if _, ok := registry[name]; ok {
		return true
	}
	return strings.HasSuffix(name, "Set") || strings.HasSuffix(name, "Sets")
}

// ---------------------------------------------------------------------------
// Call sites, removals and the Makefile
// ---------------------------------------------------------------------------

// addCallSiteRewrites updates the callers of the converted injectors.
//
// Call sites are found by text rather than by parsing every file in the project:
// they live in ordinary files that never mention wire, so scanning only the DI
// files would miss them entirely.
func (p *plan) addCallSiteRewrites(injectors []*injectorModel) error {
	if len(injectors) == 0 {
		return nil
	}
	names := make([]string, 0, len(injectors))
	injectorFiles := make(map[string]bool, len(injectors))
	for _, injector := range injectors {
		names = append(names, regexp.QuoteMeta(injector.name))
		injectorFiles[injector.src.path] = true
	}
	sort.Strings(names)
	// The optional "var" is captured so it is preserved: `var app, f, err = ...`
	// must stay a declaration, and the cleanup variable is captured too because
	// projects name it anything from f to cleanup to cl.
	callPattern := regexp.MustCompile(
		`(?m)^([ \t]*)(var[ \t]+)?([\w.]+),[ \t]*(\w+),[ \t]*err[ \t]*(:=|=)[ \t]*` +
			`((?:[\w.]+\.)?(?:` + strings.Join(names, "|") + `)\(\))`)

	var touched []string
	err := filepath.WalkDir(p.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == p.root {
				return nil
			}
			if skipDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || injectorFiles[path] {
			return nil
		}
		if base := entry.Name(); base == "wire.go" || base == "wire_gen.go" {
			return nil
		}
		src, err := p.read(path)
		if err != nil {
			return err
		}
		edits, changed := callSiteEdits(src, callPattern)
		if !changed {
			return nil
		}
		content, err := processImports(path, applyEdits(src, edits))
		if err != nil {
			return err
		}
		p.update(path, content)
		touched = append(touched, relPath(p.root, path))
		return nil
	})
	if err != nil {
		return err
	}
	if len(touched) > 0 {
		sort.Strings(touched)
		p.notes = append(p.notes, fmt.Sprintf(
			"call sites updated to *loom.Lifecycle in %s; add lifecycle.Start(ctx) there if the graph gains OnStart hooks",
			strings.Join(touched, ", ")))
	}
	return nil
}

// callSiteEdits renames each injector's cleanup result to lifecycle and turns
// the defer that called it into lifecycle.Stop.
//
// The variable is found by position rather than by name, because the second
// result is called f, cleanup, cl and anything else depending on the author.
func callSiteEdits(src []byte, callPattern *regexp.Regexp) ([]edit, bool) {
	var edits []edit
	for _, match := range callPattern.FindAllSubmatchIndex(src, -1) {
		nameStart, nameEnd := match[2*4], match[2*4+1]
		old := string(src[nameStart:nameEnd])
		// `_, err := ...` discards the lifecycle; renaming would leave the
		// variable unused, and there is no defer to convert either.
		if old == "_" || old == "lifecycle" {
			continue
		}
		edits = append(edits, edit{start: nameStart, end: nameEnd, text: "lifecycle"})

		call := []byte("defer " + old + "()")
		stop := "defer func() { _ = lifecycle.Stop(context.Background()) }()"
		if offset := bytes.Index(src[nameEnd:], call); offset >= 0 {
			start := nameEnd + offset
			edits = append(edits, edit{start: start, end: start + len(call), text: stop})
		}
	}
	return edits, len(edits) > 0
}

// addRemoveRewrites deletes Wire's generated files.
func (p *plan) addRemoveRewrites() error {
	count := 0
	err := filepath.WalkDir(p.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == p.root {
				return nil
			}
			if skipDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(path) == "wire_gen.go" {
			if err := p.remove(path); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	if err != nil {
		return err
	}
	if count > 0 {
		p.notes = append(p.notes, "deleted Wire's generated files; loom_gen.go replaces them")
	}
	return nil
}

var (
	makefileWireTarget = regexp.MustCompile(`(?m)^wire:([^\n]*)\n((?:\t[^\n]*\n)+)`)
	makefileWireInstal = regexp.MustCompile(`(?m)^\tgo install github\.com/google/wire/cmd/wire@latest\n`)
	makefilePhony      = regexp.MustCompile(`(?m)^(\.PHONY:[^\n]*?)\swire\b`)
)

// addMakefileRewrite swaps a `wire` target for a `loom` one. Projects without a
// Makefile, or with a target the pattern does not match, keep it and are
// reported instead.
func (p *plan) addMakefileRewrite() error {
	path := filepath.Join(p.root, "Makefile")
	src, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	updated := string(src)
	updated = makefileWireInstal.ReplaceAllString(updated, "")
	updated = makefilePhony.ReplaceAllString(updated, "$1 loom")
	updated = makefileWireTarget.ReplaceAllString(updated,
		"loom:$1\n\tgo tool loom generate ./...\n")
	if updated == string(src) {
		if strings.Contains(string(src), "wire") {
			p.notes = append(p.notes, "Makefile: replace the `wire` target with `go tool loom generate ./...`")
		}
		return nil
	}
	if err := p.set(path, []byte(updated)); err != nil {
		return err
	}
	p.notes = append(p.notes, "Makefile: `wire` target replaced with `loom`")
	return nil
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

func isWireCall(fun ast.Expr, name string) bool {
	selector, ok := unwrapIndex(fun).(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != name {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == "wire"
}

func unwrapIndex(expr ast.Expr) ast.Expr {
	switch value := expr.(type) {
	case *ast.IndexExpr:
		return value.X
	case *ast.IndexListExpr:
		return value.X
	default:
		return expr
	}
}

// bareName strips any package qualifier, type arguments and pointer prefix.
func bareName(text string) string {
	name := text
	if index := strings.IndexByte(name, '['); index >= 0 {
		name = name[:index]
	}
	if index := strings.LastIndexByte(name, '.'); index >= 0 {
		name = name[index+1:]
	}
	return strings.TrimPrefix(name, "*")
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func runGo(dir string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
