package main

import (
	"bufio"
	"bytes"
	"container/list"
	"io"
)

type TokenType uint

const (
	_ TokenType = iota + 127

	ttEOF

	ttLeftParen
	ttRightParen
	ttLeftBrace
	ttRightBrace
	ttLeftBracket
	ttRightBracket
	ttComma
	ttSemicolon
	ttColon

	// arithmetic
	ttAssign
	ttAddition
	ttSubstraction
	ttMultiply
	ttDivision

	// comparision
	ttGreaterThan
	ttGreaterThanOrEqual
	ttEqual
	ttNotEqual
	ttLessThan
	ttLessThanOrEqual

	// Logical
	ttNot
	ttAndAnd
	ttOrOr

	// Literals
	ttNil
	ttString
	ttNumber
	ttBoolean
	ttIdentifier

	// Keywords
	ttLet
	ttFunction
	ttReturn
	ttWhile
	ttBreak
	ttIf
	ttElse
)

var keywords map[string]TokenType

func init() {
	keywords = make(map[string]TokenType)
	keywords["let"] = ttLet
	keywords["function"] = ttFunction
	keywords["return"] = ttReturn
	keywords["while"] = ttWhile
	keywords["break"] = ttBreak
	keywords["if"] = ttIf
	keywords["else"] = ttElse
	keywords["nil"] = ttNil
	keywords["true"] = ttBoolean
	keywords["false"] = ttBoolean
}

type Token struct {
	typ  TokenType
	str  string
	num  int
	line int
}

type Tokenizer struct {
	input  *bufio.Reader
	buf    *list.List
	frames []*list.List
	line   int
}

func NewTokenizer(input io.Reader) *Tokenizer {
	return &Tokenizer{
		input: bufio.NewReader(input),
		buf:   list.New(),
		line:  1,
	}
}

func (t *Tokenizer) Next() (token Token) {
	// use frame
	defer func() {
		if len(t.frames) > 0 {
			frame := t.frames[len(t.frames)-1]
			frame.PushBack(token)
		}
	}()

	// use inner buffer
	if t.buf.Len() > 0 {
		tk := t.buf.Front()
		t.buf.Remove(tk)
		token = tk.Value.(Token)
		return
	}

	// use new
	token = t.next()
	return
}

func (t *Tokenizer) Undo(token Token) {
	t.buf.PushFront(token)
	if len(t.frames) > 0 {
		last := t.frames[len(t.frames)-1]
		if last.Len() == 0 {
			panic("cannot undo")
		}
		last.Remove(last.Back())
	}
}

func (t *Tokenizer) Peek() Token {
	token := t.Next()
	t.Undo(token)
	return token
}

func (t *Tokenizer) PushFrame() {
	t.frames = append(t.frames, list.New())
}

func (t *Tokenizer) PopFrame(putBack bool) {
	if len(t.frames) == 0 {
		panic("bad PopFrame call")
	}
	last := t.frames[len(t.frames)-1]
	t.frames = t.frames[0 : len(t.frames)-1]
	if putBack && last.Len() > 0 {
		t.buf.PushFrontList(last)
	}
}

func (t *Tokenizer) next() (token Token) {
	defer func() {
		token.line = t.line
	}()

	for {
		ch, err := t.input.ReadByte()
		if err == io.EOF {
			return Token{
				typ: ttEOF,
			}
		}

		if ch >= '0' && ch <= '9' {
			t.input.UnreadByte()
			n := t.readNumber()
			t.checkFollow()
			return Token{
				typ: ttNumber,
				num: n,
			}
		} else if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' {
			t.input.UnreadByte()
			name := t.readIdentifier()
			t.checkFollow()
			typ := ttIdentifier
			if t, ok := keywords[name]; ok {
				typ = t
			}
			return Token{
				typ: typ,
				str: name,
			}
		} else if ch == '"' {
			t.input.UnreadByte()
			s := t.readString()
			t.checkFollow()
			return Token{
				typ: ttString,
				str: s,
			}
		}

		switch ch {
		case ' ', '\t', '\r':
			continue
		case '\n':
			t.line++
			continue
		case '(':
			return Token{typ: ttLeftParen}
		case ')':
			return Token{typ: ttRightParen}
		case '[':
			return Token{typ: ttLeftBracket}
		case ']':
			return Token{typ: ttRightBracket}
		case '{':
			return Token{typ: ttLeftBrace}
		case '}':
			return Token{typ: ttRightBrace}
		case ',':
			return Token{typ: ttComma}
		case ':':
			return Token{typ: ttColon}
		case ';':
			return Token{typ: ttSemicolon}
		case '+':
			return Token{typ: ttAddition}
		case '-':
			return Token{typ: ttSubstraction}
		case '*':
			return Token{typ: ttMultiply}
		case '/':
			c := t.read()
			if c == '/' {
				for {
					c = t.read()
					if c == '\n' || c == 0 {
						if c == '\n' {
							t.line++
						}
						break
					}
				}
				continue
			} else {
				t.unread()
				return Token{typ: ttDivision}
			}
		case '=':
			return t.iif('=', ttEqual, ttAssign)
		case '>':
			return t.iif('=', ttGreaterThanOrEqual, ttGreaterThan)
		case '<':
			return t.iif('=', ttLessThanOrEqual, ttLessThan)

		}

		panic("unhandled character")
	}
}

func (t *Tokenizer) read() byte {
	ch, _ := t.input.ReadByte()
	return ch
}

func (t *Tokenizer) unread() {
	t.input.UnreadByte()
}

func (t *Tokenizer) checkFollow() {
	ch, err := t.input.ReadByte()
	if err != io.EOF {
		t.input.UnreadByte()
	}
	if ch >= '0' && ch <= '9' ||
		ch >= 'a' && ch <= 'z' ||
		ch >= 'A' && ch <= 'Z' ||
		ch == '"' {
		panic("unexpected follow character")
	}
}

func (t *Tokenizer) iif(ch byte, tt1 TokenType, tt2 TokenType) Token {
	c := t.read()
	if c == ch {
		return Token{typ: tt1}
	}
	if c != 0 {
		t.unread()
	}
	return Token{typ: tt2}
}

func (t *Tokenizer) readString() string {
	buf := bytes.NewBuffer(nil)
	t.read() // eat '"'
	for {
		ch := t.read()
		if ch == '"' {
			break
		}
		buf.WriteByte(ch)
	}
	return buf.String()
}

func (t *Tokenizer) readNumber() int {
	num := 0
	for {
		ch := t.read()
		if ch >= '0' && ch <= '9' {
			num = num*10 + (int(ch) - '0')
		} else {
			if ch != 0 {
				t.unread()
			}
			break
		}
	}
	return num
}

func (t *Tokenizer) readIdentifier() string {
	buf := bytes.NewBuffer(nil)
	for {
		ch := t.read()
		if ch >= 'a' && ch <= 'z' ||
			ch >= 'A' && ch <= 'Z' ||
			ch >= '0' && ch <= '9' {
			buf.WriteByte(ch)
		} else {
			if ch != 0 {
				t.unread()
			}
			break
		}
	}
	return buf.String()
}
