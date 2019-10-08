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
	return tokenRec{tt: String, trunk: trunk, str: str}
}

func newNameToken(trunk string, name string) tokenRec {
	return tokenRec{tt: PropertyName, trunk: trunk, str: name}
}

func newNumberToken(trunk string) tokenRec {
	return tokenRec{tt: Number, trunk: trunk}
}

func doReadNext(t *testing.T, r *Reader, exp tokenRec) {
	tt, err := r.Next()
	assert.NoError(t, err)
	assert.Equal(t, exp.tt, tt)
	assert.Equal(t, exp.trunk, string(r.Buf[r.Start:r.End]))
	if exp.tt == String || exp.tt == PropertyName {
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
	doReadTest(t, `null`, newToken(Null, `null`))
	doReadTest(t, ` null`, newToken(Null, `null`))
	doReadTest(t, " \t\r\nnull\n", newToken(Null, `null`))
}

func TestReadBool(t *testing.T) {
	doReadTest(t, `true`, newToken(Bool, `true`))
	doReadTest(t, `false`, newToken(Bool, `false`))
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
	assert.NoError(t, r.Expect(EOF))
}

func TestUndo(t *testing.T) {
	var r Reader
	r.Init([]byte(`null`))

	doReadNext(t, &r, newToken(Null, "null"))

	r.Undo()
	doReadNext(t, &r, newToken(Null, "null"))

	r.Undo()
	doReadNext(t, &r, newToken(Null, "null"))

	doReadNext(t, &r, newToken(EOF, ""))
}

func TestReadStringToken(t *testing.T) {
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

func TestReadString(t *testing.T) {
	r := NewReader([]byte(`"a\"bc"`))
	v, err := r.ReadString()
	assert.NoError(t, err)
	assert.Equal(t, `a"bc`, v)
	assert.NoError(t, r.Expect(EOF))
}

func TestReadArray(t *testing.T) {
	doReadTest(t, `[]`, newToken(BeginArray, `[`), newToken(EndArray, `]`))
	doReadTest(t, `[1]`, newToken(BeginArray, `[`), newNumberToken(`1`), newToken(EndArray, `]`))
	doReadTest(t, `[1,true,[],{"":1},false,"",[null]]`, newToken(BeginArray, `[`),
		newNumberToken(`1`),
		newToken(Bool, `true`),
		newToken(BeginArray, `[`), newToken(EndArray, `]`),
		newToken(BeginObject, `{`), newNameToken(`""`, ""), newNumberToken(`1`), newToken(EndObject, `}`),
		newToken(Bool, `false`),
		newStringToken(`""`, ""),
		newToken(BeginArray, `[`), newToken(Null, `null`), newToken(EndArray, `]`),
		newToken(EndArray, `]`))
}

func TestReadObject(t *testing.T) {
	doReadTest(t, `{}`, newToken(BeginObject, `{`), newToken(EndObject, `}`))
	doReadTest(t, `{"a":2}`, newToken(BeginObject, `{`),
		newNameToken(`"a"`, "a"), newNumberToken("2"),
		newToken(EndObject, `}`))
	doReadTest(t, `{"a":true,"b":null,"c":"","d":[],"e":{}}`, newToken(BeginObject, "{"),
		newNameToken(`"a"`, "a"), newToken(Bool, `true`),
		newNameToken(`"b"`, "b"), newToken(Null, `null`),
		newNameToken(`"c"`, "c"), newStringToken(`""`, ""),
		newNameToken(`"d"`, "d"), newToken(BeginArray, `[`), newToken(EndArray, `]`),
		newNameToken(`"e"`, "e"), newToken(BeginObject, `{`), newToken(EndObject, `}`),
		newToken(EndObject, `}`))
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
	assert.Error(t, r.Expect(BeginArray))
}

func TestExpectedNextName(t *testing.T) {
	var r Reader
	r.Init([]byte("1"))
	assert.Error(t, r.ExpectName("a"))

	r.Init([]byte(`{"Foo":2,"Bar":3}`))
	_, err := r.Next()
	assert.NoError(t, err)
	assert.Error(t, r.ExpectName("Bar"))
	_, err = r.Next()
	assert.NoError(t, err)
	assert.NoError(t, r.ExpectName("Bar"))
}
