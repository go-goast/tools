// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

var sources = []string{
	`
	package p
	import "fmt"
	import "math"
	const pi = math.Pi
	func sin(x float64) float64 {
		return math.Sin(x)
	}
	var Println = fmt.Println
	`,
	`
	package p
	import "fmt"
	func f() string {
		_ = "foo"
		return fmt.Sprintf("%d", g())
	}
	func g() (x int) { return }
	`,
	`
	package p
	import . "go/parser"
	import "sync"
	func h() Mode { return ImportsOnly }
	var _, x int = 1, 2
	func init() {}
	type T struct{ sync.Mutex; a, b, c int}
	type I interface{ m() }
	var _ = T{a: 1, b: 2, c: 3}
	func (_ T) m() {}
	`,
}

var pkgnames = []string{
	"fmt",
	"math",
}

func TestResolveQualifiedIdents(t *testing.T) {
	// parse package files
	fset := token.NewFileSet()
	var files []*ast.File
	for i, src := range sources {
		f, err := parser.ParseFile(fset, fmt.Sprintf("sources[%d]", i), src, parser.DeclarationErrors)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}

	// resolve and type-check package AST
	idents := make(map[*ast.Ident]Object)
	var ctxt Context
	ctxt.Ident = func(id *ast.Ident, obj Object) {
		if old, found := idents[id]; found && old != obj {
			t.Errorf("%s: identifier %s reported multiple times with different objects", fset.Position(id.Pos()), id.Name)
		}
		idents[id] = obj
	}
	pkg, err := ctxt.Check("testResolveQualifiedIdents", fset, files...)
	if err != nil {
		t.Fatal(err)
	}

	// check that all packages were imported
	for _, name := range pkgnames {
		if pkg.imports[name] == nil {
			t.Errorf("package %s not imported", name)
		}
	}

	// check that qualified identifiers are resolved
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok {
				if x, ok := s.X.(*ast.Ident); ok {
					obj := idents[x]
					if obj == nil {
						t.Errorf("%s: unresolved qualified identifier %s", fset.Position(x.Pos()), x.Name)
						return false
					}
					if _, ok := obj.(*Package); ok && idents[s.Sel] == nil {
						t.Errorf("%s: unresolved selector %s", fset.Position(s.Sel.Pos()), s.Sel.Name)
						return false
					}
					return false
				}
				return false
			}
			return true
		})
	}

	// check that each identifier in the source is enumerated by the Context.Ident callback
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			if x, ok := n.(*ast.Ident); ok {
				if _, found := idents[x]; found {
					delete(idents, x)
				} else {
					t.Errorf("%s: unresolved identifier %s", fset.Position(x.Pos()), x.Name)
				}
				return false
			}
			return true
		})
	}

	// any left-over identifiers didn't exist in the source
	for x := range idents {
		t.Errorf("%s: identifier %s not present in source", fset.Position(x.Pos()), x.Name)
	}

	// TODO(gri) add tests to check ImplicitObj callbacks
}