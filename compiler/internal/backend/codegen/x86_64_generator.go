package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"compiler/internal/ctx"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/types"
)

// X86_64Generator generates x86-64 assembly code from Ferret AST
type X86_64Generator struct {
	options      *GeneratorOptions
	context      *CodeGenContext
	output       strings.Builder
	dataSection  strings.Builder
	bssSection   strings.Builder
	textSection  strings.Builder
	labelCounter int
	stackOffset  int
	registers    map[string]bool // Track register usage
}

// NewX8664Generator creates a new x86-64 assembly code generator
func NewX8664Generator(options *GeneratorOptions) *X86_64Generator {
	if options == nil {
		options = &GeneratorOptions{}
	}
	return &X86_64Generator{
		options:   options,
		registers: make(map[string]bool),
	}
}

// GetTarget returns the target architecture
func (g *X86_64Generator) GetTarget() Target {
	return TargetX86_64
}

// SetOptions sets generator-specific options
func (g *X86_64Generator) SetOptions(options map[string]interface{}) {
	// Handle x86-64 specific options here
}

// Generate generates x86-64 assembly code for the entire program
func (g *X86_64Generator) Generate(program *ast.Program, compilerCtx *ctx.CompilerContext) (string, error) {
	g.context = NewCodeGenContext(compilerCtx)

	// Find the correct module name from the available modules
	var moduleName string
	for name := range compilerCtx.Modules {
		if strings.HasSuffix(name, program.Modulename) || name == program.Modulename {
			moduleName = name
			break
		}
	}
	if moduleName == "" {
		// Fallback to program modulename
		moduleName = program.Modulename
	}

	g.context.CurrentModule = moduleName
	g.labelCounter = 0
	g.stackOffset = 0

	// Reset all sections
	g.output.Reset()
	g.dataSection.Reset()
	g.bssSection.Reset()
	g.textSection.Reset()

	// Generate assembly sections
	g.generateDataSection(program, compilerCtx)
	g.generateBSSSection(program, compilerCtx)
	g.generateTextSection(program, compilerCtx)

	// Debug output for last expression if debug mode is enabled
	if g.options.DebugInfo {
		g.outputLastExpressionDebug(program, compilerCtx)
	}

	// Combine all sections
	g.combineOutput()

	return g.output.String(), nil
}

// generateDataSection generates the .data section for initialized variables
func (g *X86_64Generator) generateDataSection(program *ast.Program, compilerCtx *ctx.CompilerContext) {
	g.dataSection.WriteString("section .data\n")

	// Process AST nodes to find initialized variable declarations
	for _, node := range program.Nodes {
		if varDecl, ok := node.(*ast.VarDeclStmt); ok {
			g.generateDataVariables(varDecl, compilerCtx)
		}
	}

	// Add string literals
	g.generateStringLiterals(program, compilerCtx)

	g.dataSection.WriteString("\n")
}

// generateBSSSection generates the .bss section for uninitialized variables
func (g *X86_64Generator) generateBSSSection(program *ast.Program, compilerCtx *ctx.CompilerContext) {
	g.bssSection.WriteString("section .bss\n")

	// Process AST nodes to find uninitialized variable declarations
	for _, node := range program.Nodes {
		if varDecl, ok := node.(*ast.VarDeclStmt); ok {
			g.generateBSSVariables(varDecl, compilerCtx)
		}
	}

	g.bssSection.WriteString("\n")
}

// generateTextSection generates the .text section with executable code
func (g *X86_64Generator) generateTextSection(program *ast.Program, compilerCtx *ctx.CompilerContext) {
	g.textSection.WriteString("section .text\n")
	g.textSection.WriteString("global _start\n\n")

	// Generate functions
	g.generateFunctions(program, compilerCtx)

	// Generate main entry point
	g.generateMainEntry(program, compilerCtx)

	g.textSection.WriteString("\n")
}

// generateDataVariables generates initialized variables in .data section
func (g *X86_64Generator) generateDataVariables(varDecl *ast.VarDeclStmt, compilerCtx *ctx.CompilerContext) {
	module := compilerCtx.Modules[g.context.CurrentModule]
	if module == nil {
		return
	}

	for i, variable := range varDecl.Variables {
		if i < len(varDecl.Initializers) {
			varName := variable.Identifier.Name
			symbol, found := module.SymbolTable.Lookup(varName)
			if !found {
				continue
			}

			sanitizedName := g.sanitizeLabel(varName)
			initializer := varDecl.Initializers[i]

			// Only generate data declarations for constants
			// Variables with initializers will be handled at runtime
			if varDecl.IsConst {
				g.generateDataDeclaration(sanitizedName, symbol.Type, initializer)
			}
		}
	}
}

// generateBSSVariables generates uninitialized variables in .bss section
func (g *X86_64Generator) generateBSSVariables(varDecl *ast.VarDeclStmt, compilerCtx *ctx.CompilerContext) {
	module := compilerCtx.Modules[g.context.CurrentModule]
	if module == nil {
		return
	}

	for i, variable := range varDecl.Variables {
		varName := variable.Identifier.Name
		symbol, found := module.SymbolTable.Lookup(varName)
		if !found {
			continue
		}

		sanitizedName := g.sanitizeLabel(varName)
		size := g.getTypeSize(symbol.Type)

		// Generate BSS entries for:
		// 1. Variables without initializers
		// 2. Variables with initializers (will be initialized at runtime)
		if i >= len(varDecl.Initializers) || !varDecl.IsConst {
			comment := ""
			if i < len(varDecl.Initializers) {
				comment = "    ; variable " + varName + " (runtime initialized)"
			} else {
				comment = "    ; variable " + varName + " (uninitialized)"
			}
			g.bssSection.WriteString(fmt.Sprintf("%s: resb %d%s\n", sanitizedName, size, comment))
		}
	}
}

// generateDataDeclaration generates a data declaration for an initialized variable
func (g *X86_64Generator) generateDataDeclaration(name string, varType ctx.Type, initializer ast.Expression) {
	directive := g.getDataDirective(varType)
	value := g.generateConstantValue(initializer)
	g.dataSection.WriteString(fmt.Sprintf("%s: %s %s\n", name, directive, value))
}

// generateStringLiterals generates string literals in .data section
func (g *X86_64Generator) generateStringLiterals(program *ast.Program, compilerCtx *ctx.CompilerContext) {
	// TODO: Collect all string literals and generate them
	// For now, we'll generate them on-demand in expressions
}

// generateFunctions generates assembly code for functions
func (g *X86_64Generator) generateFunctions(program *ast.Program, compilerCtx *ctx.CompilerContext) {
	// TODO: Generate function definitions when function AST nodes are available
}

// generateMainEntry generates the main entry point
func (g *X86_64Generator) generateMainEntry(program *ast.Program, compilerCtx *ctx.CompilerContext) {
	g.textSection.WriteString("_start:\n")

	// Set up stack frame
	g.textSection.WriteString("    push rbp\n")
	g.textSection.WriteString("    mov rbp, rsp\n")

	// Generate code for main function body
	for _, node := range program.Nodes {
		switch n := node.(type) {
		case *ast.VarDeclStmt:
			// Generate runtime initialization for variables (not constants)
			if !n.IsConst {
				g.generateVariableInitialization(n, compilerCtx)
			}
		case *ast.ExpressionStmt:
			for _, expr := range *n.Expressions {
				g.generateExpressionCode(expr, compilerCtx)
			}
		case *ast.AssignmentStmt:
			g.generateAssignmentCode(n, compilerCtx)
		case *ast.ImportStmt:
			// Generate import code
			g.generateImportCode(n, compilerCtx)
		case *ast.TypeDeclStmt:
			// Skip type definitions
			continue
		}
	}

	// Exit system call
	g.textSection.WriteString("    ; Exit program\n")
	g.textSection.WriteString("    mov rax, 60      ; sys_exit\n")
	g.textSection.WriteString("    mov rdi, 0       ; exit status\n")
	g.textSection.WriteString("    syscall\n")
}

// generateVariableInitialization generates runtime initialization code for variables
func (g *X86_64Generator) generateVariableInitialization(varDecl *ast.VarDeclStmt, compilerCtx *ctx.CompilerContext) {
	module := compilerCtx.Modules[g.context.CurrentModule]
	if module == nil {
		return
	}

	for i, variable := range varDecl.Variables {
		if i < len(varDecl.Initializers) {
			varName := variable.Identifier.Name
			_, found := module.SymbolTable.Lookup(varName)
			if !found {
				continue
			}

			sanitizedName := g.sanitizeLabel(varName)
			initializer := varDecl.Initializers[i]

			g.textSection.WriteString(fmt.Sprintf("    ; Variable declaration: %s\n", varName))

			// Generate code to evaluate the initializer expression
			g.generateExpressionCode(initializer, compilerCtx)

			// Store the result in the variable
			g.textSection.WriteString(fmt.Sprintf("    mov [%s], rax    ; store computed value in %s\n", sanitizedName, varName))
		}
	}
}

// generateExpressionCode generates assembly code for an expression
func (g *X86_64Generator) generateExpressionCode(expr ast.Expression, compilerCtx *ctx.CompilerContext) {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		g.textSection.WriteString(fmt.Sprintf("    mov rax, %d\n", e.Value))

	case *ast.FloatLiteral:
		// For simplicity, convert to integer for now
		intVal := int64(e.Value)
		g.textSection.WriteString(fmt.Sprintf("    mov rax, %d    ; float %f as int\n", intVal, e.Value))

	case *ast.StringLiteral:
		label := g.getNextLabel("str")
		g.dataSection.WriteString(fmt.Sprintf("%s: db '%s', 0\n", label, g.escapeString(e.Value)))
		g.textSection.WriteString(fmt.Sprintf("    mov rax, %s\n", label))

	case *ast.BoolLiteral:
		if e.Value {
			g.textSection.WriteString("    mov rax, 1\n")
		} else {
			g.textSection.WriteString("    mov rax, 0\n")
		}

	case *ast.IdentifierExpr:
		varName := g.sanitizeLabel(e.Name)
		g.textSection.WriteString(fmt.Sprintf("    mov rax, [%s]\n", varName))

	case *ast.BinaryExpr:
		g.generateBinaryExpressionCode(e, compilerCtx)

	case *ast.UnaryExpr:
		g.generateUnaryExpressionCode(e, compilerCtx)

	case *ast.VarScopeResolution:
		// Handle module::variable access
		varName := g.sanitizeLabel(e.Identifier.Name)
		g.textSection.WriteString(fmt.Sprintf("    mov rax, [%s]\n", varName))

	default:
		g.textSection.WriteString("    ; unsupported expression\n")
	}
}

// generateBinaryExpressionCode generates assembly for binary expressions
func (g *X86_64Generator) generateBinaryExpressionCode(expr *ast.BinaryExpr, compilerCtx *ctx.CompilerContext) {
	// Generate code for left operand (result in rax)
	g.generateExpressionCode(*expr.Left, compilerCtx)
	g.textSection.WriteString("    push rax    ; save left operand\n")

	// Generate code for right operand (result in rax)
	g.generateExpressionCode(*expr.Right, compilerCtx)
	g.textSection.WriteString("    mov rbx, rax    ; move right operand to rbx\n")
	g.textSection.WriteString("    pop rax     ; restore left operand\n")

	// Perform operation
	switch expr.Operator.Kind {
	case lexer.PLUS_TOKEN:
		g.textSection.WriteString("    add rax, rbx\n")
	case lexer.MINUS_TOKEN:
		g.textSection.WriteString("    sub rax, rbx\n")
	case lexer.MUL_TOKEN:
		g.textSection.WriteString("    imul rax, rbx\n")
	case lexer.DIV_TOKEN:
		g.textSection.WriteString("    xor rdx, rdx    ; clear rdx for division\n")
		g.textSection.WriteString("    idiv rbx\n")
	case lexer.MOD_TOKEN:
		g.textSection.WriteString("    xor rdx, rdx    ; clear rdx for division\n")
		g.textSection.WriteString("    idiv rbx\n")
		g.textSection.WriteString("    mov rax, rdx    ; remainder is in rdx\n")
	case lexer.DOUBLE_EQUAL_TOKEN:
		g.textSection.WriteString("    cmp rax, rbx\n")
		g.textSection.WriteString("    sete al\n")
		g.textSection.WriteString("    movzx rax, al\n")
	case lexer.NOT_EQUAL_TOKEN:
		g.textSection.WriteString("    cmp rax, rbx\n")
		g.textSection.WriteString("    setne al\n")
		g.textSection.WriteString("    movzx rax, al\n")
	case lexer.LESS_TOKEN:
		g.textSection.WriteString("    cmp rax, rbx\n")
		g.textSection.WriteString("    setl al\n")
		g.textSection.WriteString("    movzx rax, al\n")
	case lexer.LESS_EQUAL_TOKEN:
		g.textSection.WriteString("    cmp rax, rbx\n")
		g.textSection.WriteString("    setle al\n")
		g.textSection.WriteString("    movzx rax, al\n")
	case lexer.GREATER_TOKEN:
		g.textSection.WriteString("    cmp rax, rbx\n")
		g.textSection.WriteString("    setg al\n")
		g.textSection.WriteString("    movzx rax, al\n")
	case lexer.GREATER_EQUAL_TOKEN:
		g.textSection.WriteString("    cmp rax, rbx\n")
		g.textSection.WriteString("    setge al\n")
		g.textSection.WriteString("    movzx rax, al\n")
	case lexer.BIT_AND_TOKEN:
		g.textSection.WriteString("    and rax, rbx\n")
	case lexer.BIT_OR_TOKEN:
		g.textSection.WriteString("    or rax, rbx\n")
	case lexer.BIT_XOR_TOKEN:
		g.textSection.WriteString("    xor rax, rbx\n")
	default:
		g.textSection.WriteString("    ; unsupported binary operator\n")
	}
}

// generateUnaryExpressionCode generates assembly for unary expressions
func (g *X86_64Generator) generateUnaryExpressionCode(expr *ast.UnaryExpr, compilerCtx *ctx.CompilerContext) {
	// Generate code for operand
	g.generateExpressionCode(*expr.Operand, compilerCtx)

	// Apply unary operator
	switch expr.Operator.Kind {
	case lexer.MINUS_TOKEN:
		g.textSection.WriteString("    neg rax\n")
	case lexer.NOT_TOKEN:
		g.textSection.WriteString("    test rax, rax\n")
		g.textSection.WriteString("    setz al\n")
		g.textSection.WriteString("    movzx rax, al\n")
	case lexer.BIT_XOR_TOKEN: // Bitwise NOT
		g.textSection.WriteString("    not rax\n")
	default:
		g.textSection.WriteString("    ; unsupported unary operator\n")
	}
}

// generateAssignmentCode generates assembly for assignment statements
func (g *X86_64Generator) generateAssignmentCode(assignment *ast.AssignmentStmt, compilerCtx *ctx.CompilerContext) {
	if len(*assignment.Left) != 1 || len(*assignment.Right) != 1 {
		g.textSection.WriteString("    ; unsupported assignment pattern\n")
		return
	}

	// Generate code for right side (value to assign)
	g.generateExpressionCode((*assignment.Right)[0], compilerCtx)

	// Store to left side
	switch lval := (*assignment.Left)[0].(type) {
	case *ast.IdentifierExpr:
		varName := g.sanitizeLabel(lval.Name)
		g.textSection.WriteString(fmt.Sprintf("    mov [%s], rax\n", varName))
	case *ast.FieldAccessExpr:
		// TODO: Handle struct field assignment
		g.textSection.WriteString("    ; struct field assignment not implemented\n")
	default:
		g.textSection.WriteString("    ; unsupported lvalue\n")
	}
}

// generateConstantValue generates a constant value for data declarations
func (g *X86_64Generator) generateConstantValue(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return strconv.FormatInt(e.Value, 10)
	case *ast.FloatLiteral:
		// For NASM, we need to use proper floating-point format
		return strconv.FormatFloat(e.Value, 'f', 6, 64)
	case *ast.StringLiteral:
		return fmt.Sprintf("'%s', 0", g.escapeString(e.Value))
	case *ast.BoolLiteral:
		if e.Value {
			return "1"
		}
		return "0"
	case *ast.ByteLiteral:
		return fmt.Sprintf("'%s'", g.escapeString(e.Value))
	default:
		return "0"
	}
}

// getDataDirective returns the appropriate assembly directive for a type
func (g *X86_64Generator) getDataDirective(semType ctx.Type) string {
	switch semType.TypeName() {
	case types.INT8, types.UINT8, types.BYTE:
		return "db"
	case types.INT16, types.UINT16:
		return "dw"
	case types.INT32, types.UINT32:
		return "dd"
	case types.INT64, types.UINT64:
		return "dq"
	case types.FLOAT32:
		return "dd"
	case types.FLOAT64:
		return "dq"
	case types.STRING:
		return "dq" // pointer to string
	case types.BOOL:
		return "db"
	default:
		return "dq" // default to 64-bit
	}
}

// getTypeSize returns the size in bytes for a type
func (g *X86_64Generator) getTypeSize(semType ctx.Type) int {
	switch semType.TypeName() {
	case types.INT8, types.UINT8, types.BYTE, types.BOOL:
		return 1
	case types.INT16, types.UINT16:
		return 2
	case types.INT32, types.UINT32, types.FLOAT32:
		return 4
	case types.INT64, types.UINT64, types.FLOAT64, types.STRING:
		return 8
	default:
		return 8 // default to 64-bit
	}
}

// combineOutput combines all sections into final output
func (g *X86_64Generator) combineOutput() {
	g.output.WriteString("; Generated x86-64 Assembly for Ferret\n")
	g.output.WriteString("; Target: Linux x86-64\n\n")

	g.output.WriteString(g.dataSection.String())
	g.output.WriteString(g.bssSection.String())
	g.output.WriteString(g.textSection.String())
}

// getNextLabel generates a unique label
func (g *X86_64Generator) getNextLabel(prefix string) string {
	g.labelCounter++
	return fmt.Sprintf("%s_%d", prefix, g.labelCounter)
}

// sanitizeLabel ensures the label is valid in assembly
func (g *X86_64Generator) sanitizeLabel(name string) string {
	// Replace invalid characters
	sanitized := strings.ReplaceAll(name, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, "::", "_")
	sanitized = strings.ReplaceAll(sanitized, "-", "_")

	// Ensure it starts with a letter or underscore
	if len(sanitized) > 0 && (sanitized[0] >= '0' && sanitized[0] <= '9') {
		sanitized = "_" + sanitized
	}

	return sanitized
}

// escapeString escapes special characters in strings for assembly
func (g *X86_64Generator) escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "', 10, '")
	s = strings.ReplaceAll(s, "\r", "', 13, '")
	s = strings.ReplaceAll(s, "\t", "', 9, '")
	return s
}

// outputLastExpressionDebug outputs debug information about the last expression
func (g *X86_64Generator) outputLastExpressionDebug(program *ast.Program, compilerCtx *ctx.CompilerContext) {
	var lastExpr ast.Expression
	var lastExprType string

	// Find the last expression in the program
	for i := len(program.Nodes) - 1; i >= 0; i-- {
		node := program.Nodes[i]
		switch n := node.(type) {
		case *ast.VarDeclStmt:
			// Check for initializers in variable declarations
			if len(n.Initializers) > 0 {
				lastExpr = n.Initializers[len(n.Initializers)-1]
				lastExprType = "variable declaration initializer"
				break
			}
		case *ast.ExpressionStmt:
			if len(*n.Expressions) > 0 {
				lastExpr = (*n.Expressions)[len(*n.Expressions)-1]
				lastExprType = "expression statement"
				break
			}
		case *ast.AssignmentStmt:
			if len(*n.Right) > 0 {
				lastExpr = (*n.Right)[len(*n.Right)-1]
				lastExprType = "assignment"
				break
			}
		}
		if lastExpr != nil {
			break
		}
	}

	if lastExpr != nil {
		fmt.Printf("🐛 Debug: Last expression (%s): %s\n", lastExprType, g.formatExpressionForDebug(lastExpr))
	}
}

// formatExpressionForDebug formats an expression for debug output
func (g *X86_64Generator) formatExpressionForDebug(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return fmt.Sprintf("%d (int literal)", e.Value)
	case *ast.FloatLiteral:
		return fmt.Sprintf("%f (float literal)", e.Value)
	case *ast.StringLiteral:
		return fmt.Sprintf("\"%s\" (string literal)", e.Value)
	case *ast.BoolLiteral:
		return fmt.Sprintf("%t (bool literal)", e.Value)
	case *ast.IdentifierExpr:
		return fmt.Sprintf("%s (identifier)", e.Name)
	case *ast.BinaryExpr:
		left := g.formatExpressionForDebug(*e.Left)
		right := g.formatExpressionForDebug(*e.Right)
		return fmt.Sprintf("(%s %s %s)", left, e.Operator.Value, right)
	case *ast.UnaryExpr:
		operand := g.formatExpressionForDebug(*e.Operand)
		return fmt.Sprintf("%s%s", e.Operator.Value, operand)
	default:
		return "unknown expression"
	}
}

// generateImportCode handles import statements
func (g *X86_64Generator) generateImportCode(importStmt *ast.ImportStmt, compilerCtx *ctx.CompilerContext) {
	// For now, we'll add a comment in the assembly
	importPath := importStmt.ImportPath.Value
	g.textSection.WriteString(fmt.Sprintf("    ; Import: %s\n", importPath))

	// TODO: Implement actual import linking when the module system is ready
	// This would involve:
	// 1. Finding the imported module's object file or assembly
	// 2. Adding external symbol declarations
	// 3. Linking the modules together during final compilation
}
