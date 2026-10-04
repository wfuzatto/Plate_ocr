package vision

import (
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
)

var _ = png.Decode

type Gray struct {
	W, H int
	Pix  []uint8
}

func FromImage(src image.Image) Gray {
	b := src.Bounds()
	g := Gray{W: b.Dx(), H: b.Dy(), Pix: make([]uint8, b.Dx()*b.Dy())}
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			r, gg, bb, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			y8 := uint8((299*uint64(r>>8) + 587*uint64(gg>>8) + 114*uint64(bb>>8)) / 1000)
			g.Pix[y*g.W+x] = y8
		}
	}
	return g
}

func Decode(r io.Reader) (image.Image, string, error) {
	img, format, err := image.Decode(r)
	if err != nil { return nil, "", err }
	if format != "jpeg" && format != "png" { return nil, "", errors.New("unsupported image format: "+format) }
	return img, format, nil
}

func (g Gray) At(x, y int) uint8 {
	if x < 0 || y < 0 || x >= g.W || y >= g.H { return 0 }
	return g.Pix[y*g.W+x]
}

func (g Gray) Crop(x0, y0, x1, y1 int) Gray {
	if x0 < 0 { x0 = 0 }; if y0 < 0 { y0 = 0 }
	if x1 > g.W { x1 = g.W }; if y1 > g.H { y1 = g.H }
	if x1 <= x0 || y1 <= y0 { return Gray{} }
	out := Gray{W: x1-x0, H: y1-y0, Pix: make([]uint8, (x1-x0)*(y1-y0))}
	for y := 0; y < out.H; y++ {
		copy(out.Pix[y*out.W:(y+1)*out.W], g.Pix[(y0+y)*g.W+x0:(y0+y)*g.W+x1])
	}
	return out
}

func (g Gray) MeanRect(x0, y0, x1, y1 int) float64 {
	if g.W == 0 || g.H == 0 { return 0 }
	if x0 < 0 { x0 = 0 }; if y0 < 0 { y0 = 0 }
	if x1 > g.W { x1 = g.W }; if y1 > g.H { y1 = g.H }
	if x1 <= x0 || y1 <= y0 { return 0 }
	var sum uint64
	for y:=y0; y<y1; y++ { for x:=x0; x<x1; x++ { sum += uint64(g.At(x,y)) } }
	return float64(sum) / float64((x1-x0)*(y1-y0))
}

func (g Gray) ToImage() *image.Gray {
	out := image.NewGray(image.Rect(0,0,g.W,g.H))
	copy(out.Pix, g.Pix)
	return out
}

func EncodeJPEG(w io.Writer, img image.Image, quality int) error {
	if quality < 1 || quality > 100 { quality = 80 }
	return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
}

func Solid(w,h int,v uint8) Gray {
	g:=Gray{W:w,H:h,Pix:make([]uint8,w*h)}
	for i:=range g.Pix { g.Pix[i]=v }
	return g
}

func Set(g *image.Gray,x,y int,v uint8) {
	if x>=0 && y>=0 && x<g.Bounds().Dx() && y<g.Bounds().Dy() { g.SetGray(x,y,color.Gray{Y:v}) }
}
