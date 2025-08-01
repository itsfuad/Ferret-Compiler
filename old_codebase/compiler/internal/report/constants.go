package report

// package error messages
const (
	EXPECTED_PACKAGE_KEYWORD = "Expected 'package' keyword"
	EXPECTED_IMPORT_KEYWORD  = "Expected 'import' keyword"
	EXPECTED_PACKAGE_NAME    = "Expected package name"
	EXPECTED_IMPORT_PATH     = "Expected import path"
	INVALID_IMPORT_PATH      = "Invalid import path"
)

// general error messages
const (
	MISSING_NAME      = "Expected variable name"
	MISSING_TYPE_NAME = "Expected type name"
	INVALID_TYPE_NAME = "Invalid type name"
)

// Error messages for number literals
const (
	INT_OUT_OF_RANGE   = "Integer value out of range"
	FLOAT_OUT_OF_RANGE = "Float value out of range"
	INVALID_NUMBER     = "Invalid number format"
)

// Error messages for array literals
const (
	ARRAY_EMPTY            = "Array literal must have at least one value"
	EXPECTED_ARRAY_ELEMENT = "Expected array element"
)

// Error messages for expected tokens
const (
	EXPECTED_NUMBER      = "Expected number"
	EXPECTED_STRING      = "Expected string"
	EXPECTED_BYTE        = "Expected byte"
	EXPECTED_SEMICOLON   = "Expected ';'"
	EXPECTED_COMMA       = "Expected ','"
	EXPECTED_COLON       = "Expected ':'"
	EXPECTED_EQUALS      = "Expected '='"
	EXPECTED_OPEN_BRACE  = "Expected '{'"
	EXPECTED_CLOSE_BRACE = "Expected '}'"
	EXPECTED_AT_TOKEN    = "Expected '@'"
)

// Error messages for variable declarations
const (
	MISMATCHED_VARIABLE_AND_TYPE_COUNT = "Mismatched variable and type count"
	SINGLE_VALUE_MULTIPLE_VARIABLES    = "Single value cannot be assigned to multiple variables"
)

const (
	INVALID_EXPRESSION = "Invalid expression"
	EMPTY_STATEMENT    = "Empty statement"
)

// Error messages for binary expressions
const (
	MISSING_RIGHT_OPERAND = "Missing right operand"
	MISSING_LEFT_OPERAND  = "Missing left operand"
)

// Error messages for unary expressions
const (
	INVALID_CONSECUTIVE_OPERATORS = "Invalid consecutive operators"
)

// Error messages for comparison expressions
const (
	INVALID_COMPARISON_OPERATOR = "Invalid comparison operator"
	MISSING_COMPARISON_OPERAND  = "Missing comparison operand"
	MISSING_COMPARISON_OPERATOR = "Missing comparison operator"
	MISSING_INCREMENT_OPERAND   = "Missing increment operand"
	MISSING_DECREMENT_OPERAND   = "Missing decrement operand"
)

// Error messages for increment/decrement operations
const (
	INVALID_INCREMENT_OPERAND     = "Invalid operand for increment operator"
	INVALID_DECREMENT_OPERAND     = "Invalid operand for decrement operator"
	INVALID_CONSECUTIVE_INCREMENT = "Invalid consecutive increment operators"
	INVALID_CONSECUTIVE_DECREMENT = "Invalid consecutive decrement operators"
	INVALID_MIX_OF_INCREMENT      = "Invalid mix of prefix and postfix increment operators"
	INVALID_MIX_OF_DECREMENT      = "Invalid mix of prefix and postfix decrement operators"
)

// Error messages for array operations
const (
	MISSING_INDEX_EXPRESSION   = "Missing array index expression"
	INVALID_INDEX_EXPRESSION   = "Invalid array index expression"
	INVALID_ARRAY_ELEMENT_TYPE = "Invalid array element type"
)

// Error messages for type declarations
const (
	EXPECTED_TYPE_NAME = "Expected type name"
	EXPECTED_TYPE      = "Expected type after type name"
	EXPECTED_VALUE     = "Expected value"
	UNEXPECTED_TOKEN   = "Unexpected token"
)

// Error messages for object/struct operations
const (
	EXPECTED_FIELD_NAME        = "Expected field name"
	EXPECTED_FIELD_TYPE        = "Expected field type"
	EXPECTED_FIELD_VALUE       = "Expected field value"
	DUPLICATE_FIELD_NAME       = "Duplicate field name"
	EMPTY_STRUCT_NOT_ALLOWED   = "Empty structs are not allowed - must have at least one field"
	EXPECTED_STRUCT_KEYWORD    = "Expected 'struct' keyword"
	EXPECTED_INTERFACE_KEYWORD = "Expected 'interface' keyword"
)

// Error messages for syntax errors
const (
	UNEXPECTED_CURLY_BRACE = "Unexpected '{' - object literals can only appear in expression contexts (after '=', ':', ',', or 'return')"
)

// Method declaration errors
const (
	EXPECTED_FUNCTION_KEYWORD = "Expected 'fn' keyword"
	EXPECTED_RECEIVER_NAME    = "Expected receiver name"
	EXPECTED_RECEIVER_TYPE    = "Expected receiver type"
	EXPECTED_METHOD_NAME      = "Expected method name"
	EXPECTED_RETURN_TYPE      = "Expected return type"
	DUPLICATE_METHOD_NAME     = "Duplicate method name"
	EXPECTED_PARAMETER_NAME   = "Expected parameter name"
	EXPECTED_PARAMETER_TYPE   = "Expected parameter type"
	PARAMETER_REDEFINITION    = "Parameter name already used"
)

// bracket errors
const (
	EXPECTED_OPEN_BRACKET           = "Expected '['"
	EXPECTED_CLOSE_BRACKET          = "Expected ']'"
	EXPECTED_OPEN_CURLY             = "Expected '{'"
	EXPECTED_CLOSE_CURLY            = "Expected '}'"
	EXPECTED_OPEN_PAREN             = "Expected '('"
	EXPECTED_CLOSE_PAREN            = "Expected ')'"
	EXPECTED_COMMA_OR_CLOSE_PAREN   = "Expected ',' or ')'"
	EXPECTED_COMMA_OR_CLOSE_BRACKET = "Expected ',' or ']'"
	EXPECTED_COMMA_OR_CLOSE_CURLY   = "Expected ',' or '}'"
)

// scope errors
const (
	SCOPE_MISMATCH                     = "Scope mismatch"
	INVALID_SCOPE                      = "Invalid scope"
	EXPECTED_SCOPE_RESOLUTION_OPERATOR = "Expected '::' operator"
)

// Statement errors
const (
	EXPECTED_RETURN_KEYWORD = "Expected 'return' keyword"
)

// New constant for the new error message
const (
	TRAILING_COMMA_NOT_ALLOWED = "Unnecessary trailing comma"
)

// Error messages for if statements
const (
	EXPECTED_IF   = "Expected 'if' keyword"
	EXPECTED_ELSE = "Expected 'else' keyword"
)
