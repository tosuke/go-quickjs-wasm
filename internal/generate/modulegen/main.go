// modulegen adapts wasm2go to worker-local modules over shared, growing memory.
package main

import (
	"flag"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"

	"golang.org/x/tools/go/ast/astutil"
)

func main() {
	path := flag.String("file", "", "generated module to adapt")
	libc := flag.String("libc", "", "generated libc to adapt")
	constructor := flag.String("constructor", "newModule", "name for the translated constructor")
	flag.Parse()
	rewrite(*path, *constructor)
	if *libc != "" {
		rewrite(*libc, "")
	}
}

func rewrite(path, constructor string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	renamed := false
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Recv == nil && fn.Name.Name == "New" && constructor != "" {
			fn.Name.Name = constructor
			renamed = true
		}
		if fn.Recv == nil || !ast.IsExported(fn.Name.Name) || fn.Body == nil {
			continue
		}
		fn.Body.List = append([]ast.Stmt{&ast.DeferStmt{Call: &ast.CallExpr{
			Fun:  &ast.SelectorExpr{X: ast.NewIdent("runtime"), Sel: ast.NewIdent("KeepAlive")},
			Args: []ast.Expr{fn.Recv.List[0].Names[0]},
		}}}, fn.Body.List...)
	}
	if constructor != "" && !renamed {
		log.Fatal("wasm2go constructor was not found")
	}
	// Reading a shared slice header while another worker changes its length is
	// not an atomic snapshot. Rebuild each memory view from immutable mapping
	// metadata and an atomic length, including accesses in provided libc code.
	astutil.Apply(f, func(c *astutil.Cursor) bool {
		p, ok := c.Node().(*ast.StarExpr)
		if !ok {
			return true
		}
		sel, ok := p.X.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "memory" {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok || recv.Name != "m" {
			return true
		}
		c.Replace(&ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("m"), Sel: ast.NewIdent("bytes")}})
		return false
	}, nil)
	out, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	if err := format.Node(out, fset, f); err != nil {
		log.Fatal(err)
	}
	if err := out.Close(); err != nil {
		log.Fatal(err)
	}
}
