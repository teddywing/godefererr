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

			checkFunctionReturns(pass, funcDecl.Body, errorReturnIndex, fState)

			fmt.Println()

			return true
		},
	)
}

// TODO: doc
func checkFunctionReturns(
	pass *analysis.Pass,
	funcBody *ast.BlockStmt,
	errorReturnIndex int,
	fState *functionState,
) {
	ast.Inspect(
		funcBody,
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
			// Look for error assignments in the defer closure.
			assignStmt, ok := node.(*ast.AssignStmt)
			if !ok {
				return true
			}

			// Ensure the assignment is not `:=`, or `token.DEFINE`. We only
			// want to look for issues if the closure sets an error variable
			// declared outside its scope.
			if assignStmt.Tok == token.DEFINE {
				return true
			}

			// This sentinel tracks whether we've seen an assignment to an
			// error variable in this node in the closure.
			deferAssignsError := false

			for _, lhsExpr := range assignStmt.Lhs {
				lhsIdent, ok := lhsExpr.(*ast.Ident)
				if !ok {
					continue
				}

				if lhsIdent.Obj.Decl == nil {
					continue
				}

				// Get the lhs variable's declaration so we can find out
				// whether it was declared in the closure using a `token.VAR`
				// declaration.
				var lhsIdentDecl ast.Node = lhsIdent.Obj.Decl.(ast.Node)

				// If this lhs was declared inside the defer closure, it should
				// be ignored, as it doesn't set data in the parent scope.
				if isVariableDeclaredInsideDeferClosure(deferFuncLit, lhsIdentDecl) {
					continue
				}

				// Get the type of the lhs.
				lhsNamedType, ok := pass.TypesInfo.Types[lhsExpr].Type.(*types.Named)
				if !ok {
					continue
				}

				// We only care about lhs with an `error` type.
				if lhsNamedType.Obj().Name() != "error" {
					continue
				}

				deferAssignsError = true

				// Store `lhsIdent` so we can reference its name when reporting
				// issues with subsequent `return` statements.
				fState.deferErrorVar = lhsIdent

				// Check whether the lhs variable name is the same as one of
				// the error variables declared in the function's return
				// signature.
				isErrorNameInReturnSignature := false
				for _, errorReturnIdent := range errorReturnField.Names {
					if lhsIdent.Name == errorReturnIdent.Name {
						isErrorNameInReturnSignature = true
					}
				}

				// If the variable name doesn't match any declared in the
				// function's return signature, report a diagnostic on the
				// return signature, requiring the error variable to be
				// declared with the error variable name used in the closure
				// assignment.
				if !isErrorNameInReturnSignature {
					pass.Reportf(
						errorReturnField.Pos(),
						// TODO: Get the actual signature and set the error
						// name in front of the error type.
						"return signature should be '(%s error)'",
						lhsIdent,
					)

					break
				}
			}

			if !deferAssignsError {
				return true
			}

			// Store the position of the end of the `defer` closure. We will
			// only verify `return` statements occurring after this position.
			//
			// Do this only if the `defer` closure contained an error
			// assignment.
			fState.setFirstErrorDeferEndPos(deferFuncLit.Body.Rbrace)

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
