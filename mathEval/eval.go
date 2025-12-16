package matheval

import (
	"fmt"
	"strconv"
	"unicode"
)

type tokenType int

const (
	tEOF tokenType = iota
	tNum
	tOp
	tLParen
	tRParen
)

type token struct {
	typ tokenType
	val int64 // for numbers
	op  rune  // for operators and parens
}

type lexer struct {
	s []rune
	i int
}

func newLexer(input string) *lexer {
	return &lexer{s: []rune(input)}
}

func (l *lexer) nextToken() token {
	// skip whitespace
	for l.i < len(l.s) && unicode.IsSpace(l.s[l.i]) {
		l.i++
	}
	if l.i >= len(l.s) {
		return token{typ: tEOF}
	}

	ch := l.s[l.i]

	// number
	if unicode.IsDigit(ch) {
		start := l.i
		for l.i < len(l.s) && unicode.IsDigit(l.s[l.i]) {
			l.i++
		}
		n, _ := strconv.ParseInt(string(l.s[start:l.i]), 10, 64)
		return token{typ: tNum, val: n}
	}

	// multi-character operators (>>>, >>, <<)
	if ch == '>' && l.i+2 < len(l.s) && l.s[l.i+1] == '>' && l.s[l.i+2] == '>' {
		l.i += 3
		return token{typ: tOp, op: '→'} // using → as placeholder for >>>
	}
	if ch == '>' && l.i+1 < len(l.s) && l.s[l.i+1] == '>' {
		l.i += 2
		return token{typ: tOp, op: '»'} // using » as placeholder for >>
	}
	if ch == '<' && l.i+1 < len(l.s) && l.s[l.i+1] == '<' {
		l.i += 2
		return token{typ: tOp, op: '«'} // using « as placeholder for <<
	}

	// operators and parens
	l.i++
	switch ch {
	case '+', '-', '*', '&', '|', '^', '~':
		return token{typ: tOp, op: ch}
	case '(':
		return token{typ: tLParen, op: ch}
	case ')':
		return token{typ: tRParen, op: ch}
	default:
		panic(fmt.Sprintf("unexpected character: %q", ch))
	}
}

type parser struct {
	lex *lexer
	tok token
}

func newParser(s string) *parser {
	lex := newLexer(s)
	p := &parser{lex: lex}
	p.next()
	return p
}

func (p *parser) next() {
	p.tok = p.lex.nextToken()
}

func (p *parser) expect(typ tokenType, op rune) {
	if p.tok.typ != typ || (op != 0 && p.tok.op != op) {
		panic("parse error")
	}
	p.next()
}

// Grammar (with precedence, highest to lowest):
// expr      = bitwise
// bitwise   = shift { ('&' | '|' | '^') shift }
// shift     = addSub { ('<<' | '>>' | '>>>') addSub }
// addSub    = mul { ('+' | '-') mul }
// mul       = unary { '*' unary }
// unary     = ('+' | '-' | '~') unary | primary
// primary   = number | '(' expr ')'

func (p *parser) parseExpr() int64 {
	return p.parseBitwise()
}

func (p *parser) parseBitwise() int64 {
	val := p.parseShift()
	for p.tok.typ == tOp && (p.tok.op == '&' || p.tok.op == '|' || p.tok.op == '^') {
		op := p.tok.op
		p.next()
		rhs := p.parseShift()
		switch op {
		case '&':
			val &= rhs
		case '|':
			val |= rhs
		case '^':
			val ^= rhs
		}
	}
	return val
}

func (p *parser) parseShift() int64 {
	val := p.parseAddSub()
	for p.tok.typ == tOp && (p.tok.op == '«' || p.tok.op == '»' || p.tok.op == '→') {
		op := p.tok.op
		p.next()
		rhs := p.parseAddSub()
		switch op {
		case '«': // <<
			val <<= uint64(rhs)
		case '»': // >>
			val >>= uint64(rhs)
		case '→': // >>>
			// Unsigned right shift: convert to uint64, shift, convert back
			val = int64(uint64(val) >> uint64(rhs))
		}
	}
	return val
}

func (p *parser) parseAddSub() int64 {
	val := p.parseMul()
	for p.tok.typ == tOp && (p.tok.op == '+' || p.tok.op == '-') {
		op := p.tok.op
		p.next()
		rhs := p.parseMul()
		if op == '+' {
			val += rhs
		} else {
			val -= rhs
		}
	}
	return val
}

func (p *parser) parseMul() int64 {
	val := p.parseUnary()
	for p.tok.typ == tOp && p.tok.op == '*' {
		p.next()
		rhs := p.parseUnary()
		val *= rhs
	}
	return val
}

func (p *parser) parseUnary() int64 {
	if p.tok.typ == tOp && (p.tok.op == '+' || p.tok.op == '-' || p.tok.op == '~') {
		op := p.tok.op
		p.next()
		val := p.parseUnary()
		switch op {
		case '+':
			return +val
		case '-':
			return -val
		case '~':
			return ^val // bitwise NOT
		}
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() int64 {
	switch p.tok.typ {
	case tNum:
		val := p.tok.val
		p.next()
		return val
	case tLParen:
		p.next()
		val := p.parseExpr()
		p.expect(tRParen, ')')
		return val
	default:
		panic("expected number or '('")
	}
}

func Eval(expr string) int64 {
	p := newParser(expr)
	res := p.parseExpr()
	if p.tok.typ != tEOF {
		panic("extra tokens at end")
	}
	return res
}
