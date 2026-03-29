package ui

type Rect struct {
	X float64
	Y float64
	W float64
	H float64
}

type MenuEntry struct {
	Label    string
	Disabled bool
}

func PointInRect(x, y float64, area Rect) bool {
	return x >= area.X && x <= area.X+area.W && y >= area.Y && y <= area.Y+area.H
}
