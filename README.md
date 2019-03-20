# jase93 [![GoDoc](https://godoc.org/github.com/jdknezek/jase93-go?status.svg)](https://godoc.org/github.com/jdknezek/jase93-go)

This package is the Go reference implementation of `jase93`, a JSON-string-safe base-93 encoding for binary data.

`jase93` is derived from [`basE91`](https://base91.sourceforge.net) with a modified alphabet to encode without escaping in JSON strings.

It encodes every 13 or 14 bits into 1 or 2 characters from its alphabet, so it has a variable overhead between 14.3% and 23.1%, compared to `base64`'s fixed 33.3%.

## Alphabet

The encoded alphabet is comprised of all ASCII-printable characters that can be encoded without escaping in a JSON string.

```json
" !#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_`abcdefghijklmnopqrstuvwxyz{|}~"
```

## Efficiency

The following table shows the result of encoding 1 MiB (1_048_576 bytes) of data with `base64` and `jase93`:

Encoding | Data   | Size      | Overhead
-------- | ------ | --------- | --------
`jase93` | `0x00` | 1_198_373 | 14.3%
`jase93` | random | 1_285_100 | 22.6%
`jase93` | `0xff` | 1_290_556 | 23.1%
`base64` | random | 1_398_102 | 33.3%

## Benchmarks

```
BenchmarkEncoder-8              100000000               11.3 ns/op      8883868910.70 MB/s
BenchmarkDecoder-8              300000000                5.18 ns/op     57918199809.68 MB/s
BenchmarkBase64Encoder-8        500000000                2.97 ns/op     168159304708.36 MB/s
BenchmarkBase64Decoder-8        300000000                4.50 ns/op     66614672681.03 MB/s
```

## Example

Input (269 bytes):

```
Man is distinguished, not only by his reason, but by this singular passion from other animals, which is a lust of the mind, that by a perseverance of delight in the continued and indefatigable generation of knowledge, exceeds the short vehemence of any carnal pleasure.
```

Encoded (330 bytes before quotes, +22.7%):

```json
"`}g%-`_M>0dH#;Umkrj3!`)Sv!~`0jLp~F}goORufM1`_M{]PBRKO*.7]>$5|O5&{Hu:x*6q@qo_2_4;0,%~~F;EfHoOep{kZR0jS]BH2}h0^F(Cx*.7YO1`WXF-dH2}INFI<YqfNhd7!`WMc0~F+Cq|yD*YoORu3'V&RTzFvC$0r@{m_>ZRuUINtCy:u)+r7S~3giS].,BKO*Vju>K2WM7hTC,:O*jgc('b7W4;<F1{N]}F]SD)VjLqZRuUHC~FPbA@)gg>](*T4GtC*-x*wqi>ZR`Xm;'Iv^e)kgs>$aKTLp7D2}h01,v}}hQB8{rQuCAV.Wmg~"
```

## References

1. [basE91](http://base91.sourceforge.net/)
2. [Base91, how is it calculated?](https://stackoverflow.com/a/46991272)
