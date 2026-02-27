package tests

import (
	"bytes"
	"io"
	"testing"

	"github.com/dask-58/gTorrent/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeHelper(input string) (interface{}, error) {
	var r io.Reader = bytes.NewBufferString(input)
	return internal.Decode(&r)
}

func TestDecodeInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    interface{}
		wantErr bool
	}{
		{"valid positive int", "i42e", int64(42), false},
		{"valid negative int", "i-42e", int64(-42), false},
		{"valid zero", "i0e", int64(0), false},
		{"invalid leading zero", "i03e", nil, true},
		{"invalid negative zero", "i-0e", nil, true},
		{"incomplete int", "i42", nil, true},
		{"empty int", "ie", nil, true},
		{"not an int", "ixye", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeHelper(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDecodeByteString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    interface{}
		wantErr bool
	}{
		{"valid string", "4:spam", []byte("spam"), false},
		{"empty string", "0:", []byte(""), false},
		{"invalid length string", "-1:spam", nil, true},
		{"not a string length", "a:spam", nil, true},
		{"incomplete string length", "4", nil, true},
		{"incomplete string data", "4:spa", nil, true},
		{"missing colon", "4spam", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeHelper(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDecodeList(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    interface{}
		wantErr bool
	}{
		{"valid list of ints", "li1ei2ei3ee", []interface{}{int64(1), int64(2), int64(3)}, false},
		{"valid mixed list", "l4:spami42ee", []interface{}{[]byte("spam"), int64(42)}, false},
		{"valid empty list", "le", []interface{}(nil), false},
		{"nested list", "lli1eee", []interface{}{[]interface{}{int64(1)}}, false},
		{"incomplete list", "li1e", nil, true},
		{"invalid list element", "lxe", nil, true},
		{"unterminated nested list", "lli1e", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeHelper(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDecodeDict(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    interface{}
		wantErr bool
	}{
		{"valid dict", "d3:cow3:moo4:spam4:eggse", map[string]interface{}{"cow": []byte("moo"), "spam": []byte("eggs")}, false},
		{"valid empty dict", "de", map[string]interface{}{}, false},
		{"invalid dict key (int)", "di42e4:spame", nil, true},
		{"incomplete dict", "d3:cow3:moo", nil, true},
		{"invalid element", "d3:cowxe", nil, true},
		{"unterminated dict element", "d3:cow", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeHelper(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDecodeInvalid(t *testing.T) {
	_, err := decodeHelper("")
	require.Error(t, err, "Expected error for empty reader")

	_, err = decodeHelper("x")
	require.Error(t, err, "Expected error for invalid identifier")
}
