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
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && hasError {
				t.Errorf("Expected no error but got: %v", compilerCtx.Reports)
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
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && hasError {
				t.Errorf("Expected no error but got: %v", compilerCtx.Reports)
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

func TestResolveVarDecl(t *testing.T) {
	tests := []struct {
		name           string
		stmt           *ast.VarDeclStmt
		setupSymbols   func(*semantic.SymbolTable)
		expectError    bool
		expectedSymbol string
	}{
		{
			name: "Valid variable declaration",
			stmt: &ast.VarDeclStmt{
				Variables: []*ast.VariableToDeclare{
					{
						Identifier: &ast.IdentifierExpr{
							Name: "x",
							Location: source.Location{
								Start: &source.Position{Line: 1, Column: 1},
								End:   &source.Position{Line: 1, Column: 2},
							},
						},
					},
				},
				IsConst: false,
			},
			setupSymbols: func(st *semantic.SymbolTable) {
				// No existing symbols needed for this test
			},
			expectError:    false,
			expectedSymbol: "x",
		},
		{
			name: "Valid constant declaration",
			stmt: &ast.VarDeclStmt{
				Variables: []*ast.VariableToDeclare{
					{
						Identifier: &ast.IdentifierExpr{
							Name: "PI",
							Location: source.Location{
								Start: &source.Position{Line: 1, Column: 1},
								End:   &source.Position{Line: 1, Column: 3},
							},
						},
					},
				},
				IsConst: true,
			},
			setupSymbols: func(st *semantic.SymbolTable) {
				// No existing symbols needed for this test
			},
			expectError:    false,
			expectedSymbol: "PI",
		},
		{
			name: "Variable redeclaration should fail",
			stmt: &ast.VarDeclStmt{
				Variables: []*ast.VariableToDeclare{
					{
						Identifier: &ast.IdentifierExpr{
							Name: "a",
							Location: source.Location{
								Start: &source.Position{Line: 1, Column: 1},
								End:   &source.Position{Line: 1, Column: 2},
							},
						},
					},
				},
				IsConst: false,
			},
			setupSymbols: func(st *semantic.SymbolTable) {
				// Pre-declare 'a' to test redeclaration error
				st.Declare("a", &semantic.Symbol{
					Name: "a",
					Kind: semantic.SymbolVar,
					Type: nil,
				})
			},
			expectError:    true,
			expectedSymbol: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, ctx := createTestAnalyzer(t)

			// Setup symbols for this test
			module := ctx.GetModule(r.Program.ImportPath)
			if module != nil {
				tt.setupSymbols(module.SymbolTable)
			}

			resolveVarDecl(r, tt.stmt)

			if tt.expectError {
				// Check that an error was reported
				if r.Ctx.Reports.HasErrors() == false {
					t.Errorf("Expected error to be reported for %s", tt.name)
				}
			} else {
				// Check that symbol was declared
				if tt.expectedSymbol != "" {
					module := r.Ctx.GetModule(r.Program.ImportPath)
					if module == nil {
						t.Fatalf("Module not found")
					}
					sym, found := module.SymbolTable.Lookup(tt.expectedSymbol)
					if !found {
						t.Errorf("Expected symbol %s to be declared", tt.expectedSymbol)
					} else {
						expectedKind := semantic.SymbolVar
						if tt.stmt.IsConst {
							expectedKind = semantic.SymbolConst
						}
						if sym.Kind != expectedKind {
							t.Errorf("Expected symbol kind %v, got %v", expectedKind, sym.Kind)
						}
					}
				}
			}
		})
	}
}

func TestResolveAssignment(t *testing.T) {
	tests := []struct {
		name         string
		stmt         *ast.AssignmentStmt
		setupSymbols func(*semantic.SymbolTable)
		expectError  bool
	}{
		{
			name: "Valid assignment to declared variable",
			stmt: &ast.AssignmentStmt{
				Left: &ast.ExpressionList{
					&ast.IdentifierExpr{
						Name: "a",
						Location: source.Location{
							Start: &source.Position{Line: 1, Column: 1},
							End:   &source.Position{Line: 1, Column: 2},
						},
					},
				},
				Right: &ast.ExpressionList{
					&ast.IntLiteral{Value: 42},
				},
			},
			setupSymbols: func(st *semantic.SymbolTable) {
				// Declare 'a' variable
				st.Declare("a", &semantic.Symbol{
					Name: "a",
					Kind: semantic.SymbolVar,
					Type: nil,
				})
			},
			expectError: false,
		},
		{
			name: "Assignment to undeclared variable should fail",
			stmt: &ast.AssignmentStmt{
				Left: &ast.ExpressionList{
					&ast.IdentifierExpr{
						Name: "undeclared",
						Location: source.Location{
							Start: &source.Position{Line: 1, Column: 1},
							End:   &source.Position{Line: 1, Column: 10},
						},
					},
				},
				Right: &ast.ExpressionList{
					&ast.IntLiteral{Value: 42},
				},
			},
			setupSymbols: func(st *semantic.SymbolTable) {
				// No symbols setup - testing undeclared variable
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, ctx := createTestAnalyzer(t)

			// Setup symbols for this test
			module := ctx.GetModule(r.Program.ImportPath)
			if module != nil {
				tt.setupSymbols(module.SymbolTable)
			}

			resolveAssignment(r, tt.stmt)

			if tt.expectError {
				if r.Ctx.Reports.HasErrors() == false {
					t.Errorf("Expected error to be reported for %s", tt.name)
				}
			} else {
				// Check for unexpected errors
				if r.Ctx.Reports.HasErrors() {
					t.Errorf("Unexpected error reported for %s", tt.name)
				}
			}
		})
	}
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
