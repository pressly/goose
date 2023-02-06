package filemetadata

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
)

const (
	registerGoFuncName     = "AddMigration"
	registerGoFuncNameNoTx = "AddMigrationNoTx"
)

type goMigration struct {
	name                     string
	useTx                    *bool
	upFuncName, downFuncName string
}

func (g *goMigration) isValid() error {
	switch g.name {
	case registerGoFuncName, registerGoFuncNameNoTx:
	default:
		return fmt.Errorf("goose register function must be one of: %s or %s",
			registerGoFuncName,
			registerGoFuncNameNoTx,
		)
	}
	if g.useTx == nil {
		return errors.New("validation error: failed to identify transaction: got nil bool")
	}
	// The up and down functions can either be named Go functions or "nil", an
	// empty string means there is a flaw in our parsing logic of the Go source code.
	if g.upFuncName == "" {
		return fmt.Errorf("validation error: up function is empty string")
	}
	if g.downFuncName == "" {
		return fmt.Errorf("validation error: down function is empty string")
	}
	return nil
}

func convertGoMigration(g *goMigration) *FileMetadata {
	m := &FileMetadata{
		FileType: "go",
		Tx:       *g.useTx,
	}
	if g.upFuncName != "nil" {
		m.UpCount = 1
	}
	if g.downFuncName != "nil" {
		m.DownCount = 1
	}
	return m
}

func parseGoFile(r io.Reader) (*goMigration, error) {
	astFile, err := parser.ParseFile(
		token.NewFileSet(),
		"", // filename
		r,
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
		case registerGoFuncName:
			b = true
			gf.useTx = &b
		case registerGoFuncNameNoTx:
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
				return "", fmt.Errorf("failed to assert argument identifer: got %T", arg)
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
	if err := gf.isValid(); err != nil {
		return nil, err
	}
	return gf, nil
}
