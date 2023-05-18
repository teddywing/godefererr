// TODO: doc
package defererr

import (
	"fmt"
	"go/ast"

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
				deferStmt, ok := node.(*ast.DeferStmt)
				if !ok {
					return true
				}

				// Look for a function literal after the `defer` statement.
				funcLit, ok := deferStmt.Call.Fun.(*ast.FuncLit)
				if !ok {
					return true
				}

				funcScope := pass.TypesInfo.Scopes[funcLit.Type]

				// Try to find the function where the defer is defined. Note, defer can be defined in an inner block.
				funcType, ok := funcScope.Parent().(*ast.FuncType)
				if !ok {
					return true
				}
				fmt.Printf("func: %#v\n", funcType)

				if funcLit.Type.Results == nil {
					return true
				}

				for _, returnVal := range funcLit.Type.Results.List {
					fmt.Printf("returnVal: %#v\n", returnVal)
				}

				return true
			},
		)
	}

	return nil, nil
}
