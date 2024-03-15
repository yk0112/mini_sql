package mini_sql

import (
	"errors"
	"fmt"

	"golang.org/x/text/currency"
)

func tokenFromKeyword(k Keyword) Token {
	return Token{
		Kind:  KeywordKind,
		Value: string(k),
	}
}

func tokenFromSymbol(s Symbol) Token {
	return Token{
		Kind:  SymbolKind,
		Value: string(s),
	}
}

func expectToken(tokens []*Token, cursor uint, t Token) bool {
	if cursor >= uint(len(tokens)) {
		return false
	}

	return t.equals(tokens[cursor])
}

func helpMessage(tokens []*Token, cursor uint, msg string) {
	var c *Token
	if cursor < uint(len(tokens)) {
		c = tokens[cursor]
	} else {
		c = tokens[cursor-1]
	}

	fmt.Printf("[%d,%d]: %s, got: %s\n", c.Loc.Line, c.Loc.Col, msg, c.Value)
}

func parseToken(tokens []*Token, initialCursor uint, kind tokenKind) (*Token, uint, bool) {
	cursor := initialCursor

	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	current := tokens[cursor]

	if current.Kind == kind {
		return current, cursor + 1, true
	}
	return nil, initialCursor, false
}

func parseExpressions(tokens []*token, initialCursor uint, delimiter []token) (*[]*expression, uint, bool) {
}

func parseSelectStatement(tokens []*Token, initialCursor uint, delimiter Token) (*SelectStatement, uint, bool) {
	cursor := initialCursor
	if !expectToken(tokens, cursor, tokenFromKeyword(SelectKeyword)) {
		return nil, initialCursor, false
	}
	cursor++

	slct := SelectStatement{}

	exps, newCursor, ok := parseExpressions(tokens, cursor, []Token{tokenFromKeyword(FromKeyword), delimiter})
	if !ok {
		return nil, initialCursor, false
	}

	slct.item = *exps
	cursor = newCursor

	if expectToken(tokens, cursor, tokenFromKeyword(FromKeyword)) {
		cursor++
		from, newCursor, ok := parseToken(tokens, cursor, IdentifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected From token")
			return nil, initialCursor, false
		}

		slct.from = *from
		cursor = newCursor
	}
	return &slct, cursor, true
}

func parseStatement(tokens []*Token, initialCursor uint, delimiter Token) (*Statement, uint, bool) {
	cursor := initialCursor

	// SECRET statement
	semicolonToken := tokenFromSymbol(SemicolonSymbol)
	slct, newCursor, ok := parseSelectStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:            SelectKind,
			SelectStatement: slct,
		}, newCursor, true
	}

	// INSERT statement
	inst, newCursor, ok := parserInsertStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:            InsertKind,
			InsertStatement: inst,
		}, newCursor, true
	}

	// CREATE statement
	crtTbl, newCursor, ok := parserCreateTableStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:                 CreateTableKind,
			CreateTableStatement: crtTbl,
		}, newCursor, true
	}

	return nil, initialCursor, false
}

func Parse(source string) (*Ast, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}

	a := Ast{}
	cursor := uint(0)
	for cursor < uint(len(tokens)) {
		stmt, newCursor, ok := parseStatement(tokens, cursor, tokenFromSymbol(SemicolonSymbol))
		if !ok {
			helpMessage(tokens, cursor, "Expected Statement")
			return nil, errors.New("Failed to parse, expected statement")
		}
		cursor = newCursor
		a.Statements = append(a.Statements, stmt)

		atLeastOneSemicolon := false
		for expectToken(tokens, cursor, tokenFromSymbol(SemicolonSymbol)) {
			cursor++
			atLeastOneSemicolon = true
		}

		if !atLeastOneSemicolon {
			helpMessage(tokens, cursor, "Expected semi-colon delimiter between statements")
			return nil, errors.New("Missing semi-colon between statements")
		}
	}
	return &a, nil
}
