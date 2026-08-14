## gav1d
[![Status](https://github.com/gen2brain/gav1d/actions/workflows/test.yml/badge.svg)](https://github.com/gen2brain/gav1d/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/gen2brain/gav1d.svg)](https://pkg.go.dev/github.com/gen2brain/gav1d)

[AV1](https://en.wikipedia.org/wiki/AV1) video and [AVIF](https://en.wikipedia.org/wiki/AVIF) image decoder and encoder in pure Go.

A port of [dav1d](https://code.videolan.org/videolan/dav1d), bit-exact against it. No CGo, no dependencies.

SIMD support for amd64 (AVX2, AVX-512), arm64 (NEON) and riscv64 (RVV, with `GORISCV64=rva23u64`).
Build with `-tags noasm` for pure Go everywhere.

### Decoding

```go
img, err := avif.Decode(r)
```

`avif.Decode` returns `*image.NRGBA`, or `*image.NRGBA64` above 8 bits, and registers itself with
`image.RegisterFormat`. The `av1` package decodes the bitstream on its own, a temporal unit at a time:

```go
d := av1.Decoder{}

for _, tu := range temporalUnits {
    pics, err := d.DecodeOBUs(tu)
    ...
}
```

### Encoding

```go
err := avif.Encode(w, img, avif.EncodeOptions{Quality: 70})
```

The encoder is all-intra and 8-bit, at 4:2:0 or grayscale, with alpha carried as a second
item. It searches every intra mode, partition and angle delta by rate-distortion, so it is
accurate rather than fast: roughly 0.9s for a 512x512 image.

### Supported

8/10/12-bit, 4:2:0/4:2:2/4:4:4/monochrome, film grain, scalable streams, and in the container alpha, `grid`,
`clap`/`irot`/`imir`, `colr`, image sequences, Exif and XMP, and gain maps.

### License

The decoder is a port of dav1d; the encoder's forward transform is a port of libaom. Both are BSD-2-Clause,
and both notices are in [COPYING](COPYING).

This project is an implementation of a codec. It gives you no special rights on the AV1 patents. Please read
the [AV1 patent license](PATENTS) that applies to the AV1 specification and codec.
