package typecheck

import (
	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/types"
)

// CheckProgram performs type checking on the entire program
func CheckProgram(r *analyzer.AnalyzerNode) {
	for _, node := range r.Program.Nodes {
		checkNode(r, node)
	}
	if r.Debug {
		colors.GREEN.Printf("Type checked '%s'\n", r.Program.FullPath)
	}
}

// checkNode performs type checking on a single AST node
func checkNode(r *analyzer.AnalyzerNode, node ast.Node) {
	switch n := node.(type) {
	case *ast.VarDeclStmt:
		checkVarDecl(r, n)
	case *ast.AssignmentStmt:
		checkAssignment(r, n)
	case *ast.ExpressionStmt:
		checkExpressionStmt(r, n)
	case *ast.TypeDeclStmt:
		checkTypeDecl(r, n)
	// Add more cases as needed
	default:
		// Skip nodes that don't need type checking
	}
}

// checkVarDecl performs type checking on variable declarations
func checkVarDecl(r *analyzer.AnalyzerNode, stmt *ast.VarDeclStmt) {
	currentModule := r.Ctx.GetModule(r.Program.ImportPath)
	if currentModule == nil {
		return
	}

	for i, v := range stmt.Variables {
		// Get the symbol from the symbol table (should have been added by resolver)
		sym, found := currentModule.SymbolTable.Lookup(v.Identifier.Name)
		if !found {
			continue // Error should have been reported by resolver
		}

		// Type check initializer if present
		if i < len(stmt.Initializers) && stmt.Initializers[i] != nil {
			initType := inferExpressionType(r, stmt.Initializers[i])

			// If explicit type is provided, check compatibility
			if sym.Type != nil {
				if initType != nil && !semantic.IsAssignableFrom(sym.Type, initType) {
					r.Ctx.Reports.Add(
						r.Program.FullPath,
						v.Identifier.Loc(),
						"type mismatch: cannot assign "+initType.String()+" to "+sym.Type.String(),
						report.TYPECHECK_PHASE,
					).SetLevel(report.SEMANTIC_ERROR)
				}
			}
		}
	}
}

// checkAssignment performs type checking on assignments
func checkAssignment(r *analyzer.AnalyzerNode, stmt *ast.AssignmentStmt) {
	currentModule := r.Ctx.GetModule(r.Program.ImportPath)
	if currentModule == nil {
		return
	}

	// Check each assignment pair
	leftExprs := *stmt.Left
	rightExprs := *stmt.Right

	for i, leftExpr := range leftExprs {
		if i >= len(rightExprs) {
			break // Mismatched assignment count - should be caught elsewhere
		}

		leftType := inferExpressionType(r, leftExpr)
		rightType := inferExpressionType(r, rightExprs[i])

		if leftType != nil && rightType != nil {
			if !semantic.IsAssignableFrom(leftType, rightType) {
				r.Ctx.Reports.Add(
					r.Program.FullPath,
					leftExpr.Loc(),
					"type mismatch: cannot assign "+rightType.String()+" to "+leftType.String(),
					report.TYPECHECK_PHASE,
				).SetLevel(report.SEMANTIC_ERROR)
			}
		}
	}
}

// checkExpressionStmt performs type checking on expression statements
func checkExpressionStmt(r *analyzer.AnalyzerNode, stmt *ast.ExpressionStmt) {
	if stmt.Expressions != nil {
		for _, expr := range *stmt.Expressions {
			inferExpressionType(r, expr) // This will catch type errors in expressions
		}
	}
}

// checkTypeDecl performs type checking on type declarations
func checkTypeDecl(r *analyzer.AnalyzerNode, stmt *ast.TypeDeclStmt) {
	// Basic validation - more sophisticated checks can be added here
	if stmt.BaseType != nil {
		checkTypeValidity(r, stmt.BaseType)
	}
}

// checkTypeValidity checks if a type is valid (exists and is well-formed)
func checkTypeValidity(r *analyzer.AnalyzerNode, dataType ast.DataType) bool {
	currentModule := r.Ctx.GetModule(r.Program.ImportPath)
	if currentModule == nil {
		return false
	}

	switch t := dataType.(type) {
	case *ast.UserDefinedType:
		// Check if the user-defined type exists
		_, found := currentModule.SymbolTable.Lookup(string(t.TypeName))
		if !found {
			r.Ctx.Reports.Add(
				r.Program.FullPath,
				t.Loc(),
				"undefined type: "+string(t.TypeName),
				report.TYPECHECK_PHASE,
			).SetLevel(report.SEMANTIC_ERROR)
			return false
		}
		return true
	case *ast.ArrayType:
		return checkTypeValidity(r, t.ElementType)
	case *ast.StructType:
		// Check all field types
		for _, field := range t.Fields {
			if field.FieldType != nil && !checkTypeValidity(r, field.FieldType) {
				return false
			}
		}
		return true
	default:
		// Primitive types are always valid
		return true
	}
}

// inferExpressionType infers the type of an expression
func inferExpressionType(r *analyzer.AnalyzerNode, expr ast.Expression) semantic.Type {
	if expr == nil {
		return nil
	}

	currentModule := r.Ctx.GetModule(r.Program.ImportPath)
	if currentModule == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.IdentifierExpr:
		sym, found := currentModule.SymbolTable.Lookup(e.Name)
		if found {
			return sym.Type
		}
		return nil

	case *ast.StringLiteral:
		return semantic.CreatePrimitiveType(types.STRING)

	case *ast.IntLiteral:
		return semantic.CreatePrimitiveType(types.INT32) // Default integer type

	case *ast.FloatLiteral:
		return semantic.CreatePrimitiveType(types.FLOAT64) // Default float type

	case *ast.BoolLiteral:
		return semantic.CreatePrimitiveType(types.BOOL)

	case *ast.ByteLiteral:
		return semantic.CreatePrimitiveType(types.BYTE)

	case *ast.FieldAccessExpr:
		// Get the type of the object
		objectType := inferExpressionType(r, *e.Object)
		if structType, ok := objectType.(*semantic.StructType); ok {
			fieldType := structType.GetFieldType(e.Field.Name)
			if fieldType == nil {
				r.Ctx.Reports.Add(
					r.Program.FullPath,
					e.Field.Loc(),
					"field '"+e.Field.Name+"' not found in struct '"+structType.String()+"'",
					report.TYPECHECK_PHASE,
				).SetLevel(report.SEMANTIC_ERROR)
			}
			return fieldType
		}
		return nil

	case *ast.BinaryExpr:
		// Implement proper binary operation type rules
		leftType := inferExpressionType(r, *e.Left)
		rightType := inferExpressionType(r, *e.Right)

		if leftType == nil || rightType == nil {
			return nil
		}

		resultType := inferBinaryOperationType(e.Operator.Value, leftType, rightType)

		// Report error if binary operation is invalid
		if resultType == nil {
			r.Ctx.Reports.Add(
				r.Program.FullPath,
				e.Loc(),
				"invalid binary operation: "+leftType.String()+" "+e.Operator.Value+" "+rightType.String(),
				report.TYPECHECK_PHASE,
			).SetLevel(report.SEMANTIC_ERROR)
		}

		return resultType

	// Add more expression types as needed
	default:
		return nil
	}
}

// inferBinaryOperationType infers the result type of a binary operation
func inferBinaryOperationType(operator string, leftType, rightType semantic.Type) semantic.Type {
	switch operator {
	case "+", "-", "*", "/", "%":
		return inferArithmeticOperationType(operator, leftType, rightType)
	case "==", "!=", "<", "<=", ">", ">=":
		return inferComparisonOperationType(leftType, rightType)
	case "&&", "||":
		return inferLogicalOperationType(leftType, rightType)
	case "&", "|", "^", "<<", ">>":
		return inferBitwiseOperationType(leftType, rightType)
	default:
		return nil
	}
}

// inferArithmeticOperationType handles arithmetic operations
func inferArithmeticOperationType(operator string, leftType, rightType semantic.Type) semantic.Type {
	// Arithmetic operations - return common numeric type
	commonType := semantic.GetCommonType(leftType, rightType)
	if commonType != nil {
		return commonType
	}

	// String concatenation with + (only str + str)
	if operator == "+" {
		leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
		rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

		if leftOk && rightOk &&
			leftPrim.Name == types.STRING && rightPrim.Name == types.STRING {
			return semantic.CreatePrimitiveType(types.STRING)
		}
	}

	// If we reach here, the operation is invalid
	return nil
}

// inferComparisonOperationType handles comparison operations
func inferComparisonOperationType(leftType, rightType semantic.Type) semantic.Type {
	// Check if types are comparable
	if semantic.IsAssignableFrom(leftType, rightType) ||
		semantic.IsAssignableFrom(rightType, leftType) ||
		semantic.GetCommonType(leftType, rightType) != nil {
		return semantic.CreatePrimitiveType(types.BOOL)
	}
	return nil
}

// inferLogicalOperationType handles logical operations
func inferLogicalOperationType(leftType, rightType semantic.Type) semantic.Type {
	leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
	rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

	if leftOk && rightOk &&
		leftPrim.Name == types.BOOL && rightPrim.Name == types.BOOL {
		return semantic.CreatePrimitiveType(types.BOOL)
	}
	return nil
}

// inferBitwiseOperationType handles bitwise operations
func inferBitwiseOperationType(leftType, rightType semantic.Type) semantic.Type {
	leftPrim, leftOk := leftType.(*semantic.PrimitiveType)
	rightPrim, rightOk := rightType.(*semantic.PrimitiveType)

	if leftOk && rightOk && isIntegerType(leftPrim.Name) && isIntegerType(rightPrim.Name) {
		commonType := semantic.GetCommonType(leftType, rightType)
		if commonType != nil {
			return commonType
		}
	}
	return nil
}

// isIntegerType checks if a type is an integer type
func isIntegerType(typeName types.TYPE_NAME) bool {
	switch typeName {
	case types.INT8, types.INT16, types.INT32, types.INT64,
		types.UINT8, types.UINT16, types.UINT32, types.UINT64, types.BYTE:
		return true
	default:
		return false
	}
}
