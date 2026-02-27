/*
	Bencode Decoder
	Can consume data from:
	- os.File (.torrent files)
	- net.TCPConn (BitTorrent peers)
	- bytes.Buffer (in-memory data)
	- http.Response.Body (tracker responses)
	Reference: https://wiki.theory.org/BitTorrentSpecification#Bencoding
*/

package internal

import (
	"bufio" // reduces syscalls
	"errors"
	"fmt"
	"io"
	"strconv"
)

func Decode(r io.Reader) (interface{}, error) {
	br := bufio.NewReader(r)
	return decode(br)
}

// routes to the correct reader seeing the first byte
func decode(br *bufio.Reader) (interface{}, error) {
	b, err := br.Peek(1)
	if err != nil {
		return nil, err
	}

	switch b[0] {
	case 'i':
		return rdInt(br)
	case 'l':
		return rdList(br)
	case 'd':
		return rdDict(br)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return rdByteString(br)
	default:
		return nil, fmt.Errorf("invalid bencode type: %c", b[0])
	}
}

// Int
// i<integer encoded in base ten ASCII>e
func rdInt(br *bufio.Reader) (int64, error) {
	_, err := br.ReadByte() // consume i
	if err != nil {
		return 0, err
	}

	str, err := br.ReadBytes('e')
	if err != nil {
		return 0, err
	}

	// remove e
	str = str[:len(str)-1]

	if len(str) == 0 {
		return 0, errors.New("invalid Int: empty")
	}

	// invalid leading zeroes
	if len(str) > 1 && str[0] == '0' {
		return 0, errors.New("invalid Int: Leading Zero")
	}
	if len(str) > 1 && str[0] == '-' && str[1] == '0' {
		return 0, errors.New("invalid Int: Negative Zero")
	}

	return strconv.ParseInt(string(str), 10, 64)
}

// Byte Strings
// <string length encoded in base ten ASCII>:<string data>
func rdByteString(br *bufio.Reader) ([]byte, error) {
	// read until :
	_length, err := br.ReadBytes(':')
	if err != nil {
		return nil, err
	}

	// remove :
	lengthStr := _length[:len(_length)-1]

	length, err := strconv.ParseInt(string(lengthStr), 10, 64)
	if err != nil {
		return nil, err
	}

	if length < 0 {
		return nil, errors.New("invalid String Length")
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(br, buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// Lists
// l<bencoded values>e
func rdList(br *bufio.Reader) ([]interface{}, error) {
	_, err := br.ReadByte() // consume l
	if err != nil {
		return nil, err
	}

	var list []interface{}

	for {
		b, err := br.Peek(1)
		if err != nil {
			return nil, err
		}

		if b[0] == 'e' {
			_, _err := br.ReadByte() // consume e
			if _err != nil {
				return nil, _err
			}
			break
		}

		val, err := decode(br)
		if err != nil {
			return nil, err
		}

		list = append(list, val)
	}

	return list, nil
}

// Dictionaries
// d<bencoded string><bencoded element>e
func rdDict(br *bufio.Reader) (map[string]interface{}, error) {
	_, err := br.ReadByte() // consume d
	if err != nil {
		return nil, err
	}

	dict := make(map[string]interface{})

	for {
		b, err := br.Peek(1)
		if err != nil {
			return nil, err
		}

		if b[0] == 'e' {
			_, _err := br.ReadByte()
			if _err != nil {
				return nil, _err
			}
			break
		}

		// keys are string
		key, err := rdByteString(br)
		if err != nil {
			return nil, fmt.Errorf("dict key read failed: %w", err)
		}

		// values can be anything
		val, err := decode(br)
		if err != nil {
			return nil, err
		}

		dict[string(key)] = val
	}

	return dict, nil
}
