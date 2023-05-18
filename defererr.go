// TODO: doc
package defererr

import (
	"fmt"
	"go/ast"
	"go/token"

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
								fmt.Printf("token: %d\n", token.DEFINE)

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
