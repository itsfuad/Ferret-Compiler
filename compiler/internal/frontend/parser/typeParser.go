package parser

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/fsutils/lists"
	"compiler/internal/report"
	"compiler/internal/source"
	"compiler/internal/types"
)

func parseIntegerType(p *Parser) (ast.DataType, bool) {
	token := p.advance()
	typename := types.TYPE_NAME(token.Value)
	bitSize := types.GetNumberBitSize(typename)
	if bitSize == 0 {
		p.ctx.Reports.Add(p.filePath, source.NewLocation(&token.Start, &token.End), report.INVALID_TYPE_NAME+" bitsize cannot be 0", report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}

	return &ast.IntType{
		TypeName:   typename,
		BitSize:    bitSize,
		IsUnsigned: types.IsUnsigned(typename),
		Location:   *source.NewLocation(&token.Start, &token.End),
	}, true
}

// user defined types are defined by the type keyword
// type NewType OldType;
func parseUserDefinedType(p *Parser) (ast.DataType, bool) {
	if p.match(lexer.IDENTIFIER_TOKEN) {
		token := p.advance()

		return &ast.UserDefinedType{
			TypeName: types.TYPE_NAME(token.Value),
			Location: *source.NewLocation(&token.Start, &token.End),
		}, true
	}

	return nil, false
}

func parseFloatType(p *Parser) (ast.DataType, bool) {
	token := p.advance()
	typename := types.TYPE_NAME(token.Value)
	bitSize := types.GetNumberBitSize(typename)
	if bitSize == 0 {
		p.ctx.Reports.Add(p.filePath, source.NewLocation(&token.Start, &token.End), report.INVALID_TYPE_NAME+" bitsize cannot be 0", report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}

	return &ast.FloatType{
		TypeName: typename,
		BitSize:  bitSize,
		Location: *source.NewLocation(&token.Start, &token.End),
	}, true
}

func parseStringType(p *Parser) (ast.DataType, bool) {
	token := p.advance()
	return &ast.StringType{
		TypeName: types.STRING,
		Location: *source.NewLocation(&token.Start, &token.End),
	}, true
}

func parseByteType(p *Parser) (ast.DataType, bool) {
	token := p.advance()
	return &ast.ByteType{
		TypeName: types.BYTE,
		Location: *source.NewLocation(&token.Start, &token.End),
	}, true
}

func parseBoolType(p *Parser) (ast.DataType, bool) {
	token := p.advance()
	return &ast.BoolType{
		TypeName: types.BOOL,
		Location: *source.NewLocation(&token.Start, &token.End),
	}, true
}

func parseArrayType(p *Parser) (ast.DataType, bool) {
	//consume the '[' token
	start := p.advance().Start
	// consume the ']' token
	p.consume(lexer.CLOSE_BRACKET, report.EXPECTED_CLOSE_BRACKET)

	//parse the type

	if elementType, ok := parseType(p); !ok {
		return nil, false
	} else {
		return &ast.ArrayType{
			ElementType: elementType,
			TypeName:    types.ARRAY,
			Location:    *source.NewLocation(&start, elementType.Loc().End),
		}, true
	}
}

// parseStructField parses a single struct field
func parseStructField(p *Parser) *ast.StructField {
	// Parse field name
	nameToken := p.consume(lexer.IDENTIFIER_TOKEN, report.EXPECTED_FIELD_NAME+" got "+p.peek().Value)
	fieldName := nameToken.Value

	// Expect colon
	p.consume(lexer.COLON_TOKEN, report.EXPECTED_COLON)

	// Parse field type
	if fieldType, ok := parseType(p); !ok {
		return nil
	} else {
		return &ast.StructField{
			FieldIdentifier: ast.IdentifierExpr{
				Name:     fieldName,
				Location: *source.NewLocation(&nameToken.Start, &nameToken.End),
			},
			FieldType: fieldType,
			Location:  *source.NewLocation(&nameToken.Start, fieldType.Loc().End),
		}
	}
}

// parseStructType parses a struct type definition like struct { name: str, age: i32 }
func parseStructType(p *Parser) (ast.DataType, bool) {
	// Consume 'struct' keyword
	start := p.consume(lexer.STRUCT_TOKEN, report.EXPECTED_STRUCT_KEYWORD).Start

	// Consume opening brace
	p.consume(lexer.OPEN_CURLY, report.EXPECTED_OPEN_BRACE)

	// Check for empty struct
	if p.peek().Kind == lexer.CLOSE_CURLY {
		token := p.peek()
		p.ctx.Reports.Add(p.filePath, source.NewLocation(&token.Start, &token.End),
			report.EMPTY_STRUCT_NOT_ALLOWED, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
		return nil, false
	}

	fields := make([]ast.StructField, 0)
	fieldNames := make(map[string]bool)

	for !p.match(lexer.CLOSE_CURLY) {

		// Parse field
		field := parseStructField(p)
		if field == nil {
			return nil, false
		}

		// Check for duplicate field names
		if fieldNames[field.FieldIdentifier.Name] {
			p.ctx.Reports.Add(p.filePath, source.NewLocation(field.Location.Start, field.Location.End),
				report.DUPLICATE_FIELD_NAME, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil, false
		}

		fieldNames[field.FieldIdentifier.Name] = true

		fields = append(fields, *field)

		if p.match(lexer.CLOSE_CURLY) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, report.EXPECTED_COMMA_OR_CLOSE_CURLY)
			if p.match(lexer.CLOSE_CURLY) {
				p.ctx.Reports.Add(p.filePath, source.NewLocation(&comma.Start, &comma.End), report.TRAILING_COMMA_NOT_ALLOWED, report.PARSING_PHASE).AddHint("Remove the trailing comma").SetLevel(report.WARNING)
				break
			}
		}
	}

	end := p.consume(lexer.CLOSE_CURLY, report.EXPECTED_CLOSE_BRACE).End

	return &ast.StructType{
		Fields:   fields,
		TypeName: types.STRUCT,
		Location: *source.NewLocation(&start, &end),
	}, true
}

func parseInterfaceType(p *Parser) (ast.DataType, bool) {

	start := p.consume(lexer.INTERFACE_TOKEN, report.EXPECTED_INTERFACE_KEYWORD)

	//consume the '{' token
	p.consume(lexer.OPEN_CURLY, report.EXPECTED_OPEN_BRACE)

	methods := make([]ast.InterfaceMethod, 0)

	for !p.match(lexer.CLOSE_CURLY) {

		start := p.consume(lexer.FUNCTION_TOKEN, report.EXPECTED_FUNCTION_KEYWORD).Start

		name := declareFunction(p)

		params, returnTypes := parseSignature(p, true)

		end := p.previous().End

		method := ast.InterfaceMethod{
			Name:       *name,
			Params:     params,
			ReturnType: returnTypes,
			Location:   source.Location{Start: &start, End: &end},
		}

		// check if the method name is already declared in the interface
		if lists.Has(methods, method, func(a ast.InterfaceMethod, b ast.InterfaceMethod) bool {
			return a.Name.Name == b.Name.Name
		}) {
			p.ctx.Reports.Add(p.filePath, source.NewLocation(method.Location.Start, method.Location.End), report.DUPLICATE_METHOD_NAME, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
			return nil, false
		}

		methods = append(methods, method)

		if p.match(lexer.CLOSE_CURLY) {
			break
		} else {
			//must be a comma
			comma := p.consume(lexer.COMMA_TOKEN, report.EXPECTED_COMMA_OR_CLOSE_CURLY)
			if p.match(lexer.CLOSE_CURLY) {
				p.ctx.Reports.Add(p.filePath, source.NewLocation(&comma.Start, &comma.End), report.TRAILING_COMMA_NOT_ALLOWED, report.PARSING_PHASE).AddHint("Remove the trailing comma").SetLevel(report.WARNING)
				break
			}
		}
	}

	end := p.consume(lexer.CLOSE_CURLY, "expected end of interface definition")

	return &ast.InterfaceType{
		Methods:  methods,
		TypeName: types.INTERFACE,
		Location: *source.NewLocation(&start.Start, &end.End),
	}, true
}

func parseFunctionTypeSignature(p *Parser) ([]ast.DataType, []ast.DataType) {
	// parse the parameters
	parameters := parseParameters(p)
	parameterTypes := make([]ast.DataType, len(parameters))
	// parse the return types
	returnTypes := parseReturnTypes(p)

	for i, parameter := range parameters {
		parameterTypes[i] = parameter.Type
	}

	return parameterTypes, returnTypes
}

func parseFunctionType(p *Parser) (ast.DataType, bool) {
	//consume the 'fn' token
	token := p.consume(lexer.FUNCTION_TOKEN, report.EXPECTED_FUNCTION_KEYWORD)

	// parse the parameters
	parameters, returnTypes := parseFunctionTypeSignature(p)

	return &ast.FunctionType{
		Parameters:  parameters,
		ReturnTypes: returnTypes,
		TypeName:    types.FUNCTION,
		Location:    *source.NewLocation(&token.Start, &token.End),
	}, true
}

// parseType parses a type expression
func parseType(p *Parser) (ast.DataType, bool) {
	token := p.peek()
	switch token.Value {
	case string(types.INT8), string(types.INT16), string(types.INT32), string(types.INT64), string(types.UINT8), string(types.UINT16), string(types.UINT32), string(types.UINT64):
		return parseIntegerType(p)
	case string(types.FLOAT32), string(types.FLOAT64):
		return parseFloatType(p)
	case string(types.STRING):
		return parseStringType(p)
	case string(types.BYTE):
		return parseByteType(p)
	case string(types.BOOL):
		return parseBoolType(p)
	case string(lexer.OPEN_BRACKET):
		return parseArrayType(p)
	case string(types.STRUCT):
		return parseStructType(p)
	case string(types.INTERFACE):
		return parseInterfaceType(p)
	case string(types.FUNCTION):
		return parseFunctionType(p)
	default:
		return parseUserDefinedType(p)
	}
}

// parseTypeDecl parses type declarations like "type Integer i32;"
func parseTypeDecl(p *Parser) ast.Statement {
	start := p.advance() // consume the 'type' token

	typeName := p.consume(lexer.IDENTIFIER_TOKEN, report.EXPECTED_TYPE_NAME)

	// Parse the underlying type
	underlyingType, ok := parseType(p)
	if !ok {
		token := p.peek()
		p.ctx.Reports.Add(p.filePath, source.NewLocation(&token.Start, &token.End),
			report.EXPECTED_TYPE, report.PARSING_PHASE).SetLevel(report.SYNTAX_ERROR)
		return nil
	}

	return &ast.TypeDeclStmt{
		Alias: &ast.IdentifierExpr{
			Name:     typeName.Value,
			Location: *source.NewLocation(&typeName.Start, &typeName.End),
		},
		BaseType: underlyingType,
		Location: *source.NewLocation(&start.Start, underlyingType.Loc().End),
	}
}
