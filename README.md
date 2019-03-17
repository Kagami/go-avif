# go-avif [![Build Status](https://travis-ci.org/Kagami/go-avif.svg?branch=master)](https://travis-ci.org/Kagami/go-avif) [![GoDoc](https://godoc.org/github.com/Kagami/go-avif?status.svg)](https://godoc.org/github.com/Kagami/go-avif)

go-avif implements
AVIF ([AV1 Still Image File Format](https://aomediacodec.github.io/av1-avif/))
encoder for Go using libaom, [the highest quality](https://github.com/Kagami/av1-bench)
AV1 codec at the moment.

## Requirements

Make sure libaom is installed. On typical Linux distro just run:

```bash
sudo apt-get install libaom-dev
```

## Usage

To use go-avif in your Go code:

```go
import "github.com/Kagami/go-avif"
```

To install go-avif in your $GOPATH:

```bash
go get github.com/Kagami/go-avif
```

For further details see [GoDoc documentation](https://godoc.org/github.com/Kagami/go-avif).

## Example

```go
package main

import (
	"image"
	_ "image/jpeg"
	"log"
	"os"

	"github.com/Kagami/go-avif"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s src.jpg dst.avif", os.Args[0])
	}

	srcPath := os.Args[1]
	src, err := os.Open(srcPath)
	if err != nil {
		log.Fatalf("Can't open sorce file: %v", err)
	}

	dstPath := os.Args[2]
	dst, err := os.Create(dstPath)
	if err != nil {
		log.Fatalf("Can't create destination file: %v", err)
	}

	img, _, err := image.Decode(src)
	if err != nil {
		log.Fatalf("Can't decode source file: %v", err)
	}

	err = avif.Encode(dst, img, nil)
	if err != nil {
		log.Fatalf("Can't encode source image: %v", err)
	}

	log.Printf("Encoded AVIF at %s", dstPath)
}
```

## CLI

go-avif comes with handy CLI utility `avif`. It supports encoding of JPEG and
PNG files to AVIF:

```bash
# Compile and put avif binary to $GOPATH/bin
go get github.com/Kagami/go-avif/...

# Encode JPEG to AVIF with default settings
avif -e cat.jpg -o kitty.avif

# Encode PNG with slowest speed
avif -e dog.png -o doggy.avif --best -q 15

# Lossless encoding
avif -e pig.png -o piggy.avif --lossless

# Show help
avif -h
```

## Display

To display resulting AVIF files take a look at software listed
[here](https://github.com/AOMediaCodec/av1-avif/wiki#demuxers--players). E.g.
use [avif.js](https://kagami.github.io/avif.js/) web viewer.

## License

go-avif is licensed under [CC0](COPYING).
