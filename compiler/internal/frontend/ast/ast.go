package ast

import (
	"compiler/internal/source"
	"compiler/internal/types"
)

type Node interface {
	INode() Node
	Loc() *source.Location
}

// Expression represents any node that produces a value
type Expression interface {
	Node
	Expr()
}

type BlockConstruct interface {
	Node
	Block()
}

type ExpressionList []Expression

func (el *ExpressionList) Loc() *source.Location {
	return (*el)[0].Loc()
}
func (el *ExpressionList) INode() Node { return el }

// Statement represents any node that doesn't produce a value
type Statement interface {
	Node
	Stmt()
}

// ExpressionStmt represents a statement that consists of one or more expressions
type ExpressionStmt struct {
	Expressions *ExpressionList
	source.Location
}

func (e *ExpressionStmt) Loc() *source.Location {
	return &e.Location
}

func (e *ExpressionStmt) INode() Node { return e }
func (e *ExpressionStmt) Stmt()       {} // Stmt is a marker interface for all statements

// TypeScopeResolution represents scope resolution for types (e.g., module::TypeName)
type TypeScopeResolution struct {
	Module   *IdentifierExpr
	TypeNode DataType
	source.Location
}

func (t *TypeScopeResolution) INode() Node { return t }
func (t *TypeScopeResolution) Expr()       {} // Expr is a marker interface for all expressions
func (t *TypeScopeResolution) Loc() *source.Location {
	return &t.Location
}
func (t *TypeScopeResolution) Type() types.TYPE_NAME {
	if userType, ok := t.TypeNode.(*UserDefinedType); ok {
		return userType.TypeName
	}
	return types.UNKNOWN_TYPE
}

// VarScopeResolution represents scope resolution for variables (e.g., module::variableName)
type VarScopeResolution struct {
	Module *IdentifierExpr
	Var    *IdentifierExpr
	source.Location
}

func (v *VarScopeResolution) INode() Node { return v }
func (v *VarScopeResolution) Expr()       {} // Expr is a marker interface for all expressions
func (v *VarScopeResolution) Loc() *source.Location {
	return &v.Location
}
func (v *VarScopeResolution) Type() types.TYPE_NAME {
	return types.TYPE_NAME(v.Var.Name)
}
