// Command structgen generates opaque Go structs and field address functions.
// Layout constants are compiled by Clang for wasm32, never for the host ABI.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type field struct {
	name, typ, kind string
	width           int
}
type record struct {
	name, doc string
	fields    []field
}

type includes []string

func (v *includes) String() string     { return strings.Join(*v, ",") }
func (v *includes) Set(s string) error { *v = append(*v, s); return nil }

func main() {
	input := flag.String("input", "../quickjs.go", "Go API definitions")
	output := flag.String("output", "/out/quickjs_structs_generated.go", "generated Go file")
	header := flag.String("header", "quickjs.h", "C header")
	cc := flag.String("cc", "clang", "Clang executable")
	var inc includes
	flag.Var(&inc, "I", "C include directory (repeatable)")
	flag.Parse()
	if err := run(*input, *output, *header, *cc, inc); err != nil {
		log.Fatal(err)
	}
}

func parse(input string) ([]record, error) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, input, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	aliases := map[string]ast.Expr{}
	for _, d := range f.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.TYPE {
			for _, s := range g.Specs {
				t := s.(*ast.TypeSpec)
				aliases[t.Name.Name] = t.Type
			}
		}
	}
	var records []record
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.TYPE {
			continue
		}
		for _, s := range g.Specs {
			t := s.(*ast.TypeSpec)
			// Private jsFoo definitions describe the public JSFoo C memory API.
			if !strings.HasPrefix(t.Name.Name, "js") || !ast.IsExported(strings.TrimPrefix(t.Name.Name, "js")) {
				continue
			}
			st, ok := t.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("%s: expected struct", t.Name)
			}
			doc := t.Doc
			if doc == nil {
				doc = g.Doc
			}
			r := record{name: "JS" + strings.TrimPrefix(t.Name.Name, "js"), doc: doc.Text()}
			r.fields, err = parseFields(fs, st, nil, aliases)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", r.name, err)
			}
			names := map[string]string{}
			for _, f := range r.fields {
				name := strings.ReplaceAll(f.name, ".", "_")
				if previous, exists := names[name]; exists {
					return nil, fmt.Errorf("%s: %s and %s generate the same accessor", r.name, previous, f.name)
				}
				names[name] = f.name
			}
			records = append(records, r)
		}
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("%s: no structs", input)
	}
	return records, nil
}

var cIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Anonymous structs describe C member paths, not Go or C memory layouts.
// C structs and unions can both be represented by these nested declarations.
func parseFields(fs *token.FileSet, st *ast.StructType, path []string, aliases map[string]ast.Expr) ([]field, error) {
	var fields []field
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			return nil, fmt.Errorf("embedded fields are unsupported")
		}
		for _, n := range f.Names {
			name := n.Name
			if f.Tag != nil {
				tag, err := strconv.Unquote(f.Tag.Value)
				if err != nil {
					return nil, err
				}
				var ok bool
				name, ok = reflect.StructTag(tag).Lookup("c")
				if !ok || !cIdentifier.MatchString(name) {
					return nil, fmt.Errorf("%s: expected a c tag containing a C member name", n.Name)
				}
			}
			member := append(append([]string(nil), path...), name)
			if nested, ok := f.Type.(*ast.StructType); ok {
				children, err := parseFields(fs, nested, member, aliases)
				if err != nil {
					return nil, err
				}
				if len(children) == 0 {
					return nil, fmt.Errorf("%s: empty anonymous struct", strings.Join(member, "."))
				}
				fields = append(fields, children...)
				continue
			}
			var b bytes.Buffer
			if err := format.Node(&b, fs, f.Type); err != nil {
				return nil, err
			}
			kind, width, err := representation(f.Type, aliases, map[string]bool{})
			if err != nil {
				return nil, fmt.Errorf("%s: %w", strings.Join(member, "."), err)
			}
			fields = append(fields, field{strings.Join(member, "."), b.String(), kind, width})
		}
	}
	return fields, nil
}

func representation(e ast.Expr, aliases map[string]ast.Expr, seen map[string]bool) (string, int, error) {
	if p, ok := e.(*ast.IndexExpr); ok {
		if n, ok := p.X.(*ast.Ident); ok && n.Name == "Ptr" {
			return "pointer", 4, nil
		}
	}
	if n, ok := e.(*ast.Ident); ok {
		if strings.HasPrefix(n.Name, "JS") {
			if _, ok := aliases["js"+strings.TrimPrefix(n.Name, "JS")].(*ast.StructType); ok {
				return "aggregate", 0, nil
			}
		}
		switch n.Name {
		case "byte", "uint8":
			return "unsigned", 1, nil
		case "int8":
			return "signed", 1, nil
		case "uint16":
			return "unsigned", 2, nil
		case "int16":
			return "signed", 2, nil
		case "uint32":
			return "unsigned", 4, nil
		case "int32":
			return "signed", 4, nil
		case "uint64":
			return "unsigned", 8, nil
		case "int64":
			return "signed", 8, nil
		case "float32":
			return "float", 4, nil
		case "float64":
			return "float", 8, nil
		}
		if a, ok := aliases[n.Name]; ok && !seen[n.Name] {
			seen[n.Name] = true
			kind, width, err := representation(a, aliases, seen)
			// Named i32/i64 aliases model Wasm bits. Their Go signedness need not
			// match the C typedef (for example JSAtom and JSValue).
			if err == nil && width >= 4 && (kind == "signed" || kind == "unsigned") {
				kind = "integer"
			}
			return kind, width, err
		}
	}
	return "", 0, fmt.Errorf("unsupported field type %v (use fixed-width scalars, aliases, Ptr[T], or generated opaque types)", e)
}

func probe(records []record, header string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "#include %q\n#include <stddef.h>\n_Static_assert(sizeof(void*) == 4, \"expected wasm32\");\n", header)
	for i, r := range records {
		fmt.Fprintf(&b, "const unsigned qjs_size_%d = sizeof(%s);\n", i, r.name)
		for j, f := range r.fields {
			expr := fmt.Sprintf("((%s*)0)->%s", r.name, f.name)
			fmt.Fprintf(&b, "const unsigned qjs_offset_%d_%d = offsetof(%s, %s);\n", i, j, r.name, f.name)
			fmt.Fprintf(&b, "const unsigned qjs_width_%d_%d = sizeof(%s);\n", i, j, expr)
			if f.kind == "aggregate" {
				fmt.Fprintf(&b, "_Static_assert(__builtin_types_compatible_p(__typeof__(%s), %s), \"%s.%s: aggregate type mismatch\");\n", expr, f.typ, r.name, f.name)
				continue
			}
			// Bitfields have no address; offsetof and sizeof intentionally reject them.
			fmt.Fprintf(&b, "_Static_assert(sizeof(%s) == %d, \"%s.%s: width mismatch\");\n", expr, f.width, r.name, f.name)
			class := 1
			if f.kind == "pointer" {
				class = 5
			}
			if f.kind == "float" {
				class = 8
			}
			fmt.Fprintf(&b, "_Static_assert(__builtin_classify_type(%s) == %d, \"%s.%s: type mismatch\");\n", expr, class, r.name, f.name)
			if f.kind == "signed" || f.kind == "unsigned" {
				signed := 0
				if f.kind != "unsigned" {
					signed = 1
				}
				fmt.Fprintf(&b, "_Static_assert(_Generic((%s), signed char: 1, short: 1, int: 1, long: 1, long long: 1, default: 0) == %d, \"%s.%s: signedness mismatch\");\n", expr, signed, r.name, f.name)
			}
		}
	}
	return b.String()
}

var constant = regexp.MustCompile(`(?m)^@(qjs_(?:size|offset|width)_\d+(?:_\d+)?) = .*constant i32 (\d+)`)

func layouts(ir []byte) map[string]int {
	values := map[string]int{}
	for _, m := range constant.FindAllSubmatch(ir, -1) {
		n, _ := strconv.Atoi(string(m[2]))
		values[string(m[1])] = n
	}
	return values
}

func comment(b *bytes.Buffer, s string) {
	for _, l := range strings.Split(strings.TrimSuffix(s, "\n"), "\n") {
		if s != "" {
			fmt.Fprintf(b, "// %s\n", l)
		}
	}
}

func generate(records []record, values map[string]int) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("// Code generated by internal/generate/structgen; DO NOT EDIT.\n// Layouts are compiled from the C header for wasm32-unknown-unknown.\n\npackage quickjs_wasm\n")
	for i, r := range records {
		size, ok := values[fmt.Sprintf("qjs_size_%d", i)]
		if !ok {
			return nil, fmt.Errorf("missing size for %s", r.name)
		}
		comment(&b, r.doc)
		fmt.Fprintf(&b, "type %s struct { opaque [%sSize]byte }\nconst %sSize = %d\n", r.name, r.name, r.name, size)
		for j, f := range r.fields {
			off, ok := values[fmt.Sprintf("qjs_offset_%d_%d", i, j)]
			width, widthOK := values[fmt.Sprintf("qjs_width_%d_%d", i, j)]
			if !ok || !widthOK || off+width > size {
				return nil, fmt.Errorf("invalid offset for %s.%s", r.name, f.name)
			}
			fmt.Fprintf(&b, "func %s_%s(ptr Ptr[%s]) Ptr[%s] { return ptr + %d }\n", r.name, strings.ReplaceAll(f.name, ".", "_"), r.name, f.typ, off)
		}
	}
	return format.Source(b.Bytes())
}

func run(input, output, header, cc string, inc []string) error {
	records, err := parse(input)
	if err != nil {
		return err
	}
	args := []string{"--target=wasm32-unknown-unknown", "-ffreestanding", "-nostdlib", "-S", "-emit-llvm", "-x", "c", "-o", "-", "-"}
	for _, p := range inc {
		args = append(args, "-I"+p)
	}
	cmd := exec.Command(cc, args...)
	cmd.Stdin = strings.NewReader(probe(records, header))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	ir, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("wasm32 ABI probe: %w\n%s", err, stderr.String())
	}
	out, err := generate(records, layouts(ir))
	if err != nil {
		return err
	}
	return os.WriteFile(output, out, 0644)
}
