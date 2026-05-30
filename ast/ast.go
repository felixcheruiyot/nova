package ast

import "nova/lexer"

type Node interface{ nodeType() string }
type Statement interface {
	Node
	stmtNode()
}
type Expression interface {
	Node
	exprNode()
}

// ── Statements ──────────────────────────────────────────────

type Program struct {
	Statements []Statement
}

func (p *Program) nodeType() string { return "Program" }

type AssignStatement struct {
	Line   int
	Name   string
	TypeHint string // optional
	Value  Expression
}

func (a *AssignStatement) nodeType() string { return "AssignStatement" }
func (a *AssignStatement) stmtNode()        {}

type ReturnStatement struct {
	Line  int
	Value Expression
}

func (r *ReturnStatement) nodeType() string { return "ReturnStatement" }
func (r *ReturnStatement) stmtNode()        {}

type ExpressionStatement struct {
	Line       int
	Expression Expression
}

func (e *ExpressionStatement) nodeType() string { return "ExpressionStatement" }
func (e *ExpressionStatement) stmtNode()        {}

type FunctionDeclaration struct {
	Line   int
	Name   string
	Params []string
	Body   []Statement
}

func (f *FunctionDeclaration) nodeType() string { return "FunctionDeclaration" }
func (f *FunctionDeclaration) stmtNode()        {}

type IfStatement struct {
	Line       int
	Condition  Expression
	Then       []Statement
	Else       []Statement
}

func (i *IfStatement) nodeType() string { return "IfStatement" }
func (i *IfStatement) stmtNode()        {}

type ForStatement struct {
	Line     int
	Variable string
	Iterable Expression
	Body     []Statement
}

func (f *ForStatement) nodeType() string { return "ForStatement" }
func (f *ForStatement) stmtNode()        {}

type WhileStatement struct {
	Line      int
	Condition Expression
	Body      []Statement
}

func (w *WhileStatement) nodeType() string { return "WhileStatement" }
func (w *WhileStatement) stmtNode()        {}

type BreakStatement struct{ Line int }

func (b *BreakStatement) nodeType() string { return "BreakStatement" }
func (b *BreakStatement) stmtNode()        {}

type ContinueStatement struct{ Line int }

func (c *ContinueStatement) nodeType() string { return "ContinueStatement" }
func (c *ContinueStatement) stmtNode()        {}

type TryCatchStatement struct {
	Line     int
	Try      []Statement
	ErrVar   string
	Catch    []Statement
}

func (t *TryCatchStatement) nodeType() string { return "TryCatchStatement" }
func (t *TryCatchStatement) stmtNode()        {}

type TaskStatement struct {
	Line int
	Call Expression
}

func (t *TaskStatement) nodeType() string { return "TaskStatement" }
func (t *TaskStatement) stmtNode()        {}

type WaitStatement struct{ Line int }

func (w *WaitStatement) nodeType() string { return "WaitStatement" }
func (w *WaitStatement) stmtNode()        {}

type ImportStatement struct {
	Line   int
	Module string
	Names  []string // empty means import whole module
}

func (i *ImportStatement) nodeType() string { return "ImportStatement" }
func (i *ImportStatement) stmtNode()        {}

// ── Expressions ─────────────────────────────────────────────

type Identifier struct {
	Line  int
	Token lexer.Token
	Value string
}

func (i *Identifier) nodeType() string { return "Identifier" }
func (i *Identifier) exprNode()        {}

type NumberLiteral struct {
	Line  int
	Value float64
}

func (n *NumberLiteral) nodeType() string { return "NumberLiteral" }
func (n *NumberLiteral) exprNode()        {}

type StringLiteral struct {
	Line  int
	Value string
}

func (s *StringLiteral) nodeType() string { return "StringLiteral" }
func (s *StringLiteral) exprNode()        {}

type BoolLiteral struct {
	Line  int
	Value bool
}

func (b *BoolLiteral) nodeType() string { return "BoolLiteral" }
func (b *BoolLiteral) exprNode()        {}

type NullLiteral struct{ Line int }

func (n *NullLiteral) nodeType() string { return "NullLiteral" }
func (n *NullLiteral) exprNode()        {}

type ListLiteral struct {
	Line     int
	Elements []Expression
}

func (l *ListLiteral) nodeType() string { return "ListLiteral" }
func (l *ListLiteral) exprNode()        {}

type MapLiteral struct {
	Line  int
	Pairs map[Expression]Expression
	Order []Expression // preserve insertion order
}

func (m *MapLiteral) nodeType() string { return "MapLiteral" }
func (m *MapLiteral) exprNode()        {}

type RangeLiteral struct {
	Line  int
	Start Expression
	End   Expression
}

func (r *RangeLiteral) nodeType() string { return "RangeLiteral" }
func (r *RangeLiteral) exprNode()        {}

type BinaryExpression struct {
	Line     int
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BinaryExpression) nodeType() string { return "BinaryExpression" }
func (b *BinaryExpression) exprNode()        {}

type UnaryExpression struct {
	Line     int
	Operator string
	Operand  Expression
}

func (u *UnaryExpression) nodeType() string { return "UnaryExpression" }
func (u *UnaryExpression) exprNode()        {}

type CallExpression struct {
	Line      int
	Callee    Expression
	Arguments []Expression
}

func (c *CallExpression) nodeType() string { return "CallExpression" }
func (c *CallExpression) exprNode()        {}

type MemberExpression struct {
	Line     int
	Object   Expression
	Property string
	Safe     bool // ?. operator
}

func (m *MemberExpression) nodeType() string { return "MemberExpression" }
func (m *MemberExpression) exprNode()        {}

type IndexExpression struct {
	Line  int
	Left  Expression
	Index Expression
}

func (i *IndexExpression) nodeType() string { return "IndexExpression" }
func (i *IndexExpression) exprNode()        {}

// ── HTTP server DSL ─────────────────────────────────────────────

type ServerStatement struct {
	Line   int
	Port   int // default 8080
	Routes []*RouteHandler
}

func (s *ServerStatement) nodeType() string { return "ServerStatement" }
func (s *ServerStatement) stmtNode()        {}

type RouteHandler struct {
	Line   int
	Method string // "get" | "post" | "put" | "delete"
	Path   string
	Body   []Statement
}

func (r *RouteHandler) nodeType() string { return "RouteHandler" }
func (r *RouteHandler) stmtNode()        {}
