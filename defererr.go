// Copyright (c) 2023  Teddy Wing
//
// This file is part of Godefererr.
//
// Godefererr is free software: you can redistribute it and/or
// modify it under the terms of the GNU General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// Godefererr is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Godefererr. If not, see <https://www.gnu.org/licenses/>.

// Package defererr defines an Analyzer that checks whether an error set in a
// defer closure is correctly returned.
//
// # Analyzer defererr
//
// defererr: report incorrectly returned errors from defer closures.
//
// Errors can be returned from a defer closure by setting a captured error
// variable within the closure. In order for this to work, the error variable
// must be declared in the function signature, and must be returned somewhere
// in the function. This analyzer checks to make sure that captured error
// variables assigned in defer closures are correctly declared and returned.
//
// For example:
//
//	func returnErrorFromDefer() error { // return signature should be '(err error)'
//		var err error = nil
//		if err != nil {
//			return err
//		}
//
//		defer func() {
//			err = errors.New("defer error")
//		}()
//
//		return nil // return should use 'err'
//	}
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
	// Look at each function and check if it returns an error.
	// If so, look for a defer inside the function.
	// If a captured error variable is defined in the defer's closure, ensure
	// that the variable is declared in the function's signature, and that any
	// returns after the defer closure use the assigned error variable.

	for _, file := range pass.Files {
		checkFunctions(pass, file)
	}

	return nil, nil
}

// functionState stores information about functions needed to report problems
// returning errors from defer.
type functionState struct {
	// The end position of the first defer closure.
	firstErrorDeferEndPos token.Pos

	// Error variable assigned in defer.
	deferErrorVar *ast.Ident
}

// newFunctionState initialises a new functionState.
func newFunctionState() functionState {
	return functionState{
		firstErrorDeferEndPos: -1,
	}
}

// setFirstErrorDeferEndPos sets the firstErrorDeferEndPos field of s to pos
// unless it has already been set.
func (s *functionState) setFirstErrorDeferEndPos(pos token.Pos) {
	if s.firstErrorDeferEndPos != -1 {
		return
	}

	s.firstErrorDeferEndPos = pos
}

// deferAssignsError returns true if the deferErrorVar field of s was assigned.
func (s *functionState) deferAssignsError() bool {
	return s.deferErrorVar != nil
}

// checkFunctions looks at each function and runs defer error checks.
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

// checkFunctionBody looks for defer statements in a function body and runs
// error checks.
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

// checkFunctionReturns checks whether the return statements after a defer
// closure use the error assigned in the defer.
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

// checkErrorAssignedInDefer checks whether an error value is assigned to a
// captured variable in a defer closure.
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
						"return signature should use named error parameter %s",
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

// isVariableDeclaredInsideDeferClosure returns true if the position of
// variableDecl is between the start and end of deferFuncLit.
func isVariableDeclaredInsideDeferClosure(
	deferFuncLit *ast.FuncLit,
	variableDecl ast.Node,
) bool {
	return deferFuncLit.Body.Lbrace < variableDecl.Pos() &&
		variableDecl.Pos() < deferFuncLit.Body.Rbrace
}
