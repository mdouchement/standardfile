package stormbinc

import (
	"bytes"

	"github.com/ugorji/go/codec"
)

const name = "binc"

// Codec that encodes to and decodes from Binc.
// See https://github.com/ugorji/binc
var Codec = new(bincCodec)

type bincCodec int

func (c bincCodec) Marshal(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := codec.NewEncoder(&b, &codec.BincHandle{})
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c bincCodec) Unmarshal(b []byte, v any) error {
	r := bytes.NewReader(b)
	dec := codec.NewDecoder(r, &codec.BincHandle{})
	return dec.Decode(v)
}

func (c bincCodec) Name() string {
	return name
}
