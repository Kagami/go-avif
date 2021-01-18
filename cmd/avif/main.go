package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"github.com/Kagami/go-avif"
	"github.com/docopt/docopt-go"
)

const VERSION = "0.0.0"
const USAGE = `
Usage: avif [options] -e src_filename -o dst_filename

AVIF encoder

Options:
  -h, --help                Give this help
  -V, --version             Display version number
  -e <src>, --encode=<src>  Source filename
  -o <dst>, --output=<dst>  Destination filename
  -q <qp>, --quality=<qp>   Compression level (0..63), [default: 25]
  -s <spd>, --speed=<spd>   Compression speed (0..8), [default: 4]
  -t <td>, --threads=<td>   Number of threads (1..64, 0 for all available cores), [default: 0]
  --lossless                Lossless compression (alias for -q 0)
  --best                    Slowest compression method (alias for -s 0)
  --fast                    Fastest compression method (alias for -s 8)
`

type config struct {
	Encode   string
	Output   string
	Quality  int
	Speed    int
	Threads  int
	Lossless bool
	Best     bool
	Fast     bool
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func check(cond bool, errStr string) {
	if !cond {
		fmt.Println(errStr)
		os.Exit(1)
	}
}

func main() {
	var conf config
	opts, err := docopt.ParseArgs(USAGE, nil, VERSION)
	checkErr(err)
	err = opts.Bind(&conf)
	checkErr(err)
	check(conf.Quality >= avif.MinQuality && conf.Quality <= avif.MaxQuality, "bad quality (0..63)")
	check(conf.Speed >= avif.MinSpeed && conf.Speed <= avif.MaxSpeed, "bad speed (0..8)")
	check(conf.Threads == 0 || (conf.Threads >= avif.MinThreads && conf.Threads <= avif.MaxThreads), "bad threads (0..64)")
	check(!conf.Best || !conf.Fast, "can't use both --best and --fast")
	if conf.Lossless {
		conf.Quality = 0
	}
	if conf.Best {
		conf.Speed = 0
	} else if conf.Fast {
		conf.Speed = 8
	}
	avifOpts := avif.Options{
		Speed:   conf.Speed,
		Quality: conf.Quality,
		Threads: conf.Threads,
	}

	var src io.Reader
	var dst io.Writer
	if conf.Encode == "-" {
		src = os.Stdin
	} else {
		file, err := os.Open(conf.Encode)
		checkErr(err)
		defer file.Close()
		src = file
	}
	if conf.Output == "-" {
		dst = os.Stdout
	} else {
		file, err := os.Create(conf.Output)
		checkErr(err)
		defer file.Close()
		dst = file
	}

	// TODO(Kagami): Accept y4m.
	img, _, err := image.Decode(src)
	checkErr(err)

	err = avif.Encode(dst, img, &avifOpts)
	checkErr(err)
}
