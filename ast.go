package mini_sql

type Ast struct {
	Statements []*Statement
}

type AstKind uint

const (
	SelectKind AstKind = iota
	CreateTableKind
	InsertKind
)

type expression struct {
	literal *Token
	kind    expressionKind
}

type InsertStatement struct {
	table  Token
	values *[]*expression
}

type columnDefinition struct {
	name     Token
	datatype Token
}

type CreateTableStatement struct {
	name Token
	cols *[]*columnDefinition
}

type SelectStatement struct {
	item []*expression
	from Token
}

type Statement struct {
	SelectStatement      *SelectStatement
	CreateTableStatement *CreateTableStatement
	InsertStatement      *InsertStatement
	Kind                 AstKind
}

type expressionKind uint

const (
	literalKind expressionKind = iota
)
