package parser

import (
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
	"compiler/internal/types"
	"compiler/internal/utils/lists"
	"compiler/report"
	"fmt"
)

func parseIntegerType(p *Parser) (ast.DataType, bool) {
	token := p.advance()
	typename := types.TYPE_NAME(token.Value)
	bitSize := types.GetNumberBitSize(typename)
	if bitSize == 0 {
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), fmt.Sprintf("invalid type %s", typename), report.PARSING_PHASE).AddLabel("bitsize cannot be 0")
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
		iden := &ast.IdentifierExpr{
			Name:     token.Value,
			Location: *source.NewLocation(&token.Start, &token.End),
		}
		if p.peek().Kind == lexer.SCOPE_TOKEN {
			p.advance() // consume the scope token
			// this is a type scope resolution like module::TypeName
			typeNode, ok := parseType(p) // parse the type after the scope token
			if !ok {
				return nil, false
			}
			return &ast.TypeScopeResolution{
				Module:   iden,
				TypeNode: typeNode,
				Location: *source.NewLocation(iden.Start, typeNode.Loc().End),
			}, true
		}
		return &ast.UserDefinedType{
			TypeName: types.TYPE_NAME(iden.Name),
			Location: iden.Location,
		}, true
	}
	return nil, false
}

func parseFloatType(p *Parser) (ast.DataType, bool) {
	token := p.advance()
	typename := types.TYPE_NAME(token.Value)
	bitSize := types.GetNumberBitSize(typename)
	if bitSize == 0 {
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End), fmt.Sprintf("invalid type %s", typename), report.PARSING_PHASE).AddLabel("bitsize cannot be 0")
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
	p.consume(lexer.CLOSE_BRACKET, "expected ']' after '['")

	//parse the type

	elementType, ok := parseType(p)
	if !ok {
		return nil, false
	}

	return &ast.ArrayType{
		ElementType: elementType,
		TypeName:    types.ARRAY,
		Location:    *source.NewLocation(&start, elementType.Loc().End),
	}, true
}

// parseStructField parses a single struct field
func parseStructField(p *Parser) *ast.StructField {
	// Parse field name
	nameToken := p.consume(lexer.IDENTIFIER_TOKEN, "exptected field name but got "+p.peek().Value)
	fieldName := nameToken.Value

	// Expect colon
	p.consume(lexer.COLON_TOKEN, "expected ':' after field name")

	// Parse field type
	fieldType, ok := parseType(p)
	if !ok {
		return nil
	}
	return &ast.StructField{
		FieldIdentifier: &ast.IdentifierExpr{
			Name:     fieldName,
			Location: *source.NewLocation(&nameToken.Start, &nameToken.End),
		},
		FieldType: fieldType,
		Location:  *source.NewLocation(&nameToken.Start, fieldType.Loc().End),
	}
}

// parseStructType parses a struct type definition like struct { name: str, age: i32 }
func parseStructType(p *Parser) (ast.DataType, bool) {

	start := p.advance().Start // consume the 'struct' token

	// Consume opening brace
	p.consume(lexer.OPEN_CURLY, "expected '{' after 'struct'")

	// Check for empty struct
	if p.peek().Kind == lexer.CLOSE_CURLY {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End),
			"struct is empty", report.PARSING_PHASE).AddHint("struct must have at least one field")
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
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(field.Location.Start, field.Location.End),
				fmt.Sprintf("field %q already defined", field.FieldIdentifier.Name), report.PARSING_PHASE)
			return nil, false
		}

		fieldNames[field.FieldIdentifier.Name] = true

		fields = append(fields, *field)

		if p.match(lexer.CLOSE_CURLY) {
			break
		} else {
			comma := p.consume(lexer.COMMA_TOKEN, "expected ',' or '}' after struct field")
			if p.match(lexer.CLOSE_CURLY) {
				p.ctx.Reports.AddWarning(p.fullPath, source.NewLocation(&comma.Start, &comma.End), "unnecessary trailing comma after the last property", report.PARSING_PHASE).AddHint("remove the trailing comma")
				break
			}
		}
	}

	end := p.consume(lexer.CLOSE_CURLY, "expected '}' after struct fields").End

	return &ast.StructType{
		Fields:   fields,
		TypeName: types.STRUCT,
		Location: *source.NewLocation(&start, &end),
	}, true
}

func parseInterfaceType(p *Parser) (ast.DataType, bool) {

	start := p.advance()

	//consume the '{' token
	p.consume(lexer.OPEN_CURLY, "expected '{' after 'interface'")

	methods := make([]ast.InterfaceMethod, 0)

	for !p.match(lexer.CLOSE_CURLY) {

		start := p.consume(lexer.FUNCTION_TOKEN, "expected 'fn' keyword to define the function signature").Start

		name := declareFunction(p)

		params, returnType := parseSignature(p, true)

		end := p.previous().End

		method := ast.InterfaceMethod{
			Name: name,
			Method: &ast.FunctionType{
				Parameters: params,
				ReturnType: returnType,
				TypeName:   types.FUNCTION,
				Location:   *source.NewLocation(&start, &end),
			},
			Location: *source.NewLocation(&start, &end),
		}

		// check if the method name is already declared in the interface
		if lists.Has(methods, method, func(a ast.InterfaceMethod, b ast.InterfaceMethod) bool {
			return a.Name.Name == b.Name.Name
		}) {
			p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(method.Location.Start, method.Location.End), fmt.Sprintf("method %q already defined", name), report.PARSING_PHASE).AddHint("remove the duplicate method or rename it")
			return nil, false
		}

		methods = append(methods, method)

		if p.match(lexer.CLOSE_CURLY) {
			break
		} else {
			//must be a comma
			comma := p.consume(lexer.COMMA_TOKEN, "expected ',' or '}' after method signature")
			if p.match(lexer.CLOSE_CURLY) {
				p.ctx.Reports.AddWarning(p.fullPath, source.NewLocation(&comma.Start, &comma.End), "unnecessary trailing comma after last method", report.PARSING_PHASE).AddHint("remove the trailing comma")
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

func parseFunctionType(p *Parser) (ast.DataType, bool) {

	token := p.advance()

	// parse the parameters
	parameters, returnType := parseSignature(p, true)

	return &ast.FunctionType{
		Parameters: parameters,
		ReturnType: returnType,
		TypeName:   types.FUNCTION,
		Location:   *source.NewLocation(&token.Start, &token.End),
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

	typeName := p.consume(lexer.IDENTIFIER_TOKEN, "expected type name after 'type'")

	// Parse the underlying type
	underlyingType, ok := parseType(p)
	if !ok {
		token := p.peek()
		p.ctx.Reports.AddSyntaxError(p.fullPath, source.NewLocation(&token.Start, &token.End),
			"expected underlying type after type name", report.PARSING_PHASE)
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
