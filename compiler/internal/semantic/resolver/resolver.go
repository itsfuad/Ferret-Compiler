package resolver

import (
	"fmt"
	"os"

	"compiler/colors"
	"compiler/internal/frontend/ast"
	"compiler/internal/report"
	"compiler/internal/semantic"
	"compiler/internal/semantic/analyzer"
	"compiler/internal/types"
)

func ResolveProgram(r *analyzer.AnalyzerNode) {
	for _, node := range r.Program.Nodes {
		resolveNode(r, node)
	}
	if r.Debug {
		colors.GREEN.Printf("Resolved '%s'\n", r.Program.FullPath)
	}
}

func resolveNode(r *analyzer.AnalyzerNode, node ast.Node) {
	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		r.Ctx.Reports.Add(r.Program.FullPath, r.Program.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}
	switch n := node.(type) {
	case *ast.ImportStmt:
		resolveImport(r, currentModule, n)
	case *ast.VarDeclStmt:
		resolveVarDecl(r, n)
	case *ast.AssignmentStmt:
		resolveAssignment(r, n)
	case *ast.ExpressionStmt:
		resolveExpressionStmt(r, n)
	case *ast.TypeDeclStmt:
		resolveTypeDecl(r, n)
	// Note: Type-related cases are handled by resolveType function, not here
	default:
		fmt.Printf("[Resolver] Node <%T> is not implemented yet\n", n)
		os.Exit(-1)
	}
}

// resolveType resolves type expressions (separate from statement/declaration resolution)
func resolveType(r *analyzer.AnalyzerNode, dataType ast.DataType) {
	if dataType == nil {
		return
	}

	switch t := dataType.(type) {
	case *ast.UserDefinedType:
		resolveUserDefinedType(r, t)
	case *ast.TypeScopeResolution:
		resolveTypeScopeResolution(r, t)
	case *ast.ArrayType:
		// Resolve the element type
		if t.ElementType != nil {
			resolveType(r, t.ElementType)
		}
	case *ast.StructType:
		// Resolve field types
		for _, field := range t.Fields {
			if field.FieldType != nil {
				resolveType(r, field.FieldType)
			}
		}
	case *ast.StringType, *ast.IntType, *ast.FloatType, *ast.BoolType, *ast.ByteType:
		// Primitive types don't need resolution
	default:
		// For any unhandled type, just ignore (primitive types are handled above)
	}
}

func resolveUserDefinedType(r *analyzer.AnalyzerNode, t *ast.UserDefinedType) {

	// Check if this is a built-in type first
	typeName := string(t.TypeName)
	if types.IsPrimitiveType(typeName) {
		// Built-in types don't need resolution
		return
	}

	// Check if the type exists in the current module's symbol table
	currentModule, err := r.Ctx.GetModule(r.Program.ImportPath)
	if err != nil {
		r.Ctx.Reports.Add(r.Program.FullPath, t.Loc(), err.Error(), report.RESOLVER_PHASE).SetLevel(report.CRITICAL_ERROR)
		return
	}

	// Look up the type in the symbol table
	sym, found := currentModule.SymbolTable.Lookup(typeName)
	if !found {
		r.Ctx.Reports.Add(r.Program.FullPath, t.Loc(), "undefined type: "+typeName, report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}

	// Verify it's actually a type
	if sym.Kind != semantic.SymbolType {
		r.Ctx.Reports.Add(r.Program.FullPath, t.Loc(), fmt.Sprintf("expected type but found %s '%s'", sym.Kind, typeName), report.RESOLVER_PHASE).SetLevel(report.SEMANTIC_ERROR)
		return
	}
}
