package bencode

import (
	"bytes"
	"errors"
	"sort"
	"strconv"
)

// Encode turns a Go interface back into bencoded bytes
func Encode(val interface{}) ([]byte, error) {
	switch v := val.(type) {
	case string:
		return encodeBytes([]byte(v)), nil
	case []byte:
		return encodeBytes(v), nil
	case int64:
		return encodeInt(v), nil
	case []interface{}:
		return encodeList(v)
	case map[string]interface{}:
		return encodeDict(v)
	default:
		return nil, errors.New("unsupported encode type")
	}
}

func encodeBytes(v []byte) []byte {
	prefix := []byte(strconv.Itoa(len(v)) + ":")
	return append(prefix, v...)
}

func encodeInt(v int64) []byte {
	return []byte("i" + strconv.FormatInt(v, 10) + "e")
}

func encodeList(v []interface{}) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('l')
	for _, item := range v {
		enc, err := Encode(item)
		if err != nil {
			return nil, err
		}
		buf.Write(enc)
	}
	buf.WriteByte('e')
	return buf.Bytes(), nil
}

func encodeDict(v map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('d')

	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		buf.Write(encodeBytes([]byte(k)))
		enc, err := Encode(v[k])
		if err != nil {
			return nil, err
		}
		buf.Write(enc)
	}
	buf.WriteByte('e')
	return buf.Bytes(), nil
}
