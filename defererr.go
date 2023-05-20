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
		checkFunctions(pass, file)
	}

	return nil, nil
}

func checkFunctions(pass *analysis.Pass, node ast.Node) {
	ast.Inspect(
		node,
		func(node ast.Node) bool {
			funcDecl, ok := node.(*ast.FuncDecl)
			if !ok {
				return true
			}

			if funcDecl.Type.Results == nil {
				return true
			}

			funcReturnsError := false
			errorReturnIndex := -1
			for i, returnVal := range funcDecl.Type.Results.List {
				fmt.Printf("returnVal Type: %#v\n", returnVal.Type)

				returnIdent, ok := returnVal.Type.(*ast.Ident)
				if !ok {
					return true
				}

				if returnIdent.Name == "error" {
					funcReturnsError = true
					errorReturnIndex = i
				}
			}

			// Can we do the same for non-error types?
			// for _, returnVal := range funcType.Results.List {
			// }

			if !funcReturnsError || errorReturnIndex == -1 {
				return true
			}

			if len(funcDecl.Type.Results.List[errorReturnIndex].Names) > 0 {
				fmt.Printf("return error var name: %#v\n", funcDecl.Type.Results.List[errorReturnIndex].Names[0])
			}
			errorReturnField := funcDecl.Type.Results.List[errorReturnIndex]

			// TODO: Move to checkDeferFunc()
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

					// TODO: funcall
					checkErrorAssignedInDefer(pass, funcLit, errorReturnField)

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

			fmt.Println()

			return true
		},
	)
}

// TODO: doc
func checkErrorAssignedInDefer(
	pass *analysis.Pass,
	deferFuncLit *ast.FuncLit,
	errorReturnField *ast.Field,
) {
	ast.Inspect(
		deferFuncLit.Body,
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

				// TODO: Was error lhs declared in defer closure? Then it
				// should be ignored.
				if isVariableDeclaredInsideDeferClosure(deferFuncLit, valueSpec) {
					continue
				}

				if named.Obj().Name() == "error" {
					deferAssignsError = true

					isErrorNameInReturnSignature := false

					for _, errorReturnIdent := range errorReturnField.Names {
						if ident.Name == errorReturnIdent.Name {
							// Report if no matches
							isErrorNameInReturnSignature = true
						}
					}

					// Maybe don't report the error if it was declared in the closure using a GenDecl? -> We already don't. Should test for these things.

					if !isErrorNameInReturnSignature {
						pass.Reportf(
							errorReturnField.Pos(),
							"return signature should be '(err error)'", // TODO: Use name from ident.Name
							// errorReturnField,
						)

						break
					}
				}
			}

			if !deferAssignsError {
				return true
			}

			// TODO: Check that funcDecl declares error in signature (check before ast.Inspect on function body, report here)

			// isErrorNameInReturnSignature := false
			//
			// for _, errorReturnIdent := range errorReturnField.Names {
			// 	if ident.Name == errorReturnIdent.Name {
			// 		// Report if no matches
			// 		isErrorNameInReturnSignature = true
			// 	}
			// }
			//
			// if !isErrorNameInReturnSignature {
			// 	pass.Reportf(
			// 		errorReturnField.Pos(),
			// 		"return signature should be '(err error)' (TODO)",
			// 		errorReturnField,
			// 	)
			// }

			return true
		},
	)
}

// TODO: doc
func isVariableDeclaredInsideDeferClosure(
	deferFuncLit *ast.FuncLit,
	variableDecl *ast.ValueSpec,
) bool {
	return deferFuncLit.Body.Lbrace < variableDecl.Pos() &&
		variableDecl.Pos() < deferFuncLit.Body.Rbrace
}
