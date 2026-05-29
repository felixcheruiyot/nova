package lexer

type TokenType string

const (
	// Literals
	IDENTIFIER TokenType = "IDENTIFIER"
	NUMBER     TokenType = "NUMBER"
	STRING     TokenType = "STRING"

	// Keywords
	IF       TokenType = "if"
	ELSE     TokenType = "else"
	FOR      TokenType = "for"
	WHILE    TokenType = "while"
	FUNC     TokenType = "func"
	RETURN   TokenType = "return"
	BREAK    TokenType = "break"
	CONTINUE TokenType = "continue"
	TASK     TokenType = "task"
	WAIT     TokenType = "wait"
	IMPORT   TokenType = "import"
	FROM     TokenType = "from"
	CLASS    TokenType = "class"
	TRUE     TokenType = "true"
	FALSE    TokenType = "false"
	NULL     TokenType = "null"
	SERVER   TokenType = "server"
	GET      TokenType = "get"
	POST     TokenType = "post"
	PUT      TokenType = "put"
	DELETE   TokenType = "delete"
	TRY      TokenType = "try"
	CATCH    TokenType = "catch"
	IN       TokenType = "in"

	// Operators
	PLUS      TokenType = "+"
	MINUS     TokenType = "-"
	STAR      TokenType = "*"
	SLASH     TokenType = "/"
	PERCENT   TokenType = "%"
	EQ        TokenType = "=="
	NEQ       TokenType = "!="
	GT        TokenType = ">"
	LT        TokenType = "<"
	GTE       TokenType = ">="
	LTE       TokenType = "<="
	AND       TokenType = "&&"
	OR        TokenType = "||"
	BANG      TokenType = "!"
	ASSIGN    TokenType = "="
	COLON     TokenType = ":"
	COMMA     TokenType = ","
	DOT       TokenType = "."
	DOTDOT    TokenType = ".."
	QUESTION  TokenType = "?"
	QMARK_DOT TokenType = "?."

	// Delimiters
	LPAREN   TokenType = "("
	RPAREN   TokenType = ")"
	LBRACE   TokenType = "{"
	RBRACE   TokenType = "}"
	LBRACKET TokenType = "["
	RBRACKET TokenType = "]"

	// Structure
	NEWLINE TokenType = "NEWLINE"
	INDENT  TokenType = "INDENT"
	DEDENT  TokenType = "DEDENT"
	EOF     TokenType = "EOF"
)

var keywords = map[string]TokenType{
	"if":       IF,
	"else":     ELSE,
	"for":      FOR,
	"while":    WHILE,
	"func":     FUNC,
	"return":   RETURN,
	"break":    BREAK,
	"continue": CONTINUE,
	"task":     TASK,
	"wait":     WAIT,
	"import":   IMPORT,
	"from":     FROM,
	"class":    CLASS,
	"true":     TRUE,
	"false":    FALSE,
	"null":     NULL,
	"server":   SERVER,
	"get":      GET,
	"post":     POST,
	"put":      PUT,
	"delete":   DELETE,
	"try":      TRY,
	"catch":    CATCH,
	"in":       IN,
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENTIFIER
}
