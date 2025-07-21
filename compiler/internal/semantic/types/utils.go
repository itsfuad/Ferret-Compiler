package types

// UnwrapType resolves user types to their underlying types A -> B -> C(not user type), return C
// Note: This function requires access to symbol tables to properly resolve type aliases.
// For now, it only checks the Definition field. For full resolution, use resolveTypeAlias instead.
func UnwrapType(t Type) Type {
	if userType, ok := t.(*UserType); ok {
		if userType.Definition != nil {
			return UnwrapType(userType.Definition)
		}
	}
	return t
}