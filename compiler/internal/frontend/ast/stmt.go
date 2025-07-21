package ast

import (
	"encoding/json"
	"fmt"
	"os"

	"ferret/compiler/internal/source"
)

type Program struct {
	FullPath   string // the physical full path to the file
	ImportPath string // the logical path to the module
	Modulename string // the module name derived from the full path
	Nodes      []Node
	source.Location
}

func (m *Program) SaveAST() error {
	file, err := os.Create(m.FullPath + ".ast.json")
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // pretty-print
	if err := encoder.Encode(m); err != nil {
		return fmt.Errorf("failed to encode AST to JSON: %w", err)
	}
	return nil
}

func (m *Program) INode()                {} // Impliments Node interface
func (m *Program) Stmt()                 {} // Stmt is a marker interface for all statements
func (m *Program) Loc() *source.Location { return &m.Location }

// Statement nodes
type VarDeclStmt struct {
	Variables    []*VariableToDeclare
	Initializers []Expression
	IsConst      bool
	source.Location
}

func (v *VarDeclStmt) INode()                {} // Impliments Node interface
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

func (a *AssignmentStmt) INode()                {} // Impliments Node interface
func (a *AssignmentStmt) Stmt()                 {} // Stmt is a marker interface for all statements
func (a *AssignmentStmt) Loc() *source.Location { return &a.Location }

// TypeDeclStmt represents a type declaration statement
type TypeDeclStmt struct {
	Alias    *IdentifierExpr // The name of the type
	BaseType DataType        // The underlying type
	source.Location
}

func (t *TypeDeclStmt) INode()                {} // Impliments Node interface
func (t *TypeDeclStmt) Stmt()                 {} // Stmt is a marker interface for all statements
func (t *TypeDeclStmt) Loc() *source.Location { return &t.Location }

// ReturnStmt represents a return statement
type ReturnStmt struct {
	Value *Expression
	source.Location
}

func (r *ReturnStmt) INode()                {} // Impliments Node interface
func (r *ReturnStmt) Stmt()                 {} // Stmt is a marker method for statements
func (r *ReturnStmt) Loc() *source.Location { return &r.Location }

// ImportStmt represents an import statement
type ImportStmt struct {
	ImportPath *StringLiteral // The import path as written in source (e.g., "code/data")
	ModuleName string         // The alias or last part of the import path (e.g., "data")
	FullPath   string         // The fully resolved, normalized file path (always with .fer)

	source.Location
}

func (i *ImportStmt) INode()                {} // Impliments Node interface
func (i *ImportStmt) Stmt()                 {} // Stmt is a marker interface for all statements
func (i *ImportStmt) Loc() *source.Location { return &i.Location }

type ModuleDeclStmt struct {
	ModuleName *IdentifierExpr
	source.Location
}

func (m *ModuleDeclStmt) INode()                {} // Impliments Node interface
func (m *ModuleDeclStmt) Stmt()                 {} // Stmt is a marker interface for all statements
func (m *ModuleDeclStmt) Loc() *source.Location { return &m.Location }
