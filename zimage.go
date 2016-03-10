package gimg

import (
	_ "fmt"

	"gopkg.in/gographics/imagick.v2/imagick"
)

type ZImage struct {
	MW *imagick.MagickWand
}

func NewImage() *ZImage {
	return &ZImage{MW: imagick.NewMagickWand()}
}

func (z *ZImage) Destroy() {
	z.MW.Destroy()
}
