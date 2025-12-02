package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// analyzer reports usage of panic anywhere in the project and
// reports calls to log.Fatal / os.Exit outside main.main.
var analyzer = &analysis.Analyzer{
	Name: "projectlinter",
	Doc:  "reports panic usage and log.Fatal/os.Exit calls outside main.main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	pkgName := pass.Pkg.Name()

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			funcName := fn.Name.Name
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				checkPanic(pass, call)
				checkFatalOrExit(pass, call, pkgName, funcName)
				return true
			})
		}
	}

	return nil, nil
}

// checkPanic reports usage of builtin panic.
func checkPanic(pass *analysis.Pass, call *ast.CallExpr) {
	id, ok := call.Fun.(*ast.Ident)
	if !ok {
		return
	}

	obj := pass.TypesInfo.Uses[id]
	if obj == nil {
		return
	}

	if builtin, ok := obj.(*types.Builtin); ok && builtin.Name() == "panic" {
		pass.Reportf(call.Lparen, "use of builtin panic is forbidden")
	}
}

// checkFatalOrExit reports usage of log.Fatal / os.Exit outside main.main.
func checkFatalOrExit(pass *analysis.Pass, call *ast.CallExpr, pkgName, funcName string) {
	// allowed only in main.main
	allowedInThisContext := pkgName == "main" && funcName == "main"

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	fnObj, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fnObj.Pkg() == nil {
		return
	}

	pkgPath := fnObj.Pkg().Path()
	name := fnObj.Name()

	isLogFatal := pkgPath == "log" && name == "Fatal"
	isOsExit := pkgPath == "os" && name == "Exit"

	if !isLogFatal && !isOsExit {
		return
	}

	if allowedInThisContext {
		return
	}

	if isLogFatal {
		pass.Reportf(call.Lparen, "log.Fatal should not be called outside main.main")
		return
	}

	pass.Reportf(call.Lparen, "os.Exit should not be called outside main.main")
}
