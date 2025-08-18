package ast

import (
	"compiler/internal/source"
	"compiler/internal/types"
	"fmt"
)

type Node interface {
	INode()
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
func (el *ExpressionList) INode() {} // Impliments Node interface

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

func (e *ExpressionStmt) INode() {} // Impliments Node interface
func (e *ExpressionStmt) Stmt()  {} // Stmt is a marker interface for all statements

// TypeScopeResolution represents scope resolution for types (e.g., module::TypeName)
type TypeScopeResolution struct {
	Module   *IdentifierExpr
	TypeNode DataType
	source.Location
}

func (t *TypeScopeResolution) INode() {} // Impliments Node interface
func (t *TypeScopeResolution) Expr()  {} // Expr is a marker interface for all expressions
func (t *TypeScopeResolution) Loc() *source.Location {
	return &t.Location
}
func (t *TypeScopeResolution) Type() types.TYPE_NAME {
	return types.TYPE_NAME(fmt.Sprintf("%s::%s", t.Module.Name, t.TypeNode.Type()))
}

// VarScopeResolution represents scope resolution for variables (e.g., module::variableName)
type VarScopeResolution struct {
	Module     *IdentifierExpr
	Identifier *IdentifierExpr
	source.Location
}

func (v *VarScopeResolution) INode() {} // Impliments Node interface
func (v *VarScopeResolution) Expr()  {} // Expr is a marker interface for all expressions
func (v *VarScopeResolution) Loc() *source.Location {
	return &v.Location
}
func (v *VarScopeResolution) Type() types.TYPE_NAME {
	return types.TYPE_NAME(fmt.Sprintf("%s::%s", v.Module.Name, v.Identifier.Name))
}
