package ast

import (
	"ferret/compiler/internal/source"
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

// MultiProgram represents a program that may span multiple files.
// It holds a map of file paths to their corresponding Program ASTs.
type MultiProgram struct {
	Programs map[string]*Program
}

// Loc returns the source location of the MultiProgram.
// For now, this returns nil as the concept of a single location for a multi-file program is ambiguous.
// It might represent the entry file or a span of all files in the future.
func (mp *MultiProgram) Loc() *source.Location {
	return nil // Placeholder
}

func (mp *MultiProgram) INode() Node { return mp }
