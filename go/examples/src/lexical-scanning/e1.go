package main

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
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
	OPERATOR
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
	l := newLexer("SELECT * FROM table1 WHERE id = 1;")
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
	l.acceptRun(" ")
	l.ignore()

	l.acceptRun("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ*")
	if isKeyword(l.input[l.start:l.pos]) {
		l.emit(KEYWORD)
		return lexText
	}

	// identifier
	l.acceptRun("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ*")
	if int(l.pos) > int(l.start) {
		l.emit(IDENT)
		return lexText
	}

	// operator
	l.acceptRun("=><")
	if int(l.pos) > int(l.start) {
		l.emit(OPERATOR)
		return lexText
	}

	return l.errorf("Illegal expression `%s`, start:pos => %d:%d", l.input[l.start:], l.start, l.pos)
}

// SELECT INSERT UPDATE DELETE FROM WHERE
func isKeyword(in string) bool {
	switch strings.ToUpper(in) {
	case "SELECT", "INSERT", "UPDATE", "DELETE", "FROM", "WHERE":
		return true
	}

	return false
}

func isAlphaBeta(r rune) bool {
	downcase := (r >= 'a') && (r <= 'z')
	upcase := (r >= 'A') && (r <= 'Z')
	return downcase || upcase
}
