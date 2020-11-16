package bencode

import (
	"io"
	"io/ioutil"
)

type Encoder struct {
	w io.Writer
}

func (e *Encoder) Encode(v interface{}) error {
	b, err := Marshal(v)
	if err != nil {
		return err
	}
	_, err = e.w.Write(b)
	return err
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

type Decoder struct {
	r io.Reader
}

func (d *Decoder) Decode(v interface{}) error {
	// TODO replace this
	b, err := ioutil.ReadAll(d.r)
	if err != nil {
		return err
	}

	_, err = Unmarshal(b, v)
	return err
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}
