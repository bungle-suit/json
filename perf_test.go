package json_test

import (
	"bytes"
	sysjs "encoding/json"
	"io"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/redforks/spork/json"
	"github.com/stretchr/testify/assert"
)

func BenchmarkParse(b *testing.B) {
	var json []byte
	_, f, _, ok := runtime.Caller(0)
	assert.True(b, ok, "Can not get current benchmark source file directory")

	file := filepath.Join(filepath.Dir(f), "ec2-2016-11-15.normal.json")
	json, err := ioutil.ReadFile(file)
	assert.NoError(b, err)

	b.Run("Standard lib", func(b *testing.B) {
		decoder := sysjs.NewDecoder(bytes.NewReader(json))
		var v = make(map[string]interface{})
		if err := decoder.Decode(&v); err != nil {
			panic(err)
		}
	})

	b.Run("Standard lib used as stream", func(b *testing.B) {
		decoder := sysjs.NewDecoder(bytes.NewReader(json))
		for {
			_, err := decoder.Token()
			if err != nil {
				if err == io.EOF {
					break
				}
				panic(err)
			}
		}
	})

	b.Run("Mine", func(b *testing.B) {
		r := Reader{}
		r.Init(json)
		for {
			tt, err := r.Next()
			if err != nil {
				panic(err)
			}
			if tt == EOF {
				break
			}
		}
	})
}
