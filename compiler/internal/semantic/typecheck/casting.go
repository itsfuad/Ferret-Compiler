package typecheck

import (
	"errors"
	"ferret/compiler/colors"
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic"
	"ferret/compiler/internal/semantic/analyzer"
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/types"
	"fmt"
	"strings"
)

// checkCastExprType validates type cast expressions and returns the target type
func checkCastExprType(r *analyzer.AnalyzerNode, cast *ast.CastExpr, cm *modules.Module) stype.Type {
	// Evaluate the source expression type
	sourceType := evaluateExpressionType(r, *cast.Value, cm)
	if sourceType == nil {
		return nil
	}

	// Convert AST target type to semantic type
	targetType, err := semantic.DeriveSemanticType(cast.TargetType, cm)
	if err != nil || targetType == nil {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			cast.Loc(),
			fmt.Sprintf("invalid target type in cast expression: %v", err),
			report.TYPECHECK_PHASE,
		)
		return nil
	}

	// Check if the cast is valid
	isValid, err := isExplicitCastable(sourceType, targetType)
	if err != nil || !isValid {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			cast.Loc(),
			fmt.Sprintf("cannot cast from '%s' to '%s': %v", sourceType.String(), targetType.String(), err),
			report.TYPECHECK_PHASE,
		)
		return nil // Return nil if cast is invalid
	}

	return targetType
}

// --- EXPLICIT CASTING CHECKS ---

func isExplicitCastable(sourceType, targetType stype.Type) (bool, error) {
	// primitive types can be casted to each other
	if sourceType == nil || targetType == nil {
		return false, errors.New("source or target type is nil")
	}

	if isImplicitCastable(sourceType, targetType) {
		return true, nil // Implicit cast is also valid for explicit cast
	}

	sourceUnwrapped := semantic.UnwrapType(sourceType) // Unwrap any type aliases
	targetUnwrapped := semantic.UnwrapType(targetType) // Unwrap any type aliases

	// Check if both are primitive types
	if _, ok := sourceUnwrapped.(*stype.PrimitiveType); ok {
		if _, ok := targetUnwrapped.(*stype.PrimitiveType); ok {
			// Both are primitive types, check if they are castable
			return isPrimitiveExplicitCastable(sourceUnwrapped, targetUnwrapped)
		}
	}

	//structs can be casted to other struct and interfaces and interfaces can be casted to structs
	if ss, ok := sourceUnwrapped.(*stype.StructType); ok {
		//target must be a struct or interface. for now skip interface
		if ts, ok := targetUnwrapped.(*stype.StructType); ok {
			//both are struct
			return isStructExplicitCastable(ss, ts)
		}
	}

	return false, fmt.Errorf("no valid cast found from '%s' to '%s'", sourceType.String(), targetType.String())
}

func isStructExplicitCastable(sourceType, targetType *stype.StructType) (bool, error) {
	//targets all properties must present

	fieldErrors := make([]string, 0, len(targetType.Fields))

	for fieldName, fieldType := range targetType.Fields {
		if sourceFieldType, exists := sourceType.Fields[fieldName]; !exists {
			fieldErrors = append(fieldErrors, colors.RED.Sprintf(" - missing field '%s'", fieldName))
		} else if !isImplicitCastable(sourceFieldType, fieldType) {
			fieldErrors = append(fieldErrors, colors.PURPLE.Sprintf(" - field '%s' type required '%s', but got '%s'", fieldName, fieldType.String(), sourceFieldType.String()))
		}
	}

	if len(fieldErrors) > 0 {
		errMsg := colors.WHITE.Sprintf("\n%s and %s has field missmatch\n", sourceType.String(), targetType.String())
		return false, fmt.Errorf("%s%s", errMsg, strings.Join(fieldErrors, "\n"))
	}

	fmt.Printf("Successfully casted struct '%s' to '%s'\n", sourceType.String(), targetType.String())

	return true, nil
}

func isPrimitiveExplicitCastable(sourceType, targetType stype.Type) (bool, error) {
	sourcePrim, sOk := sourceType.(*stype.PrimitiveType)
	targetPrim, tOk := targetType.(*stype.PrimitiveType)

	if !sOk || !tOk {
		return false, errors.New("both source and target must be primitive types")
	}

	// Allow ALL numeric to numeric casting with explicit "as" keyword
	// The developer explicitly requests the conversion, so allow both widening and narrowing
	if types.IsNumericTypeName(sourcePrim.Name) && types.IsNumericTypeName(targetPrim.Name) {
		return true, nil
	}

	// No valid cast found
	return false, fmt.Errorf("no valid cast found from '%s' to '%s'", sourceType.String(), targetType.String())
}

// -- IMPLICIT CASTING CHECKS ---

// isImplicitCastable checks if a value of type 'source' can be assigned to 'target'
// Note: This function has limited type resolution capability. For full alias resolution,
// the type checker should use resolveTypeAlias with analyzer context.
func isImplicitCastable(target, source stype.Type) bool {
	// Handle user types (aliases) - limited resolution without symbol table access
	resolvedTarget := semantic.UnwrapType(target)
	resolvedSource := semantic.UnwrapType(source)

	colors.PURPLE.Printf("Checking Implicit Cast: %v → %v ", resolvedSource, resolvedTarget)

	if isPrimitiveImplicitCastable(resolvedTarget, resolvedSource) || isArrayImplicitCastable(resolvedTarget, resolvedSource) || isFunctionImplicitCastable(resolvedTarget, resolvedSource) || isStructImpliticCastable(resolvedTarget, resolvedSource) {
		colors.GREEN.Println(" ✔ ")
		return true
	}

	colors.RED.Println(" ✘ ")

	return false
}

// ===== HELPER FUNCTIONS =====

// isPrimitiveImplicitCastable checks if source can be promoted to target (implicit conversion)
func isPrimitiveImplicitCastable(target, source stype.Type) bool {
	targetPrim, targetOk := target.(*stype.PrimitiveType)
	sourcePrim, sourceOk := source.(*stype.PrimitiveType)

	if !targetOk || !sourceOk {
		return false
	}

	if targetPrim.TypeName() == sourcePrim.TypeName() {
		return true // Same type is always assignable
	}

	// Define promotion rules
	promotions := map[types.TYPE_NAME][]types.TYPE_NAME{
		// Integer promotions (smaller -> larger)
		types.INT16:  {types.INT8},
		types.INT32:  {types.INT8, types.INT16},
		types.INT64:  {types.INT8, types.INT16, types.INT32},
		types.UINT16: {types.UINT8},
		types.UINT32: {types.UINT8, types.UINT16},
		types.UINT64: {types.UINT8, types.UINT16, types.UINT32},

		// Float promotions (int -> float, smaller float -> larger float)
		types.FLOAT32: {},
		types.FLOAT64: {types.FLOAT32},
	}

	if allowedSources, exists := promotions[targetPrim.Name]; exists {
		for _, allowedSource := range allowedSources {
			if sourcePrim.Name == allowedSource {
				return true
			}
		}
	}

	return false
}

// isArrayImplicitCastable checks array type compatibility
func isArrayImplicitCastable(target, source stype.Type) bool {
	targetArray, targetOk := target.(*stype.ArrayType)
	sourceArray, sourceOk := source.(*stype.ArrayType)

	if !targetOk || !sourceOk {
		return false
	}

	return isImplicitCastable(targetArray.ElementType, sourceArray.ElementType)
}

// isFunctionImplicitCastable checks function type compatibility
func isFunctionImplicitCastable(target, source stype.Type) bool {
	targetFunc, targetOk := target.(*stype.FunctionType)
	sourceFunc, sourceOk := source.(*stype.FunctionType)

	if !targetOk || !sourceOk {
		return false
	}

	// Parameter and return type counts must match
	if len(targetFunc.Parameters) != len(sourceFunc.Parameters) {
		return false
	}

	// Parameters are contravariant, returns are covariant
	for i := range targetFunc.Parameters {
		if !isImplicitCastable(sourceFunc.Parameters[i], targetFunc.Parameters[i]) {
			return false
		}
	}

	// Compare return types
	if targetFunc.ReturnType == nil && sourceFunc.ReturnType == nil {
		return true
	}
	if targetFunc.ReturnType == nil || sourceFunc.ReturnType == nil {
		return false
	}

	return isImplicitCastable(targetFunc.ReturnType, sourceFunc.ReturnType)
}

// isStructImpliticCastable checks structural compatibility of structs
func isStructImpliticCastable(target, source stype.Type) bool {
	targetStruct, targetOk := target.(*stype.StructType)
	sourceStruct, sourceOk := source.(*stype.StructType)

	if !targetOk || !sourceOk {
		return false
	}

	// Source must have all fields that target requires, with compatible types
	for fieldName, targetFieldType := range targetStruct.Fields {
		sourceFieldType, exists := sourceStruct.Fields[fieldName]
		if !exists || !isImplicitCastable(targetFieldType, sourceFieldType) {
			return false
		}
	}

	return true
}
