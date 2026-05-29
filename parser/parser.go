package parser

import (
	"fmt"
	"nova/ast"
	"nova/lexer"
	"strconv"
)

type Parser struct {
	tokens []lexer.Token
	pos    int
}

func New(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens}
}

func (p *Parser) peek() lexer.Token {
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == lexer.NEWLINE {
		// Only skip newlines when not at the block level — handled by callers
		return p.tokens[p.pos]
	}
	return p.tokens[p.pos]
}

func (p *Parser) peekSkipNL() lexer.Token {
	i := p.pos
	for i < len(p.tokens) && p.tokens[i].Type == lexer.NEWLINE {
		i++
	}
	if i >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[i]
}

func (p *Parser) cur() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() lexer.Token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *Parser) skipNewlines() {
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == lexer.NEWLINE {
		p.pos++
	}
}

func (p *Parser) expect(t lexer.TokenType) (lexer.Token, error) {
	tok := p.cur()
	if tok.Type != t {
		return tok, fmt.Errorf("line %d: expected %s, got %s (%q)", tok.Line, t, tok.Type, tok.Literal)
	}
	p.advance()
	return tok, nil
}

func (p *Parser) Parse() (*ast.Program, error) {
	prog := &ast.Program{}
	p.skipNewlines()
	for p.cur().Type != lexer.EOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			prog.Statements = append(prog.Statements, stmt)
		}
		p.skipNewlines()
	}
	return prog, nil
}

func (p *Parser) parseStatement() (ast.Statement, error) {
	tok := p.cur()
	switch tok.Type {
	case lexer.FUNC:
		return p.parseFuncDecl()
	case lexer.RETURN:
		return p.parseReturn()
	case lexer.IF:
		return p.parseIf()
	case lexer.FOR:
		return p.parseFor()
	case lexer.WHILE:
		return p.parseWhile()
	case lexer.BREAK:
		p.advance()
		return &ast.BreakStatement{Line: tok.Line}, nil
	case lexer.CONTINUE:
		p.advance()
		return &ast.ContinueStatement{Line: tok.Line}, nil
	case lexer.TRY:
		return p.parseTryCatch()
	case lexer.TASK:
		return p.parseTask()
	case lexer.WAIT:
		p.advance()
		return &ast.WaitStatement{Line: tok.Line}, nil
	case lexer.IMPORT:
		return p.parseImport()
	case lexer.FROM:
		return p.parseFromImport()
	case lexer.NEWLINE, lexer.DEDENT:
		p.advance()
		return nil, nil
	default:
		return p.parseExprOrAssign()
	}
}

func (p *Parser) parseFuncDecl() (*ast.FunctionDeclaration, error) {
	line := p.cur().Line
	p.advance() // consume 'func'
	nameTok, err := p.expect(lexer.IDENTIFIER)
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(lexer.LPAREN); err != nil {
		return nil, err
	}
	params, err := p.parseParams()
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(lexer.RPAREN); err != nil {
		return nil, err
	}
	if _, err = p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.FunctionDeclaration{Line: line, Name: nameTok.Literal, Params: params, Body: body}, nil
}

func (p *Parser) parseParams() ([]string, error) {
	var params []string
	for p.cur().Type != lexer.RPAREN && p.cur().Type != lexer.EOF {
		tok, err := p.expect(lexer.IDENTIFIER)
		if err != nil {
			return nil, err
		}
		// optional type hint: name: type
		if p.cur().Type == lexer.COLON {
			p.advance()
			p.advance() // consume type name
		}
		params = append(params, tok.Literal)
		if p.cur().Type == lexer.COMMA {
			p.advance()
		}
	}
	return params, nil
}

func (p *Parser) parseBlock() ([]ast.Statement, error) {
	p.skipNewlines()
	var stmts []ast.Statement

	// Brace-delimited block
	if p.cur().Type == lexer.LBRACE {
		p.advance()
		p.skipNewlines()
		for p.cur().Type != lexer.RBRACE && p.cur().Type != lexer.EOF {
			stmt, err := p.parseStatement()
			if err != nil {
				return nil, err
			}
			if stmt != nil {
				stmts = append(stmts, stmt)
			}
			p.skipNewlines()
		}
		if _, err := p.expect(lexer.RBRACE); err != nil {
			return nil, err
		}
		return stmts, nil
	}

	// Indentation-delimited block
	if _, err := p.expect(lexer.INDENT); err != nil {
		return nil, err
	}
	for p.cur().Type != lexer.DEDENT && p.cur().Type != lexer.EOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
		p.skipNewlines()
	}
	if p.cur().Type == lexer.DEDENT {
		p.advance()
	}
	return stmts, nil
}

func (p *Parser) parseReturn() (*ast.ReturnStatement, error) {
	line := p.cur().Line
	p.advance()
	if p.cur().Type == lexer.NEWLINE || p.cur().Type == lexer.EOF {
		return &ast.ReturnStatement{Line: line, Value: &ast.NullLiteral{Line: line}}, nil
	}
	val, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return &ast.ReturnStatement{Line: line, Value: val}, nil
}

func (p *Parser) parseIf() (*ast.IfStatement, error) {
	line := p.cur().Line
	p.advance() // consume 'if'
	cond, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	then, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	var elseBlock []ast.Statement
	p.skipNewlines()
	if p.cur().Type == lexer.ELSE {
		p.advance()
		if p.cur().Type == lexer.IF {
			nested, err := p.parseIf()
			if err != nil {
				return nil, err
			}
			elseBlock = []ast.Statement{nested}
		} else {
			if _, err = p.expect(lexer.COLON); err != nil {
				return nil, err
			}
			elseBlock, err = p.parseBlock()
			if err != nil {
				return nil, err
			}
		}
	}
	return &ast.IfStatement{Line: line, Condition: cond, Then: then, Else: elseBlock}, nil
}

func (p *Parser) parseFor() (*ast.ForStatement, error) {
	line := p.cur().Line
	p.advance() // consume 'for'
	varTok, err := p.expect(lexer.IDENTIFIER)
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(lexer.IN); err != nil {
		return nil, err
	}
	iter, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.ForStatement{Line: line, Variable: varTok.Literal, Iterable: iter, Body: body}, nil
}

func (p *Parser) parseWhile() (*ast.WhileStatement, error) {
	line := p.cur().Line
	p.advance()
	cond, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.WhileStatement{Line: line, Condition: cond, Body: body}, nil
}

func (p *Parser) parseTryCatch() (*ast.TryCatchStatement, error) {
	line := p.cur().Line
	p.advance() // try
	if _, err := p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	tryBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	p.skipNewlines()
	if _, err := p.expect(lexer.CATCH); err != nil {
		return nil, err
	}
	errTok, err := p.expect(lexer.IDENTIFIER)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	catchBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.TryCatchStatement{Line: line, Try: tryBlock, ErrVar: errTok.Literal, Catch: catchBlock}, nil
}

func (p *Parser) parseTask() (*ast.TaskStatement, error) {
	line := p.cur().Line
	p.advance()
	call, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	return &ast.TaskStatement{Line: line, Call: call}, nil
}

func (p *Parser) parseImport() (*ast.ImportStatement, error) {
	line := p.cur().Line
	p.advance()
	mod, err := p.expect(lexer.IDENTIFIER)
	if err != nil {
		return nil, err
	}
	return &ast.ImportStatement{Line: line, Module: mod.Literal}, nil
}

func (p *Parser) parseFromImport() (*ast.ImportStatement, error) {
	line := p.cur().Line
	p.advance() // from
	mod, err := p.expect(lexer.IDENTIFIER)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.IMPORT); err != nil {
		return nil, err
	}
	var names []string
	for {
		n, err := p.expect(lexer.IDENTIFIER)
		if err != nil {
			return nil, err
		}
		names = append(names, n.Literal)
		if p.cur().Type != lexer.COMMA {
			break
		}
		p.advance()
	}
	return &ast.ImportStatement{Line: line, Module: mod.Literal, Names: names}, nil
}

func (p *Parser) parseExprOrAssign() (ast.Statement, error) {
	line := p.cur().Line

	// Check for typed assignment: name: type = value
	if p.cur().Type == lexer.IDENTIFIER {
		name := p.cur().Literal
		// peek ahead
		next := p.tokens[p.pos+1]
		if next.Type == lexer.ASSIGN {
			p.advance() // consume name
			p.advance() // consume =
			val, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			return &ast.AssignStatement{Line: line, Name: name, Value: val}, nil
		}
		if next.Type == lexer.COLON {
			// typed: name: type = value
			p.advance() // name
			p.advance() // :
			typeHint := p.cur().Literal
			p.advance() // type name
			if _, err := p.expect(lexer.ASSIGN); err != nil {
				return nil, err
			}
			val, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			return &ast.AssignStatement{Line: line, Name: name, TypeHint: typeHint, Value: val}, nil
		}
	}

	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	// Could be a plain assignment via call result — not re-assignable here
	return &ast.ExpressionStatement{Line: line, Expression: expr}, nil
}

// ── Pratt expression parser ──────────────────────────────────

type precedence int

const (
	_ precedence = iota
	PREC_LOWEST
	PREC_OR
	PREC_AND
	PREC_EQUALS
	PREC_COMPARE
	PREC_SUM
	PREC_PRODUCT
	PREC_UNARY
	PREC_CALL
	PREC_INDEX
)

func tokenPrec(t lexer.TokenType) precedence {
	switch t {
	case lexer.OR:
		return PREC_OR
	case lexer.AND:
		return PREC_AND
	case lexer.EQ, lexer.NEQ:
		return PREC_EQUALS
	case lexer.GT, lexer.LT, lexer.GTE, lexer.LTE:
		return PREC_COMPARE
	case lexer.PLUS, lexer.MINUS:
		return PREC_SUM
	case lexer.STAR, lexer.SLASH, lexer.PERCENT:
		return PREC_PRODUCT
	case lexer.LPAREN:
		return PREC_CALL
	case lexer.LBRACKET:
		return PREC_INDEX
	case lexer.DOT, lexer.QMARK_DOT:
		return PREC_INDEX
	case lexer.DOTDOT:
		return PREC_SUM
	}
	return 0 // not an infix operator
}

func (p *Parser) parseExpression(prec precedence) (ast.Expression, error) {
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}
	for {
		next := p.cur()
		if next.Type == lexer.NEWLINE || next.Type == lexer.EOF {
			break
		}
		nextPrec := tokenPrec(next.Type)
		if nextPrec <= prec {
			break
		}
		left, err = p.parseInfix(left, next.Type, nextPrec)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) parsePrefix() (ast.Expression, error) {
	tok := p.cur()
	switch tok.Type {
	case lexer.NUMBER:
		p.advance()
		v, err := strconv.ParseFloat(tok.Literal, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid number %q", tok.Line, tok.Literal)
		}
		return &ast.NumberLiteral{Line: tok.Line, Value: v}, nil
	case lexer.STRING:
		p.advance()
		return &ast.StringLiteral{Line: tok.Line, Value: tok.Literal}, nil
	case lexer.TRUE:
		p.advance()
		return &ast.BoolLiteral{Line: tok.Line, Value: true}, nil
	case lexer.FALSE:
		p.advance()
		return &ast.BoolLiteral{Line: tok.Line, Value: false}, nil
	case lexer.NULL:
		p.advance()
		return &ast.NullLiteral{Line: tok.Line}, nil
	case lexer.IDENTIFIER:
		p.advance()
		return &ast.Identifier{Line: tok.Line, Token: tok, Value: tok.Literal}, nil
	case lexer.BANG, lexer.MINUS:
		p.advance()
		operand, err := p.parseExpression(PREC_UNARY)
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpression{Line: tok.Line, Operator: tok.Literal, Operand: operand}, nil
	case lexer.LPAREN:
		p.advance()
		expr, err := p.parseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		if _, err = p.expect(lexer.RPAREN); err != nil {
			return nil, err
		}
		return expr, nil
	case lexer.LBRACKET:
		return p.parseListLiteral()
	case lexer.LBRACE:
		return p.parseMapLiteral()
	}
	return nil, fmt.Errorf("line %d: unexpected token %s (%q) in expression", tok.Line, tok.Type, tok.Literal)
}

func (p *Parser) parseInfix(left ast.Expression, op lexer.TokenType, prec precedence) (ast.Expression, error) {
	line := p.cur().Line
	switch op {
	case lexer.LPAREN:
		p.advance()
		args, err := p.parseArgs()
		if err != nil {
			return nil, err
		}
		if _, err = p.expect(lexer.RPAREN); err != nil {
			return nil, err
		}
		return &ast.CallExpression{Line: line, Callee: left, Arguments: args}, nil
	case lexer.LBRACKET:
		p.advance()
		idx, err := p.parseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		if _, err = p.expect(lexer.RBRACKET); err != nil {
			return nil, err
		}
		return &ast.IndexExpression{Line: line, Left: left, Index: idx}, nil
	case lexer.DOT, lexer.QMARK_DOT:
		safe := op == lexer.QMARK_DOT
		p.advance()
		prop, err := p.expect(lexer.IDENTIFIER)
		if err != nil {
			return nil, err
		}
		return &ast.MemberExpression{Line: line, Object: left, Property: prop.Literal, Safe: safe}, nil
	case lexer.DOTDOT:
		p.advance()
		right, err := p.parseExpression(prec)
		if err != nil {
			return nil, err
		}
		return &ast.RangeLiteral{Line: line, Start: left, End: right}, nil
	default:
		opStr := p.cur().Literal
		p.advance()
		right, err := p.parseExpression(prec)
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpression{Line: line, Left: left, Operator: opStr, Right: right}, nil
	}
}

func (p *Parser) parseArgs() ([]ast.Expression, error) {
	var args []ast.Expression
	for p.cur().Type != lexer.RPAREN && p.cur().Type != lexer.EOF {
		arg, err := p.parseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.cur().Type == lexer.COMMA {
			p.advance()
		}
	}
	return args, nil
}

func (p *Parser) parseListLiteral() (*ast.ListLiteral, error) {
	line := p.cur().Line
	p.advance() // [
	var elems []ast.Expression
	p.skipNewlines()
	for p.cur().Type != lexer.RBRACKET && p.cur().Type != lexer.EOF {
		e, err := p.parseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		elems = append(elems, e)
		p.skipNewlines()
		if p.cur().Type == lexer.COMMA {
			p.advance()
			p.skipNewlines()
		}
	}
	if _, err := p.expect(lexer.RBRACKET); err != nil {
		return nil, err
	}
	return &ast.ListLiteral{Line: line, Elements: elems}, nil
}

func (p *Parser) parseMapLiteral() (*ast.MapLiteral, error) {
	line := p.cur().Line
	p.advance() // {
	m := &ast.MapLiteral{Line: line, Pairs: make(map[ast.Expression]ast.Expression)}
	p.skipNewlines()
	for p.cur().Type != lexer.RBRACE && p.cur().Type != lexer.EOF {
		key, err := p.parseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		if _, err = p.expect(lexer.COLON); err != nil {
			return nil, err
		}
		val, err := p.parseExpression(PREC_LOWEST)
		if err != nil {
			return nil, err
		}
		m.Pairs[key] = val
		m.Order = append(m.Order, key)
		p.skipNewlines()
		if p.cur().Type == lexer.COMMA {
			p.advance()
			p.skipNewlines()
		}
	}
	if _, err := p.expect(lexer.RBRACE); err != nil {
		return nil, err
	}
	return m, nil
}
