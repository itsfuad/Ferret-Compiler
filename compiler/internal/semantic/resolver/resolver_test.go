package resolver

import (
	"testing"

	"compiler/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/source"
)

const (
	EXPECTED_ERROR   = "Expected error but got none"
	UNEXPECTED_ERROR = "Expected no error but got: %v"
)

// createTestAnalyzer creates a minimal analyzer for unit testing
func createTestAnalyzer(t *testing.T) (*analyzer.AnalyzerNode, *ctx.CompilerContext) {
	t.Helper()

	compilerCtx := &ctx.CompilerContext{
		Modules: make(map[string]*ctx.Module),
		Reports: make(report.Reports, 0),
	}

	// Create a test module
	importPath := "test"
	fullPath := "/test/main.fer"

	program := &ast.Program{
		ImportPath: importPath,
		FullPath:   fullPath,
		Nodes:      []ast.Node{},
	}

	module := &ctx.Module{
		AST:         program,
		SymbolTable: semantic.NewSymbolTable(nil),
	}

	compilerCtx.Modules[importPath] = module
	compilerCtx.Modules[fullPath] = module

	analyzer := analyzer.NewAnalyzerNode(program, compilerCtx, false)
	return analyzer, compilerCtx
}

func TestResolveIdentifierExpr(t *testing.T) {
	tests := []struct {
		name           string
		setupSymbols   func(*semantic.SymbolTable)
		identifierName string
		expectError    bool
	}{
		{
			name: "Valid identifier - declared variable",
			setupSymbols: func(st *semantic.SymbolTable) {
				st.Declare("x", &semantic.Symbol{
					Name: "x",
					Kind: semantic.SymbolVar,
					Type: nil,
				})
			},
			identifierName: "x",
			expectError:    false,
		},
		{
			name: "Undeclared identifier",
			setupSymbols: func(st *semantic.SymbolTable) {
				// No symbols setup - testing undeclared variable
			},
			identifierName: "undeclaredVar",
			expectError:    true,
		},
		{
			name: "Valid identifier - declared constant",
			setupSymbols: func(st *semantic.SymbolTable) {
				st.Declare("PI", &semantic.Symbol{
					Name: "PI",
					Kind: semantic.SymbolConst,
					Type: nil,
				})
			},
			identifierName: "PI",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, compilerCtx := createTestAnalyzer(t)

			// Setup symbols in the module's symbol table
			module := compilerCtx.Modules["test"]
			tt.setupSymbols(module.SymbolTable)
			// Create identifier expression with location
			identifierExpr := &ast.IdentifierExpr{
				Name: tt.identifierName,
				Location: source.Location{
					Start: &source.Position{Line: 1, Column: 1, Index: 0},
					End:   &source.Position{Line: 1, Column: len(tt.identifierName), Index: len(tt.identifierName) - 1},
				},
			}

			// Test the resolver function
			resolveIdentifierExpr(analyzer, identifierExpr)

			// Check results
			hasError := compilerCtx.Reports.HasErrors()
			if tt.expectError && !hasError {
				t.Error(EXPECTED_ERROR)
			}
			if !tt.expectError && hasError {
				t.Errorf(UNEXPECTED_ERROR, compilerCtx.Reports)
			}
		})
	}
}

func TestResolveBinaryExpr(t *testing.T) {
	tests := []struct {
		name         string
		setupSymbols func(*semantic.SymbolTable)
		leftVar      string
		rightVar     string
		expectError  bool
	}{
		{
			name: "Valid binary expression - both operands declared",
			setupSymbols: func(st *semantic.SymbolTable) {
				st.Declare("a", &semantic.Symbol{Name: "a", Kind: semantic.SymbolVar})
				st.Declare("b", &semantic.Symbol{Name: "b", Kind: semantic.SymbolVar})
			},
			leftVar:     "a",
			rightVar:    "b",
			expectError: false,
		},
		{
			name: "Invalid binary expression - left operand undeclared",
			setupSymbols: func(st *semantic.SymbolTable) {
				st.Declare("b", &semantic.Symbol{Name: "b", Kind: semantic.SymbolVar})
			},
			leftVar:     "undeclaredA",
			rightVar:    "b",
			expectError: true,
		},
		{
			name: "Invalid binary expression - right operand undeclared",
			setupSymbols: func(st *semantic.SymbolTable) {
				st.Declare("a", &semantic.Symbol{Name: "a", Kind: semantic.SymbolVar})
			},
			leftVar:     "a",
			rightVar:    "undeclaredB",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, compilerCtx := createTestAnalyzer(t)

			// Setup symbols
			module := compilerCtx.Modules["test"]
			tt.setupSymbols(module.SymbolTable)
			// Create binary expression with identifier operands
			leftIdent := ast.Expression(&ast.IdentifierExpr{
				Name: tt.leftVar,
				Location: source.Location{
					Start: &source.Position{Line: 1, Column: 1, Index: 0},
					End:   &source.Position{Line: 1, Column: len(tt.leftVar), Index: len(tt.leftVar) - 1},
				},
			})
			rightIdent := ast.Expression(&ast.IdentifierExpr{
				Name: tt.rightVar,
				Location: source.Location{
					Start: &source.Position{Line: 1, Column: 1, Index: 0},
					End:   &source.Position{Line: 1, Column: len(tt.rightVar), Index: len(tt.rightVar) - 1},
				},
			})

			binaryExpr := &ast.BinaryExpr{
				Left:  &leftIdent,
				Right: &rightIdent,
			}

			// Test resolver
			resolveExpr(analyzer, binaryExpr)

			// Check results
			hasError := compilerCtx.Reports.HasErrors()
			if tt.expectError && !hasError {
				t.Error(EXPECTED_ERROR)
			}
			if !tt.expectError && hasError {
				t.Errorf(UNEXPECTED_ERROR, compilerCtx.Reports)
			}
		})
	}
}

func TestResolveExprWithLiterals(t *testing.T) {
	tests := []struct {
		name       string
		expression ast.Expression
	}{
		{
			name:       "String literal",
			expression: &ast.StringLiteral{Value: "hello"},
		},
		{
			name:       "Integer literal",
			expression: &ast.IntLiteral{Value: 42},
		},
		{
			name:       "Float literal",
			expression: &ast.FloatLiteral{Value: 3.14},
		},
		{
			name:       "Boolean literal",
			expression: &ast.BoolLiteral{Value: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, compilerCtx := createTestAnalyzer(t)

			// Test resolver - literals should not cause errors
			resolveExpr(analyzer, tt.expression)

			// Check that no errors occurred
			if compilerCtx.Reports.HasErrors() {
				t.Errorf("Expected no error for literal but got: %v", compilerCtx.Reports)
			}
		})
	}
}

func TestResolveExprWithNil(t *testing.T) {
	t.Run("Nil expression should panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for nil expression")
			}
		}()

		analyzer, _ := createTestAnalyzer(t)
		resolveExpr(analyzer, nil)
	})
}

// createVarDeclStmt creates a variable declaration statement for testing
func createVarDeclStmt(name string, isConst bool) *ast.VarDeclStmt {
	return &ast.VarDeclStmt{
		Variables: []*ast.VariableToDeclare{
			{
				Identifier: &ast.IdentifierExpr{
					Name: name,
					Location: source.Location{
						Start: &source.Position{Line: 1, Column: 1},
						End:   &source.Position{Line: 1, Column: len(name) + 1},
					},
				},
			},
		},
		IsConst: isConst,
	}
}

// verifySymbolDeclaration checks if a symbol was correctly declared with expected properties
func verifySymbolDeclaration(t *testing.T, symbolTable *semantic.SymbolTable, name string, isConst bool) {
	t.Helper()
	sym, found := symbolTable.Lookup(name)
	if !found {
		t.Errorf("Expected symbol %s to be declared", name)
		return
	}

	expectedKind := semantic.SymbolVar
	if isConst {
		expectedKind = semantic.SymbolConst
	}

	if sym.Kind != expectedKind {
		t.Errorf("Expected symbol kind %v, got %v", expectedKind, sym.Kind)
	}
}

func TestResolveVarDecl(t *testing.T) {
	tests := []struct {
		name           string
		varName        string
		isConst        bool
		preDeclareName string // if non-empty, pre-declare this variable
		expectError    bool
	}{
		{
			name:        "Valid variable declaration",
			varName:     "x",
			isConst:     false,
			expectError: false,
		},
		{
			name:        "Valid constant declaration",
			varName:     "PI",
			isConst:     true,
			expectError: false,
		},
		{
			name:           "Variable redeclaration should fail",
			varName:        "a",
			isConst:        false,
			preDeclareName: "a",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, ctx := createTestAnalyzer(t)

			// Setup pre-declared symbol if needed
			module := ctx.GetModule(r.Program.ImportPath)
			if tt.preDeclareName != "" {
				module.SymbolTable.Declare(tt.preDeclareName, &semantic.Symbol{
					Name: tt.preDeclareName,
					Kind: semantic.SymbolVar,
					Type: nil,
				})
			}

			// Create and resolve the statement
			stmt := createVarDeclStmt(tt.varName, tt.isConst)
			resolveVarDecl(r, stmt)

			// Verify the results
			hasErrors := ctx.Reports.HasErrors()
			if tt.expectError && !hasErrors {
				t.Error(EXPECTED_ERROR)
			} else if !tt.expectError && hasErrors {
				t.Errorf(UNEXPECTED_ERROR, ctx.Reports)
			}

			// Check symbol declaration if no error was expected
			if !tt.expectError {
				verifySymbolDeclaration(t, module.SymbolTable, tt.varName, tt.isConst)
			}
		})
	}
}

// createAssignmentStmt creates an assignment statement for testing
func createAssignmentStmt(varName string, value interface{}) *ast.AssignmentStmt {
	var rightExpr ast.Expression
	switch v := value.(type) {
	case int64:
		rightExpr = &ast.IntLiteral{Value: v}
	default:
		rightExpr = &ast.IntLiteral{Value: 42}
	}

	return &ast.AssignmentStmt{
		Left: &ast.ExpressionList{
			&ast.IdentifierExpr{
				Name: varName,
				Location: source.Location{
					Start: &source.Position{Line: 1, Column: 1},
					End:   &source.Position{Line: 1, Column: len(varName) + 1},
				},
			},
		},
		Right: &ast.ExpressionList{rightExpr},
	}
}

// setupDeclaredVariable declares a variable in the symbol table
func setupDeclaredVariable(st *semantic.SymbolTable, name string) {
	st.Declare(name, &semantic.Symbol{
		Name: name,
		Kind: semantic.SymbolVar,
		Type: nil,
	})
}

// runAssignmentTest runs a single assignment test case
func runAssignmentTest(t *testing.T, name string, varName string, declareVar bool, expectError bool) {
	t.Helper()
	r, ctx := createTestAnalyzer(t)

	module := ctx.GetModule(r.Program.ImportPath)
	if module != nil && declareVar {
		setupDeclaredVariable(module.SymbolTable, varName)
	}

	stmt := createAssignmentStmt(varName, 42)
	resolveAssignment(r, stmt)

	hasError := r.Ctx.Reports.HasErrors()
	if expectError != hasError {
		if expectError {
			t.Errorf("Expected error to be reported for %s", name)
		} else {
			t.Errorf("Unexpected error reported for %s", name)
		}
	}
}

func TestResolveAssignment(t *testing.T) {
	t.Run("Valid assignment to declared variable", func(t *testing.T) {
		runAssignmentTest(t, "Valid assignment", "a", true, false)
	})

	t.Run("Assignment to undeclared variable should fail", func(t *testing.T) {
		runAssignmentTest(t, "Undeclared assignment", "undeclared", false, true)
	})
}

// Benchmark core resolver functions
func BenchmarkResolveIdentifierExpr(b *testing.B) {
	analyzer, _ := createTestAnalyzer(&testing.T{})

	// Setup a symbol
	module := analyzer.Ctx.Modules["test"]
	module.SymbolTable.Declare("x", &semantic.Symbol{
		Name: "x",
		Kind: semantic.SymbolVar,
	})

	identifierExpr := &ast.IdentifierExpr{Name: "x"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolveIdentifierExpr(analyzer, identifierExpr)
	}
}

func BenchmarkResolveBinaryExpr(b *testing.B) {
	analyzer, _ := createTestAnalyzer(&testing.T{})

	// Setup symbols
	module := analyzer.Ctx.Modules["test"]
	module.SymbolTable.Declare("a", &semantic.Symbol{Name: "a", Kind: semantic.SymbolVar})
	module.SymbolTable.Declare("b", &semantic.Symbol{Name: "b", Kind: semantic.SymbolVar})

	leftIdent := ast.Expression(&ast.IdentifierExpr{
		Name: "a",
		Location: source.Location{
			Start: &source.Position{Line: 1, Column: 1, Index: 0},
			End:   &source.Position{Line: 1, Column: 1, Index: 0},
		},
	})
	rightIdent := ast.Expression(&ast.IdentifierExpr{
		Name: "b",
		Location: source.Location{
			Start: &source.Position{Line: 1, Column: 1, Index: 0},
			End:   &source.Position{Line: 1, Column: 1, Index: 0},
		},
	})
	binaryExpr := &ast.BinaryExpr{
		Left:  &leftIdent,
		Right: &rightIdent,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolveExpr(analyzer, binaryExpr)
	}
}
