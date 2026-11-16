package provider

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// frameworkPathPackage is the import path of the framework's path package.
const frameworkPathPackage = "github.com/hashicorp/terraform-plugin-framework/path"

// rawPathFormattingExempt are the functions that render paths themselves.
var rawPathFormattingExempt = map[string]bool{"displayPath": true, "displayPathText": true, "quotedInner": true}

// formatFunctions are the fmt functions whose arguments end up in a message.
var formatFunctions = map[string]bool{"Sprintf": true, "Errorf": true, "Sprint": true, "Sprintln": true}

// findRawPathFormatting returns, as file:line, each argument of a
// fmt.Sprintf, Errorf, Sprint or Sprintln call and each operand of a +
// that is a framework path not passed through displayPath, and each String()
// call on a path anywhere, which would reach a message some other way, such
// as through strings.Join or as a diagnostic's detail. A path is
// recognized syntactically: a selector named Path (req.Path), a field
// declared as path.Path in any of the files, a parameter or variable of the
// enclosing function declared as path.Path or assigned a path, a call to the
// path package (path.Root), or a method call on any of these
// (req.Path.AtName("x"), p.ParentPath(), p.String()).
func findRawPathFormatting(fset *token.FileSet, files []*ast.File) []string {
	pathFields := map[string]bool{}
	for _, f := range files {
		pkg := pathImportName(f)
		ast.Inspect(f, func(n ast.Node) bool {
			if st, ok := n.(*ast.StructType); ok {
				for _, field := range st.Fields.List {
					if isPathTypeExpr(field.Type, pkg) {
						for _, name := range field.Names {
							pathFields[name.Name] = true
						}
					}
				}
			}
			return true
		})
	}

	var hits []string
	for _, f := range files {
		pkg := pathImportName(f)
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || rawPathFormattingExempt[fn.Name.Name] {
				continue
			}
			c := pathChecker{pkg: pkg, fields: pathFields, idents: map[string]bool{}}
			ast.Inspect(fn, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.Field:
					if isPathTypeExpr(n.Type, pkg) {
						for _, name := range n.Names {
							c.idents[name.Name] = true
						}
					}
				case *ast.ValueSpec:
					if isPathTypeExpr(n.Type, pkg) || (len(n.Values) == len(n.Names) && anyPath(c, n.Values)) {
						for _, name := range n.Names {
							c.idents[name.Name] = true
						}
					}
				case *ast.AssignStmt:
					if len(n.Lhs) == len(n.Rhs) {
						for i, lhs := range n.Lhs {
							if id, ok := lhs.(*ast.Ident); ok && c.isPath(n.Rhs[i]) {
								c.idents[id.Name] = true
							}
						}
					}
				case *ast.CallExpr:
					if isDisplayPathCall(n) {
						return false
					}
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "String" && c.isPath(sel.X) {
						hits = append(hits, position(fset, n))
					}
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && formatFunctions[sel.Sel.Name] {
						if x, ok := sel.X.(*ast.Ident); ok && x.Name == "fmt" {
							for _, arg := range n.Args {
								if c.isPath(arg) {
									hits = append(hits, position(fset, arg))
								}
							}
						}
					}
				case *ast.BinaryExpr:
					if n.Op == token.ADD {
						for _, operand := range []ast.Expr{n.X, n.Y} {
							if c.isPath(operand) {
								hits = append(hits, position(fset, operand))
							}
						}
					}
				}
				return true
			})
		}
	}
	return hits
}

// pathChecker recognizes path expressions within one function.
type pathChecker struct {
	pkg    string
	fields map[string]bool
	idents map[string]bool
}

func (c pathChecker) isPath(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return c.isPath(e.X)
	case *ast.Ident:
		return c.idents[e.Name]
	case *ast.SelectorExpr:
		return e.Sel.Name == "Path" || c.fields[e.Sel.Name]
	case *ast.CallExpr:
		if isDisplayPathCall(e) {
			return false
		}
		sel, ok := e.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == c.pkg {
			return true
		}
		return c.isPath(sel.X)
	}
	return false
}

func anyPath(c pathChecker, exprs []ast.Expr) bool {
	for _, e := range exprs {
		if c.isPath(e) {
			return true
		}
	}
	return false
}

func isDisplayPathCall(call *ast.CallExpr) bool {
	id, ok := call.Fun.(*ast.Ident)
	return ok && id.Name == "displayPath"
}

// isPathTypeExpr reports whether typ is pkg.Path, where pkg is the name the
// file imports the framework's path package as.
func isPathTypeExpr(typ ast.Expr, pkg string) bool {
	sel, ok := typ.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Path" {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && pkg != "" && x.Name == pkg
}

// pathImportName returns the name f imports the framework's path package
// as, or "" when it doesn't import it.
func pathImportName(f *ast.File) string {
	for _, imp := range f.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == frameworkPathPackage {
			if imp.Name != nil {
				return imp.Name.Name
			}
			return "path"
		}
	}
	return ""
}

func position(fset *token.FileSet, n ast.Node) string {
	p := fset.Position(n.Pos())
	return filepath.Base(p.Filename) + ":" + strconv.Itoa(p.Line)
}

// TestNoRawPathFormatting fails when provider code formats a framework path
// into a message without displayPath, which would print the whole value of
// each set element on the path.
func TestNoRawPathFormatting(t *testing.T) {
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		t.Fatal("no source files found")
	}

	for _, hit := range findRawPathFormatting(fset, files) {
		t.Errorf("%s: path formatted without displayPath", hit)
	}
}

// TestFindRawPathFormatting checks that the scan in TestNoRawPathFormatting
// finds each kind of raw path it is meant to, and accepts displayPath.
func TestFindRawPathFormatting(t *testing.T) {
	const src = `package provider

import (
	"fmt"

	tfpath "github.com/hashicorp/terraform-plugin-framework/path"
)

type thresholds struct {
	upper tfpath.Path
}

func planted(req request, p tfpath.Path, v thresholds) {
	_ = fmt.Sprintf("%s", req.Path)
	_ = fmt.Errorf("%s", req.Path.AtName("x"))
	_ = fmt.Sprint(p)
	_ = fmt.Sprintln(v.upper)
	_ = "at " + p.String()
	parent := req.Path.ParentPath()
	_ = fmt.Sprintf("%s", parent)
	var q tfpath.Path
	_ = fmt.Sprintf("%s", q)
	_ = fmt.Sprintf("%s", tfpath.Root("name"))
	_ = fmt.Sprintf("%s %s", displayPath(req.Path), displayPath(p.AtName("x")))
	_ = "at " + displayPath(p)
	_ = fmt.Sprintf("%s", req.ConfigValue)
	_ = strings.Join([]string{req.Path.String()}, ", ")
	b.WriteString(p.AtName("x").String())
	_ = errors.New(v.upper.String())
	diags.AddAttributeError(p, "Summary", p.String())
	_ = req.ConfigValue.String()
}

func displayPath(p tfpath.Path) string { return "" + p.String() }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "planted.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	got := strings.Join(findRawPathFormatting(fset, []*ast.File{f}), " ")
	want := "planted.go:14 planted.go:15 planted.go:16 planted.go:17 planted.go:18 planted.go:18 planted.go:20 planted.go:22 planted.go:23 " +
		"planted.go:27 planted.go:28 planted.go:29 planted.go:30"
	if got != want {
		t.Errorf("hits = %s\nwant   %s", got, want)
	}
}
