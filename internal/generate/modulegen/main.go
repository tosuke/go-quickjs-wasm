// modulegen adapts wasm2go's constructor and keeps mmap owners live across calls.
package main

import (
	"flag"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
)

func main() {
	path := flag.String("file", "", "generated module to adapt")
	flag.Parse()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, *path, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	renamed := false
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Recv == nil && fn.Name.Name == "New" {
			fn.Name.Name = "newModule"
			renamed = true
		}
		if fn.Recv == nil || !ast.IsExported(fn.Name.Name) || fn.Body == nil {
			continue
		}
		// Public entry points retain the module while translated code uses raw
		// mmap slices. Private calls inherit that lifetime from their caller.
		fn.Body.List = append([]ast.Stmt{&ast.DeferStmt{Call: &ast.CallExpr{
			Fun:  &ast.SelectorExpr{X: ast.NewIdent("runtime"), Sel: ast.NewIdent("KeepAlive")},
			Args: []ast.Expr{fn.Recv.List[0].Names[0]},
		}}}, fn.Body.List...)
	}
	if !renamed {
		log.Fatal("wasm2go constructor was not found")
	}
	out, err := os.Create(*path)
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
