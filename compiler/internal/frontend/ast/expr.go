package ast

import (
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
)

// Basic expression nodes
type BinaryExpr struct {
	Left     *Expression
	Operator lexer.Token
	Right    *Expression
	source.Location
}

func (b *BinaryExpr) INode()                {} // Impliments Node interface
func (b *BinaryExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (b *BinaryExpr) Loc() *source.Location { return &b.Location }

type UnaryExpr struct {
	Operator lexer.Token
	Operand  *Expression
	source.Location
}

func (u *UnaryExpr) INode()                {} // Impliments Node interface
func (u *UnaryExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (u *UnaryExpr) Loc() *source.Location { return &u.Location }

type PrefixExpr struct {
	Operator lexer.Token // The operator token (++, --)
	Operand  *Expression
	source.Location
}

func (p *PrefixExpr) INode()                {} // Impliments Node interface
func (p *PrefixExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (p *PrefixExpr) Loc() *source.Location { return &p.Location }

type PostfixExpr struct {
	Operand  *Expression
	Operator lexer.Token // The operator token (++, --)
	source.Location
}

func (p *PostfixExpr) INode()                {} // Impliments Node interface
func (p *PostfixExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (p *PostfixExpr) Loc() *source.Location { return &p.Location }

type IdentifierExpr struct {
	Name string
	source.Location
}

func (i *IdentifierExpr) INode()                {} // Impliments Node interface
func (i *IdentifierExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (i *IdentifierExpr) LValue()               {} // LValue is a marker interface for all lvalues
func (i *IdentifierExpr) Loc() *source.Location { return &i.Location }

// FunctionCallExpr represents a function call expression
type FunctionCallExpr struct {
	Caller    *Expression  // The function being called (can be an identifier or other expression)
	Arguments []Expression // The arguments passed to the function
	source.Location
}

func (f *FunctionCallExpr) INode()                {} // Impliments Node interface
func (f *FunctionCallExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (f *FunctionCallExpr) Loc() *source.Location { return &f.Location }

// FieldAccessExpr represents a field access expression like struct.field
type FieldAccessExpr struct {
	Object *Expression     // The struct being accessed
	Field  *IdentifierExpr // The field being accessed
	source.Location
}

func (f *FieldAccessExpr) INode()                {} // Impliments Node interface
func (f *FieldAccessExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (f *FieldAccessExpr) LValue()               {} // LValue is a marker interface for all lvalues
func (f *FieldAccessExpr) Loc() *source.Location { return &f.Location }

// CastExpr represents a type cast expression like value as TargetType
type CastExpr struct {
	Value      *Expression // The value being cast
	TargetType DataType    // The target type to cast to
	source.Location
}

func (c *CastExpr) INode()                {} // Impliments Node interface
func (c *CastExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (c *CastExpr) Loc() *source.Location { return &c.Location }
