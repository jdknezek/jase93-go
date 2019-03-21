// Package jase93 implements a very efficient JSON-string-safe base-93 encoding for binary data.
package jase93 // import "github.com/jdknezek/jase93-go"

import (
	"errors"
	"io"
	"math"
)

func invertAlphabet(alphabet string) []int8 {
	decode := make([]int8, 256)
	for i := range decode {
		decode[i] = -1
	}

	for i, c := range encode {
		decode[c] = int8(i)
	}

	return decode
}

var (
	encode   = " !#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_`abcdefghijklmnopqrstuvwxyz{|}~"
	decode   = invertAlphabet(encode)
	base     = uint32(len(encode))                  // 93
	wordMax  = (base * base) - 1                    // 8648
	wordBits = uint8(math.Log2(float64(wordMax)))   // 13
	wordMask = uint32((1 << wordBits) - 1)          // 0x1fff
	wordFull = uint32(wordMax - (wordMask + 1) + 1) // 457
)

// MaxEncodedLen returns the maximum number of bytes necessary to encode n source bytes.
func MaxEncodedLen(n int) int {
	return int(math.Ceil(float64(n) * 16 / float64(wordBits)))
}

type encoder struct {
	state     uint32
	stateBits uint8
}

func (e *encoder) reset() {
	e.state = 0
	e.stateBits = 0
}

// write encodes src and appends it to dst.
func (e *encoder) write(dst, src []byte) []byte {
	for _, c := range src {
		e.state |= uint32(c) << e.stateBits
		e.stateBits += 8

		// Ensure we have an extra bit in case we need it
		for e.stateBits > wordBits {
			word := e.state & wordMask
			e.state >>= wordBits
			e.stateBits -= wordBits

			if word < wordFull {
				// We can fit one more bit into word without exceeding wordMax
				word |= (e.state & 1) << wordBits
				e.state >>= 1
				e.stateBits--
			}

			mod := word % base
			div := word / base
			dst = append(dst, encode[mod], encode[div])
		}
	}

	return dst
}

// flush flushes the encoding state and appends it to dst.
func (e *encoder) flush(dst []byte) []byte {
	if e.stateBits > 0 {
		mod := e.state % base
		dst = append(dst, encode[mod])

		if e.stateBits > 8 || e.state >= base {
			div := e.state / base
			dst = append(dst, encode[div])
		}
	}

	return dst
}

// Encode encodes src and appends it to dst.
func Encode(dst, src []byte) []byte {
	var enc encoder
	dst = enc.write(dst, src)
	return enc.flush(dst)
}

// Encoder encodes data to a wrapped io.Writer.
type Encoder struct {
	w   io.Writer
	enc encoder
	buf []byte
}

// NewEncoder creates a new Encoder that encodes to w.
func NewEncoder(w io.Writer) *Encoder {
	return new(Encoder).Reset(w)
}

// Reset sets the Encoder to encode to w and resets its encoding state.
func (e *Encoder) Reset(w io.Writer) *Encoder {
	e.w = w
	e.enc.reset()
	e.buf = nil
	return e
}

// Write encodes data to the wrapped io.Writer.
func (e *Encoder) Write(data []byte) (int, error) {
	e.buf = e.enc.write(e.buf[:0], data)
	_, err := e.w.Write(e.buf)
	return len(data), err
}

// Close flushes the encoding state to the wrapped io.Writer. It does not close the wrapped io.Writer.
func (e *Encoder) Close() error {
	e.buf = e.enc.flush(e.buf[:0])
	_, err := e.w.Write(e.buf)
	return err
}

// ErrInvalidData indicates that non-jase93 characters were encountered while decoding.
var ErrInvalidData = errors.New("jase93: invalid data")

type decoder struct {
	word      int16
	state     uint32
	stateBits uint8
}

func (d *decoder) reset() {
	d.word = -1
	d.state = 0
	d.stateBits = 0
}

// write decodes src and appends it to dst.
func (d *decoder) write(dst, src []byte) ([]byte, error) {
	for _, c := range src {
		nibble := decode[c]
		if nibble == -1 {
			return dst, ErrInvalidData
		}

		if d.word == -1 {
			d.word = int16(nibble)
			continue
		}

		d.word += int16(nibble) * int16(base)

		// If the lower wordBits aren't a full word, then we know this word includes an extra bit
		currentWordBits := wordBits
		if (uint32(d.word) & wordMask) < wordFull {
			currentWordBits++
		}

		d.state |= uint32(d.word) << d.stateBits
		d.stateBits += currentWordBits

		for d.stateBits >= 8 {
			dst = append(dst, byte(d.state))
			d.state >>= 8
			d.stateBits -= 8
		}

		d.word = -1
	}

	return dst, nil
}

// flush flushes the decoding state and appends it to dst.
func (d *decoder) flush(dst []byte) []byte {
	if d.word != -1 {
		dst = append(dst, byte(d.state)|(byte(d.word)<<d.stateBits))
	}

	return dst
}

// Decode decodes src and appends it to dst.
func Decode(dst, src []byte) ([]byte, error) {
	var dec decoder
	dec.reset()
	var err error
	dst, err = dec.write(dst, src)
	if err != nil {
		return dst, err
	}
	return dec.flush(dst), nil
}

// Decoder decodes data from a wrapped io.Reader.
type Decoder struct {
	r   io.Reader
	eof bool
	dec decoder
	buf []byte
}

// NewDecoder creates a new Decoder that decodes from r.
func NewDecoder(r io.Reader) *Decoder {
	return new(Decoder).Reset(r)
}

// Reset sets the Decoder to decode from r and resets its decoding state.
func (d *Decoder) Reset(r io.Reader) *Decoder {
	d.r = r
	d.eof = false
	d.dec.reset()
	d.buf = nil
	return d
}

// Read decodes data from the wrapped io.Reader.
func (d *Decoder) Read(data []byte) (n int, err error) {
	if len(data) == 0 {
		return 0, nil
	}

	// Read outstanding data
	if len(d.buf) > 0 {
		n += copy(data, d.buf)
		data = data[n:]
		if n == len(d.buf) {
			// All buffered data was read
			d.buf = d.buf[:0]
			if d.eof {
				return n, io.EOF
			}
		} else {
			copy(d.buf, d.buf[n:])
			d.buf = d.buf[:len(d.buf)-n]
			// data must have been too small
			return
		}
	}

	rn, rerr := d.r.Read(data)
	if rn > 0 {
		d.buf, err = d.dec.write(d.buf, data[:rn])
	}

	if rerr == io.EOF {
		d.eof = true
		d.buf = d.dec.flush(d.buf)
	}

	cn := copy(data, d.buf)
	n += cn
	if cn == len(d.buf) {
		// All buffered data was read
		d.buf = d.buf[:0]
		if d.eof {
			err = io.EOF
		}
	} else {
		copy(d.buf, d.buf[cn:])
		d.buf = d.buf[:len(d.buf)-cn]
	}

	return
}
