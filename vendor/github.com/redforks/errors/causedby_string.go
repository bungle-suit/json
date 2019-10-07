// Code generated by "stringer -type=CausedBy"; DO NOT EDIT.

package errors

import "fmt"

const (
	_CausedBy_name_0 = "ByBug"
	_CausedBy_name_1 = "ByRuntime"
	_CausedBy_name_2 = "ByExternal"
	_CausedBy_name_3 = "ByInput"
	_CausedBy_name_4 = "ByClientBug"
	_CausedBy_name_5 = "NoError"
)

var (
	_CausedBy_index_0 = [...]uint8{0, 5}
	_CausedBy_index_1 = [...]uint8{0, 9}
	_CausedBy_index_2 = [...]uint8{0, 10}
	_CausedBy_index_3 = [...]uint8{0, 7}
	_CausedBy_index_4 = [...]uint8{0, 11}
	_CausedBy_index_5 = [...]uint8{0, 7}
)

func (i CausedBy) String() string {
	switch {
	case i == 16777216:
		return _CausedBy_name_0
	case i == 33554432:
		return _CausedBy_name_1
	case i == 50331648:
		return _CausedBy_name_2
	case i == 67108864:
		return _CausedBy_name_3
	case i == 83886080:
		return _CausedBy_name_4
	case i == 100663296:
		return _CausedBy_name_5
	default:
		return fmt.Sprintf("CausedBy(%d)", i)
	}
}
