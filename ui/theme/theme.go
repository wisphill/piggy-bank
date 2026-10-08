package theme

import (
	"image/color"

	"gioui.org/text"
	"gioui.org/widget/material"
)

var (
	Primary     = color.NRGBA{R: 0x00, G: 0x6A, B: 0x60, A: 0xFF}
	OnPrimary   = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	PrimarySoft = color.NRGBA{R: 0xCC, G: 0xE8, B: 0xE2, A: 0xFF}

	Sidebar     = color.NRGBA{R: 0xE8, G: 0xF1, B: 0xEF, A: 0xFF}
	Background  = color.NRGBA{R: 0xF4, G: 0xFB, B: 0xF9, A: 0xFF}
	Surface     = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	Outline     = color.NRGBA{R: 0x6F, G: 0x79, B: 0x77, A: 0xFF}
	OutlineSoft = color.NRGBA{R: 0xDA, G: 0xE5, B: 0xE2, A: 0xFF}

	Text      = color.NRGBA{R: 0x17, G: 0x1D, B: 0x1C, A: 0xFF}
	TextMuted = color.NRGBA{R: 0x5C, G: 0x66, B: 0x64, A: 0xFF}

	Success = color.NRGBA{R: 0x2E, G: 0x9E, B: 0x5B, A: 0xFF}
	Danger  = color.NRGBA{R: 0xC6, G: 0x3D, B: 0x32, A: 0xFF}
	Offline = color.NRGBA{R: 0x9A, G: 0xA4, B: 0xA2, A: 0xFF}
	PowerOn = color.NRGBA{R: 0x2B, G: 0x68, B: 0x60, A: 0xFF}

	ToastBg = color.NRGBA{R: 0x24, G: 0x2B, B: 0x2A, A: 0xFF}
)

func New(shaper *text.Shaper) *material.Theme {
	th := material.NewTheme()
	th.Shaper = shaper
	th.Palette = material.Palette{
		Bg:         Background,
		Fg:         Text,
		ContrastBg: Primary,
		ContrastFg: OnPrimary,
	}
	return th
}

func WithAlpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}
