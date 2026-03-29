package ui

import (
	"bytes"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	faceSource *text.GoTextFaceSource
	titleFace  *text.GoTextFace
	bodyFace   *text.GoTextFace
	smallFace  *text.GoTextFace
)

func init() {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	faceSource = source
	titleFace = &text.GoTextFace{Source: faceSource, Size: 28}
	bodyFace = &text.GoTextFace{Source: faceSource, Size: 18}
	smallFace = &text.GoTextFace{Source: faceSource, Size: 14}
}

func TitleFace() *text.GoTextFace {
	return titleFace
}

func BodyFace() *text.GoTextFace {
	return bodyFace
}

func SmallFace() *text.GoTextFace {
	return smallFace
}

func DrawText(screen *ebiten.Image, content string, face *text.GoTextFace, x, y float64, clr color.Color) {
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

func DrawTextCentered(screen *ebiten.Image, content string, face *text.GoTextFace, centerX, y float64, clr color.Color) {
	width, _ := MeasureText(content, face)
	DrawText(screen, content, face, centerX-width/2, y, clr)
}

func MeasureText(content string, face *text.GoTextFace) (float64, float64) {
	return text.Measure(content, face, face.Size*1.25)
}
