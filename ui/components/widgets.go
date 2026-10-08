package components

import (
	"image"
	"image/color"
	"math"
	"piggy-bank/ui/theme"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Text returns a label using the app typography defaults.
func Text(th *material.Theme, size unit.Sp, weight font.Weight, c color.NRGBA, txt string) material.LabelStyle {
	lbl := material.Label(th, size, txt)
	lbl.Font.Weight = weight
	lbl.Color = c
	return lbl
}

// Surface draws a rounded background (and optional 1dp border) sized to w.
func Surface(gtx layout.Context, radius unit.Dp, bg, border color.NRGBA, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()

	rect := image.Rectangle{Max: dims.Size}
	rr := min(gtx.Dp(radius), dims.Size.X/2, dims.Size.Y/2)
	if bg.A > 0 {
		paint.FillShape(gtx.Ops, bg, clip.UniformRRect(rect, rr).Op(gtx.Ops))
	}
	if border.A > 0 {
		paint.FillShape(gtx.Ops, border, clip.Stroke{
			Path:  clip.UniformRRect(rect, rr).Path(gtx.Ops),
			Width: float32(gtx.Dp(1)),
		}.Op())
	}
	call.Add(gtx.Ops)
	return dims
}

// Card is a white rounded container with a soft outline.
func Card(gtx layout.Context, w layout.Widget) layout.Dimensions {
	return Surface(gtx, 12, theme.Surface, theme.OutlineSoft, w)
}

// Icon lays out ic as a square of the given size.
func Icon(gtx layout.Context, ic *widget.Icon, size unit.Dp, c color.NRGBA) layout.Dimensions {
	sz := gtx.Dp(size)
	gtx.Constraints = layout.Exact(image.Pt(sz, sz))
	return ic.Layout(gtx, c)
}

// Dot draws a filled circle, used for status indicators.
func Dot(gtx layout.Context, size unit.Dp, c color.NRGBA) layout.Dimensions {
	sz := gtx.Dp(size)
	paint.FillShape(gtx.Ops, c, clip.Ellipse{Max: image.Pt(sz, sz)}.Op(gtx.Ops))
	return layout.Dimensions{Size: image.Pt(sz, sz)}
}

// IconButton is a circular, borderless icon button.
func IconButton(gtx layout.Context, btn *widget.Clickable, ic *widget.Icon, c color.NRGBA) layout.Dimensions {
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		sz := gtx.Dp(40)
		if btn.Hovered() {
			paint.FillShape(gtx.Ops, theme.WithAlpha(theme.Primary, 0x1A), clip.Ellipse{Max: image.Pt(sz, sz)}.Op(gtx.Ops))
		}
		pointer.CursorPointer.Add(gtx.Ops)

		iconSz := gtx.Dp(24)
		off := (sz - iconSz) / 2
		stack := op.Offset(image.Pt(off, off)).Push(gtx.Ops)
		Icon(gtx, ic, 24, c)
		stack.Pop()
		return layout.Dimensions{Size: image.Pt(sz, sz)}
	})
}

// PillButton is a rounded button with an optional leading icon, in the style
// of LocalSend's "Receive via link" button.
type PillButton struct {
	Icon     *widget.Icon
	Label    string
	Color    color.NRGBA
	Filled   bool
	Disabled bool
	Compact  bool
}

func (b PillButton) Layout(gtx layout.Context, th *material.Theme, btn *widget.Clickable) layout.Dimensions {
	gtx.Constraints.Min = image.Point{}
	fg := b.Color
	if b.Disabled {
		gtx = gtx.Disabled()
		fg = theme.Offline
	}

	padX, padY, textSize, iconSize := unit.Dp(22), unit.Dp(11), unit.Sp(15), unit.Dp(20)
	if b.Compact {
		padX, padY, textSize, iconSize = unit.Dp(14), unit.Dp(7), unit.Sp(13), unit.Dp(16)
	}

	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg := color.NRGBA{}
		border := theme.Outline
		textColor := fg
		switch {
		case b.Filled && !b.Disabled:
			bg, border, textColor = fg, color.NRGBA{}, theme.OnPrimary
		case b.Disabled:
			border = theme.OutlineSoft
		case btn.Hovered():
			bg = theme.WithAlpha(fg, 0x14)
		}
		if !b.Disabled {
			pointer.CursorPointer.Add(gtx.Ops)
		}

		return Surface(gtx, 24, bg, border, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: padX, Right: padX, Top: padY, Bottom: padY}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if b.Icon == nil {
							return layout.Dimensions{}
						}
						return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return Icon(gtx, b.Icon, iconSize, textColor)
						})
					}),
					layout.Rigid(Text(th, textSize, font.Medium, textColor, b.Label).Layout),
				)
			})
		})
	})
}

// NavItem is a sidebar destination: an icon inside a pill indicator followed
// by a label, like the Material 3 navigation rail used by LocalSend.
func NavItem(gtx layout.Context, th *material.Theme, btn *widget.Clickable, ic *widget.Icon, label string, selected bool) layout.Dimensions {
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		pointer.CursorPointer.Add(gtx.Ops)
		gtx.Constraints.Min.X = gtx.Constraints.Max.X

		return layout.Inset{Left: unit.Dp(14), Top: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					pill := image.Pt(gtx.Dp(64), gtx.Dp(36))
					var bg color.NRGBA
					switch {
					case selected:
						bg = theme.PrimarySoft
					case btn.Hovered():
						bg = theme.WithAlpha(theme.PrimarySoft, 0x80)
					}
					if bg.A > 0 {
						paint.FillShape(gtx.Ops, bg, clip.UniformRRect(image.Rectangle{Max: pill}, pill.Y/2).Op(gtx.Ops))
					}

					iconSz := gtx.Dp(24)
					stack := op.Offset(image.Pt((pill.X-iconSz)/2, (pill.Y-iconSz)/2)).Push(gtx.Ops)
					Icon(gtx, ic, 24, theme.Text)
					stack.Pop()
					return layout.Dimensions{Size: pill}
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					weight := font.Medium
					if selected {
						weight = font.Bold
					}
					return Text(th, 14, weight, theme.Text, label).Layout(gtx)
				}),
			)
		})
	})
}

// StatusHalo draws the LocalSend-style emblem: a solid disc surrounded by a
// ring of arc segments. rotation is in radians so callers can animate it.
func StatusHalo(gtx layout.Context, size unit.Dp, c color.NRGBA, rotation float32, center *widget.Icon) layout.Dimensions {
	sz := gtx.Dp(size)
	s := float32(sz)
	mid := f32.Pt(s/2, s/2)

	discR := s * 0.28
	paint.FillShape(gtx.Ops, c, clip.Ellipse{
		Min: image.Pt(int(mid.X-discR), int(mid.Y-discR)),
		Max: image.Pt(int(mid.X+discR), int(mid.Y+discR)),
	}.Op(gtx.Ops))

	const segments = 8
	step := 2 * math.Pi / segments
	gap := float32(step) * 0.32
	ringR := s * 0.44

	var p clip.Path
	p.Begin(gtx.Ops)
	for i := 0; i < segments; i++ {
		a := rotation + float32(i)*float32(step) + gap/2
		p.MoveTo(f32.Pt(
			mid.X+ringR*float32(math.Cos(float64(a))),
			mid.Y+ringR*float32(math.Sin(float64(a))),
		))
		p.ArcTo(mid, mid, float32(step)-gap)
	}
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: s * 0.075}.Op())

	if center != nil {
		iconSz := int(s * 0.26)
		off := (sz - iconSz) / 2
		stack := op.Offset(image.Pt(off, off)).Push(gtx.Ops)
		cgtx := gtx
		cgtx.Constraints = layout.Exact(image.Pt(iconSz, iconSz))
		center.Layout(cgtx, theme.OnPrimary)
		stack.Pop()
	}

	return layout.Dimensions{Size: image.Pt(sz, sz)}
}

// Toast is a floating dark pill used for transient status messages.
func Toast(gtx layout.Context, th *material.Theme, msg string, busy bool) layout.Dimensions {
	return Surface(gtx, 20, theme.ToastBg, color.NRGBA{}, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(18), Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !busy {
						return layout.Dimensions{}
					}
					return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						sz := gtx.Dp(16)
						gtx.Constraints = layout.Exact(image.Pt(sz, sz))
						loader := material.Loader(th)
						loader.Color = theme.PrimarySoft
						return loader.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := Text(th, 13, font.Normal, theme.OnPrimary, msg)
					lbl.MaxLines = 1
					lbl.Alignment = text.Start
					return lbl.Layout(gtx)
				}),
			)
		})
	})
}

// Divider draws a 1dp horizontal line across the available width.
func Divider(gtx layout.Context) layout.Dimensions {
	size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(1))
	paint.FillShape(gtx.Ops, theme.OutlineSoft, clip.Rect{Max: size}.Op())
	return layout.Dimensions{Size: size}
}
