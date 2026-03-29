package ui

import (
	"image/color"
	"testing"

	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestPointInRectIncludesEdges(t *testing.T) {
	area := Rect{X: 10, Y: 20, W: 30, H: 40}
	for _, point := range [][2]float64{{10, 20}, {40, 60}, {25, 45}} {
		if !PointInRect(point[0], point[1], area) {
			t.Fatalf("expected point %v inside rect %+v", point, area)
		}
	}
	if PointInRect(9, 20, area) || PointInRect(41, 60, area) {
		t.Fatalf("expected outside points to be excluded")
	}
}

func TestTextFacesAndMeasureText(t *testing.T) {
	if TitleFace() == nil || BodyFace() == nil || SmallFace() == nil {
		t.Fatalf("expected loaded text faces")
	}
	width, height := MeasureText("Go Hockey", BodyFace())
	if width <= 0 || height <= 0 {
		t.Fatalf("expected positive text measurement, got width=%f height=%f", width, height)
	}
}

func TestModalMenuLayout(t *testing.T) {
	panel := ModalMenuPanelRect(3)
	if panel.W != 460 || panel.H <= 0 {
		t.Fatalf("unexpected modal panel rect %+v", panel)
	}
	option := ModalMenuOptionRect(2, 3)
	if option.X < panel.X || option.Y < panel.Y || option.X+option.W > panel.X+panel.W || option.Y+option.H > panel.Y+panel.H {
		t.Fatalf("expected option rect inside panel, panel=%+v option=%+v", panel, option)
	}
}

func TestDrawHelpersSmoke(t *testing.T) {
	screen := ebiten.NewImage(int(sim.WindowWidth), int(sim.WindowHeight))
	DrawRoundedFill(screen, 10, 10, 120, 60, 12, color.RGBA{0x12, 0x34, 0x56, 0xff})
	DrawLine(screen, 10, 10, 100, 100, 4, color.White)
	DrawText(screen, "Go Hockey", BodyFace(), 24, 24, color.White)
	DrawTextCentered(screen, "Centered", SmallFace(), sim.CenterX, 80, color.Black)
	DrawOverlayButton(screen, Rect{X: 30, Y: 100, W: 140, H: 40}, "Resume", true, true)
	DrawModalMenu(screen, "Pause Menu", "Choose what to do next.", "Enter selects.", []MenuEntry{{Label: "Resume"}, {Label: "Quit", Disabled: true}}, 0)
}
