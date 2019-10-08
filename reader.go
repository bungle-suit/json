//go:generate stringer -type=TokenType

package json

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode/utf8"
)

type readerStatus int

const (
	statusBof readerStatus = iota
	statusTopValue
	statusBeginArray
	statusArrayValue
	statusBeginObject
	statusObjectName
	statusObjectValue
	statusComma
	statusColon
	statusEOF
)

func errInvalidJSONFormat() error {
	return fmt.Errorf("[%s] Invalid json format", tag)
}

// Read json string as a sequence of Json Token. After create Reader struct,
// must call Init() method.
type Reader struct {
	Buf []byte

	// Current parsed trunk start position
	Start int

	// Current parsed trunk end position, Buf[Start:End] is the current parsed trunk
	End int

	// Last parsed string/property name.
	Str []byte

	stack []readerStatus

	// current state
	state func(*Reader, TokenType) (TokenType, error)

	// last token type, used by Undo
	lastToken TokenType

	// true after calling Undo()
	undo bool
}

// Json Token type
type TokenType int

const (
	ttInvalid TokenType = iota
	ttComma
	ttColon
	NULL
	BOOL
	// If the token is number, the number value not parsed, caller should parse
	// itself from Buf[Start:End]. Why Reader not parse number as double? Because:
	//
	//  1. Caller may parse number in different way, such as parse it as decimal,
	//		 parse to double, then decimal will lose precision.
	//  2. Performance, strconv.ParseFloat() is expensive, caller can use cheaper
	//     parse function such as ParseInt() or just treat it as string because
	//     she knows the underlay type exactly.
	//
	// And also means, the number format maybe invalid even the Reader report it
	// as a valid NUMBER, because Reader use a very simple way to parse number
	// string. Caller should process the error, such as check the err result of
	// ParseFloat().
	NUMBER
	// If the token is string or property name, Buf[Start:End] contains the raw
	// json string trunk include `"` quote. The parsed value stores in Buf.Str field.
	STRING
	BEGIN_OBJECT
	PROPERTY_NAME // Json object name
	END_OBJECT
	BEGIN_ARRAY
	END_ARRAY
	EOF
)

// NewReader create a json reader from buf
func NewReader(json []byte) *Reader {
	r := &Reader{}
	r.Init(json)
	return r
}

// Init Reader struct
func (r *Reader) Init(json []byte) {
	r.Buf = json
	r.stack = r.stack[0:0]
	r.push(statusBof)
	r.state = (*Reader).topLevelState
	r.Start, r.End = 0, 0
}

// push new status to stack
func (r *Reader) push(status readerStatus) {
	r.stack = append(r.stack, status)
}

func (r *Reader) pop() {
	r.stack = r.stack[0 : len(r.stack)-1]
	switch r.current() {
	case statusTopValue:
		r.state = (*Reader).topLevelState
	case statusArrayValue:
		r.state = (*Reader).arrayState
	case statusObjectValue:
		r.state = (*Reader).objectState
	}
}

//return current stack top
func (r *Reader) current() readerStatus {
	return r.stack[len(r.stack)-1]
}

// Change current stack top
func (r *Reader) changeCurrent(status readerStatus) {
	r.stack[len(r.stack)-1] = status
}

func (r *Reader) topLevelState(tt TokenType) (TokenType, error) {
	switch tt {
	case NULL, BOOL, NUMBER, STRING:
		if r.current() != statusBof {
			return ttInvalid, errInvalidJSONFormat()
		}
		r.changeCurrent(statusTopValue)
	case EOF:
		if r.current() != statusTopValue || len(r.stack) != 1 {
			return ttInvalid, errInvalidJSONFormat()
		}
		r.changeCurrent(statusEOF)
	case BEGIN_ARRAY:
		r.changeCurrent(statusTopValue)
		r.state = (*Reader).arrayState
		r.push(statusBeginArray)
	case BEGIN_OBJECT:
		r.changeCurrent(statusTopValue)
		r.state = (*Reader).objectState
		r.push(statusBeginObject)
	default:
		return ttInvalid, errInvalidJSONFormat()
	}
	return tt, nil
}

func (r *Reader) arrayState(tt TokenType) (TokenType, error) {
	switch tt {
	case END_ARRAY:
		if r.current() != statusBeginArray && r.current() != statusArrayValue {
			return ttInvalid, errInvalidJSONFormat()
		}
		r.pop()
	case ttComma:
		if r.current() != statusArrayValue {
			return ttInvalid, errInvalidJSONFormat()
		}
		r.changeCurrent(statusComma)
	case NULL, BOOL, NUMBER, STRING:
		if r.current() != statusBeginArray && r.current() != statusComma {
			return ttInvalid, errInvalidJSONFormat()
		}
		r.changeCurrent(statusArrayValue)
	case BEGIN_ARRAY:
		r.changeCurrent(statusArrayValue)
		r.push(statusBeginArray)
		r.state = (*Reader).arrayState
	case BEGIN_OBJECT:
		r.changeCurrent(statusArrayValue)
		r.push(statusBeginObject)
		r.state = (*Reader).objectState
	default:
		return ttInvalid, errInvalidJSONFormat()
	}
	return tt, nil
}

func (r *Reader) objectState(tt TokenType) (TokenType, error) {
	switch tt {
	case END_OBJECT:
		if r.current() != statusBeginObject && r.current() != statusObjectValue {
			return ttInvalid, errInvalidJSONFormat()
		}
		r.pop()
	case STRING:
		switch r.current() {
		case statusBeginObject, statusComma:
			r.changeCurrent(statusObjectName)
			return PROPERTY_NAME, nil
		}
		fallthrough
	case NUMBER, BOOL, NULL:
		if r.current() != statusColon {
			return ttInvalid, errInvalidJSONFormat()
		}
		r.changeCurrent(statusObjectValue)
	case ttColon:
		if r.current() != statusObjectName {
			return ttInvalid, errInvalidJSONFormat()
		}
		r.changeCurrent(statusColon)
	case ttComma:
		if r.current() != statusObjectValue {
			return ttInvalid, errInvalidJSONFormat()
		}
		r.changeCurrent(statusComma)
	case BEGIN_ARRAY:
		if r.current() != statusColon {
			return ttInvalid, errInvalidJSONFormat()
		}
		r.changeCurrent(statusObjectValue)
		r.state = (*Reader).arrayState
		r.push(statusBeginArray)
	case BEGIN_OBJECT:
		if r.current() != statusColon {
			return ttInvalid, errInvalidJSONFormat()
		}
		r.changeCurrent(statusObjectValue)
		r.state = (*Reader).objectState
		r.push(statusBeginObject)
	default:
		return ttInvalid, errInvalidJSONFormat()
	}
	return tt, nil
}

// Parse next json token, return token type. If json string is invalid,
// Next() return non-nil error.
func (r *Reader) Next() (tt TokenType, err error) {
	if r.undo {
		tt = r.lastToken
		r.undo = false
		return
	}

	tt, err = r.doNext()
	r.lastToken = tt
	return
}

// If next token is not tt, returns non-nil error
func (r *Reader) Expect(tt TokenType) error {
	if t, err := r.Next(); err != nil {
		return err
	} else if t != tt {
		return fmt.Errorf("[%s] Expect %s token, but got %s", tag, tt, t)
	}
	return nil
}

// If next token is not PROPERTY_NAME, or its name not expected, return non-nil error
func (r *Reader) ExpectName(name string) error {
	if err := r.Expect(PROPERTY_NAME); err != nil {
		return err
	}
	if bytes.Equal([]byte(name), r.Str) {
		return nil
	}
	return fmt.Errorf("[%s] Expect next is property name '%s', but got '%s'", tag, name, string(r.Str))
}

// ReadNumber return next float value, return non-nil error
// if next token not number.
func (r *Reader) ReadNumber() (float64, error) {
	if err := r.Expect(NUMBER); err != nil {
		return 0, err
	}

	return strconv.ParseFloat(string(r.Buf[r.Start:r.End]), 64)
}

// ReadString return next string value, return non-nil error
// if next token not string.
func (r *Reader) ReadString() (string, error) {
	if err := r.Expect(STRING); err != nil {
		return "", err
	}

	return string(r.Str), nil
}

func (r *Reader) doNext() (tt TokenType, err error) {
	r.Start = r.End
	for i := r.Start; i < len(r.Buf); i++ {
		switch r.Buf[i] {
		case ' ', '\t', '\n', '\r':
			continue
		case 'n':
			r.Start = i
			return r.filterState(r.parseNull())
		case 't':
			r.Start = i
			return r.filterState(r.parseTrue())
		case 'f':
			r.Start = i
			return r.filterState(r.parseFalse())
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			r.Start = i
			return r.filterState(r.parseNumber())
		case '"':
			r.Start = i
			return r.filterState(r.parseString())
		case '[':
			r.Start, r.End = i, i+1
			return r.state(r, BEGIN_ARRAY)
		case ']':
			r.Start, r.End = i, i+1
			return r.state(r, END_ARRAY)
		case '{':
			r.Start, r.End = i, i+1
			return r.state(r, BEGIN_OBJECT)
		case '}':
			r.Start, r.End = i, i+1
			return r.state(r, END_OBJECT)
		case ',':
			r.Start, r.End = i, i+1
			if tt, err = r.state(r, ttComma); err != nil {
				return
			}
			continue
		case ':':
			r.Start, r.End = i, i+1
			if tt, err = r.state(r, ttColon); err != nil {
				return
			}
			continue
		default:
			return ttInvalid, errInvalidJSONFormat()
		}
	}
	return r.state(r, EOF)
}

func (r *Reader) filterState(tt TokenType, err error) (TokenType, error) {
	if err != nil {
		return tt, err
	}
	return r.state(r, tt)
}

// Undo last Next() call.
func (r *Reader) Undo() {
	r.undo = true
}

// start position is '"'
func (r *Reader) parseString() (TokenType, error) {
	r.Str = r.Str[0:0]
	for i := r.Start + 1; i < len(r.Buf); i++ {
		switch r.Buf[i] {
		case '"':
			r.End = i + 1
			return STRING, nil
		case '\\':
			if len(r.Buf)-(i+1) <= 0 {
				return ttInvalid, errInvalidJSONFormat()
			}
			if r.Buf[i+1] == 'u' {
				ch, err := convertUnicodeChar(r.Buf, i+1)
				if err != nil {
					return ttInvalid, err
				}
				r.Str = appendRune(r.Str, ch)
				i += 5
			} else {
				r.Str = append(r.Str, convertEscapedChar(r.Buf[i+1]))
				i++
			}
		default:
			r.Str = append(r.Str, r.Buf[i])
		}
	}
	return ttInvalid, errInvalidJSONFormat()
}

func (r *Reader) parseNumber() (TokenType, error) {
	var dotCount, eCount, i int
outFor:
	for i = r.Start + 1; i < len(r.Buf); i++ {
		switch r.Buf[i] {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '+', '-':
		case '.':
			dotCount++
			if dotCount == 2 {
				return ttInvalid, errInvalidJSONFormat()
			}
		case 'e', 'E':
			eCount++
			if eCount == 2 {
				return ttInvalid, errInvalidJSONFormat()
			}
		default:
			break outFor
		}
	}
	r.End = i
	return NUMBER, nil
}

func (r *Reader) parseTrue() (TokenType, error) {
	if err := r.requireAtLeast(4); err != nil {
		return ttInvalid, err
	}

	buf, s := r.Buf, r.Start
	if buf[s+1] != 'r' || buf[s+2] != 'u' || buf[s+3] != 'e' {
		return ttInvalid, errInvalidJSONFormat()
	}
	if err := r.requireEndToken(); err != nil {
		return ttInvalid, err
	}
	return BOOL, nil
}

func (r *Reader) parseFalse() (TokenType, error) {
	if err := r.requireAtLeast(5); err != nil {
		return ttInvalid, err
	}
	buf, s := r.Buf, r.Start
	if buf[s+1] != 'a' || buf[s+2] != 'l' || buf[s+3] != 's' || buf[s+4] != 'e' {
		return ttInvalid, errInvalidJSONFormat()
	}
	if err := r.requireEndToken(); err != nil {
		return ttInvalid, err
	}
	return BOOL, nil
}

func (r *Reader) parseNull() (TokenType, error) {
	if err := r.requireAtLeast(4); err != nil {
		return ttInvalid, err
	}

	buf, s := r.Buf, r.Start
	if buf[s+1] != 'u' || buf[s+2] != 'l' || buf[s+3] != 'l' {
		return ttInvalid, errInvalidJSONFormat()
	}
	if err := r.requireEndToken(); err != nil {
		return ttInvalid, err
	}
	return NULL, nil
}

// Check does buffer has at least l bytes in buffer
func (r *Reader) requireAtLeast(l int) error {
	r.End = r.Start + l
	if r.End > len(r.Buf) {
		return errInvalidJSONFormat()
	}
	return nil
}

// Check current end position is end token
func (r *Reader) requireEndToken() error {
	if !r.isEndToken() {
		return errInvalidJSONFormat()
	}
	return nil
}

// return true if current end position is token boundary:
//
//  1. eof
//  2. white space
//  3. ,
//  4. :
//  5. ]
//  6. }
func (r *Reader) isEndToken() bool {
	if r.End >= len(r.Buf) {
		return true
	}

	switch r.Buf[r.End] {
	case ' ', '\t', '\r', '\n':
		fallthrough
	case ',':
		fallthrough
	case ':':
		fallthrough
	case ']':
		fallthrough
	case '}':
		return true
	}
	return false
}

func appendRune(buf []byte, r rune) []byte {
	l := utf8.RuneLen(r)
	var b []byte
	if cap(buf)-len(buf) < l {
		b = make([]byte, len(buf)+l, (len(buf)+l)*2)
		copy(b, buf)
	} else {
		b = buf
	}
	utf8.EncodeRune(b[len(buf):len(buf)+l], r)
	return b[0 : len(buf)+l]
}

func convertEscapedChar(ch byte) byte {
	switch ch {
	case 'b':
		return '\b'
	case 'f':
		return '\f'
	case 'r':
		return '\r'
	case 'n':
		return '\n'
	case 't':
		return '\t'
	default:
		return ch
	}
}

func convertUnicodeChar(buf []byte, i int) (rune, error) {
	if len(buf)-i < 5 {
		return 0, errInvalidJSONFormat()
	}

	s := string(buf[i+1 : i+5])
	ch, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, errInvalidJSONFormat()
	}
	return rune(ch), nil
}
