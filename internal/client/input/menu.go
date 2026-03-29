package input

import (
	"hockeyv2/internal/client/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func UpdateSelectableMenu(selected int, entries []ui.MenuEntry, rectFor func(index int) ui.Rect) (int, int, bool) {
	if len(entries) == 0 {
		return 0, 0, false
	}
	selected = clampSelection(selected, entries)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cursorX, cursorY := ebiten.CursorPosition()
		x := float64(cursorX)
		y := float64(cursorY)
		for index := range entries {
			if !ui.PointInRect(x, y, rectFor(index)) {
				continue
			}
			selected = index
			if !entries[index].Disabled {
				return selected, index, true
			}
			return selected, 0, false
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		selected = nextSelection(selected, -1, entries)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		selected = nextSelection(selected, 1, entries)
	}
	if (inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)) && !entries[selected].Disabled {
		return selected, selected, true
	}
	return selected, 0, false
}

func clampSelection(selected int, entries []ui.MenuEntry) int {
	if len(entries) == 0 {
		return 0
	}
	if selected >= 0 && selected < len(entries) && !entries[selected].Disabled {
		return selected
	}
	for index, entry := range entries {
		if !entry.Disabled {
			return index
		}
	}
	return 0
}

func nextSelection(selected, delta int, entries []ui.MenuEntry) int {
	if len(entries) == 0 {
		return 0
	}
	selected = clampSelection(selected, entries)
	for step := 0; step < len(entries); step++ {
		selected = (selected + delta + len(entries)) % len(entries)
		if !entries[selected].Disabled {
			return selected
		}
	}
	return selected
}
