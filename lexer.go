package mini_sql

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2/middleware/skip"
)

type location struct {
	line uint
	col  uint
}

type keyword string

const (
	selectKeyword keyword = "select"
	fromKeyword   keyword = "from"
	asKeyword     keyword = "as"
	tableKeyword  keyword = "table"
	createKeyword keyword = "create"
	insertKeyword keyword = "insert"
	whereKeyword  keyword = "where"
	intoKeyword   keyword = "into"
	valuesKeyword keyword = "values"
	intKeyword    keyword = "int"
	textKeyword   keyword = "text"
)

type symbol string

const (
	semicolonSymbol  symbol = ";"
	asteriskSymbol   symbol = "*"
	commaSymbol      symbol = ","
	leftparenSymbol  symbol = "("
	rightparenSymbol symbol = ")"
)

type tokenKind uint

const (
	keywordKind tokenKind = iota
	symbolKind
	identifierKind
	stringKind
	numericKind
)

type token struct {
	value string
	kind  tokenKind
	loc   location
}

type cursor struct {
	pointer uint
	loc     location
}

func (t *token) equals(other *token) bool {
	return t.value == other.value && t.kind == other.kind
}

type lexer func(string, cursor) (*token, cursor, bool)

func lexNumeric(source string, ic cursor) (*token, cursor, bool) {
	cur := ic
	periodFound := false    // 小数点
	expMarkerFound := false // 指数
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]
		cur.loc.col++
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

			// No periods allowed after expMarker
			periodFound = true
			expMarkerFound = true

			// expMarker must be followed by digits
			if cur.pointer == uint(len(source)-1) {
				return nil, ic, false
			}

			cNext := source[cur.pointer+1]
			if cNext == '-' || cNext == '+' {
				cur.pointer++
				cur.loc.col++
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

	return &token{
		value: source[ic.pointer:cur.pointer],
		loc:   ic.loc,
		kind:  numericKind,
	}, cur, true
}

func lexCharacterDelimited(source string, ic cursor, delimiter byte) (*token, cursor, bool) {
	cur := ic

	if len(source[cur.pointer:]) == 0 {
		return nil, ic, false
	}

	if source[cur.pointer] != delimiter {
		return nil, ic, false
	}

	cur.loc.col++
	cur.pointer++

	var value []byte

	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]

		if c == delimiter {
			if cur.pointer+1 >= uint(len(source)) || source[cur.pointer+1] != delimiter {
				return &token{
					value: string(value),
					loc:   ic.loc,
					kind:  stringKind,
				}, cur, true
			} else { // 連続するデリミタはエスケープとみなす
				value = append(value, delimiter)
				cur.pointer++
				cur.loc.col++
			}
		}
		value = append(value, c)
		cur.loc.col++
	}
	return nil, ic, false
}

func lexString(source string, ic cursor) (*token, cursor, bool) {
	return lexCharacterDelimited(source, ic, '\'')
}

func lexSymbol(source string, ic cursor) (*token, cursor, bool) {
	c := source[ic.pointer]
	cur := ic

	cur.pointer++
	cur.loc.col++

	switch c {
	case '\n':
		cur.loc.line++
		cur.loc.col = 0
		fallthrough
	case '\t':
		fallthrough
	case ' ':
		return nil, cur, true
	}

	symbols := []symbol{
		commaSymbol,
		leftparenSymbol,
		rightparenSymbol,
		semicolonSymbol,
		asteriskSymbol,
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
	cur.loc.col = ic.loc.col + uint(len(match))

	return &token{
		value: match,
		loc:   ic.loc,
		kind:  symbolKind,
	}, cur, true
}

func lexKeyword(source string, ic cursor) (*token, cursor, bool) {
	cur := ic
	keywords := []keyword{
		selectKeyword,
		insertKeyword,
		valuesKeyword,
		tableKeyword,
		createKeyword,
		whereKeyword,
		fromKeyword,
		intoKeyword,
		textKeyword,
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
	cur.loc.col = ic.loc.col + uint(len(match))

	return &token{
		value: match,
		kind:  kind,
		loc:   ic.loc,
	}, cur, true
}

func longestMatch(source string, ic cursor, options []string) {
	var value []byte
	var skipList []int
	var match string

	cur := ic

	for cur.pointer < uint(len(source)) {
		value = append(value, strings.ToLower(strings(source[cur.pointer]))...)
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

		}

	}
}

func lex(source string) ([]*token, error) {
	tokens := []*token{}
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
			hint = "after" + tokens[len(tokens)-1].value
		}
		return nil, fmt.Errorf("Unable to lex token%s, at %d:%d", hint, cur.loc.line, cur.loc.col)
	}
	return tokens, nil
}
