package ast

import "ferret/compiler/internal/source"

type IntLiteral struct {
	Value int64  // Actual numeric value
	Raw   string // Original source text
	Base  int    // Number base (10, 16, 8, or 2)
	source.Location
}

func (i *IntLiteral) INode() Node           { return i }
func (i *IntLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (i *IntLiteral) Loc() *source.Location { return &i.Location }

type FloatLiteral struct {
	Value float64 // Actual numeric value
	Raw   string  // Original source text
	source.Location
}

func (f *FloatLiteral) INode() Node           { return f }
func (f *FloatLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (f *FloatLiteral) Loc() *source.Location { return &f.Location }

type StringLiteral struct {
	Value string
	source.Location
}

func (s *StringLiteral) INode() Node           { return s }
func (s *StringLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (s *StringLiteral) Loc() *source.Location { return &s.Location }

type BoolLiteral struct {
	Value bool
	source.Location
}

func (b *BoolLiteral) INode() Node           { return b }
func (b *BoolLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (b *BoolLiteral) Loc() *source.Location { return &b.Location }

type ByteLiteral struct {
	Value string
	source.Location
}

func (b *ByteLiteral) INode() Node           { return b }
func (b *ByteLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (b *ByteLiteral) Loc() *source.Location { return &b.Location }

type IndexableExpr struct {
	Indexable Expression // The expression being indexed (array, map, etc.)
	Index     Expression // The index expression
	source.Location
}

func (i *IndexableExpr) INode() Node           { return i }
func (i *IndexableExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (i *IndexableExpr) Loc() *source.Location { return &i.Location }

type ArrayLiteralExpr struct {
	Elements []Expression
	source.Location
}

func (a *ArrayLiteralExpr) INode() Node           { return a }
func (a *ArrayLiteralExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (a *ArrayLiteralExpr) Loc() *source.Location { return &a.Location }

// StructLiteralExpr represents a struct literal expression like Point{x: 10, y: 20}
type StructLiteralExpr struct {
	StructName  IdentifierExpr
	Fields      []StructField
	IsAnonymous bool
	source.Location
}

func (s *StructLiteralExpr) INode() Node           { return s }
func (s *StructLiteralExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (s *StructLiteralExpr) Loc() *source.Location { return &s.Location }

type FunctionLiteral struct {
	Params     []Parameter
	ReturnType []DataType
	Body       *Block
	source.Location
}

func (f *FunctionLiteral) INode() Node           { return f }
func (f *FunctionLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (f *FunctionLiteral) Loc() *source.Location { return &f.Location }
