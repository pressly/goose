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
	name                     string
	useTx                    *bool
	upFuncName, downFuncName string
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
		getNameFromExpr := func(expr ast.Expr) (string, error) {
			arg, ok := expr.(*ast.Ident)
			if !ok {
				return "", fmt.Errorf("failed to assert argument identifier: got %T", arg)
			}
			return arg.Name, nil
		}
		var err error
		gf.upFuncName, err = getNameFromExpr(call.Args[0])
		if err != nil {
			return nil, err
		}
		gf.downFuncName, err = getNameFromExpr(call.Args[1])
		if err != nil {
			return nil, err
		}
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
	// The up and down functions can either be named Go functions or "nil", an
	// empty string means there is a flaw in our parsing logic of the Go source code.
	if gf.upFuncName == "" {
		return nil, fmt.Errorf("validation error: up function is empty string")
	}
	if gf.downFuncName == "" {
		return nil, fmt.Errorf("validation error: down function is empty string")
	}
	return gf, nil
}
