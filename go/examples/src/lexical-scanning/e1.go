package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	EOF = -1
)

const (
	KEYWORD tokenType = iota
	STAR
	IDENT
	ILLEGAL
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
	l.tokens <- token{typ: t, text: l.input[l.start:l.pos]}
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

func (l *lexer) nextItem() (t token) {
	t = <-l.tokens
	l.lastPos = l.pos

	return
}

func (l *lexer) run() {
	for l.state = lexEntrypoint; l.state != nil; {
		l.state = l.state(l)
	}

	l.shutdown()
}

func main() {
	l := newLexer("SELECT * FROM table1")
	l.run()
}

func newLexer(sql string) (l *lexer) {
	return &lexer{
		start:  Pos(0),
		pos:    Pos(0),
		input:  sql,
		tokens: make(chan token),
	}
}

func lexEntrypoint(l *lexer) (fn stateFn) {
	return
}
