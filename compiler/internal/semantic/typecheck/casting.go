package typecheck

import (
	"ferret/compiler/internal/frontend/ast"
	"ferret/compiler/internal/modules"
	"ferret/compiler/internal/report"
	"ferret/compiler/internal/semantic"
	"ferret/compiler/internal/semantic/analyzer"
	"ferret/compiler/internal/semantic/stype"
	"ferret/compiler/internal/types"
	"ferret/compiler/internal/utils"
	str "ferret/compiler/internal/utils/strings"
	"fmt"
	"slices"
	"strings"
)

// all types with same structname with lower size will fit into the larger type
var auto_promote_map = map[types.TYPE_NAME][]types.TYPE_NAME{
	// Integer promotions (smaller -> larger)
	types.INT8:   {}, // INT8 can be assigned to nothing smaller
	types.INT16:  {types.INT8},
	types.INT32:  {types.INT8, types.INT16},
	types.INT64:  {types.INT8, types.INT16, types.INT32},
	types.UINT8:  {}, // UINT8 can be assigned to nothing smaller
	types.UINT16: {types.UINT8},
	types.UINT32: {types.UINT8, types.UINT16},
	types.UINT64: {types.UINT8, types.UINT16, types.UINT32},

	// Float promotions (int -> float, smaller float -> larger float)
	types.FLOAT32: {},
	types.FLOAT64: {types.FLOAT32},
}

func checkCastExprType(r *analyzer.AnalyzerNode, cast *ast.CastExpr, cm *modules.Module) stype.Type {
	// Evaluate the source expression type
	sourceType := evaluateExpressionType(r, *cast.Value, cm)
	if sourceType == nil {
		return &stype.Invalid{}
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
		return &stype.Invalid{}
	}

	// Check if the cast is valid
	isValid, err := isExplicitCastable(targetType, sourceType)
	if err != nil || !isValid {
		r.Ctx.Reports.AddSemanticError(
			r.Program.FullPath,
			cast.Loc(),
			fmt.Sprintf("cannot cast from '%s' to '%s': %v", sourceType, targetType, err),
			report.TYPECHECK_PHASE,
		)
		return &stype.Invalid{}
	}

	return targetType
}

func isImplicitCastable(target, source stype.Type) (bool, error) {
	// Handle primitive types
	if _, ok := target.(*stype.PrimitiveType); ok {
		return isPrimitiveImplicitCastable(target, source)
	}

	// Handle array types
	if _, ok := target.(*stype.ArrayType); ok {
		return isArrayImplicitCastable(target, source)
	}

	// function types
	if targetFunction, ok := target.(*stype.FunctionType); ok {
		if sourceFunction, ok := source.(*stype.FunctionType); ok {
			return isFunctionCompatible(targetFunction, sourceFunction)
		}
	}

	// Handle interfaces (source implements target)
	if _, ok := target.(*stype.InterfaceType); ok {
		return isInterfaceCompatible(target, source)
	}

	// Handle structs (struct-to-struct compatibility)
	if _, ok := target.(*stype.StructType); ok {
		return isStructCompatible(target, source, true)
	}

	//if target or source user type, unwrap
	if a, ok := target.(*stype.UserType); ok {
		if b, ok := source.(*stype.UserType); ok {
			if a.Name == b.Name {
				return true, nil
			}
		}
	}

	return false, fmt.Errorf("type '%s' and '%s' are not compatible", source, target)
}

func isExplicitCastable(target, source stype.Type) (bool, error) {
	// Handle primitive types
	if _, ok := target.(*stype.PrimitiveType); ok {
		return isPrimitiveExplicitCastable(target, source)
	}

	// Handle array types
	if _, ok := target.(*stype.ArrayType); ok {
		return isArrayExplicitCastable(target, source)
	}

	// Allow interface checks via the same logic as implicit
	if _, ok := target.(*stype.InterfaceType); ok {
		return isInterfaceCompatible(target, source)
	}

	// Allow struct-to-struct if explicitly compatible
	if _, ok := target.(*stype.StructType); ok {
		return isStructCompatible(target, source, false)
	}

	if _, ok := target.(*stype.UserType); ok {
		return isExplicitCastable(semantic.UnwrapType(target), source)
	}

	return false, fmt.Errorf("explicit cast not supported between %s and %s", source, target)
}

// --- PRIMITIVES ---

func isPrimitiveImplicitCastable(target, source stype.Type) (bool, error) {
	targetPrim, targetOk := target.(*stype.PrimitiveType)
	sourcePrim, sourceOk := source.(*stype.PrimitiveType)

	if !targetOk || !sourceOk {
		return false, fmt.Errorf("implicit cast not possible between types: %s to %s", target, source)
	}

	// if both typename are the same, it's trivially castable
	if targetPrim.TypeName == sourcePrim.TypeName {
		return true, nil
	}

	if allowedSources, exists := auto_promote_map[targetPrim.TypeName]; exists {
		if slices.Contains(allowedSources, sourcePrim.TypeName) {
			return true, nil
		}
	}

	// so the source is larger than target
	return false, fmt.Errorf("expected %s but got %s", target, source)
}

func isPrimitiveExplicitCastable(target, source stype.Type) (bool, error) {

	//unwrap type aliases
	targetUnwrapped := semantic.UnwrapType(target)
	sourceUnwrapped := semantic.UnwrapType(source)

	targetPrim, targetOk := targetUnwrapped.(*stype.PrimitiveType)
	sourcePrim, sourceOk := sourceUnwrapped.(*stype.PrimitiveType)

	if !targetOk || !sourceOk {
		return false, fmt.Errorf("explicit cast not possible between non-primitive types: %s to %s", target, source)
	}

	//any numeric primitive type can be explicitly cast to any other numeric primitive type
	if semantic.IsNumericTypeName(targetPrim.TypeName) && semantic.IsNumericTypeName(sourcePrim.TypeName) {
		return true, nil
	}

	if targetPrim.TypeName == sourcePrim.TypeName {
		return true, nil
	}

	return false, fmt.Errorf("explicit cast not possible between types: %s to %s", source, target)
}

// --- ARRAYS ---

func isArrayImplicitCastable(target, source stype.Type) (bool, error) {
	targetArray, targetOk := target.(*stype.ArrayType)
	sourceArray, sourceOk := source.(*stype.ArrayType)

	if !targetOk || !sourceOk {
		return false, fmt.Errorf("implicit cast not possible between non-array types: %s to %s", target, source)
	}

	return isImplicitCastable(targetArray.ElementType, sourceArray.ElementType)
}

func isArrayExplicitCastable(target, source stype.Type) (bool, error) {
	targetArray, targetOk := target.(*stype.ArrayType)
	sourceArray, sourceOk := source.(*stype.ArrayType)

	if !targetOk || !sourceOk {
		return false, fmt.Errorf("explicit cast not possible between non-array types: %s to %s", target, source)
	}

	// Check if the element types are compatible
	return isExplicitCastable(targetArray.ElementType, sourceArray.ElementType)
}

// --- STRUCTS ---
func isStructCompatible(target, source stype.Type, isImplicit bool) (bool, error) {

	targetStruct, targetStructOk := semantic.UnwrapType(target).(*stype.StructType)
	//targetInterface, targetInterfaceOk := semantic.UnwrapType(target).(*stype.InterfaceType)
	sourceStruct, sourceOk := semantic.UnwrapType(source).(*stype.StructType)

	if !targetStructOk || !sourceOk {
		return false, fmt.Errorf("%s cast not possible between non-struct types: %s to %s", str.Ternary(isImplicit, "implicit", "explicit"), target, source)
	}

	problems := &[]string{}

	if isImplicit {
		checkImplicitFields(targetStruct, sourceStruct, problems)
	} else {
		checkExplicitFields(targetStruct, sourceStruct, problems)
	}

	fmt.Printf("Struct compatibility check: target=%s, source=%s, problems=%v\n", targetStruct, sourceStruct, problems)

	if len(*problems) > 0 {
		return false, fmt.Errorf("\n- %s", strings.Join(*problems, "\n- "))
	}

	// If we reach here, the structs are compatible
	return true, nil
}

func checkImplicitFields(targetStruct, sourceStruct *stype.StructType, problems *[]string) {
	// Check for extra fields in source struct that are not in target struct
	for fieldName := range sourceStruct.Fields {
		field, exists := targetStruct.Fields[fieldName]
		if !exists {
			*problems = append(*problems, fmt.Sprintf("extra field %s", fieldName))
			continue // Skip to next field if it doesn't exist in target
		}
		// Check if the field type is implicitly castable
		if ok, err := isImplicitCastable(field, sourceStruct.Fields[fieldName]); !ok {
			*problems = append(*problems, fmt.Sprintf("field %s type mismatch: %s", fieldName, err.Error()))
		}
	}
}

func checkExplicitFields(targetStruct, sourceStruct *stype.StructType, problems *[]string) {
	// target struct's fields must be a subset of source struct's fields
	for fieldName, targetFieldType := range targetStruct.Fields {
		sourceFieldType, exists := sourceStruct.Fields[fieldName]

		if !exists {
			*problems = append(*problems, fmt.Sprintf("missing field %s", fieldName))
			continue // Skip to next field if it doesn't exist in source
		}

		if ok, err := isExplicitCastable(targetFieldType, sourceFieldType); !ok {
			*problems = append(*problems, fmt.Sprintf("field %s type mismatch: %s", fieldName, err.Error()))
		}
	}
}

// --- INTERFACES ---
func isInterfaceCompatible(target, source stype.Type) (bool, error) {
	// target must be an interface type
	targetInterface, targetOk := target.(*stype.InterfaceType)
	if !targetOk {
		return false, fmt.Errorf("type %s is not an interface type", target)
	}

	var problems []string

	var sourceMethods map[string]*stype.FunctionType
	// source must be an interface type or struct type
	sourceInterface, sourceOk := source.(*stype.InterfaceType)
	if !sourceOk {
		sourceUser, sourceStructOk := source.(*stype.UserType)
		if !sourceStructOk {
			return false, fmt.Errorf("type %s is neither an interface nor a user defined type", source)
		}
		fmt.Printf("set methods for user type %s\n", sourceUser.Name)
		sourceMethods = sourceUser.Methods
		fmt.Printf("sourceMethods: %v\n", sourceMethods)
	} else {
		sourceMethods = sourceInterface.Methods
	}

	// Check if source implements all methods of target interface
	for methodName, targetMethod := range targetInterface.Methods {
		sourceMethod, exists := sourceMethods[methodName]
		if !exists {
			problems = append(problems, fmt.Sprintf("method %s not found in source type %s", methodName, source))
			continue // Skip to next method if it doesn't exist in source
		}
		// Check if the method signatures match
		if ok, err := isFunctionCompatible(targetMethod, sourceMethod); !ok {
			problems = append(problems, fmt.Sprintf("method %s signature mismatch: %s", methodName, err.Error()))
		}
	}

	if len(problems) > 0 {
		return false, fmt.Errorf("\n- %s", strings.Join(problems, "\n- "))
	}

	return true, nil
}

// --- FUNCTIONS ---
func isFunctionCompatible(target, source *stype.FunctionType) (bool, error) {

	if target == nil || source == nil {
		return false, fmt.Errorf("function type cannot be nil")
	}

	// Check if the number of parameters match
	if len(target.Parameters) != len(source.Parameters) {
		return false, fmt.Errorf("function parameter count mismatch: expected %d, got %d", len(target.Parameters), len(source.Parameters))
	}

	// Check if each parameter type is compatible
	for i, targetParam := range target.Parameters {
		sourceParam := source.Parameters[i]
		if ok, err := isExplicitCastable(targetParam.Type, sourceParam.Type); !ok {
			return false, fmt.Errorf("%s parameter type mismatch: %s", utils.NumericToOrdinal(i+1), err.Error())
		}
	}

	// Check return type compatibility
	if ok, err := isExplicitCastable(target.ReturnType, source.ReturnType); !ok {
		return false, fmt.Errorf("function return type mismatch: %s", err.Error())
	}

	return true, nil
}
