package avif_test

import (
	"image"
	_ "image/jpeg"
	"log"
	"os"

	"github.com/kagami/go-avif"
)

const usageHelp = "Usage: %s src.jpg dst.avif"

func Example() {
	if len(os.Args) != 3 {
		log.Fatalf(usageHelp, os.Args[0])
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
