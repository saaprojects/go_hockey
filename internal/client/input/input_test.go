package input

import (
	"math"
	"testing"

	"hockeyv2/internal/client/ui"
)

func TestUpdateSelectableMenuWithNoEntries(t *testing.T) {
	selected, activatedIndex, activated := UpdateSelectableMenu(4, nil, func(index int) ui.Rect {
		return ui.Rect{}
	})
	if selected != 0 || activatedIndex != 0 || activated {
		t.Fatalf("expected empty menu result, got selected=%d activatedIndex=%d activated=%v", selected, activatedIndex, activated)
	}
}

func TestClampSelection(t *testing.T) {
	entries := []ui.MenuEntry{{Label: "One", Disabled: true}, {Label: "Two"}, {Label: "Three", Disabled: true}}
	if got := clampSelection(1, entries); got != 1 {
		t.Fatalf("expected existing valid selection 1, got %d", got)
	}
	if got := clampSelection(0, entries); got != 1 {
		t.Fatalf("expected disabled selection to clamp to 1, got %d", got)
	}
	if got := clampSelection(8, entries); got != 1 {
		t.Fatalf("expected out of range selection to clamp to 1, got %d", got)
	}
	if got := clampSelection(0, []ui.MenuEntry{{Disabled: true}}); got != 0 {
		t.Fatalf("expected all-disabled menu to keep 0, got %d", got)
	}
}

func TestNextSelectionSkipsDisabledAndWraps(t *testing.T) {
	entries := []ui.MenuEntry{{Label: "One"}, {Label: "Two", Disabled: true}, {Label: "Three"}}
	if got := nextSelection(0, 1, entries); got != 2 {
		t.Fatalf("expected next selectable item 2, got %d", got)
	}
	if got := nextSelection(2, 1, entries); got != 0 {
		t.Fatalf("expected wrapped selection 0, got %d", got)
	}
	if got := nextSelection(0, -1, entries); got != 2 {
		t.Fatalf("expected backward wrapped selection 2, got %d", got)
	}
}

func TestMovementVectorFromKeys(t *testing.T) {
	if got := movementVectorFromKeys(false, false, false, false); got.X != 0 || got.Y != 0 {
		t.Fatalf("expected no movement, got %#v", got)
	}
	if got := movementVectorFromKeys(true, false, false, false); got.X != -1 || got.Y != 0 {
		t.Fatalf("expected left movement, got %#v", got)
	}
	got := movementVectorFromKeys(true, false, true, false)
	want := math.Sqrt(0.5)
	if math.Abs(got.X+want) > 0.0001 || math.Abs(got.Y+want) > 0.0001 {
		t.Fatalf("expected normalized diagonal movement, got %#v", got)
	}
}

func TestReadyOverlayActionAt(t *testing.T) {
	prevRect := ui.Rect{X: 10, Y: 10, W: 20, H: 20}
	nextRect := ui.Rect{X: 40, Y: 10, W: 20, H: 20}
	readyRect := ui.Rect{X: 70, Y: 10, W: 20, H: 20}

	if got := readyOverlayActionAt(15, 15, prevRect, nextRect, readyRect); !got.ColorPrev || got.ColorNext || got.Ready {
		t.Fatalf("expected prev action, got %+v", got)
	}
	if got := readyOverlayActionAt(45, 15, prevRect, nextRect, readyRect); got.ColorPrev || !got.ColorNext || got.Ready {
		t.Fatalf("expected next action, got %+v", got)
	}
	if got := readyOverlayActionAt(75, 15, prevRect, nextRect, readyRect); got.ColorPrev || got.ColorNext || !got.Ready {
		t.Fatalf("expected ready action, got %+v", got)
	}
	if got := readyOverlayActionAt(0, 0, prevRect, nextRect, readyRect); got != (ReadyOverlayAction{}) {
		t.Fatalf("expected no action, got %+v", got)
	}
}

func TestApplyRoomCodeEditUppercasesAndTrimsInvalidCharacters(t *testing.T) {
	got := applyRoomCodeEdit("ab", []rune{'c', '-', '2', '!'}, false, false, 5)
	if got != "ABC2" {
		t.Fatalf("expected uppercase alphanumeric room code, got %q", got)
	}
	got = applyRoomCodeEdit("ABCDE", []rune{'F'}, false, false, 5)
	if got != "ABCDE" {
		t.Fatalf("expected max length enforcement, got %q", got)
	}
}

func TestApplyRoomNameEditAllowsCommonRoomNameCharacters(t *testing.T) {
	got := applyRoomNameEdit("Friday", []rune{' ', 'N', 'i', 'g', 'h', 't', '#'}, false, false, 32)
	if got != "Friday Night" {
		t.Fatalf("expected sanitized room name, got %q", got)
	}
	got = applyRoomNameEdit("Go Hockey", nil, true, false, 32)
	if got != "Go Hocke" {
		t.Fatalf("expected backspace to remove final rune, got %q", got)
	}
}
