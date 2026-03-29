package input

import (
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func UpdateAddressField(current string, maxRunes int) string {
	return applyAddressFieldEdit(current, ebiten.AppendInputChars(nil), inpututil.IsKeyJustPressed(ebiten.KeyBackspace), inpututil.IsKeyJustPressed(ebiten.KeyDelete), maxRunes)
}

func UpdateRoomCodeField(current string, maxRunes int) string {
	return applyRoomCodeEdit(current, ebiten.AppendInputChars(nil), inpututil.IsKeyJustPressed(ebiten.KeyBackspace), inpututil.IsKeyJustPressed(ebiten.KeyDelete), maxRunes)
}

func UpdateRoomNameField(current string, maxRunes int) string {
	return applyRoomNameEdit(current, ebiten.AppendInputChars(nil), inpututil.IsKeyJustPressed(ebiten.KeyBackspace), inpututil.IsKeyJustPressed(ebiten.KeyDelete), maxRunes)
}

func applyAddressFieldEdit(current string, chars []rune, backspace, clear bool, maxRunes int) string {
	value := current
	if clear {
		value = ""
	}
	if backspace {
		runes := []rune(value)
		if len(runes) > 0 {
			value = string(runes[:len(runes)-1])
		}
	}
	if len(chars) == 0 {
		return value
	}

	count := len([]rune(value))
	var builder strings.Builder
	builder.WriteString(value)
	for _, r := range chars {
		if !allowedAddressRune(r) {
			continue
		}
		if maxRunes > 0 && count >= maxRunes {
			break
		}
		builder.WriteRune(r)
		count++
	}
	return builder.String()
}

func applyRoomCodeEdit(current string, chars []rune, backspace, clear bool, maxRunes int) string {
	value := strings.ToUpper(current)
	if clear {
		value = ""
	}
	if backspace {
		runes := []rune(value)
		if len(runes) > 0 {
			value = string(runes[:len(runes)-1])
		}
	}
	if len(chars) == 0 {
		return value
	}

	count := len([]rune(value))
	var builder strings.Builder
	builder.WriteString(value)
	for _, r := range chars {
		upper := unicode.ToUpper(r)
		if !allowedRoomCodeRune(upper) {
			continue
		}
		if maxRunes > 0 && count >= maxRunes {
			break
		}
		builder.WriteRune(upper)
		count++
	}
	return builder.String()
}

func applyRoomNameEdit(current string, chars []rune, backspace, clear bool, maxRunes int) string {
	value := current
	if clear {
		value = ""
	}
	if backspace {
		runes := []rune(value)
		if len(runes) > 0 {
			value = string(runes[:len(runes)-1])
		}
	}
	if len(chars) == 0 {
		return value
	}

	count := len([]rune(value))
	var builder strings.Builder
	builder.WriteString(value)
	for _, r := range chars {
		if !allowedRoomNameRune(r) {
			continue
		}
		if maxRunes > 0 && count >= maxRunes {
			break
		}
		builder.WriteRune(r)
		count++
	}
	return builder.String()
}

func allowedAddressRune(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	switch r {
	case '.', '-', ':', '[', ']':
		return true
	default:
		return false
	}
}

func allowedRoomCodeRune(r rune) bool {
	return strings.ContainsRune("ABCDEFGHJKLMNPQRSTUVWXYZ23456789", r)
}
func allowedRoomNameRune(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	switch r {
	case ' ', '-', '_', '\'', '.', '!', '&':
		return true
	default:
		return false
	}
}
