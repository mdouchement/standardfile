package stormcbor

import (
	"bytes"

	"github.com/ugorji/go/codec"
)

const name = "cbor"

// Codec that encodes to and decodes from CBOR (Concise Binary Object Representation).
// http://cbor.io/
// https://tools.ietf.org/html/rfc7049
var Codec = new(cborCodec)

type cborCodec int

func (c cborCodec) Marshal(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := codec.NewEncoder(&b, &codec.CborHandle{})
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c cborCodec) Unmarshal(b []byte, v any) error {
	r := bytes.NewReader(b)
	dec := codec.NewDecoder(r, &codec.CborHandle{})
	return dec.Decode(v)
}

func (c cborCodec) Name() string {
	return name
}
