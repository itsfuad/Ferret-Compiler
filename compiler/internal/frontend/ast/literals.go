package ast

import "compiler/internal/source"

type IntLiteral struct {
	Value int64  // Actual numeric value
	Raw   string // Original source text
	Base  int    // Number base (10, 16, 8, or 2)
	source.Location
}

func (i *IntLiteral) INode()                {} // Impliments Node interface
func (i *IntLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (i *IntLiteral) Loc() *source.Location { return &i.Location }

type FloatLiteral struct {
	Value float64 // Actual numeric value
	Raw   string  // Original source text
	source.Location
}

func (f *FloatLiteral) INode()                {} // Impliments Node interface
func (f *FloatLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (f *FloatLiteral) Loc() *source.Location { return &f.Location }

type StringLiteral struct {
	Value string
	source.Location
}

func (s *StringLiteral) INode()                {} // Impliments Node interface
func (s *StringLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (s *StringLiteral) Loc() *source.Location { return &s.Location }

type BoolLiteral struct {
	Value bool
	source.Location
}

func (b *BoolLiteral) INode()                {} // Impliments Node interface
func (b *BoolLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (b *BoolLiteral) Loc() *source.Location { return &b.Location }

type ByteLiteral struct {
	Value string
	source.Location
}

func (b *ByteLiteral) INode()                {} // Impliments Node interface
func (b *ByteLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (b *ByteLiteral) Loc() *source.Location { return &b.Location }

type IndexableExpr struct {
	Indexable *Expression // The expression being indexed (array, map, etc.)
	Index     *Expression // The index expression
	source.Location
}

func (i *IndexableExpr) INode()                {} // Impliments Node interface
func (i *IndexableExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (i *IndexableExpr) Loc() *source.Location { return &i.Location }

type ArrayLiteralExpr struct {
	Elements []Expression
	source.Location
}

func (a *ArrayLiteralExpr) INode()                {} // Impliments Node interface
func (a *ArrayLiteralExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (a *ArrayLiteralExpr) Loc() *source.Location { return &a.Location }

// CompositeLiteralExpr represents a composite literal expression like Point{x: 10} or Map{key => val}
type CompositeLiteralExpr struct {
	TypeName    *IdentifierExpr // nil for anonymous structs
	IsAnonymous bool            // true for @struct{...} syntax
	Fields      []CompositeField
	source.Location
}

func (c *CompositeLiteralExpr) INode()                {} // Impliments Node interface
func (c *CompositeLiteralExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (c *CompositeLiteralExpr) Loc() *source.Location { return &c.Location }

type CompositeField struct {
	Key       *Expression // Key expression (identifier for structs, any expr for maps)
	Value     *Expression // Value expression
	Separator string      // ":" or "=>"
	source.Location
}

type FunctionLiteral struct {
	ID         string // Unique identifier for this literal
	Params     []Parameter
	ReturnType DataType
	Body       *Block
	source.Location
}

func (f *FunctionLiteral) INode()                {} // Impliments Node interface
func (f *FunctionLiteral) Expr()                 {} // Expr is a marker interface for all expressions
func (f *FunctionLiteral) Loc() *source.Location { return &f.Location }
