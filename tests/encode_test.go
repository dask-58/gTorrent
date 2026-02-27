package tests

import (
	"testing"

	"github.com/dask-58/gTorrent/internal/bencode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{"valid string", "spam", []byte("4:spam"), false},
		{"empty string", "", []byte("0:"), false},
		{"unicode string", "🚀", []byte("4:🚀"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bencode.Encode(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEncodeByteSlice(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    []byte
		wantErr bool
	}{
		{"valid byte slice", []byte("spam"), []byte("4:spam"), false},
		{"empty byte slice", []byte(""), []byte("0:"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bencode.Encode(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEncodeInt(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		want    []byte
		wantErr bool
	}{
		{"valid positive int", 42, []byte("i42e"), false},
		{"valid negative int", -42, []byte("i-42e"), false},
		{"valid zero", 0, []byte("i0e"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bencode.Encode(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEncodeList(t *testing.T) {
	tests := []struct {
		name    string
		input   []interface{}
		want    []byte
		wantErr bool
	}{
		{"valid list of ints", []interface{}{int64(1), int64(2), int64(3)}, []byte("li1ei2ei3ee"), false},
		{"valid mixed list", []interface{}{"spam", int64(42)}, []byte("l4:spami42ee"), false},
		{"valid empty list", []interface{}{}, []byte("le"), false},
		{"nested list", []interface{}{[]interface{}{int64(1)}}, []byte("lli1eee"), false},
		{"invalid element in list", []interface{}{float64(3.14)}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bencode.Encode(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEncodeDict(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		want    []byte
		wantErr bool
	}{
		{"valid dict", map[string]interface{}{"cow": "moo", "spam": "eggs"}, []byte("d3:cow3:moo4:spam4:eggse"), false}, // tests sorting too
		{"valid empty dict", map[string]interface{}{}, []byte("de"), false},
		{"nested dict", map[string]interface{}{"nested": map[string]interface{}{"key": int64(42)}}, []byte("d6:nestedd3:keyi42eee"), false},
		{"invalid element in dict", map[string]interface{}{"cow": float64(3.14)}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bencode.Encode(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEncodeUnsupported(t *testing.T) {
	_, err := bencode.Encode(float64(3.14))
	require.Error(t, err, "Expected error for unsupported type")

	_, err = bencode.Encode(struct{}{})
	require.Error(t, err, "Expected error for unsupported type")
}
