package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type param struct{ K, V, P string }

type parse struct {
	Path    string
	Package string
	// Imports      []string
	Imports map[string]*param
	// Structs    []*param
	// Interfaces []string
	Funcs []*struct {
		Name                   string
		Method, Params, Result []*param
	}
}

func parseArgs(args []string, res *[]string, index int) (err error) {
	if len(args) <= index {
		return
	}
	if strings.HasPrefix(args[index], "-") {
		index += 2
		parseArgs(args, res, index)
		return
	}
	var f os.FileInfo
	if f, err = os.Stat(args[index]); err != nil {
		return
	}
	if f.IsDir() {
		if !strings.HasSuffix(args[index], "/") {
			args[index] += "/"
		}
		var fs []os.DirEntry
		if fs, err = os.ReadDir(args[index]); err != nil {
			return
		}
		for _, f = range fs {
			path, _ := filepath.Abs(args[index] + f.Name())
			args = append(args, path)
		}
	} else {
		if strings.HasSuffix(args[index], ".go") &&
			!strings.HasSuffix(args[index], "_test.go") {
			*res = append(*res, args[index])
		}
	}
	index++
	return parseArgs(args, res, index)
}

func parseFile(files ...string) (parses []*parse, err error) {
	for _, file := range files {
		var (
			astFile *ast.File
			fSet    = token.NewFileSet()
			parse   = &parse{
				Imports: make(map[string]*param),
			}
		)
		if astFile, err = parser.ParseFile(fSet, file, nil, 0); err != nil {
			return
		}
		if astFile.Name != nil {
			parse.Path = file
			parse.Package = astFile.Name.Name
		}
		for _, decl := range astFile.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				if specs := decl.Specs; len(specs) > 0 {
					parse.Imports = parseImports(specs)
				}
			case *ast.FuncDecl:
				var (
					dec       = decl
					parseFunc = &struct {
						Name                   string
						Method, Params, Result []*param
					}{Name: dec.Name.Name}
				)
				if dec.Recv != nil {
					parseFunc.Method = parserParams(dec.Recv.List)
				}
				if dec.Type.Params != nil {
					parseFunc.Params = parserParams(dec.Type.Params.List)
				}
				if dec.Type.Results != nil {
					parseFunc.Result = parserParams(dec.Type.Results.List)
				}
				parse.Funcs = append(parse.Funcs, parseFunc)
			}
		}
		parses = append(parses, parse)
	}
	return
}

func parserParams(fields []*ast.Field) (params []*param) {
	for _, field := range fields {
		p := &param{}
		p.V = parseType(field.Type)
		if field.Names == nil {
			params = append(params, p)
		}
		for _, name := range field.Names {
			sp := &param{}
			sp.K = name.Name
			sp.V = p.V
			sp.P = p.P
			params = append(params, sp)
		}
	}
	return
}

func parseType(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		return "*" + parseType(expr.X)
	case *ast.ArrayType:
		return "[" + parseType(expr.Len) + "]" + parseType(expr.Elt)
	case *ast.SelectorExpr:
		return parseType(expr.X) + "." + expr.Sel.Name
	case *ast.MapType:
		return "map[" + parseType(expr.Key) + "]" + parseType(expr.Value)
	case *ast.StructType:
		return "struct{}"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		var (
			pTemp string
			rTemp string
		)
		pTemp = parseFuncType(pTemp, expr.Params)
		if expr.Results != nil {
			rTemp = parseFuncType(rTemp, expr.Results)
			return fmt.Sprintf("func(%s) (%s)", pTemp, rTemp)
		}
		return fmt.Sprintf("func(%s)", pTemp)
	case *ast.ChanType:
		return fmt.Sprintf("make(chan %s)", parseType(expr.Value))
	case *ast.Ellipsis:
		return parseType(expr.Elt)
	}
	return ""
}

func parseFuncType(temp string, data *ast.FieldList) string {
	params := parserParams(data.List)
	for i, param := range params {
		if i == 0 {
			temp = param.K + " " + param.V
			continue
		}
		t := param.K + " " + param.V
		temp = fmt.Sprintf("%s, %s", temp, t)
	}
	return temp
}

func parseImports(specs []ast.Spec) (params map[string]*param) {
	params = make(map[string]*param)
	for _, spec := range specs {
		switch spec := spec.(type) {
		case *ast.ImportSpec:
			p := &param{V: strings.Replace(spec.Path.Value, "\"", "", -1)}
			if spec.Name != nil {
				p.K = spec.Name.Name
				params[p.K] = p
			} else {
				vs := strings.Split(p.V, "/")
				params[vs[len(vs)-1]] = p
			}
		}
	}
	return
}
