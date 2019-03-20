package jase93

import (
	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"
	"math/rand"
	"testing"
	"testing/quick"
)

func TestMaxEncodedLen(t *testing.T) {
	for _, tc := range []struct {
		n, len int
	}{
		{0, 0},
		{1, 2},
		{2, 3},
		{3, 4},
	} {
		if l := MaxEncodedLen(tc.n); l != tc.len {
			t.Errorf("MaxEncodedLen(%d) = %d != %d", tc.n, l, tc.len)
		}
	}
}

func TestEncode(t *testing.T) {
	for _, tc := range []struct {
		in, out []byte
	}{
		{[]byte{}, []byte{}},
		{[]byte{0x00}, []byte(" ")},
		{[]byte{0x00, 0x00}, []byte("   ")},
		{[]byte{0xff}, []byte("g#")},
		{[]byte{0xff, 0xff}, []byte("(z(")},
	} {
		enc := Encode(nil, tc.in)
		if !bytes.Equal(enc, tc.out) {
			t.Errorf("Encode(%q) = %q != %q", tc.in, enc, tc.out)
		}
	}
}

func TestEncoder(t *testing.T) {
	for _, tc := range []struct {
		in, out []byte
	}{
		{[]byte{}, []byte{}},
		{[]byte{0x00}, []byte(" ")},
		{[]byte{0x00, 0x00}, []byte("   ")},
		{[]byte{0xff}, []byte("g#")},
		{[]byte{0xff, 0xff}, []byte("(z(")},
	} {
		var buf bytes.Buffer
		enc := NewEncoder(&buf)
		if _, err := enc.Write(tc.in); err != nil {
			t.Error(err)
		}
		if err := enc.Close(); err != nil {
			t.Error(err)
		}
		if !bytes.Equal(buf.Bytes(), tc.out) {
			t.Errorf("Encoder(%q) = %q != %q", tc.in, buf.Bytes(), tc.out)
		}
	}
}

func TestDecode(t *testing.T) {
	for _, tc := range []struct {
		in, out []byte
		err     error
	}{
		{[]byte{}, []byte{}, nil},
		{[]byte(" "), []byte{0x00}, nil},
		{[]byte("   "), []byte{0x00, 0x00}, nil},
		{[]byte("g#"), []byte{0xff}, nil},
		{[]byte("(z("), []byte{0xff, 0xff}, nil},
		{[]byte(`"`), []byte{}, ErrInvalidData},
	} {
		dec, err := Decode(nil, tc.in)
		if err != nil {
			if err != tc.err {
				t.Error(err)
			}
			continue
		}
		if !bytes.Equal(dec, tc.out) {
			t.Errorf("Decode(%q) = %q != %q", tc.in, dec, tc.out)
		}
	}
}

func TestDecoder(t *testing.T) {
	for _, tc := range []struct {
		in, out []byte
		err     error
	}{
		{[]byte{}, []byte{}, nil},
		{[]byte(" "), []byte{0x00}, nil},
		{[]byte("   "), []byte{0x00, 0x00}, nil},
		{[]byte("g#"), []byte{0xff}, nil},
		{[]byte("(z("), []byte{0xff, 0xff}, nil},
		{[]byte(`"`), []byte{}, ErrInvalidData},
	} {
		src := bytes.NewBuffer(tc.in)
		dec := NewDecoder(src)

		dst, err := ioutil.ReadAll(dec)
		if err != tc.err {
			t.Error(err)
			continue
		}

		if !bytes.Equal(dst, tc.out) {
			t.Errorf("Decode(%q) = %q != %q", tc.in, dst, tc.out)
		}
	}
}

func TestEncodeDecode(t *testing.T) {
	var inBytes, encBytes int

	quick.Check(func(in []byte) bool {
		inBytes += len(in)

		enc := Encode(nil, in)
		encBytes += len(enc)

		dec, err := Decode(nil, enc)
		if err != nil {
			t.Error(err)
			return false
		}

		return bytes.Equal(in, dec)
	}, &quick.Config{MaxCountScale: 100})

	t.Logf("%d / %d = %g", encBytes, inBytes, float64(encBytes)/float64(inBytes))
}

func TestPartialDecode(t *testing.T) {
	src := []byte(`Man is distinguished, not only by his reason, but by this singular passion from other animals, which is a lust of the mind, that by a perseverance of delight in the continued and indefatigable generation of knowledge, exceeds the short vehemence of any carnal pleasure.`)
	enc := Encode(nil, src)
	d := NewDecoder(bytes.NewReader(enc))

	var dec []byte
	var buf = make([]byte, 10)

	i := 0
	for len(dec) < len(src) {
		l := i % 11
		n, err := d.Read(buf[:l])
		dec = append(dec, buf[:n]...)
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		i++
	}

	if !bytes.Equal(dec, src) {
		t.Errorf("decoder.Read(%q) = %q != %q", enc, dec, src)
	}
}

func TestVector(t *testing.T) {
	src := []byte(`Man is distinguished, not only by his reason, but by this singular passion from other animals, which is a lust of the mind, that by a perseverance of delight in the continued and indefatigable generation of knowledge, exceeds the short vehemence of any carnal pleasure.`)
	t.Log(string(src))

	enc := Encode(nil, src)
	t.Log(string(enc))
	t.Logf("%d / %d = %g", len(enc), len(src), float64(len(enc))/float64(len(src)))

	dec, err := Decode(nil, enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec, src) {
		t.Fatalf("Decode(Encode(%q)) = %q", src, dec)
	}

	var buf bytes.Buffer
	e := NewEncoder(&buf)
	if _, err := e.Write(src); err != nil {
		t.Fatal(err)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}

	d := NewDecoder(&buf)
	dec, err = ioutil.ReadAll(d)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec, src) {
		t.Fatalf("decoder.Read(encoder.Write(%q)) = %q", src, dec)
	}
}

type Forever struct {
	c byte
}

func (f *Forever) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = f.c
	}

	return len(buf), nil
}

type Discard struct {
	n int
}

func (d *Discard) Write(buf []byte) (int, error) {
	d.n += len(buf)
	return len(buf), nil
}

func TestBestCase(t *testing.T) {
	src := &Forever{0x00}
	dst := &Discard{}
	enc := NewEncoder(dst)

	n, err := io.CopyN(enc, src, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}

	t.Logf("best: %d / %d = %g", dst.n, n, float64(dst.n)/float64(n))
}

func TestWorstCase(t *testing.T) {
	src := &Forever{0xff}
	dst := &Discard{}
	enc := NewEncoder(dst)

	n, err := io.CopyN(enc, src, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}

	t.Logf("worst: %d / %d = %g", dst.n, n, float64(dst.n)/float64(n))
}

func TestAverageCase(t *testing.T) {
	src := rand.New(rand.NewSource(0))
	dst := &Discard{}
	enc := NewEncoder(dst)

	n, err := io.CopyN(enc, src, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}

	t.Logf("average: %d / %d = %g", dst.n, n, float64(dst.n)/float64(n))
}

func BenchmarkEncoder(b *testing.B) {
	src := rand.New(rand.NewSource(0))
	dst := &Discard{}
	enc := NewEncoder(dst)

	if _, err := io.CopyN(enc, src, int64(b.N)); err != nil {
		b.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(b.N))

	b.Logf("%d / %d = %g", dst.n, b.N, float64(dst.n)/float64(b.N))
}

func BenchmarkDecoder(b *testing.B) {
	src := &Forever{' '}
	dst := &Discard{}
	dec := NewDecoder(src)

	if _, err := io.CopyN(dst, dec, int64(b.N)); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(b.N))
}

func BenchmarkBase64Encoder(b *testing.B) {
	src := rand.New(rand.NewSource(0))
	dst := &Discard{}
	enc := base64.NewEncoder(base64.RawStdEncoding, dst)

	if _, err := io.CopyN(enc, src, int64(b.N)); err != nil {
		b.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(b.N))

	b.Logf("%d / %d = %g", dst.n, b.N, float64(dst.n)/float64(b.N))
}

func BenchmarkBase64Decoder(b *testing.B) {
	src := &Forever{'A'}
	dst := &Discard{}
	dec := base64.NewDecoder(base64.RawStdEncoding, src)

	if _, err := io.CopyN(dst, dec, int64(b.N)); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(b.N))
}
