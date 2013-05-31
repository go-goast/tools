// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"fmt"
	"go/ast"
)

// TODO(gri) Provide scopes with a name or other mechanism so that
//           objects can use that information for better printing.

// A Scope maintains a set of objects and a link to its containing (parent)
// scope. Objects may be inserted and looked up by name, or by package path
// and name. A nil *Scope acts like an empty scope for operations that do not
// modify the scope or access a scope's parent scope.
type Scope struct {
	parent  *Scope
	entries []Object
}

// NewScope returns a new, empty scope.
func NewScope(parent *Scope) *Scope {
	return &Scope{parent, nil}
}

// Parent returns the scope's containing (parent) scope.
func (s *Scope) Parent() *Scope {
	return s.parent
}

// NumEntries() returns the number of scope entries.
// If s == nil, the result is 0.
func (s *Scope) NumEntries() int {
	if s == nil {
		return 0 // empty scope
	}
	return len(s.entries)
}

// IsEmpty reports whether the scope is empty.
// If s == nil, the result is true.
func (s *Scope) IsEmpty() bool {
	return s == nil || len(s.entries) == 0
}

// At returns the i'th scope entry for 0 <= i < NumEntries().
func (s *Scope) At(i int) Object {
	return s.entries[i]
}

// Index returns the index of the scope entry with the given package
// (path) and name if such an entry exists in s; otherwise the result
// is negative. A nil scope acts like an empty scope, and parent scopes
// are ignored.
//
// If pkg != nil, both pkg.Path() and name are used to identify an
// entry, per the Go rules for identifier equality. If pkg == nil,
// only the name is used and the package path is ignored.
func (s *Scope) Index(pkg *Package, name string) int {
	if s == nil {
		return -1 // empty scope
	}

	// fast path: only the name must match
	if pkg == nil {
		for i, obj := range s.entries {
			if obj.Name() == name {
				return i
			}
		}
		return -1
	}

	// slow path: both pkg path and name must match
	// TODO(gri) if packages were canonicalized, we could just compare the packages
	for i, obj := range s.entries {
		// spec:
		// "Two identifiers are different if they are spelled differently,
		// or if they appear in different packages and are not exported.
		// Otherwise, they are the same."
		if obj.Name() == name && (ast.IsExported(name) || obj.Pkg().path == pkg.path) {
			return i
		}
	}

	// not found
	return -1

	// TODO(gri) Optimize Lookup by also maintaining a map representation
	//           for larger scopes.
}

// Lookup returns the scope entry At(i) for i = Index(pkg, name), if i >= 0.
// Otherwise it returns nil.
func (s *Scope) Lookup(pkg *Package, name string) Object {
	if i := s.Index(pkg, name); i >= 0 {
		return s.At(i)
	}
	return nil
}

// LookupParent follows the parent chain of scopes starting with s until it finds
// a scope where Lookup(nil, name) returns a non-nil entry, and then returns that
// entry. If no such scope exists, the result is nil.
func (s *Scope) LookupParent(name string) Object {
	for s != nil {
		if i := s.Index(nil, name); i >= 0 {
			return s.At(i)
		}
		s = s.parent
	}
	return nil
}

// Insert attempts to insert an object obj into scope s.
// If s already contains an object with the same package path
// and name, Insert leaves s unchanged and returns that object.
// Otherwise it inserts obj, sets the object's scope to s, and
// returns nil.
//
func (s *Scope) Insert(obj Object) Object {
	if alt := s.Lookup(obj.Pkg(), obj.Name()); alt != nil {
		return alt
	}
	s.entries = append(s.entries, obj)
	obj.setParent(s)
	return nil
}

// String returns a string representation of the scope, for debugging.
func (s *Scope) String() string {
	if s == nil {
		return "scope {}"
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "scope %p {", s)
	if s != nil && len(s.entries) > 0 {
		fmt.Fprintln(&buf)
		for _, obj := range s.entries {
			fmt.Fprintf(&buf, "\t%s\t%T\n", obj.Name(), obj)
		}
	}
	fmt.Fprintf(&buf, "}\n")
	return buf.String()
}