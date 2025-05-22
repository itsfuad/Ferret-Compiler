package ast

import "ferret/compiler/internal/source"

// BlockStmt represents a block of statements
type Block struct {
	Nodes []Node
	source.Location
}

func (b *Block) INode() Node           { return b }
func (b *Block) Block()                {} // Block is a marker method for statements
func (b *Block) Loc() *source.Location { return &b.Location }

// FunctionDecl represents both named and anonymous function declarations
type FunctionDecl struct {
	Identifier IdentifierExpr
	Function   *FunctionLiteral // Function literal
	source.Location
}

func (f *FunctionDecl) INode() Node           { return f }
func (f *FunctionDecl) Block()                {} // Block is a marker interface for all expressions
func (f *FunctionDecl) Loc() *source.Location { return &f.Location }

// IfStmt represents an if statement with optional else and else-if branches
type IfStmt struct {
	Condition   Expression
	Body        *Block
	Alternative Node
	source.Location
}

func (i *IfStmt) INode() Node           { return i }
func (i *IfStmt) Block()                {} // Block is a marker interface for all statements
func (i *IfStmt) Loc() *source.Location { return &i.Location }

// MethodDecl represents a method declaration
type MethodDecl struct {
	Method   IdentifierExpr
	Receiver *Parameter // Receiver parameter: e.g. in `fn (t *T) M(n int)`, `t` is the receiver
	IsRRef   bool       // Whether the receiver is a reference
	Function *FunctionLiteral
	source.Location
}

func (m *MethodDecl) INode() Node           { return m }
func (m *MethodDecl) Block()                {} // Block is a marker interface for all statements
func (m *MethodDecl) Loc() *source.Location { return &m.Location }
