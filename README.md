## gav1d
[![Status](https://github.com/gen2brain/gav1d/actions/workflows/test.yml/badge.svg)](https://github.com/gen2brain/gav1d/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/gen2brain/gav1d.svg)](https://pkg.go.dev/github.com/gen2brain/gav1d)

[AV1](https://en.wikipedia.org/wiki/AV1) video and [AVIF](https://en.wikipedia.org/wiki/AVIF) image decoder in pure Go.

A port of [dav1d](https://code.videolan.org/videolan/dav1d), bit-exact against it. No CGo, no assembly.

### Decoding

```go
img, err := avif.Decode(r)
```

`avif.Decode` returns `*image.NRGBA`, or `*image.NRGBA64` above 8 bits. The package registers itself with
`image.RegisterFormat`, so `image.Decode` works once it is imported for side effects.

```go
opts := avif.Options{
    AutoRotate:     true,    // apply the clap/irot/imir transforms
    FrameSizeLimit: 1 << 24, // bound a frame's area, in pixels
}

anim, err := avif.DecodeAll(r, opts)
```

`DecodeAll` returns every frame of an image sequence with its duration; a still image gives one frame.

The `av1` package decodes the bitstream on its own, a temporal unit at a time:

```go
d := av1.Decoder{Threads: 4}

for _, tu := range temporalUnits {
    pics, err := d.DecodeOBUs(tu)
    ...
}
```

`Decoder.Threads` bounds the goroutines decoding a frame's tiles, and defaults to `GOMAXPROCS`. It only helps
streams that have more than one tile.

For video, `Send` and `Receive` decode several frames at once:

```go
d := av1.Decoder{}

for _, tu := range temporalUnits {
    if err := d.Send(tu); err != nil {
        ...
    }
    pic, err := d.Receive()
    ...
}
```

`Receive` returns pictures in output order, and nil once the queue is empty. `Decoder.FrameThreads` bounds
the frames in flight, and `Flush` drops them to seek. Call `Send` and `Receive` from one goroutine.

### Supported

8/10/12-bit, 4:2:0/4:2:2/4:4:4/monochrome, film grain, scalable streams, and in the container alpha, `grid`,
`clap`/`irot`/`imir`, `colr` and image sequences.

### License

The decoder is a port of dav1d and carries its BSD-2-Clause notice, in [COPYING](COPYING).

This project is an implementation of a decoder. It gives you no special rights on the AV1 patents. Please read
the [AV1 patent license](PATENTS) that applies to the AV1 specification and codec.
