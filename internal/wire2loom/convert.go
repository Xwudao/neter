// Package wire2loom rewrites a project's dependency injection declarations from
// github.com/google/wire to github.com/Xwudao/loom.
//
// The converter is deliberately conservative. It understands the shapes this
// project family uses — wire.NewSet provider sets, wire.Bind interface
// bindings, wireinject injector stubs, and the "value, cleanup, error" call
// pattern — and refuses, without writing anything, when it meets anything else.
// Every rewrite is validated by regenerating the graphs and building the
// project; a failure restores the tree exactly as it was.
package wire2loom

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Result summarises what a conversion did, or would do.
type Result struct {
	// Rewritten lists existing files whose contents changed.
	Rewritten []string
	// Created lists files that did not exist before, such as each graph.go.
	Created []string
	// Removed lists files that were deleted, such as wire.go and wire_gen.go.
	Removed []string
	// Graphs maps a package directory to the constructor names generated in it.
	Graphs map[string][]string
	// Notes lists follow-ups the converter could not do automatically.
	Notes []string
	// DryRun reports that nothing was written.
	DryRun bool
}

// Options controls a conversion.
type Options struct {
	// Root is the project directory containing go.mod.
	Root string
	// Apply writes the changes. When false the conversion is only planned.
	Apply bool
	// LoomVersion is the module version to depend on.
	LoomVersion string
	// SkipVerify leaves `go mod tidy`, graph generation and the build to the
	// caller. Useful offline, at the cost of losing the safety net.
	SkipVerify bool
}

const (
	defaultLoomVersion = "latest"
	wireImport         = "github.com/google/wire"
	loomImport         = "github.com/Xwudao/loom"
)

// Convert rewrites the project at opts.Root.
func Convert(opts Options) (*Result, error) {
	plan, err := buildPlan(opts.Root)
	if err != nil {
		return nil, err
	}
	result := plan.result()
	result.DryRun = !opts.Apply
	if !opts.Apply {
		return result, nil
	}

	snapshot, err := takeSnapshot(plan.touched())
	if err != nil {
		return nil, err
	}
	if err := plan.apply(); err != nil {
		return nil, errors.Join(err, snapshot.restore())
	}
	if err := opts.verify(plan.root); err != nil {
		return nil, errors.Join(err, snapshot.restore())
	}
	return result, nil
}

// verify regenerates the project so a conversion that does not compile is never
// left on disk.
func (o Options) verify(root string) error {
	if o.SkipVerify {
		return nil
	}
	version := o.LoomVersion
	if version == "" {
		version = defaultLoomVersion
	}
	steps := [][]string{
		{"mod", "edit", "-droprequire=" + wireImport},
		{"get", loomImport + "@" + version},
		{"get", "-tool", loomImport + "/cmd/loom"},
		{"mod", "tidy"},
		{"tool", "loom", "generate", "./..."},
		{"build", "./..."},
		{"vet", "./..."},
	}
	for _, args := range steps {
		if out, err := runGo(root, args...); err != nil {
			return fmt.Errorf("go %s failed after rewriting:\n%s\n%w", strings.Join(args, " "), out, err)
		}
	}
	return nil
}

// change is one planned file operation. A nil content means delete.
type change struct {
	path    string
	content []byte
}

type plan struct {
	root    string
	changes []change
	graphs  map[string][]string
	notes   []string
}

func (p *plan) touched() []string {
	paths := make([]string, 0, len(p.changes)+2)
	for _, c := range p.changes {
		paths = append(paths, c.path)
	}
	// go mod tidy rewrites these, so they have to be restorable too.
	paths = append(paths, filepath.Join(p.root, "go.mod"), filepath.Join(p.root, "go.sum"))
	return paths
}

func (p *plan) result() *Result {
	res := &Result{Graphs: p.graphs, Notes: p.notes}
	for _, c := range p.changes {
		rel := relPath(p.root, c.path)
		switch {
		case c.content == nil:
			res.Removed = append(res.Removed, rel)
		default:
			if _, err := os.Stat(c.path); errors.Is(err, fs.ErrNotExist) {
				res.Created = append(res.Created, rel)
				continue
			}
			res.Rewritten = append(res.Rewritten, rel)
		}
	}
	sort.Strings(res.Created)
	sort.Strings(res.Rewritten)
	sort.Strings(res.Removed)
	return res
}

func (p *plan) apply() error {
	for _, c := range p.changes {
		if c.content == nil {
			if err := os.Remove(c.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("remove %s: %w", c.path, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(c.path, c.content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", c.path, err)
		}
	}
	return nil
}

// snapshot records the original contents of every path a conversion may touch.
type snapshot struct {
	files map[string]fileState
}

type fileState struct {
	content []byte
	existed bool
}

func takeSnapshot(paths []string) (*snapshot, error) {
	snap := &snapshot{files: make(map[string]fileState, len(paths))}
	for _, path := range paths {
		if _, seen := snap.files[path]; seen {
			continue
		}
		content, err := os.ReadFile(path)
		switch {
		case err == nil:
			snap.files[path] = fileState{content: content, existed: true}
		case errors.Is(err, fs.ErrNotExist):
			snap.files[path] = fileState{}
		default:
			return nil, err
		}
	}
	return snap, nil
}

func (s *snapshot) restore() error {
	var errs []error
	for path, state := range s.files {
		if !state.existed {
			if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				errs = append(errs, err)
			}
			continue
		}
		if err := os.WriteFile(path, state.content, 0o644); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("restore after failed conversion: %w", errors.Join(errs...))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Discovery
// ---------------------------------------------------------------------------

// skipDirs are directories that never hold hand-written DI code.
var skipDirs = map[string]bool{
	".git": true, ".github": true, ".idea": true, ".vscode": true,
	"node_modules": true, "vendor": true, "testdata": true, "dist": true, "build": true,
}

type source struct {
	path    string
	dir     string
	pkgName string
	src     []byte
	file    *ast.File
	fset    *token.FileSet
}

func (s *source) offset(pos token.Pos) int { return s.fset.Position(pos).Offset }

func (s *source) text(node ast.Node) string {
	return string(s.src[s.offset(node.Pos()):s.offset(node.End())])
}

// parse reads every Go file in the project, skipping generated Wire output.
func parseSources(root string) ([]*source, error) {
	var sources []*source
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == root {
				return nil
			}
			if skipDirs[entry.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Wire's generated file is deleted, never rewritten.
		if filepath.Base(path) == "wire_gen.go" {
			return nil
		}
		if !bytes.Contains(src, []byte("wire.")) {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		sources = append(sources, &source{
			path: path, dir: filepath.Dir(path), pkgName: file.Name.Name,
			src: src, file: file, fset: fset,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].path < sources[j].path })
	return sources, nil
}
