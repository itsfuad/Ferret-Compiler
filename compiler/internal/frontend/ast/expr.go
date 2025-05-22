package ast

import (
	"ferret/compiler/internal/frontend/lexer"
	"ferret/compiler/internal/source"
)

// Basic expression nodes
type BinaryExpr struct {
	Left     Expression
	Operator lexer.Token
	Right    Expression
	source.Location
}

func (b *BinaryExpr) INode() Node           { return b }
func (b *BinaryExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (b *BinaryExpr) Loc() *source.Location { return &b.Location }

type UnaryExpr struct {
	Operator lexer.Token
	Operand  Expression
	source.Location
}

func (u *UnaryExpr) INode() Node           { return u }
func (u *UnaryExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (u *UnaryExpr) Loc() *source.Location { return &u.Location }

type PrefixExpr struct {
	Operator lexer.Token // The operator token (++, --)
	Operand  Expression
	source.Location
}

func (p *PrefixExpr) INode() Node           { return p }
func (p *PrefixExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (p *PrefixExpr) Loc() *source.Location { return &p.Location }

type PostfixExpr struct {
	Operand  Expression
	Operator lexer.Token // The operator token (++, --)
	source.Location
}

func (p *PostfixExpr) INode() Node           { return p }
func (p *PostfixExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (p *PostfixExpr) Loc() *source.Location { return &p.Location }

type IdentifierExpr struct {
	Name string
	source.Location
}

func (i *IdentifierExpr) INode() Node           { return i }
func (i *IdentifierExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (i *IdentifierExpr) LValue()               {} // LValue is a marker interface for all lvalues
func (i *IdentifierExpr) Loc() *source.Location { return &i.Location }

// FunctionCallExpr represents a function call expression
type FunctionCallExpr struct {
	Caller    Expression   // The function being called (can be an identifier or other expression)
	Arguments []Expression // The arguments passed to the function
	source.Location
}

func (f *FunctionCallExpr) INode() Node           { return f }
func (f *FunctionCallExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (f *FunctionCallExpr) Loc() *source.Location { return &f.Location }

// FieldAccessExpr represents a field access expression like struct.field
type FieldAccessExpr struct {
	Object Expression      // The struct being accessed
	Field  *IdentifierExpr // The field being accessed
	source.Location
}

func (f *FieldAccessExpr) INode() Node           { return f }
func (f *FieldAccessExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (f *FieldAccessExpr) LValue()               {} // LValue is a marker interface for all lvalues
func (f *FieldAccessExpr) Loc() *source.Location { return &f.Location }

// ScopeResolutionExpr represents a scope resolution expression like fmt::Println
type ScopeResolutionExpr struct {
	Module     *IdentifierExpr
	Identifier *IdentifierExpr
	source.Location
}

func (s *ScopeResolutionExpr) INode() Node           { return s }
func (s *ScopeResolutionExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (s *ScopeResolutionExpr) Loc() *source.Location { return &s.Location }
