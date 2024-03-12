package mini_sql

import (
	"fmt"
	"strings"
)

type Location struct {
	Line uint
	Col  uint
}

type Keyword string

const (
	SelectKeyword     Keyword = "select"
	FromKeyword       Keyword = "from"
	AsKeyword         Keyword = "as"
	TableKeyword      Keyword = "table"
	CreateKeyword     Keyword = "create"
	DropKeyword       Keyword = "drop"
	InsertKeyword     Keyword = "insert"
	IntoKeyword       Keyword = "into"
	ValuesKeyword     Keyword = "values"
	IntKeyword        Keyword = "int"
	TextKeyword       Keyword = "text"
	BoolKeyword       Keyword = "boolean"
	WhereKeyword      Keyword = "where"
	AndKeyword        Keyword = "and"
	OrKeyword         Keyword = "or"
	TrueKeyword       Keyword = "true"
	FalseKeyword      Keyword = "false"
	UniqueKeyword     Keyword = "unique"
	IndexKeyword      Keyword = "index"
	OnKeyword         Keyword = "on"
	PrimarykeyKeyword Keyword = "primary key"
	NullKeyword       Keyword = "null"
	LimitKeyword      Keyword = "limit"
	OffsetKeyword     Keyword = "offset"
)

type symbol string

const (
	SemicolonSymbol  symbol = ";"
	AsteriskSymbol   symbol = "*"
	CommaSymbol      symbol = ","
	LeftparenSymbol  symbol = "("
	RightparenSymbol symbol = ")"
)

type tokenKind uint

const (
	KeywordKind tokenKind = iota
	SymbolKind
	IdentifierKind
	StringKind
	NumericKind
	BoolKind
	NullKind
)

type Token struct {
	Value string
	Kind  tokenKind
	Loc   Location
}

type cursor struct {
	pointer uint
	loc     Location
}

func (t *Token) equals(other *Token) bool {
	return t.Value == other.Value && t.Kind == other.Kind
}

type lexer func(string, cursor) (*Token, cursor, bool)

func lexNumeric(source string, ic cursor) (*Token, cursor, bool) {
	cur := ic
	periodFound := false    // 小数点
	expMarkerFound := false // 指数
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]
		cur.loc.Col++
		isDigit := c >= '0' && c <= '9'
		isPeriod := c == '.'
		isExpMarker := c == 'e'

		if cur.pointer == ic.pointer { // 先頭の文字?
			if !isDigit && !isPeriod {
				return nil, ic, false
			}

			periodFound = isPeriod
			continue
		}

		if isPeriod {
			if periodFound {
				return nil, ic, false
			}
			periodFound = true
			continue
		}

		if isExpMarker {
			if expMarkerFound {
				return nil, ic, false
			}

			periodFound = true
			expMarkerFound = true

			if cur.pointer == uint(len(source)-1) {
				return nil, ic, false
			}

			cNext := source[cur.pointer+1]
			if cNext == '-' || cNext == '+' {
				cur.pointer++
				cur.loc.Col++
			}

			continue
		}

		if !isDigit {
			break
		}
	}

	if cur.pointer == ic.pointer {
		return nil, ic, false
	}

	return &Token{
		Value: source[ic.pointer:cur.pointer],
		Loc:   ic.loc,
		Kind:  NumericKind,
	}, cur, true
}

func lexCharacterDelimited(source string, ic cursor, delimiter byte) (*Token, cursor, bool) {
	cur := ic

	if len(source[cur.pointer:]) == 0 {
		return nil, ic, false
	}

	if source[cur.pointer] != delimiter {
		return nil, ic, false
	}

	cur.loc.Col++
	cur.pointer++

	var value []byte

	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]

		if c == delimiter {
			if cur.pointer+1 >= uint(len(source)) || source[cur.pointer+1] != delimiter {
				return &Token{
					Value: string(value),
					Loc:   ic.loc,
					Kind:  StringKind,
				}, cur, true
			} else { // 連続するデリミタはエスケープとみなす
				value = append(value, delimiter)
				cur.pointer++
				cur.loc.Col++
			}
		}
		value = append(value, c)
		cur.loc.Col++
	}
	return nil, ic, false
}

func lexString(source string, ic cursor) (*Token, cursor, bool) {
	return lexCharacterDelimited(source, ic, '\'')
}

func lexSymbol(source string, ic cursor) (*Token, cursor, bool) {
	c := source[ic.pointer]
	cur := ic

	cur.pointer++
	cur.loc.Col++

	switch c {
	case '\n':
		cur.loc.Line++
		cur.loc.Col = 0
		fallthrough
	case '\t':
		fallthrough
	case ' ':
		return nil, cur, true
	}

	symbols := []symbol{
		CommaSymbol,
		LeftparenSymbol,
		RightparenSymbol,
		SemicolonSymbol,
		AsteriskSymbol,
	}

	var options []string
	for _, s := range symbols {
		options = append(options, string(s))
	}

	match := longestMatch(source, ic, options)
	if match == "" {
		return nil, ic, false
	}

	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.Col = ic.loc.Col + uint(len(match))

	return &Token{
		Value: match,
		Loc:   ic.loc,
		Kind:  SymbolKind,
	}, cur, true
}

func lexKeyword(source string, ic cursor) (*Token, cursor, bool) {
	cur := ic
	keywords := []Keyword{
		SelectKeyword,
		InsertKeyword,
		ValuesKeyword,
		TableKeyword,
		CreateKeyword,
		WhereKeyword,
		FromKeyword,
		IntoKeyword,
		TextKeyword,
	}
	var options []string

	for _, k := range keywords {
		options = append(options, string(k))
	}

	match := longestMatch(source, ic, options)
	if match == "" {
		return nil, ic, false
	}

	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.Col = ic.loc.Col + uint(len(match))

	Kind := KeywordKind

	if match == string(TrueKeyword) || match == string(FalseKeyword) {
		Kind = BoolKind
	}

	if match == string(NullKeyword) {
		Kind = NullKind
	}
	return &Token{
		Value: match,
		Kind:  Kind,
		Loc:   ic.loc,
	}, cur, true
}

func lexIdentifier(source string, ic cursor) (*Token, cursor, bool) {
	if token, newCursor, ok := lexCharacterDelimited(source, ic, '"'); ok {
		return token, newCursor, true
	}

	cur := ic

	c := source[cur.pointer]

	isAlphabetical := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')

	if !isAlphabetical {
		return nil, ic, false
	}
	cur.pointer++
	cur.loc.Col++

	value := []byte{c}

	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c = source[cur.pointer]
		isAlphabetical := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		isNumeric := c >= '0' && c <= '9'
		if isAlphabetical || isNumeric || c == '$' || c == '_' {
			value = append(value, c)
			cur.loc.Col++
			continue
		}
		break
	}

	if len(value) == 0 {
		return nil, ic, false
	}

	return &Token{
		Value: strings.ToLower(string(value)),
		Loc:   ic.loc,
		Kind:  IdentifierKind,
	}, cur, true
}

func longestMatch(source string, ic cursor, options []string) string {
	var value []byte
	var skipList []int
	var match string

	cur := ic

	for cur.pointer < uint(len(source)) {
		value = append(value, strings.ToLower(string(source[cur.pointer]))...)
		cur.pointer++
	match:
		for i, option := range options {
			for _, skip := range skipList {
				if i == skip {
					continue match
				}
			}

			if option == string(value) {
				skipList = append(skipList, i)
				if len(option) > len(match) {
					match = option
				}
				continue
			}
			sharesPrefix := string(value) == option[:cur.pointer-ic.pointer]
			tooLong := len(value) > len(option)
			if tooLong || !sharesPrefix {
				skipList = append(skipList, i)
			}
		}
		if len(skipList) == len(options) {
			break
		}
	}
	return match
}

func lex(source string) ([]*Token, error) {
	tokens := []*Token{}
	cur := cursor{}

lex:
	for cur.pointer < uint(len(source)) {
		lexers := []lexer{lexKeyword, lexSymbol, lexString, lexNumeric, lexIdentifier}
		for _, l := range lexers {
			if token, newCursor, ok := l(source, cur); ok {
				cur = newCursor
				if token != nil {
					tokens = append(tokens, token)
				}
				continue lex
			}
		}
		hint := ""
		if len(tokens) > 0 {
			hint = "after" + tokens[len(tokens)-1].Value
		}
		return nil, fmt.Errorf("Unable to lex token%s, at %d:%d", hint, cur.loc.Line, cur.loc.Col)
	}
	return tokens, nil
}
