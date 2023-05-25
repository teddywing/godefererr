// TODO: doc
package defererr

import (
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

type functionState struct {
	firstErrorDeferEndPos token.Pos
	deferErrorVar         *ast.Ident
}

func newFunctionState() functionState {
	return functionState{
		firstErrorDeferEndPos: -1,
	}
}

func (s *functionState) setFirstErrorDeferEndPos(pos token.Pos) {
	if s.firstErrorDeferEndPos != -1 {
		return
	}

	s.firstErrorDeferEndPos = pos
}

func (s *functionState) deferAssignsError() bool {
	return s.deferErrorVar != nil
}

func checkFunctions(pass *analysis.Pass, node ast.Node) {
	ast.Inspect(
		node,
		func(node ast.Node) bool {
			// Begin by looking at each declared function.
			funcDecl, ok := node.(*ast.FuncDecl)
			if !ok {
				return true
			}

			// var buf bytes.Buffer
			// err := printer.Fprint(&buf, pass.Fset, funcDecl)
			// if err != nil {
			// 	panic(err)
			// }
			// fmt.Println(buf.String())

			// Since we only care about functions that return errors, ignore
			// those that don't have return values.
			if funcDecl.Type.Results == nil {
				return true
			}

			// Look for a return value that has an `error` type. Store the
			// index of the last one.
			errorReturnIndex := -1
			for i, returnVal := range funcDecl.Type.Results.List {
				returnIdent, ok := returnVal.Type.(*ast.Ident)
				if !ok {
					continue
				}

				if returnIdent.Name == "error" {
					errorReturnIndex = i
				}
			}

			// If the function doesn't return an error, ignore the function.
			if errorReturnIndex == -1 {
				return true
			}

			// Get the error return field in case we need to report a problem
			// with error declaration.
			errorReturnField := funcDecl.Type.Results.List[errorReturnIndex]

			fState := newFunctionState()

			checkFunctionBody(
				pass,
				funcDecl.Body,
				errorReturnField,
				&fState,
			)

			// Stop if the `defer` closure does not assign to an error
			// variable.
			if !fState.deferAssignsError() {
				return true
			}

			checkFunctionReturns(
				pass,
				funcDecl.Body,
				errorReturnIndex,
				&fState,
			)

			return true
		},
	)
}

// TODO: doc
func checkFunctionBody(
	pass *analysis.Pass,
	funcBody *ast.BlockStmt,
	errorReturnField *ast.Field,
	fState *functionState,
) {
	ast.Inspect(
		funcBody,
		func(node ast.Node) bool {
			deferStmt, ok := node.(*ast.DeferStmt)
			if !ok {
				return true
			}

			// Get a function closure run by `defer`.
			funcLit, ok := deferStmt.Call.Fun.(*ast.FuncLit)
			if !ok {
				return true
			}

			checkErrorAssignedInDefer(pass, funcLit, errorReturnField, fState)

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

			// Ignore any `return` statements before the end of the `defer`
			// closure.
			if returnStmt.Pos() <= fState.firstErrorDeferEndPos {
				return true
			}

			if returnStmt.Results == nil {
				return true
			}

			// Get the value used when returning the error.
			returnErrorExpr := returnStmt.Results[errorReturnIndex]
			returnErrorIdent, ok := returnErrorExpr.(*ast.Ident)
			if !ok {
				return true
			}

			_, isReturnErrorNamedType :=
				pass.TypesInfo.Types[returnErrorExpr].Type.(*types.Named)

			// Ensure the value used when returning the error is a named type
			// (checking that no nil constant is used), and that the name of
			// the error variable used in the `return` statement matches the
			// name of the error variable assigned in the `defer` closure.
			if !isReturnErrorNamedType ||
				returnErrorIdent.Name != fState.deferErrorVar.Name {

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
