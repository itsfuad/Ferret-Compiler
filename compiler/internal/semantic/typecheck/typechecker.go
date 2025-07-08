package typecheck

import (
	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/types"
	"fmt"
	"os"
	"reflect"
)

type TypeChecker struct {
	Symbols      *semantic.SymbolTable
	Reports      *report.Reports
	Debug        bool
	Ctx          *ctx.CompilerContext
	ModuleTables map[string]*semantic.SymbolTable // module file path -> symbol table
	AliasToPath  map[string]string                // import alias -> file path
	CheckedMods  map[string]bool                  // file path -> checked
	CurrentFile  string                           // current file being typechecked
}

func NewTypeChecker(symbols *semantic.SymbolTable, reports *report.Reports, debug bool) *TypeChecker {
	return &TypeChecker{
		Symbols:      symbols,
		Reports:      reports,
		Debug:        debug,
		ModuleTables: make(map[string]*semantic.SymbolTable),
		AliasToPath:  make(map[string]string),
		CheckedMods:  make(map[string]bool),
	}
}

func (tc *TypeChecker) SetContext(ctx *ctx.CompilerContext) {
	tc.Ctx = ctx
}

func (tc *TypeChecker) CheckProgram(prog *ast.Program) {
	if prog == nil {
		return
	}
	tc.CurrentFile = prog.FilePath
	if tc.Debug {
		fmt.Printf("[TypeChecker] Starting type checking for %s\n", tc.CurrentFile)
	}
	// Set current file for error reporting
	// Build import alias map: alias -> file path
	for _, node := range prog.Nodes {
		if imp, ok := node.(*ast.ImportStmt); ok {
			if imp.ModuleName != "" && imp.FilePath != "" {
				tc.AliasToPath[imp.ModuleName] = imp.FilePath
				if tc.Debug {
					fmt.Printf("[TypeChecker] Import alias: %s -> %s\n", imp.ModuleName, imp.FilePath)
				}
			}
		}
	}
	// Cache the current module's symbol table
	if prog != nil && prog.FilePath != "" {
		modName := prog.FilePath
		tc.ModuleTables[modName] = tc.Symbols
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
		if tc.Ctx == nil {
			fmt.Println("[TypeChecker] CompilerContext is not set for import type checking.")
			os.Exit(1)
		}
		alias := n.ModuleName
		filePath, ok := tc.AliasToPath[alias]
		if !ok {
			tc.Reports.Add(tc.CurrentFile, n.Loc(), fmt.Sprintf("unknown module: %s", alias), report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
			if tc.Debug {
				fmt.Printf("[TypeChecker] Import alias '%s' not found in alias map.\n", alias)
			}
			return
		}

		if tc.CheckedMods[filePath] {
			return
		}
		modAST := tc.Ctx.GetModule(ctx.LocalModuleKey(filePath))
		if modAST == nil {
			modAST = tc.Ctx.GetModule(ctx.RemoteModuleKey(filePath))
		}
		if modAST == nil {
			tc.Reports.Add(tc.CurrentFile, n.Loc(), fmt.Sprintf("module not found for alias '%s' (file: %s)", alias, filePath), report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
			if tc.Debug {
				fmt.Printf("[TypeChecker] Module file '%s' not found for alias '%s'\n", filePath, alias)
			}
			return
		}
		modTable, found := tc.ModuleTables[filePath]
		if !found {
			modTable = semantic.NewSymbolTable(nil)
			for _, node := range modAST.Nodes {
				if v, ok := node.(*ast.VarDeclStmt); ok {
					for _, varDecl := range v.Variables {
						modTable.Declare(varDecl.Identifier.Name, &semantic.Symbol{Name: varDecl.Identifier.Name, Kind: semantic.SymbolVar, Type: varDecl.ExplicitType})
					}
				}
			}
			tc.ModuleTables[filePath] = modTable
		}
		// Recursively type check the imported module
		checker := NewTypeChecker(modTable, tc.Reports, tc.Debug)
		checker.SetContext(tc.Ctx)
		checker.ModuleTables = tc.ModuleTables
		checker.AliasToPath = tc.AliasToPath
		checker.CheckedMods = tc.CheckedMods
		checker.CheckProgram(modAST)
		tc.CheckedMods[filePath] = true
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
				tc.Reports.Add(tc.CurrentFile, v.Identifier.Loc(), fmt.Sprintf("type mismatch: expected %s, got %s", declType, initType), report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
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
				tc.Reports.Add(tc.CurrentFile, lhs.Loc(), fmt.Sprintf("type mismatch in assignment: %s = %s", lhsType, rhsType), report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
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
		sym, found := tc.Symbols.Lookup(e.Name)
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
			tc.Reports.Add(tc.CurrentFile, e.Loc(), fmt.Sprintf("type mismatch in binary expr: %s %s %s", leftType, e.Operator.Value, rightType), report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
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
		importedTable, ok := tc.Symbols.Imports[alias]
		if !ok {
			tc.Reports.Add(tc.CurrentFile, e.Module.Loc(), fmt.Sprintf("unknown module: %s", alias), report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
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
		tc.Reports.Add(tc.CurrentFile, e.Identifier.Loc(), fmt.Sprintf("undeclared symbol in module '%s': %s", alias, e.Identifier.Name), report.SEMANTIC_PHASE).SetLevel(report.SEMANTIC_ERROR)
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
