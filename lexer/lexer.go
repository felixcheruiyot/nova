package lexer

import (
	"fmt"
	"strings"
)

type Lexer struct {
	source  []rune
	pos     int
	line    int
	col     int
	indents []int
	tokens  []Token
}

func New(source string) *Lexer {
	return &Lexer{
		source:  []rune(source),
		pos:     0,
		line:    1,
		col:     0,
		indents: []int{0},
	}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.source) {
		return 0
	}
	return l.source[l.pos]
}

func (l *Lexer) peekAt(offset int) rune {
	i := l.pos + offset
	if i >= len(l.source) {
		return 0
	}
	return l.source[i]
}

func (l *Lexer) advance() rune {
	ch := l.source[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 0
	} else {
		l.col++
	}
	return ch
}

func (l *Lexer) tok(t TokenType, lit string) Token {
	return Token{Type: t, Literal: lit, Line: l.line, Col: l.col}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	for l.pos < len(l.source) {
		ch := l.peek()

		// Handle newlines and indentation
		if ch == '\n' {
			l.advance()
			l.tokens = append(l.tokens, l.tok(NEWLINE, "\n"))
			// Count indent of next line
			indent := 0
			for l.pos < len(l.source) && l.peek() == ' ' {
				indent++
				l.advance()
			}
			// Skip blank lines and comment lines
			if l.pos < len(l.source) && (l.peek() == '\n' || l.peek() == '#') {
				continue
			}
			curr := l.indents[len(l.indents)-1]
			if indent > curr {
				l.indents = append(l.indents, indent)
				l.tokens = append(l.tokens, l.tok(INDENT, ""))
			} else {
				for indent < l.indents[len(l.indents)-1] {
					l.indents = l.indents[:len(l.indents)-1]
					l.tokens = append(l.tokens, l.tok(DEDENT, ""))
				}
			}
			continue
		}

		// Skip spaces and tabs (within a line)
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
			continue
		}

		// Comments
		if ch == '#' {
			for l.pos < len(l.source) && l.peek() != '\n' {
				l.advance()
			}
			continue
		}

		// String literals
		if ch == '"' || ch == '\'' {
			tok, err := l.readString(ch)
			if err != nil {
				return nil, err
			}
			l.tokens = append(l.tokens, tok)
			continue
		}

		// Numbers
		if isDigit(ch) {
			l.tokens = append(l.tokens, l.readNumber())
			continue
		}

		// Identifiers / keywords
		if isLetter(ch) {
			l.tokens = append(l.tokens, l.readIdent())
			continue
		}

		// Operators and delimiters
		tok, err := l.readSymbol()
		if err != nil {
			return nil, err
		}
		l.tokens = append(l.tokens, tok)
	}

	// Emit remaining DEDENTs
	for len(l.indents) > 1 {
		l.indents = l.indents[:len(l.indents)-1]
		l.tokens = append(l.tokens, l.tok(DEDENT, ""))
	}
	l.tokens = append(l.tokens, l.tok(EOF, ""))
	return l.tokens, nil
}

func (l *Lexer) readString(quote rune) (Token, error) {
	l.advance() // consume opening quote
	var sb strings.Builder
	for l.pos < len(l.source) {
		ch := l.peek()
		if ch == quote {
			l.advance()
			return l.tok(STRING, sb.String()), nil
		}
		if ch == '\\' {
			l.advance()
			esc := l.advance()
			switch esc {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case '\\':
				sb.WriteRune('\\')
			default:
				sb.WriteRune(esc)
			}
			continue
		}
		sb.WriteRune(l.advance())
	}
	return Token{}, fmt.Errorf("line %d: unterminated string", l.line)
}

func (l *Lexer) readNumber() Token {
	var sb strings.Builder
	for l.pos < len(l.source) && (isDigit(l.peek()) || l.peek() == '.') {
		// Prevent reading ".." range operator as part of number
		if l.peek() == '.' && l.peekAt(1) == '.' {
			break
		}
		sb.WriteRune(l.advance())
	}
	return l.tok(NUMBER, sb.String())
}

func (l *Lexer) readIdent() Token {
	var sb strings.Builder
	for l.pos < len(l.source) && (isLetter(l.peek()) || isDigit(l.peek())) {
		sb.WriteRune(l.advance())
	}
	lit := sb.String()
	return l.tok(LookupIdent(lit), lit)
}

func (l *Lexer) readSymbol() (Token, error) {
	ch := l.advance()
	switch ch {
	case '+':
		return l.tok(PLUS, "+"), nil
	case '-':
		return l.tok(MINUS, "-"), nil
	case '*':
		return l.tok(STAR, "*"), nil
	case '/':
		return l.tok(SLASH, "/"), nil
	case '%':
		return l.tok(PERCENT, "%"), nil
	case '(':
		return l.tok(LPAREN, "("), nil
	case ')':
		return l.tok(RPAREN, ")"), nil
	case '{':
		return l.tok(LBRACE, "{"), nil
	case '}':
		return l.tok(RBRACE, "}"), nil
	case '[':
		return l.tok(LBRACKET, "["), nil
	case ']':
		return l.tok(RBRACKET, "]"), nil
	case ',':
		return l.tok(COMMA, ","), nil
	case ':':
		return l.tok(COLON, ":"), nil
	case '.':
		if l.pos < len(l.source) && l.peek() == '.' {
			l.advance()
			return l.tok(DOTDOT, ".."), nil
		}
		return l.tok(DOT, "."), nil
	case '=':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			return l.tok(EQ, "=="), nil
		}
		return l.tok(ASSIGN, "="), nil
	case '!':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			return l.tok(NEQ, "!="), nil
		}
		return l.tok(BANG, "!"), nil
	case '>':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			return l.tok(GTE, ">="), nil
		}
		return l.tok(GT, ">"), nil
	case '<':
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			return l.tok(LTE, "<="), nil
		}
		return l.tok(LT, "<"), nil
	case '&':
		if l.pos < len(l.source) && l.peek() == '&' {
			l.advance()
			return l.tok(AND, "&&"), nil
		}
	case '|':
		if l.pos < len(l.source) && l.peek() == '|' {
			l.advance()
			return l.tok(OR, "||"), nil
		}
	case '?':
		if l.pos < len(l.source) && l.peek() == '.' {
			l.advance()
			return l.tok(QMARK_DOT, "?."), nil
		}
		return l.tok(QUESTION, "?"), nil
	}
	return Token{}, fmt.Errorf("line %d: unexpected character %q", l.line, ch)
}

func isDigit(ch rune) bool { return ch >= '0' && ch <= '9' }
func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}
