package input

import (
	"hockeyv2/internal/client/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type ReadyOverlayAction struct {
	ColorPrev bool
	ColorNext bool
	Ready     bool
}

func ReadyOverlayMouseAction(prevRect, nextRect, readyRect ui.Rect) ReadyOverlayAction {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return ReadyOverlayAction{}
	}
	cursorX, cursorY := ebiten.CursorPosition()
	return readyOverlayActionAt(float64(cursorX), float64(cursorY), prevRect, nextRect, readyRect)
}

func readyOverlayActionAt(x, y float64, prevRect, nextRect, readyRect ui.Rect) ReadyOverlayAction {
	if ui.PointInRect(x, y, prevRect) {
		return ReadyOverlayAction{ColorPrev: true}
	}
	if ui.PointInRect(x, y, nextRect) {
		return ReadyOverlayAction{ColorNext: true}
	}
	if ui.PointInRect(x, y, readyRect) {
		return ReadyOverlayAction{Ready: true}
	}
	return ReadyOverlayAction{}
}
