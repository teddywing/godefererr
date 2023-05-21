// TODO: doc
package defererr

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
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

type functionState struct {
	firstErrorDeferEndPos token.Pos
	deferErrorVar         *ast.Ident
}

func newFunctionState() *functionState {
	return &functionState{
		firstErrorDeferEndPos: -1,
	}
}

func (s *functionState) setFirstErrorDeferEndPos(pos token.Pos) {
	if s.firstErrorDeferEndPos != -1 {
		return
	}

	s.firstErrorDeferEndPos = pos
}

func (s *functionState) isfirstErrorDeferEndPosSet() bool {
	return s.firstErrorDeferEndPos == -1
}

func checkFunctions(pass *analysis.Pass, node ast.Node) {
	ast.Inspect(
		node,
		func(node ast.Node) bool {
			funcDecl, ok := node.(*ast.FuncDecl)
			if !ok {
				return true
			}

			var buf bytes.Buffer
			err := printer.Fprint(&buf, pass.Fset, funcDecl)
			if err != nil {
				panic(err)
			}
			fmt.Println(buf.String())

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

			// Idea: Set this to the end token.Pos of the first `defer`
			// closure. Look for `return`s after that in funcDecl.Body and
			// ensure they include the error variable.
			// firstErrorDeferEndPos := -1

			fState := newFunctionState()

			// Is it possible to generalise this to other types, and look for
			// anything set in `defer` with the same type as a result in the
			// return signature?

			// TODO: Move to checkDeferFunc()
			// Should we make this an ast.Visitor to store some state for `return` checking?
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
					checkErrorAssignedInDefer(pass, funcLit, errorReturnField, fState)

					return true
				},
			)

			fmt.Printf("fState: %#v\n", fState)

			if fState.isfirstErrorDeferEndPosSet() {
				return true
			}

			ast.Inspect(
				funcDecl.Body,
				func(node ast.Node) bool {
					returnStmt, ok := node.(*ast.ReturnStmt)
					if !ok {
						return true
					}

					if returnStmt.Pos() <= fState.firstErrorDeferEndPos {
						return true
					}

					// TODO: Check whether returnStmt uses error variable.
					fmt.Printf("returnStmt: %#v\n", returnStmt)

					if returnStmt.Results == nil {
						return true
					}

					fmt.Printf("returnStmt.Results: %#v\n", returnStmt.Results)

					for _, expr := range returnStmt.Results {
						fmt.Printf("returnStmt expr: %#v\n", expr)

						t := pass.TypesInfo.Types[expr]
						fmt.Printf("returnStmt expr type: %#v\n", t)
					}

					// TODO: Get returnStmt.Results[error index from function result signature]
					// If not variable and name not [error variable name from defer], report diagnostic
					returnErrorExpr := returnStmt.Results[errorReturnIndex]
					t := pass.TypesInfo.Types[returnErrorExpr]
					fmt.Printf("returnStmt value type: %#v\n", t)
					fmt.Printf("returnStmt type type: %#v\n", t.Type)

					returnErrorIdent, ok := returnErrorExpr.(*ast.Ident)
					if !ok {
						return true
					}

					// TODO: Require t.Type to be *types.Named
					_, ok = t.Type.(*types.Named)
					if !ok {
						// TODO: report

						pass.Reportf(
							returnErrorIdent.Pos(),
							"does not return '%s'",
							fState.deferErrorVar,
						)

						return true
					}

					// Or, we want to compare with the error declared in the function signature.
					fmt.Printf("returnError: %#v\n", returnErrorExpr)

					if returnErrorIdent.Name == fState.deferErrorVar.Name {
						fmt.Printf(
							"names: return:%#v : defer:%#v\n",
							returnErrorIdent.Name,
							fState.deferErrorVar.Name,
						)
					}

					if returnErrorIdent.Name != fState.deferErrorVar.Name {
						fmt.Printf(
							"names: return:%#v : defer:%#v\n",
							returnErrorIdent.Name,
							fState.deferErrorVar.Name,
						)

						pass.Reportf(
							returnErrorIdent.Pos(),
							"does not return '%s'",
							fState.deferErrorVar,
						)
					}

					return true
				},
			)

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
	fState *functionState,
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

				// TODO: Figure out why doesDeclareErrInSignature doesn't
				// continue past here.
				fmt.Printf("variable: %#v\n", ident)
				fmt.Printf("variable.obj: %#v\n", ident.Obj)
				fmt.Printf("variable.obj.type: %#v\n", ident.Obj.Type)
				fmt.Printf("variable.obj.decl: %#v\n", ident.Obj.Decl)

				obj := pass.TypesInfo.Defs[ident]
				fmt.Printf("obj: %#v\n", obj)

				if ident.Obj.Decl == nil {
					continue
				}

				var variableDecl ast.Node = ident.Obj.Decl.(ast.Node)
				// variableDecl, ok := ident.Obj.Decl.(*ast.DeclStmt)
				// if !ok {
				// 	fmt.Println("NOK")
				// 	continue
				// }

				fmt.Printf("variable.obj.valuespec: %#v\n", variableDecl)
				fmt.Printf("variable.obj.valuespec.type: %#v\n", variableDecl)

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
				if isVariableDeclaredInsideDeferClosure(deferFuncLit, variableDecl) {
					continue
				}

				if named.Obj().Name() == "error" {
					deferAssignsError = true

					isErrorNameInReturnSignature := false

					// TODO: Use the var name in return statement checks.
					for _, errorReturnIdent := range errorReturnField.Names {
						if ident.Name == errorReturnIdent.Name {
							// Report if no matches
							isErrorNameInReturnSignature = true
						}
					}

					// fmt.Printf("named: %#v\n", named)
					// if len(errorReturnField.Names) > 0 {
					// 	fState.deferErrorVar = errorReturnField.Names[0]
					// }
					fState.deferErrorVar = ident

					// Maybe don't report the error if it was declared in the closure using a GenDecl? -> We already don't. Should test for these things.

					if !isErrorNameInReturnSignature {
						pass.Reportf(
							errorReturnField.Pos(),
							"return signature should be '(%s error)'", // TODO: Use name from ident.Name
							ident,
						)

						break
					}

					// TODO: Check `return`s in rest of function to find out whether this error name is included
				}
			}

			if !deferAssignsError {
				return true
			}

			fState.setFirstErrorDeferEndPos(deferFuncLit.Body.Rbrace)

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
	variableDecl ast.Node,
) bool {
	return deferFuncLit.Body.Lbrace < variableDecl.Pos() &&
		variableDecl.Pos() < deferFuncLit.Body.Rbrace
}
