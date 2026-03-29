package client

import (
	"bytes"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	uiFaceSource *text.GoTextFaceSource
	uiTitleFace  *text.GoTextFace
	uiBodyFace   *text.GoTextFace
	uiSmallFace  *text.GoTextFace
)

func init() {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	uiFaceSource = source
	uiTitleFace = &text.GoTextFace{Source: uiFaceSource, Size: 28}
	uiBodyFace = &text.GoTextFace{Source: uiFaceSource, Size: 18}
	uiSmallFace = &text.GoTextFace{Source: uiFaceSource, Size: 14}
}

func drawUIText(screen *ebiten.Image, content string, face *text.GoTextFace, x, y float64, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.Filter = ebiten.FilterLinear
	r, g, b, a := clr.RGBA()
	op.ColorScale.Scale(
		float32(r)/65535.0,
		float32(g)/65535.0,
		float32(b)/65535.0,
		float32(a)/65535.0,
	)
	text.Draw(screen, content, face, op)
}

func drawUITextCentered(screen *ebiten.Image, content string, face *text.GoTextFace, centerX, y float64, clr color.Color) {
	width, _ := measureUIText(content, face)
	drawUIText(screen, content, face, centerX-width/2, y, clr)
}

func measureUIText(content string, face *text.GoTextFace) (float64, float64) {
	return text.Measure(content, face, face.Size*1.25)
}
