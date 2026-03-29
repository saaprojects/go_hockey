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
	bodySource    *text.GoTextFaceSource
	displaySource *text.GoTextFaceSource
	displayFace   *text.GoTextFace
	titleFace     *text.GoTextFace
	headingFace   *text.GoTextFace
	bodyFace      *text.GoTextFace
	smallFace     *text.GoTextFace
	tinyFace      *text.GoTextFace
)

func init() {
	mplusSource, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	pressStartSource, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.PressStart2P_ttf))
	if err != nil {
		log.Fatal(err)
	}
	bodySource = mplusSource
	displaySource = pressStartSource
	displayFace = &text.GoTextFace{Source: displaySource, Size: 24}
	titleFace = &text.GoTextFace{Source: bodySource, Size: 30}
	headingFace = &text.GoTextFace{Source: bodySource, Size: 24}
	bodyFace = &text.GoTextFace{Source: bodySource, Size: 18}
	smallFace = &text.GoTextFace{Source: bodySource, Size: 14}
	tinyFace = &text.GoTextFace{Source: bodySource, Size: 12}
}

func DisplayFace() *text.GoTextFace {
	return displayFace
}

func TitleFace() *text.GoTextFace {
	return titleFace
}

func HeadingFace() *text.GoTextFace {
	return headingFace
}

func BodyFace() *text.GoTextFace {
	return bodyFace
}

func SmallFace() *text.GoTextFace {
	return smallFace
}

func TinyFace() *text.GoTextFace {
	return tinyFace
}

func DrawText(screen *ebiten.Image, content string, face *text.GoTextFace, x, y float64, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.Filter = ebiten.FilterLinear
	if face == displayFace {
		op.Filter = ebiten.FilterNearest
	}
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
	return text.Measure(content, face, face.Size*1.22)
}
