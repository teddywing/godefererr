// TODO: doc
package defererr

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "defererr",
	Doc:  "reports issues returning errors from defer",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// TODO: Find defer closure
	// Does it set error defined in outer scope?
	// Does outer scope declare error variable in signature?
	// Is err variable returned after closure?

	for _, file := range pass.Files {
		ast.Inspect(
			file,
			func(node ast.Node) bool {
				funcDecl, ok := node.(*ast.FuncDecl)
				if !ok {
					return true
				}

				if funcDecl.Type.Results == nil {
					return true
				}

				funcReturnsError := false
				for _, returnVal := range funcDecl.Type.Results.List {
					fmt.Printf("returnVal: %#v\n", returnVal.Type)

					returnIdent, ok := returnVal.Type.(*ast.Ident)
					if !ok {
						return true
					}

					if returnIdent.Name == "error" {
						funcReturnsError = true
					}
				}

				// Can we do the same for non-error types?
				// for _, returnVal := range funcType.Results.List {
				// }

				if !funcReturnsError {
					return true
				}

				ast.Inspect(
					funcDecl.Body,
					func(node ast.Node) bool {
						// fmt.Printf("node: %#v\n", node)
						deferStmt, ok := node.(*ast.DeferStmt)
						if !ok {
							return true
						}

						fmt.Printf("defer: %#v\n", deferStmt)

						// TODO: Find out if defer uses assigns an error variable without declaring it

						funcLit, ok := deferStmt.Call.Fun.(*ast.FuncLit)
						if !ok {
							return true
						}

						ast.Inspect(
							funcLit.Body,
							func(node ast.Node) bool {
								assignStmt, ok := node.(*ast.AssignStmt)
								if !ok {
									return true
								}

								if assignStmt.Tok == token.DEFINE {
									return true
								}

								fmt.Printf("assignStmt: %#v\n", assignStmt)

								// TODO: Get type of Lhs, check if "error"
								// If "error", then ensure error return is declared in signature

								deferAssignsError := false
								for _, variable := range assignStmt.Lhs {
									ident, ok := variable.(*ast.Ident)
									if !ok {
										continue
									}

									obj := pass.TypesInfo.Defs[ident]

									valueSpec, ok := ident.Obj.Decl.(*ast.ValueSpec)
									if !ok {
										continue
									}

									fmt.Printf("variable: %#v\n", ident)
									fmt.Printf("variable.obj: %#v\n", ident.Obj)
									fmt.Printf("variable.obj.type: %#v\n", ident.Obj.Type)
									fmt.Printf("variable.obj.valuespec: %#v\n", valueSpec)
									fmt.Printf("variable.obj.valuespec.type: %#v\n", valueSpec.Type)
									fmt.Printf("obj: %#v\n", obj)

									t := pass.TypesInfo.Types[variable]
									fmt.Printf("type: %#v\n", t)
									fmt.Printf("type.type: %#v\n", t.Type)

									named, ok := t.Type.(*types.Named)
									if !ok {
										continue
									}

									fmt.Printf("type.type.obj: %#v\n", named.Obj())
									fmt.Printf("type.type.obj: %#v\n", named.Obj().Name())

									if named.Obj().Name() == "error" {
										deferAssignsError = true
									}
								}

								if !deferAssignsError {
									return true
								}

								// TODO: Check that funcDecl declares error in signature (check before ast.Inspect on function body, report here)

								return true
							},
						)

						return true
					},
				)

				// // Look for a function literal after the `defer` statement.
				// funcLit, ok := deferStmt.Call.Fun.(*ast.FuncLit)
				// if !ok {
				// 	return true
				// }
				//
				// funcScope := pass.TypesInfo.Scopes[funcLit.Type]
				//
				// // Try to find the function where the defer is defined. Note, defer can be defined in an inner block.
				// funcType, ok := funcScope.Parent().(*ast.FuncType)
				// if !ok {
				// 	return true
				// }
				// fmt.Printf("func: %#v\n", funcType)
				//
				// if funcLit.Type.Results == nil {
				// 	return true
				// }
				//
				// for _, returnVal := range funcLit.Type.Results.List {
				// 	fmt.Printf("returnVal: %#v\n", returnVal)
				// }

				return true
			},
		)
	}

	return nil, nil
}
