package main

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	_MeaninglessRunes     = " \n\r\t"
	_NumberRunes          = "0123456789"
	_Star                 = "*"
	_RawQuote             = "`"
	_Quote                = "'"
	_OpValueRunes         = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ'"
	_OpValueRunesNoQuotes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	_Ident                = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_"
	_Keywords             = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	_Operator             = "=><"
)

const (
	EOF    = -1
	C_WS   = ' '
	C_STAR = '*'
)

const (
	ILLEGAL tokenType = iota
	KEYWORD
	IDENT
	OPERATOR // operator
	DOT
	OPV_NUMBER // operator value of numbers
	OPV_QUOTED // operator value of quoted text
)

type lexFn func(*lexer) lexFn
type parseFn func(*parser) parseFn

type tokenType int

type token struct {
	typ  tokenType
	pos  Pos
	text string
}

type Pos int

type lexer struct {
	start   Pos
	pos     Pos
	lastPos Pos
	width   Pos
	input   string
	state   lexFn
	tokens  chan token
}

func (l *lexer) peek() (r rune) {
	r = l.next()
	l.backup()
	return
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) next() (r rune) {
	if int(l.pos) >= len(l.input) {
		return EOF
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
	l.pos += l.width

	return
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) forget() {
	l.pos = l.start
}

func (l *lexer) emit(t tokenType) {
	l.tokens <- token{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) accept(valid string) (v bool) {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()

	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

func (l *lexer) errorf(format string, args ...interface{}) lexFn {
	l.tokens <- token{ILLEGAL, l.start, fmt.Sprintf(format, args...)}
	return nil
}

func (l *lexer) shutdown() {
	close(l.tokens)
}

func (l *lexer) tokenChan() <-chan token {
	return l.tokens
}

func (l *lexer) run() {
	for l.state = lexText; l.state != nil; {
		l.state = l.state(l)
	}

	l.shutdown()
}

type query struct {
}

type parser struct {
	*lexer
	state       parseFn
	buf         []token
	bufSize     int
	parsedQuery query
}

func (p *parser) run() {
	for p.state = parseSQL; p.state != nil; {
		p.state = p.state(p)
	}
}

func (p *parser) fill() {
	n := p.bufSize - len(p.buf)
	for i := 0; i < n; i++ {
		p.buf = append(p.buf, <-p.tokenChan())
	}
}

func (p *parser) peek() (t token) {
	p.fill()
	if len(p.buf) > 0 {
		t = p.buf[0]
	}

	return
}

// buf is an array acts as a FIFO queue
func (p *parser) next() (t token) {
	p.fill()
	if len(p.buf) > 0 {
		t = p.buf[0]
		p.buf = p.buf[1:]
	}
	p.fill()

	return
}

func newParser(l *lexer) *parser {
	return &parser{
		lexer:   l,
		bufSize: 3,
	}
}

func main() {
	wg := sync.WaitGroup{}
	l := newLexer(`SELECT t1.c1, t1.c2, t2.c1 FROM` + " `table1` " + `t1 INNER JOIN table2
	t2 ON t1.t2_id = t2.id WHERE id = 1 AND name = 'abc' AND age >= 123`)
	p := newParser(l)
	wg.Add(1)
	go func() {
		p.run()
		wg.Done()
	}()

	go l.run()
	wg.Wait()
}

func newLexer(sql string) (l *lexer) {
	if !strings.HasSuffix(sql, ";") {
		sql = sql + ";"
	}

	return &lexer{
		start:  Pos(0),
		pos:    Pos(0),
		input:  sql,
		tokens: make(chan token, 3),
	}
}

// Scan expressions
// 1. * -> STAR
func lexText(l *lexer) (fn lexFn) {
	omitSpaces(l)
	// reaches the EOF
	if l.accept(";") {
		return nil
	}

	l.acceptRun(_Keywords)
	if isKeyword(l.input[l.start:l.pos]) {
		l.emit(KEYWORD)
		return lexText
	}
	l.forget()

	if l.accept(_Star) {
		l.emit(IDENT)
		return lexText
	}

	// identifier
	if l.peek() == '`' {
		return lexIdentLeftQuote
	}
	var isQuoted, isOperator, isNumber bool
	isQuoted = l.accept(_Quote)
	if isQuoted {
		l.backup()
	}

	isOperator = l.accept(_Operator)
	if isOperator {
		l.backup()
	}

	isNumber = l.accept(_NumberRunes)
	if isNumber {
		l.backup()
	}

	if !isQuoted && !isOperator && !isNumber {
		return lexIdent
	}

	if isOperator {
		return lexOperator
	}

	// after a valid operator, there should be a valid op value
	if isQuoted || isNumber {
		return lexOpValue
	}

	return l.errorf("Illegal expression `%s`, start:pos => %d:%d", l.input[l.start:], l.start, l.pos)
}

func lexIdentLeftQuote(l *lexer) lexFn {
	omitSpaces(l)
	l.acceptRun(_Ident + _RawQuote)

	if int(l.pos) > int(l.start) {
		l.emit(IDENT)
		return lexText
	}

	return nil
}

// identifiers without raw quotes
func lexIdent(l *lexer) lexFn {
	omitSpaces(l)
	l.accept(_RawQuote)
	l.acceptRun(_Ident)
	if l.accept(_MeaninglessRunes) {
		l.backup()
		l.emit(IDENT)
		return lexText
	}

	l.emit(IDENT)
	if l.accept(".") {
		l.emit(DOT)
		return lexIdent
	}

	if l.accept(",") {
		l.ignore()
		return lexIdent
	}

	return l.errorf("Illegal identifier `%s`, start:pos => %d:%d", l.input[l.start:], l.start, l.pos)
}

func lexOperator(l *lexer) lexFn {
	// operator
	l.acceptRun(_Operator)
	if int(l.pos) > int(l.start) {
		l.emit(OPERATOR)
		return lexText
	}

	return nil
}

// scan numbers or quoted values
func lexOpValue(l *lexer) lexFn {
	omitSpaces(l)
	// handle quoted values
	if l.peek() == '\'' {
		return lexOpQuoted
	}

	return lexOpNumber
}

// scan identifier start with ', and ensure it's closed by '
func lexOpQuoted(l *lexer) lexFn {
	omitSpaces(l)

	l.acceptRun(_OpValueRunes)
	l.emit(OPV_QUOTED)

	return lexText
}

func lexOpNumber(l *lexer) lexFn {
	omitSpaces(l)
	// handler numbers, decimals
	// it must reach EOF or a space
	l.acceptRun(_NumberRunes)

	if int(l.pos) > int(l.start) {
		l.emit(OPV_NUMBER)
		return lexText
	}

	return l.errorf("Illegal number `%s`, start:pos => %d:%d", l.input[l.start:], l.start, l.pos)
}

func parseSQL(p *parser) parseFn {
	if t := p.next(); t.typ == KEYWORD {
		switch strings.ToUpper(t.text) {
		case "SELECT":
			return parseSelColumns
		case "FROM":
			return parseFrom
		}
	}

	return nil
}

func parseSelColumns(p *parser) parseFn {
	return parseSQL
}

func parseFrom(p *parser) parseFn {
	return nil
}

// SELECT INSERT UPDATE DELETE FROM WHERE
func isKeyword(in string) bool {
	switch strings.ToUpper(in) {
	case "SELECT", "INSERT", "UPDATE", "DELETE":
		return true
	case "FROM", "WHERE", "AND", "OR", "IS", "NOT", "IN":
		return true
	case "INNER", "JOIN":
		return true
	}

	return false
}

func isAlphaBeta(r rune) bool {
	downcase := (r >= 'a') && (r <= 'z')
	upcase := (r >= 'A') && (r <= 'Z')
	return downcase || upcase
}

func omitSpaces(l *lexer) {
	l.acceptRun(_MeaninglessRunes)
	l.ignore()
}
