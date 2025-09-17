# Ferret Parser: Resolving Ambiguous `{ ... }` Syntax

## Problem Statement

The Ferret programming language faced a significant parsing ambiguity with the `{ ... }` syntax, which serves dual purposes:

1. **Composite Literals**: Struct and map initialization (e.g., `Point{x: 10, y: 20}`)
2. **Code Blocks**: Statement containers in control structures (e.g., `if condition { let x = 1; }`)

### The Ambiguity Issue

The parser could not reliably distinguish between these two contexts, leading to false positives where code blocks containing certain tokens (particularly `:` and `=>`) were incorrectly interpreted as composite literals.

#### Example of the Problem

```ferret
fn printAny(value: any) {
    let a, b := 10, 29;
    if a > b {
        let big : i32 = a;  // ❌ Parser incorrectly saw this as composite literal
        printAny("a is greater");
    }
}
```

The parser would encounter `let big : i32 = a;` and misinterpret the `{ let big : i32 = a; ... }` block as a composite literal because:
- It started with an identifier (`printAny`)
- Followed by parentheses (function call)
- The block contained `:` tokens

This caused parsing errors like:
- "expected expression, found unexpected token `let`"
- "Expected key expression"

### Impact

- **False Positives**: Legitimate code blocks were rejected as invalid syntax
- **Poor Developer Experience**: Confusing error messages that didn't reflect actual syntax issues
- **Limited Language Expressiveness**: Developers had to work around the parser limitations

## Current Working Solution

### Look-Ahead Disambiguation Strategy

The solution implements a **look-ahead function** that examines the content inside `{ ... }` blocks to determine if they represent composite literals or code blocks.

#### Key Components

1. **Look-Ahead Function**: `isPotentialCompositeLiteral(p *Parser) bool`
2. **Keyword Rejection**: Blocks starting with statement keywords are immediately rejected
3. **Pattern Detection**: Scans for `identifier: value` or `identifier => value` patterns

### Implementation Details

#### Core Algorithm

```go
func isPotentialCompositeLiteral(p *Parser) bool {
    // Save current position for restoration
    savedPos := p.tokenNo
    defer func() { p.tokenNo = savedPos }()

    // Skip identifier and opening brace
    p.tokenNo += 2

    // Handle empty literals: { }
    if p.tokenNo < len(p.tokens) && p.tokens[p.tokenNo].Kind == lexer.CLOSE_CURLY {
        return true
    }

    // Reject blocks starting with statement keywords
    if p.tokenNo < len(p.tokens) && isStatementKeyword(p.tokens[p.tokenNo].Kind) {
        return false
    }

    // Scan for key-value separator patterns
    for p.tokenNo < len(p.tokens) && p.tokens[p.tokenNo].Kind != lexer.CLOSE_CURLY {
        token := p.tokens[p.tokenNo]

        if token.Kind == lexer.IDENTIFIER_TOKEN || token.Kind == lexer.STRING_TOKEN {
            if p.tokenNo+1 < len(p.tokens) {
                next := p.tokens[p.tokenNo+1]
                if next.Kind == lexer.COLON_TOKEN || next.Kind == lexer.FAT_ARROW_TOKEN {
                    return true
                }
            }
        }

        p.tokenNo++
    }

    return false
}
```

#### Statement Keyword Detection

```go
func isStatementKeyword(kind lexer.TOKEN) bool {
    switch kind {
    case lexer.LET_TOKEN, lexer.CONST_TOKEN, lexer.IF_TOKEN, lexer.ELSE_TOKEN,
         lexer.FOR_TOKEN, lexer.WHILE_TOKEN, lexer.RETURN_TOKEN, lexer.TYPE_TOKEN,
         lexer.FUNCTION_TOKEN, lexer.IMPORT_TOKEN:
        return true
    }
    return false
}
```

### Parser Integration

The look-ahead function is called in the primary expression parser:

```go
case lexer.IDENTIFIER_TOKEN:
    if p.next().Kind == lexer.OPEN_CURLY {
        if isPotentialCompositeLiteral(p) {
            return parseCompositeLiteral(p)
        }
    }
    return parseIdentifier(p)
```

### Solution Characteristics

#### Strengths
- **Conservative Approach**: Prefers code blocks over composite literals when ambiguous
- **Fast Look-Ahead**: Minimal token scanning with early termination
- **Extensible**: Easy to add new statement keywords or patterns
- **Non-Breaking**: Maintains backward compatibility with existing code

#### Limitations
- **Heuristic-Based**: Relies on pattern detection rather than formal grammar
- **Potential Edge Cases**: Complex expressions might still cause ambiguity
- **Performance**: Small look-ahead cost for each potential composite literal

### Testing and Validation

The solution was validated through:
1. **Unit Tests**: Parser test suite passes
2. **Integration Tests**: Full compilation pipeline works
3. **Real Code Examples**: Successfully parses the problematic `printAny` function

### Future Considerations

#### Potential Improvements
- **Formal Grammar**: Replace heuristic with context-aware grammar rules
- **Type-Guided Parsing**: Use type information to resolve ambiguities
- **Syntax Extensions**: Consider alternative syntax for composite literals (e.g., `Point{x: 10, y: 20}` vs `Point{x = 10, y = 20}`)

#### Monitoring
- Watch for new edge cases in user code
- Consider performance impact on large codebases
- Track parser accuracy metrics

## Conclusion

The implemented solution successfully resolves the `{ ... }` syntax ambiguity in Ferret by using a conservative look-ahead strategy that prioritizes code blocks over composite literals. This maintains language flexibility while ensuring reliable parsing, demonstrating that careful heuristic-based disambiguation can effectively handle syntactic ambiguities in programming languages.