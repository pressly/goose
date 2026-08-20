package migrationstats

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"strings"
)

const (
	registerGoFuncName            = "AddMigration"
	registerGoFuncNameNoTx        = "AddMigrationNoTx"
	registerGoFuncNameContext     = "AddMigrationContext"
	registerGoFuncNameNoTxContext = "AddMigrationNoTxContext"
)

type goMigration struct {
	name                   string
	useTx                  *bool
	upFuncNil, downFuncNil bool
}

func parseGoFile(r io.Reader) (*goMigration, error) {
	astFile, err := parser.ParseFile(
		token.NewFileSet(),
		"", // filename
		r,
		// We don't need to resolve imports, so we can skip it.
		// This speeds up the parsing process.
		// See https://github.com/golang/go/issues/46485
		parser.SkipObjectResolution,
	)
	if err != nil {
		return nil, err
	}
	for _, decl := range astFile.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn == nil || fn.Name == nil {
			continue
		}
		if fn.Name.Name == "init" {
			return parseInitFunc(fn)
		}
	}
	return nil, errors.New("no init function")
}

func parseInitFunc(fd *ast.FuncDecl) (*goMigration, error) {
	if fd == nil {
		return nil, fmt.Errorf("function declaration must not be nil")
	}
	if fd.Body == nil {
		return nil, fmt.Errorf("no function body")
	}
	if len(fd.Body.List) == 0 {
		return nil, fmt.Errorf("no registered goose functions")
	}
	gf := new(goMigration)
	for _, statement := range fd.Body.List {
		expr, ok := statement.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := expr.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel == nil {
			continue
		}
		funcName := sel.Sel.Name
		b := false
		switch funcName {
		case registerGoFuncName, registerGoFuncNameContext:
			b = true
			gf.useTx = &b
		case registerGoFuncNameNoTx, registerGoFuncNameNoTxContext:
			gf.useTx = &b
		default:
			continue
		}
		if gf.name != "" {
			return nil, fmt.Errorf("found duplicate registered functions:\nprevious: %v\ncurrent: %v", gf.name, funcName)
		}
		gf.name = funcName

		if len(call.Args) != 2 {
			return nil, fmt.Errorf("registered goose functions have 2 arguments: got %d", len(call.Args))
		}
		// The only thing we need to know about each argument is whether a
		// migration function was supplied or not. Anything that is not the nil
		// identifier counts as a function, which covers named functions,
		// qualified names such as pkg.Up001 and inlined function literals.
		gf.upFuncNil = isNilIdent(call.Args[0])
		gf.downFuncNil = isNilIdent(call.Args[1])
	}
	// validation
	switch gf.name {
	case registerGoFuncName, registerGoFuncNameNoTx, registerGoFuncNameContext, registerGoFuncNameNoTxContext:
	default:
		return nil, fmt.Errorf("goose register function must be one of: %s",
			strings.Join([]string{
				registerGoFuncName,
				registerGoFuncNameNoTx,
				registerGoFuncNameContext,
				registerGoFuncNameNoTxContext,
			}, ", "),
		)
	}
	if gf.useTx == nil {
		return nil, errors.New("validation error: failed to identify transaction: got nil bool")
	}
	return gf, nil
}

// isNilIdent reports whether the expression is the nil identifier.
func isNilIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}
