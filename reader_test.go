package json_test

import (
	"testing"

	. "github.com/bungle-suit/json"
	"github.com/stretchr/testify/assert"
)

type tokenRec struct {
	tt    TokenType
	trunk string
	str   string
}

func newToken(tt TokenType, trunk string) tokenRec {
	return tokenRec{tt: tt, trunk: trunk}
}

func newStringToken(trunk string, str string) tokenRec {
	return tokenRec{tt: STRING, trunk: trunk, str: str}
}

func newNameToken(trunk string, name string) tokenRec {
	return tokenRec{tt: PROPERTY_NAME, trunk: trunk, str: name}
}

func newNumberToken(trunk string) tokenRec {
	return tokenRec{tt: NUMBER, trunk: trunk}
}

func doReadNext(t *testing.T, r *Reader, exp tokenRec) {
	tt, err := r.Next()
	assert.NoError(t, err)
	assert.Equal(t, exp.tt, tt)
	assert.Equal(t, exp.trunk, string(r.Buf[r.Start:r.End]))
	if exp.tt == STRING || exp.tt == PROPERTY_NAME {
		assert.Equal(t, exp.str, string(r.Str))
	}
}

func doReadTest(t *testing.T, json string, tokens ...tokenRec) {
	var r Reader
	r.Init([]byte(json))

	for _, rec := range tokens {
		doReadNext(t, &r, rec)
	}

	tt, err := r.Next()
	assert.NoError(t, err)
	assert.Equal(t, EOF, tt)
}

func TestReadNull(t *testing.T) {
	doReadTest(t, `null`, newToken(NULL, `null`))
	doReadTest(t, ` null`, newToken(NULL, `null`))
	doReadTest(t, " \t\r\nnull\n", newToken(NULL, `null`))
}

func TestReadBool(t *testing.T) {
	doReadTest(t, `true`, newToken(BOOL, `true`))
	doReadTest(t, `false`, newToken(BOOL, `false`))
}

func TestReadNumberToken(t *testing.T) {
	for _, json := range []string{
		`0`, `-0`, `234`, `-34`, `12.34`, `1E10`, `1.10`, `1e+10`, `13.4e-10`,
	} {
		doReadTest(t, json, newNumberToken(json))
	}
}

func TestReadNumber(t *testing.T) {
	buf := []byte("1")
	r := NewReader(buf)

	v, err := r.ReadNumber()
	assert.NoError(t, err)
	assert.Equal(t, float64(1), v)
}

func TestUndo(t *testing.T) {
	var r Reader
	r.Init([]byte(`null`))

	doReadNext(t, &r, newToken(NULL, "null"))

	r.Undo()
	doReadNext(t, &r, newToken(NULL, "null"))

	r.Undo()
	doReadNext(t, &r, newToken(NULL, "null"))

	doReadNext(t, &r, newToken(EOF, ""))
}

func TestReadString(t *testing.T) {
	for json, val := range map[string]string{
		`""`: "", `"abc"`: "abc", `"\""`: `"`,
		`"\\\b\f\n\r\t"`: "\\\b\f\n\r\t",
		`"\/"`:           "/", `"\u4e00a"`: "一a",
		`"\u4e00\u4e00"`: "一一", `"\u0013\u4e00"`: "\x13一",
		`"一一"`: "一一",
	} {
		doReadTest(t, json, newStringToken(json, val))
	}
}

func TestReadArray(t *testing.T) {
	doReadTest(t, `[]`, newToken(BEGIN_ARRAY, `[`), newToken(END_ARRAY, `]`))
	doReadTest(t, `[1]`, newToken(BEGIN_ARRAY, `[`), newNumberToken(`1`), newToken(END_ARRAY, `]`))
	doReadTest(t, `[1,true,[],{"":1},false,"",[null]]`, newToken(BEGIN_ARRAY, `[`),
		newNumberToken(`1`),
		newToken(BOOL, `true`),
		newToken(BEGIN_ARRAY, `[`), newToken(END_ARRAY, `]`),
		newToken(BEGIN_OBJECT, `{`), newNameToken(`""`, ""), newNumberToken(`1`), newToken(END_OBJECT, `}`),
		newToken(BOOL, `false`),
		newStringToken(`""`, ""),
		newToken(BEGIN_ARRAY, `[`), newToken(NULL, `null`), newToken(END_ARRAY, `]`),
		newToken(END_ARRAY, `]`))
}

func TestReadObject(t *testing.T) {
	doReadTest(t, `{}`, newToken(BEGIN_OBJECT, `{`), newToken(END_OBJECT, `}`))
	doReadTest(t, `{"a":2}`, newToken(BEGIN_OBJECT, `{`),
		newNameToken(`"a"`, "a"), newNumberToken("2"),
		newToken(END_OBJECT, `}`))
	doReadTest(t, `{"a":true,"b":null,"c":"","d":[],"e":{}}`, newToken(BEGIN_OBJECT, "{"),
		newNameToken(`"a"`, "a"), newToken(BOOL, `true`),
		newNameToken(`"b"`, "b"), newToken(NULL, `null`),
		newNameToken(`"c"`, "c"), newStringToken(`""`, ""),
		newNameToken(`"d"`, "d"), newToken(BEGIN_ARRAY, `[`), newToken(END_ARRAY, `]`),
		newNameToken(`"e"`, "e"), newToken(BEGIN_OBJECT, `{`), newToken(END_OBJECT, `}`),
		newToken(END_OBJECT, `}`))
}

func TestReadBadJson(t *testing.T) {
	for _, json := range []string{
		``, `n`, `nu`, `nuLl`, `null1`, `true1`, `false1`, `1n`, `+1`,
		`tRue`, `faLse`,
		`1.2.3`, `-2EE3`, `:`, `,`, `}`, `]`,
		`"abc`, `''`, `"\u"`, `"\u123`, `"\u123x`, `"\`,
		"1 2", "null null", `1 ""`, `true false`, `false true`,
		`[`, `[,`, `[1 2]`, `[1,]`, `[,1]`, `[[],1`,
		`{`, `{:}`, `{,}`, `{1:2}`, `{1 2}`, `{"":3 4}`, `{""}`, `{[]}`, `{{}}`, `{]`, `[}`, `[{]}`,
	} {
		var r Reader
		r.Init([]byte(json))
		for {
			tt, err := r.Next()
			if err != nil {
				break
			}
			if tt == EOF {
				assert.FailNowf(t, "Expected bad json %#v, but no parse error", json)
			}
		}
	}
}

func TestReadExpectedNext(t *testing.T) {
	var r Reader
	r.Init([]byte("1"))
	assert.Equal(t, GenericFormatError(), r.Expect(BEGIN_ARRAY))
}

func TestExpectedNextName(t *testing.T) {
	var r Reader
	r.Init([]byte("1"))
	assert.Equal(t, GenericFormatError(), r.ExpectName("a"))

	r.Init([]byte(`{"Foo":2,"Bar":3}`))
	_, err := r.Next()
	assert.NoError(t, err)
	assert.Equal(t, GenericFormatError(), r.ExpectName("Bar"))
	_, err = r.Next()
	assert.NoError(t, err)
	assert.NoError(t, r.ExpectName("Bar"))
}
