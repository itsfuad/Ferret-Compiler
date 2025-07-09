package lexer

import (
	"fmt"

	"compiler/colors"
	"compiler/internal/source"
	"compiler/internal/types"
)

type TOKEN string

const (
	//keywords
	LET_TOKEN        TOKEN = "let"
	CONST_TOKEN      TOKEN = "const"
	TYPE_TOKEN       TOKEN = "type"
	IF_TOKEN         TOKEN = "if"
	ELSE_TOKEN       TOKEN = "else"
	FOR_TOKEN        TOKEN = "for"
	FOREACH_TOKEN    TOKEN = "foreach"
	WHILE_TOKEN      TOKEN = "while"
	DO_TOKEN         TOKEN = "do"
	IDENTIFIER_TOKEN TOKEN = "identifier"
	PRIVATE_TOKEN    TOKEN = "priv"
	RETURN_TOKEN     TOKEN = "return"
	IMPORT_TOKEN     TOKEN = "import"
	AS_TOKEN         TOKEN = "as"
	MODULE_TOKEN     TOKEN = "mod"
	//data types
	NUMBER_TOKEN    TOKEN = "numeric literal"
	STRING_TOKEN    TOKEN = "string literal"
	BYTE_TOKEN      TOKEN = "byte literal"
	STRUCT_TOKEN    TOKEN = TOKEN(types.STRUCT)
	FUNCTION_TOKEN  TOKEN = TOKEN(types.FUNCTION)
	INTERFACE_TOKEN TOKEN = TOKEN(types.INTERFACE)

	//array range operator
	RANGE_TOKEN TOKEN = ".."
	//increment and decrement
	PLUS_PLUS_TOKEN   TOKEN = "++"
	MINUS_MINUS_TOKEN TOKEN = "--"
	//Binary operators
	AND_TOKEN TOKEN = "&&"
	OR_TOKEN  TOKEN = "||"
	//bitwise operators
	BIT_AND_TOKEN TOKEN = "&"
	BIT_OR_TOKEN  TOKEN = "|"
	BIT_XOR_TOKEN TOKEN = "^"
	//unary operators
	NOT_TOKEN TOKEN = "!"
	//arithmetic operators
	EXP_TOKEN   TOKEN = "**"
	MINUS_TOKEN TOKEN = "-"
	PLUS_TOKEN  TOKEN = "+"
	MUL_TOKEN   TOKEN = "*"
	DIV_TOKEN   TOKEN = "/"
	MOD_TOKEN   TOKEN = "%"
	//logical operators
	LESS_EQUAL_TOKEN    TOKEN = "<="
	GREATER_EQUAL_TOKEN TOKEN = ">="
	NOT_EQUAL_TOKEN     TOKEN = "!="
	DOUBLE_EQUAL_TOKEN  TOKEN = "=="
	LESS_TOKEN          TOKEN = "<"
	GREATER_TOKEN       TOKEN = ">"
	//assignment
	SCOPE_TOKEN        TOKEN = "::"
	COLON_TOKEN        TOKEN = ":"
	EQUALS_TOKEN       TOKEN = "="
	PLUS_EQUALS_TOKEN  TOKEN = "+="
	MINUS_EQUALS_TOKEN TOKEN = "-="
	MUL_EQUALS_TOKEN   TOKEN = "*="
	DIV_EQUALS_TOKEN   TOKEN = "/="
	MOD_EQUALS_TOKEN   TOKEN = "%="
	EXP_EQUALS_TOKEN   TOKEN = "^="
	//delimiters
	OPEN_PAREN      TOKEN = "("
	CLOSE_PAREN     TOKEN = ")"
	OPEN_BRACKET    TOKEN = "["
	CLOSE_BRACKET   TOKEN = "]"
	OPEN_CURLY      TOKEN = "{"
	CLOSE_CURLY     TOKEN = "}"
	COMMA_TOKEN     TOKEN = ","
	DOT_TOKEN       TOKEN = "."
	SEMICOLON_TOKEN TOKEN = ";"
	ARROW_TOKEN     TOKEN = "->"
	FAT_ARROW_TOKEN TOKEN = "=>"

	AT_TOKEN TOKEN = "@"

	EOF_TOKEN TOKEN = "end_of_file"
)

var keyWordsMap map[TOKEN]bool = map[TOKEN]bool{
	LET_TOKEN:       true,
	CONST_TOKEN:     true,
	IF_TOKEN:        true,
	ELSE_TOKEN:      true,
	FOR_TOKEN:       true,
	FOREACH_TOKEN:   true,
	WHILE_TOKEN:     true,
	DO_TOKEN:        true,
	TYPE_TOKEN:      true,
	STRUCT_TOKEN:    true,
	PRIVATE_TOKEN:   true,
	INTERFACE_TOKEN: true,
	FUNCTION_TOKEN:  true,
	RETURN_TOKEN:    true,
	IMPORT_TOKEN:    true,
	MODULE_TOKEN:    true,
	AS_TOKEN:        true,
}

func IsKeyword(token string) bool {
	if _, ok := keyWordsMap[TOKEN(token)]; ok {
		return true
	}
	return false
}

type Token struct {
	Kind  TOKEN
	Value string
	Start source.Position
	End   source.Position
}

func (t *Token) Debug(filename string) {
	colors.GREY.Printf("%s:%d:%d ", filename, t.Start.Line, t.Start.Column)
	if t.Value == string(t.Kind) {
		fmt.Printf("'%s'\n", t.Value)
	} else {
		fmt.Printf("'%s' ('%v')\n", t.Value, t.Kind)
	}
}

func NewToken(kind TOKEN, value string, start source.Position, end source.Position) Token {
	return Token{
		Kind:  kind,
		Value: value,
		Start: start,
		End:   end,
	}
}
