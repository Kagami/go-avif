// Package avif implements a AVIF image encoder.
//
// AVIF is defined in https://aomediacodec.github.io/av1-avif/
package avif

// #cgo CFLAGS: -Wall -O2 -DNDEBUG
// #cgo LDFLAGS: -laom
// #include <stdlib.h>
// #include "av1.h"
import "C"
import (
	"fmt"
	"image"
	"io"
	"runtime"
)

type Options struct {
	Threads        int
	Speed          int
	Quality        int
	SubsampleRatio *image.YCbCrSubsampleRatio
}

const (
	MinSpeed   = 0
	MaxSpeed   = 8
	MinQuality = 0
	MaxQuality = 63
)

var (
	DefaultOptions = Options{
		Threads:        0,
		Speed:          4,
		Quality:        25,
		SubsampleRatio: nil,
	}
)

type OptionsError string

func (e OptionsError) Error() string {
	return fmt.Sprintf("options error: %s", string(e))
}

type EncoderError int

func (e EncoderError) ToString() string {
	switch e {
	case C.AVIF_ERROR_GENERAL:
		return "general error"
	case C.AVIF_ERROR_CODEC_INIT:
		return "codec init error"
	case C.AVIF_ERROR_CODEC_DESTROY:
		return "codec destroy error"
	case C.AVIF_ERROR_FRAME_ENCODE:
		return "frame encode error"
	default:
		return "unknown error"
	}
}

func (e EncoderError) Error() string {
	return fmt.Sprintf("encoder error: %s", e.ToString())
}

type MuxerError string

func (e MuxerError) Error() string {
	return fmt.Sprintf("muxer error: %s", string(e))
}

// RGB to BT.709 YCbCr limited range.
// https://web.archive.org/web/20180421030430/http://www.equasys.de/colorconversion.html
// TODO(Kagami): Use fixed point, don't calc chroma values for skipped pixels.
func rgb2yuv(r16, g16, b16 uint32) (uint8, uint8, uint8) {
	r, g, b := float32(r16)/256, float32(g16)/256, float32(b16)/256
	y := 0.183*r + 0.614*g + 0.062*b + 16
	cb := -0.101*r - 0.339*g + 0.439*b + 128
	cr := 0.439*r - 0.399*g - 0.040*b + 128
	return uint8(y), uint8(cb), uint8(cr)
}

// Encode writes the Image m to w in AVIF format with the given options.
// Default parameters are used if a nil *Options is passed.
//
// NOTE: Image pixels are converted to RGBA first using standard Go
// library. This is no-op for PNG images and does the right thing for
// JPEG since they are normally stored as BT.601 full range with some
// chroma subsampling. Then pixels are converted to BT.709 limited range
// with specified chroma subsampling.
//
// Alpha channel and monochrome are not supported at the moment. Only
// 4:2:0 8-bit images are supported at the moment.
func Encode(w io.Writer, m image.Image, o *Options) error {
	// TODO(Kagami): More subsamplings, 10/12 bitdepth, monochrome, alpha.
	// TODO(Kagami): Allow to pass BT.709 YCbCr without extra conversions.
	if o == nil {
		o2 := DefaultOptions
		o = &o2
	} else {
		o2 := *o
		o = &o2
	}
	if o.Threads == 0 {
		o.Threads = runtime.NumCPU()
	}
	if o.SubsampleRatio == nil {
		s := image.YCbCrSubsampleRatio420
		o.SubsampleRatio = &s
		// if yuvImg, ok := m.(*image.YCbCr); ok {
		// 	o.SubsampleRatio = &yuvImg.SubsampleRatio
		// }
	}
	if o.Threads < 1 {
		// return OptionsError("bad threads number")
	}
	if o.Speed < MinSpeed || o.Speed > MaxSpeed {
		return OptionsError("bad speed value")
	}
	if o.Quality < MinQuality || o.Quality > MaxQuality {
		return OptionsError("bad quality value")
	}
	if *o.SubsampleRatio != image.YCbCrSubsampleRatio420 {
		return OptionsError("unsupported subsampling")
	}
	if m.Bounds().Empty() {
		return OptionsError("empty image")
	}

	rec := m.Bounds()
	width := rec.Max.X - rec.Min.X
	height := rec.Max.Y - rec.Min.Y
	ySize := width * height
	uSize := ((width + 1) / 2) * ((height + 1) / 2)
	dataSize := ySize + uSize*2
	// Can't pass normal slice inside a struct, see
	// https://github.com/golang/go/issues/14210
	dataPtr := C.malloc(C.size_t(dataSize))
	defer C.free(dataPtr)
	data := (*[1 << 30]byte)(dataPtr)[:dataSize:dataSize]

	yPos := 0
	uPos := ySize
	for j := rec.Min.Y; j < rec.Max.Y; j++ {
		for i := rec.Min.X; i < rec.Max.X; i++ {
			r16, g16, b16, _ := m.At(i, j).RGBA()
			y, u, v := rgb2yuv(r16, g16, b16)
			data[yPos] = y
			yPos++
			// TODO(Kagami): Resample chroma planes with some better filter.
			if (i-rec.Min.X)&1 == 0 && (j-rec.Min.Y)&1 == 0 {
				data[uPos] = u
				data[uPos+uSize] = v
				uPos++
			}
		}
	}

	cfg := C.avif_config{
		threads: C.int(o.Threads),
		speed:   C.int(o.Speed),
		quality: C.int(o.Quality),
	}
	frame := C.avif_frame{
		width:       C.uint16_t(width),
		height:      C.uint16_t(height),
		subsampling: C.AVIF_SUBSAMPLING_I420,
		data:        (*C.uint8_t)(dataPtr),
	}
	obu := C.avif_buffer{
		buf: nil,
		sz:  0,
	}
	defer C.free(obu.buf)
	// TODO(Kagami): Error description.
	if eErr := C.avif_encode_frame(&cfg, &frame, &obu); eErr != 0 {
		return EncoderError(eErr)
	}

	obuData := (*[1 << 30]byte)(obu.buf)[:obu.sz:obu.sz]
	if mErr := muxFrame(w, m, *o.SubsampleRatio, obuData); mErr != nil {
		return MuxerError(mErr.Error())
	}

	return nil
}
