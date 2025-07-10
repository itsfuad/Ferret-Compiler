package ast

import (
	"compiler/internal/source"
)

type Program struct {
	FullPath   string // the physical full path to the file
	ImportPath string // the logical path to the module
	Nodes      []Node
	source.Location
}

func (m *Program) INode() Node           { return m }
func (m *Program) Stmt()                 {} // Stmt is a marker interface for all statements
func (m *Program) Loc() *source.Location { return &m.Location }

// Statement nodes
type VarDeclStmt struct {
	Variables    []*VariableToDeclare
	Initializers []Expression
	IsConst      bool
	source.Location
}

func (v *VarDeclStmt) INode() Node           { return v }
func (v *VarDeclStmt) Stmt()                 {} // Stmt is a marker interface for all statements
func (v *VarDeclStmt) Loc() *source.Location { return &v.Location }

type VariableToDeclare struct {
	Identifier   *IdentifierExpr
	ExplicitType DataType
}

type AssignmentStmt struct {
	Left  *ExpressionList
	Right *ExpressionList
	source.Location
}

func (a *AssignmentStmt) INode() Node           { return a }
func (a *AssignmentStmt) Stmt()                 {} // Stmt is a marker interface for all statements
func (a *AssignmentStmt) Loc() *source.Location { return &a.Location }

// TypeDeclStmt represents a type declaration statement
type TypeDeclStmt struct {
	Alias    *IdentifierExpr // The name of the type
	BaseType DataType        // The underlying type
	source.Location
}

func (t *TypeDeclStmt) INode() Node           { return t }
func (t *TypeDeclStmt) Stmt()                 {} // Stmt is a marker interface for all statements
func (t *TypeDeclStmt) Loc() *source.Location { return &t.Location }

// ReturnStmt represents a return statement
type ReturnStmt struct {
	Values *ExpressionList
	source.Location
}

func (r *ReturnStmt) INode() Node           { return r }
func (r *ReturnStmt) Stmt()                 {} // Stmt is a marker method for statements
func (r *ReturnStmt) Loc() *source.Location { return &r.Location }

// ImportStmt represents an import statement
type ImportStmt struct {
	ImportPath   *StringLiteral // The import path as written in source (e.g., "code/data")
	ModuleName   string         // The alias or last part of the import path (e.g., "data")
	FullPath     string         // The fully resolved, normalized file path (always with .fer)
	OriginalPath string         // The original path as written in source (e.g., "code/data")
	IsRemote     bool           // Whether this is a remote import
	source.Location
}

func (i *ImportStmt) INode() Node           { return i }
func (i *ImportStmt) Stmt()                 {} // Stmt is a marker interface for all statements
func (i *ImportStmt) Loc() *source.Location { return &i.Location }

type ModuleDeclStmt struct {
	ModuleName *IdentifierExpr
	source.Location
}

func (m *ModuleDeclStmt) INode() Node           { return m }
func (m *ModuleDeclStmt) Stmt()                 {} // Stmt is a marker interface for all statements
func (m *ModuleDeclStmt) Loc() *source.Location { return &m.Location }
