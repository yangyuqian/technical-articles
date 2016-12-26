package main

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	_NumberRunes          = "0123456789"
	_OpValueRunes         = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ'"
	_OpValueRunesNoQuotes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	_Identifier           = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`*._"
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
	OPERATOR   // operator
	OPV_NUMBER // operator value of numbers
	OPV_QUOTED // operator value of quoted text
)

type stateFn func(*lexer) stateFn

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
	state   stateFn
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

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.tokens <- token{ILLEGAL, l.start, fmt.Sprintf(format, args...)}
	return nil
}

func (l *lexer) shutdown() {
	close(l.tokens)
}

func (l *lexer) nextToken() (t token) {
	t = <-l.tokens
	l.lastPos = l.pos

	return
}

func (l *lexer) run() {
	for l.state = lexText; l.state != nil; {
		l.state = l.state(l)
	}

	l.shutdown()
}

func main() {
	wg := sync.WaitGroup{}
	l := newLexer("SELECT * FROM table1 t1 INNER JOIN table2 t2 ON t1.t2_id = t2.id WHERE id = 1 AND name = 'abc' AND age >= 2")
	wg.Add(1)
	go func() {
		for t := l.nextToken(); len(t.text) > 0; t = l.nextToken() {
			fmt.Println(t)
		}
		wg.Done()
	}()

	go l.run()
	wg.Wait()
}

func newLexer(sql string) (l *lexer) {
	return &lexer{
		start:  Pos(0),
		pos:    Pos(0),
		input:  sql,
		tokens: make(chan token),
	}
}

// Scan expressions
// 1. * -> STAR
func lexText(l *lexer) (fn stateFn) {
	omitSpaces(l)

	l.acceptRun(_Keywords)
	if isKeyword(l.input[l.start:l.pos]) {
		l.emit(KEYWORD)
		return lexText
	}

	// identifier, should
	l.acceptRun(_Identifier)
	if int(l.pos) > int(l.start) {
		l.emit(IDENT)
		return lexText
	}

	if l.accept(_Operator) {
		l.backup()
		return lexOperator
	}

	// is a valid op value
	if l.accept(_OpValueRunes) {
		l.backup()
		return lexOpValue
	}

	return l.errorf("Illegal expression `%s`, start:pos => %d:%d", l.input[l.start:], l.start, l.pos)
}

func lexOperator(l *lexer) stateFn {
	// operator
	l.acceptRun(_Operator)
	if int(l.pos) > int(l.start) {
		l.emit(OPERATOR)
		return lexText
	}

	return nil
}

// scan numbers or quoted values
func lexOpValue(l *lexer) stateFn {
	omitSpaces(l)
	// handle quoted values
	if l.accept("'") {
		return lexOpQuoted
	}

	return lexOpNumber
}

// scan identifier start with ', and ensure it's closed by '
func lexOpQuoted(l *lexer) stateFn {
	omitSpaces(l)

	if l.peek() == '\'' {
		l.ignore()
	}
	l.acceptRun(_OpValueRunesNoQuotes)
	l.emit(OPV_QUOTED)
	// ignore end quote
	if l.accept("'") {
		l.ignore()
	} else {
		return l.errorf("Illegal quoted value`%s`, start:pos => %d:%d", l.input[l.start:], l.start, l.pos)
	}

	return lexText
}

func lexOpNumber(l *lexer) stateFn {
	omitSpaces(l)
	// handler numbers, decimals
	// it must reach EOF or a space
	l.acceptRun(_NumberRunes)
	if r := l.next(); r >= '0' && r <= '9' {
		switch l.next() {
		case EOF, ' ':
			l.emit(OPV_NUMBER)
			return nil
		}
	}
	return l.errorf("Illegal number `%s`, start:pos => %d:%d", l.input[l.start:], l.start, l.pos)
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
	l.acceptRun(" ")
	l.ignore()
}
