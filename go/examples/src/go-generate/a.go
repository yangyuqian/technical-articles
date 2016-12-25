package main

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"reflect"
)

func main() {
	fs := token.NewFileSet()

	// func ParseFile(fset *token.FileSet, filename string, src interface{}, mode parser.Mode) (f *ast.File, err error)
	if f, err := goparser.ParseFile(fs, "./src/go-generate/s.go", nil, goparser.ParseComments); err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Println("file:", f)
		fmt.Println("------------")
		fmt.Println("f.Decls:", f.Decls)
		for _, decls := range f.Decls {
			fmt.Println("decls:", decls)
		}
		fmt.Println("------------")
		fmt.Println("f.Name:", f.Name.String())
		fmt.Println("------------")
		fmt.Println("f.Doc:", f.Doc)
		fmt.Println("------------")
		fmt.Println("f.Scope:", f.Scope)
		fmt.Println("f.Scope.Outer:", f.Scope.Outer)
		fmt.Println("f.Scope.Objects:", f.Scope.Objects)
		for k, v := range f.Scope.Objects {
			fmt.Println(
				"object:", k,
				"obj:", v.Name,
				"kind:", v.Kind,
				"decl:", v.Decl,
				"data:", v.Data,
				"pos:", v.Pos(),
			)

			switch v.Decl.(type) {
			case *ast.ValueSpec:
				decl, _ := v.Decl.(*ast.ValueSpec)
				fmt.Println("decl:", decl)
				fmt.Println("decl.Names", decl.Names)
				for _, name := range decl.Names {
					fmt.Println("decl.Name:", name.Name)
					fmt.Println("decl.String():", name.String())
					fmt.Println("name.Obj.Kind.String():", name.Obj.Kind.String())
					fmt.Println("name.Obj.Type:", name.Obj.Type)
				}

				fmt.Println("decl.Values", decl.Values)
				break
			case *ast.TypeSpec:
				decl, _ := v.Decl.(*ast.TypeSpec)
				fmt.Println("decl.Name", decl.Name)
				switch decl.Type.(type) {
				case *ast.StructType:
					tt, _ := decl.Type.(*ast.StructType)

					for _, f := range tt.Fields.List {
						if f.Tag != nil {
							fmt.Println("field tag:", f.Tag.Value)
							for _, nn := range f.Names {
								fmt.Println("field name:", nn.Name)
							}
						}
					}
				default:
					fmt.Println("type of decl type:", reflect.TypeOf(decl.Type))
				}

				break
			default:
				fmt.Println("decl type:", reflect.TypeOf(v.Decl))
			}

		}
	}
}
