package json

import (
	"bufio"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/redforks/errors"
)

const hex = "0123456789ABCDEF"

type blockType int

const (
	none blockType = iota
	array
	object
)

type writerState interface {
	NeedWriteSep() bool
	Type() blockType
	NeedWriteName() bool
}

// Writer write json encoded string.
//
// Writer stores error occurred during write, WriteXXX() functions do not return error. If
// an error stored in Writer, all WriteXXX() functions quit immediately. Must call .Flush()
// periodically to check does error have occurred.
type Writer struct {
	stateStack []writerState

	// use bufio.Writer because it will cache writer error, and do not need check error
	// on every Write().
	*bufio.Writer

	e error
}

// Create a new writer that output json string to a IO Writer.
// Argument `w' must also implement io.ByteWriter interface.
func NewWriter(w io.Writer) *Writer {
	r := &Writer{Writer: bufio.NewWriter(w)}
	r.stateStack = []writerState{&topWriterState{w: r}}
	return r
}

// Flush written content to underlay writer, and report whether an underlay error happened?
func (w *Writer) Flush() error {
	if w.e != nil {
		return w.e
	}
	return w.Writer.Flush()
}

func (w *Writer) WriteNull() {
	if w.e != nil {
		return
	}

	w.writeSep()
	w.writeLiteral([]byte(`null`))
}

func (w *Writer) WriteTrue() {
	if w.e != nil {
		return
	}

	w.writeSep()
	w.writeLiteral([]byte(`true`))
}

func (w *Writer) WriteFalse() {
	if w.e != nil {
		return
	}

	w.writeSep()
	w.writeLiteral([]byte(`false`))
}

func (w *Writer) WriteBool(val bool) {
	if val {
		w.WriteTrue()
	} else {
		w.WriteFalse()
	}
}

func (w *Writer) WriteNumber(num float64) {
	if w.e != nil {
		return
	}

	w.writeSep()
	// must use 'f' format, other format broke jsonrpc datetime format.
	// jsonrpc datetime format expected number is a integer, not accept Exponent.
	w.writeString(strconv.FormatFloat(num, 'f', -1, 64))
}

func (w *Writer) WriteString(s string) {
	if w.e != nil {
		return
	}

	w.writeSep()
	w.writeStringOnly(s)
}

// WriteRaw write directly to the underlay writer.
func (w *Writer) WriteRaw(raw string) {
	if w.e != nil {
		return
	}

	w.writeSep()
	w.writeString(raw)
}

func (w *Writer) BeginArray() {
	if w.e != nil {
		return
	}

	w.writeSep()
	w.writeChar('[')
	w.pushState(new(arrayWriterState))
}

func (w *Writer) EndArray() {
	if w.e != nil {
		return
	}

	w.writeChar(']')
	w.popState(array)
}

func (w *Writer) EmptyArray() {
	if w.e != nil {
		return
	}

	w.writeSep()
	w.writeChar('[')
	w.writeChar(']')
}

func (w *Writer) BeginObject() {
	if w.e != nil {
		return
	}

	w.writeSep()
	w.writeChar('{')
	w.pushState(new(objectWriterState))
}

func (w *Writer) EndObject() {
	if w.e != nil {
		return
	}

	w.writeChar('}')
	w.popState(object)
}

func (w *Writer) WriteName(name string) {
	if w.e != nil {
		return
	}

	state := w.currentState()
	if state.Type() != object {
		w.e = errors.Bug("WriteName() not called in Object context")
		return
	}
	if !state.NeedWriteName() {
		w.e = errors.Bug("Can not call WriteName() in current state")
		return
	}

	if state.NeedWriteSep() {
		w.writeChar(',')
	}
	w.writeStringOnly(name)

	w.writeChar(':')
}

func (w *Writer) EmptyObject() {
	if w.e != nil {
		return
	}

	w.writeSep()
	w.writeChar('{')
	w.writeChar('}')
}

func (w *Writer) writeLiteral(bytes []byte) {
	_, err := w.Write(bytes)
	if err != nil {
		w.e = err
	}
}

func (w *Writer) writeSep() {
	if w.currentState().NeedWriteName() {
		w.e = errors.Bug(`WriteName() expected`)
		return
	}

	if w.currentState().NeedWriteSep() {
		w.writeChar(',')
	}
}

func (w *Writer) currentState() writerState {
	return w.stateStack[len(w.stateStack)-1]
}

func (w *Writer) pushState(state writerState) {
	w.stateStack = append(w.stateStack, state)
}

func (w *Writer) popState(expectedCurrentState blockType) writerState {
	if w.stateStack[len(w.stateStack)-1].Type() != expectedCurrentState {
		w.e = errors.Bug("BeginObject/EndObject, BeginArray/EndArray not paired")
		return w.stateStack[len(w.stateStack)-1]
	}

	result := w.stateStack[len(w.stateStack)-1]
	w.stateStack = w.stateStack[:len(w.stateStack)-1]
	return result
}

func (w *Writer) writeEscapedChar(ch byte) {
	w.writeChar('\\')
	w.writeChar(ch)
}

func (w *Writer) writeEscaped(b byte) {
	switch b {
	case '\\', '"':
		w.writeEscapedChar(b)
	case '\n':
		w.writeEscapedChar('n')
	case '\r':
		w.writeEscapedChar('r')
	case '\b':
		w.writeEscapedChar('b')
	case '\f':
		w.writeEscapedChar('f')
	case '\t':
		w.writeEscapedChar('t')
	default:
		// This encodes bytes < 0x20 except for \n and \r,
		// as well as < and >. The latter are escaped because they
		// can lead to security holes when user-controlled strings
		// are rendered into JSON and served to some browsers.
		w.writeString(`\u00`)
		w.writeChar(hex[b>>4])
		w.writeChar(hex[b&0xF])
	}
}

func (w *Writer) writeStringOnly(s string) {
	w.writeChar('"')

	// copied from encoding/json/encode.go encodeState.string()
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' && b != '&' {
				i++
				continue
			}

			if start < i {
				w.writeString(s[start:i])
			}
			w.writeEscaped(b)
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				w.writeString(s[start:i])
			}
			w.writeString(`\ufffd`)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				w.writeString(s[start:i])
			}
			w.writeString(`\u202`)
			w.writeChar(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}

	if start < len(s) {
		w.writeString(s[start:])
	}

	w.writeChar('"')
}

type topWriterState struct {
	hit bool

	w *Writer
}

func (s *topWriterState) NeedWriteSep() bool {
	if s.hit {
		s.w.e = errors.Bug("Only one value allowed")
		return false
	}

	s.hit = true
	return false
}

func (s *topWriterState) Type() blockType {
	return none
}

func (s *topWriterState) NeedWriteName() bool {
	return false
}

type arrayWriterState struct {
	topWriterState
}

func (s *arrayWriterState) NeedWriteSep() bool {
	if s.hit {
		return true
	}
	s.hit = true
	return false
}

func (s *arrayWriterState) Type() blockType {
	return array
}

type objectWriterState struct {
	hit  bool
	hits int
}

func (s *objectWriterState) NeedWriteName() bool {
	return !s.hit || (s.hits%2) == 1
}

func (s *objectWriterState) NeedWriteSep() bool {
	if s.hit {
		s.hits++
		return (s.hits)%2 == 0
	}
	s.hit = true
	return false
}

func (s *objectWriterState) Type() blockType {
	return object
}

func (w *Writer) writeString(s string) {
	// no need to check bufio.Writer's return value, it will checked in .Flush()
	_, _ = io.WriteString(w, s)
}

func (w *Writer) writeChar(c byte) {
	// no need to check bufio.Writer's return value, it will checked in .Flush()
	_ = w.WriteByte(c)
}
