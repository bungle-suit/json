package json_test

import (
	"bytes"
	"sort"
	"testing"

	. "github.com/redforks/spork/json"
	"github.com/stretchr/testify/assert"
)

type writeFunc func(*Writer)

type Case struct {
	expected string
	f        writeFunc
}

func testInvalidWrites(t *testing.T, errMsg string, f writeFunc) {
	w := NewWriter(&bytes.Buffer{})
	f(w)

	assert.EqualError(t, w.Flush(), errMsg)
}

func test(t *testing.T, expected string, f writeFunc) {
	o := bytes.Buffer{}
	w := NewWriter(&o)
	f(w)
	assert.NoError(t, w.Flush())
	assert.Equal(t, expected, o.String())
}

func runCases(t *testing.T, cases []Case) {
	for _, rec := range cases {
		test(t, rec.expected, rec.f)
	}
}

func writeValue(t *testing.T, w *Writer, val interface{}) {
	switch v := val.(type) {
	case bool:
		w.WriteBool(v)
	case float64:
		w.WriteNumber(v)
	case string:
		w.WriteString(v)
	case map[string]interface{}:
		if len(v) == 0 {
			w.EmptyObject()
		} else {
			w.BeginObject()

			names := make([]string, 0, len(v))
			for k := range v {
				names = append(names, k)
			}
			sort.Strings(names)

			for _, k := range names {
				w.WriteName(k)
				writeValue(t, w, v[k])
			}

			w.EndObject()
		}
	case []interface{}:
		if len(v) == 0 {
			w.EmptyArray()
		} else {
			w.BeginArray()

			for _, val := range v {
				writeValue(t, w, val)
			}

			w.EndArray()
		}
	default:
		assert.Nil(t, v)
		w.WriteNull()
	}
}

func TestWriteNull(t *testing.T) {
	test(t, `null`, (*Writer).WriteNull)
}

func TestWriteBool(t *testing.T) {
	runCases(t, []Case{
		{`true`, (*Writer).WriteTrue},
		{`false`, (*Writer).WriteFalse},
		{`true`, func(w *Writer) { w.WriteBool(true) }},
		{`false`, func(w *Writer) { w.WriteBool(false) }},
	})
}

func TestNumber(t *testing.T) {
	writeNumber := func(num float64) writeFunc {
		return func(w *Writer) {
			w.WriteNumber(num)
		}
	}

	runCases(t, []Case{
		{`0`, writeNumber(0)},
		{`-1`, writeNumber(-1)},
		{`1`, writeNumber(1)},
		{`1.1`, writeNumber(1.1)},
		{"1000000000000000", writeNumber(1E15)},
		{"0.00001", writeNumber(1E-5)},
	})
}

func TestWriteString(t *testing.T) {
	writeString := func(s string) writeFunc {
		return func(w *Writer) {
			w.WriteString(s)
		}
	}

	runCases(t, []Case{
		{`""`, writeString(``)},
		{`" "`, writeString(` `)},
		{`" abc "`, writeString(` abc `)},
		{`"\""`, writeString(`"`)},
		{`"\\"`, writeString(`\`)},
		{`"han汉"`, writeString(`han汉`)},
		{`"\b\f\n\r\t\u0007\u000E\u001E"`, writeString("\b\f\n\r\t\x07\x0e\x1e")},
	})
}

func TestWriteRaw(t *testing.T) {
	writeRaw := func(s string) writeFunc {
		return func(w *Writer) {
			w.WriteRaw(s)
		}
	}

	runCases(t, []Case{
		{"3", writeRaw("3")},
		{"3,4", writeRaw("3,4")},
		{"\"foo\"", writeRaw("\"foo\"")},
	})
}

func TestWriteArray(t *testing.T) {
	writeArray := func(items ...interface{}) writeFunc {
		return func(w *Writer) {
			w.BeginArray()

			for _, item := range items {
				writeValue(t, w, item)
			}

			w.EndArray()
		}
	}

	emptyMap := make(map[string]interface{})
	runCases(t, []Case{
		{`[]`, writeArray()},
		{`[false]`, writeArray(false)},
		{`[false,1]`, writeArray(false, 1.0)},
		{`[false,1,"",null]`, writeArray(false, 1.0, "", nil)},
		{`[0,[null,false],[],""]`, func(w *Writer) {
			w.BeginArray()
			w.WriteNumber(0)
			w.BeginArray()
			w.WriteNull()
			w.WriteFalse()
			w.EndArray()
			w.EmptyArray()
			w.WriteString(``)
			w.EndArray()
		}},
		{`[{},{},{}]`, writeArray(emptyMap, emptyMap, emptyMap)},
	})
}

func TestWriteEmptyArray(t *testing.T) {
	test(t, "[]", (*Writer).EmptyArray)
	test(t, "[]", func(w *Writer) {
		w.BeginArray()
		w.EndArray()
	})
}

func TestWriteObject(t *testing.T) {
	writeObject := func(pairs map[string]interface{}) writeFunc {
		return func(w *Writer) {
			writeValue(t, w, pairs)
		}
	}

	runCases(t, []Case{
		{`{}`, writeObject(nil)},
		{`{"":null}`, writeObject(map[string]interface{}{"": nil})},
		{`{"":null,"a":[{"1":true}]}`, writeObject(map[string]interface{}{"": nil, "a": []interface{}{
			map[string]interface{}{"1": true},
		}})},
	})
}

func TestWriteEmptyObject(t *testing.T) {
	test(t, "{}", (*Writer).EmptyObject)
	test(t, "{}", func(w *Writer) {
		w.BeginObject()
		w.EndObject()
	})
}

func TestWriteOneValue(t *testing.T) {
	testInvalidWrites(t, "Only one value allowed", func(w *Writer) {
		w.WriteNull()
		w.WriteNull()
	})
}

func TestWriteCrossBeginEnd(t *testing.T) {
	testInvalidWrites(t, "BeginObject/EndObject, BeginArray/EndArray not paired", func(w *Writer) {
		w.BeginObject()
		w.EndArray()
	})
}

func TestWriteEndWithoutBegin(t *testing.T) {
	testInvalidWrites(t, `BeginObject/EndObject, BeginArray/EndArray not paired`, (*Writer).EndObject)
	testInvalidWrites(t, `BeginObject/EndObject, BeginArray/EndArray not paired`, (*Writer).EndArray)
	testInvalidWrites(t, `BeginObject/EndObject, BeginArray/EndArray not paired`, func(w *Writer) {
		w.BeginObject()
		w.EndObject()
		w.EndArray()
	})
}

func TestWriteNameNotInObjectContext(t *testing.T) {
	testInvalidWrites(t, `WriteName() not called in Object context`, func(w *Writer) {
		w.WriteName(``)
	})
}

func TestWriteNameName(t *testing.T) {
	testInvalidWrites(t, `Can not call WriteName() in current state`, func(w *Writer) {
		w.BeginObject()
		w.WriteName(``)
		w.WriteName(``)
	})
}

func TestWriteValueWithoutWriteName(t *testing.T) {
	testInvalidWrites(t, `WriteName() expected`, func(w *Writer) {
		w.BeginObject()
		w.WriteTrue()
	})

	testInvalidWrites(t, `WriteName() expected`, func(w *Writer) {
		w.BeginObject()
		w.WriteName(``)
		w.WriteNumber(1.0)
		w.WriteFalse()
	})
}
