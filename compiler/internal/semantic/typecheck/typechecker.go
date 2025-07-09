package typecheck

import (
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/types"
	"fmt"
	"os"
	"reflect"
)

type TypeChecker struct {
	program     *ast.Program
	Debug       bool
	ctx         *ctx.CompilerContext
	CheckedMods map[string]bool // file path -> checked
	CurrentFile string          // current file being typechecked
}

func NewTypeChecker(program *ast.Program, ctx *ctx.CompilerContext, debug bool) *TypeChecker {
	return &TypeChecker{
		program:     program,
		ctx:         ctx,
		Debug:       debug,
		CheckedMods: make(map[string]bool),
	}
}

func (tc *TypeChecker) CheckProgram(prog *ast.Program) {
	tc.CurrentFile = prog.FilePath
	if tc.CurrentFile == "" {
		fmt.Println("[TypeChecker] Current file is empty. Skipping type checking.")
		return
	}
	if tc.Debug {
		fmt.Printf("[TypeChecker] Starting type checking for %s\n", tc.CurrentFile)
	}
	// Build import alias map: alias -> file path (for error reporting only)
	for _, node := range prog.Nodes {
		if imp, ok := node.(*ast.ImportStmt); ok {
			if imp.ModuleName != "" && imp.FilePath != "" {
				tc.ctx.AliasToModuleName[imp.ModuleName] = imp.FilePath
				if tc.Debug {
					fmt.Printf("[TypeChecker] Import alias: %s -> %s\n", imp.ModuleName, imp.FilePath)
				}
			}
		}
	}
	for _, node := range prog.Nodes {
		tc.checkNode(node)
	}
	if tc.Debug {
		fmt.Printf("[TypeChecker] Finished type checking for %s\n", tc.CurrentFile)
	}
}

func (tc *TypeChecker) checkNode(node ast.Node) {
	switch n := node.(type) {
	case *ast.VarDeclStmt:
		tc.checkVarDecl(n)
	case *ast.AssignmentStmt:
		tc.checkAssignment(n)
	case *ast.ExpressionStmt:
		if n.Expressions != nil {
			for _, expr := range *n.Expressions {
				tc.checkExpr(expr)
			}
		}
	case *ast.Block:
		for _, sub := range n.Nodes {
			tc.checkNode(sub)
		}
	case *ast.ImportStmt:
		if tc.ctx == nil {
			fmt.Println("[TypeChecker] CompilerContext is not set for import type checking.")
			os.Exit(1)
		}
		alias := n.ModuleName
		_, ok := tc.ctx.Modules[tc.CurrentFile].SymbolTable.Imports[alias]
		if !ok {
			tc.ctx.Reports.Add(tc.CurrentFile, n.Loc(), fmt.Sprintf("unknown module: %s", alias), report.TYPECHECK_PHASE).SetLevel(report.SEMANTIC_ERROR)
			if tc.Debug {
				fmt.Printf("[TypeChecker] Import alias '%s' not found in Imports map.\n", alias)
			}
			return
		}
		// Only typecheck the imported module if not already checked
		if tc.CheckedMods[alias] {
			return
		}
		modAST := tc.ctx.GetModule(tc.ctx.AliasToModuleName[alias]).AST
		if modAST == nil {
			modAST = tc.ctx.GetModule(tc.ctx.AliasToModuleName[alias]).AST
		}
		if modAST == nil {
			tc.ctx.Reports.Add(tc.CurrentFile, n.Loc(), fmt.Sprintf("module not found for alias '%s' (file: %s)", alias, tc.ctx.AliasToModuleName[alias]), report.TYPECHECK_PHASE).SetLevel(report.SEMANTIC_ERROR)
			if tc.Debug {
				fmt.Printf("[TypeChecker] Module file '%s' not found for alias '%s'\n", tc.ctx.AliasToModuleName[alias], alias)
			}
			return
		}
		checker := NewTypeChecker(modAST, tc.ctx, tc.Debug)
		checker.CheckedMods = tc.CheckedMods
		checker.CheckProgram(modAST)
		tc.CheckedMods[alias] = true
	default:
		fmt.Printf("[TypeChecker] type checking for %s is not implemented yet.\n", reflect.TypeOf(node))
		os.Exit(0)
	}
}

func (tc *TypeChecker) checkVarDecl(stmt *ast.VarDeclStmt) {
	for i, v := range stmt.Variables {
		if v.ExplicitType != nil && i < len(stmt.Initializers) && stmt.Initializers[i] != nil {
			initType := tc.checkExpr(stmt.Initializers[i])
			declType := v.ExplicitType.Type()
			if initType != declType {
				tc.ctx.Reports.Add(tc.CurrentFile, v.Identifier.Loc(), fmt.Sprintf("type mismatch: expected %s, got %s", declType, initType), report.TYPECHECK_PHASE).SetLevel(report.SEMANTIC_ERROR)
				if tc.Debug {
					fmt.Printf("[TypeChecker] : %s -> VarDecl type mismatch: expected %s, got %s\n", tc.CurrentFile, declType, initType)
				}
			}
		}
	}
}

func (tc *TypeChecker) checkAssignment(stmt *ast.AssignmentStmt) {
	for i, lhs := range stmt.Left {
		lhsType := tc.checkExpr(lhs)
		if i < len(stmt.Right) {
			rhsType := tc.checkExpr(stmt.Right[i])
			if lhsType != rhsType {
				tc.ctx.Reports.Add(tc.CurrentFile, lhs.Loc(), fmt.Sprintf("type mismatch in assignment: %s = %s", lhsType, rhsType), report.TYPECHECK_PHASE).SetLevel(report.SEMANTIC_ERROR)
				if tc.Debug {
					fmt.Printf("[TypeChecker] Assignment type mismatch: %s = %s\n", lhsType, rhsType)
				}
			}
		}
	}
}

func (tc *TypeChecker) checkExpr(expr ast.Expression) types.TYPE_NAME {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return types.INT32
	case *ast.FloatLiteral:
		return types.FLOAT64
	case *ast.StringLiteral:
		return types.STRING
	case *ast.BoolLiteral:
		return types.BOOL
	case *ast.ByteLiteral:
		return types.BYTE
	case *ast.IdentifierExpr:
		sym, found := tc.ctx.Modules[tc.CurrentFile].SymbolTable.Lookup(e.Name)
		if found && sym.Type != nil {
			if dt, ok := sym.Type.(ast.DataType); ok {
				return dt.Type()
			}
		}
		return ""
	case *ast.BinaryExpr:
		leftType := tc.checkExpr(e.Left)
		rightType := tc.checkExpr(e.Right)
		if leftType != rightType {
			tc.ctx.Reports.Add(tc.CurrentFile, e.Loc(), fmt.Sprintf("type mismatch in binary expr: %s %s %s", leftType, e.Operator.Value, rightType), report.TYPECHECK_PHASE).SetLevel(report.SEMANTIC_ERROR)
			if tc.Debug {
				fmt.Printf("[TypeChecker] BinaryExpr type mismatch: %s %s %s\n", leftType, e.Operator.Value, rightType)
			}
		}
		return leftType
	case *ast.UnaryExpr:
		return tc.checkExpr(e.Operand)
	case *ast.PrefixExpr:
		return tc.checkExpr(e.Operand)
	case *ast.PostfixExpr:
		return tc.checkExpr(e.Operand)
	case *ast.FunctionCallExpr:
		fmt.Println("[TypeChecker] type checking for FunctionCallExpr is not implemented yet.")
		os.Exit(0)
		return types.VOID
	case *ast.FieldAccessExpr:
		fmt.Println("[TypeChecker] type checking for FieldAccessExpr is not implemented yet.")
		os.Exit(0)
		return ""
	case *ast.ScopeResolutionExpr:
		alias := e.Module.Name
		importedTable, ok := tc.ctx.Modules[tc.CurrentFile].SymbolTable.Imports[alias]
		if !ok {
			tc.ctx.Reports.Add(tc.CurrentFile, e.Module.Loc(), fmt.Sprintf("unknown module: %s", alias), report.TYPECHECK_PHASE).SetLevel(report.SEMANTIC_ERROR)
			if tc.Debug {
				fmt.Printf("[TypeChecker] Alias '%s' not found in Imports map\n", alias)
			}
			return ""
		}
		sym, found := importedTable.Lookup(e.Identifier.Name)
		if found && sym.Type != nil {
			if dt, ok := sym.Type.(ast.DataType); ok {
				return dt.Type()
			}
		}
		tc.ctx.Reports.Add(tc.CurrentFile, e.Identifier.Loc(), fmt.Sprintf("undeclared symbol in module '%s': %s", alias, e.Identifier.Name), report.TYPECHECK_PHASE).SetLevel(report.SEMANTIC_ERROR)
		if tc.Debug {
			fmt.Printf("[TypeChecker] Symbol '%s' not found in module '%s'\n", e.Identifier.Name, alias)
		}
		return ""
	default:
		fmt.Printf("[TypeChecker] type checking for <%s> is not implemented yet.\n", reflect.TypeOf(expr))
		os.Exit(0)
		return ""
	}
}
