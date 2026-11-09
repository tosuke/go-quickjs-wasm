// modulegen renames wasm2go's constructor for caller-owned worker setup.
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
