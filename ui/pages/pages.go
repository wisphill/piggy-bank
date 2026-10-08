package pages

import (
	"piggy-bank/ui/components"
	"piggy-bank/ui/theme"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type ID int

const (
	Host ID = iota
	WSL
	Settings
	History
	About
)

type Navigator func(ID)

type Page interface {
	Layout(gtx layout.Context, th *material.Theme) layout.Dimensions
}

// contentInset leaves room for the custom title bar above every page.
var contentInset = layout.Inset{Top: unit.Dp(44), Left: unit.Dp(32), Right: unit.Dp(32), Bottom: unit.Dp(24)}

func pageTitle(th *material.Theme, txt string) material.LabelStyle {
	return components.Text(th, 26, font.Bold, theme.Text, txt)
}

// header renders a page title with an optional back button and trailing actions.
func header(gtx layout.Context, th *material.Theme, back *widget.Clickable, title string, trailing layout.Widget) layout.Dimensions {
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if back == nil {
				return layout.Dimensions{}
			}
			return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return components.IconButton(gtx, back, components.IconBack, theme.Text)
			})
		}),
		layout.Flexed(1, pageTitle(th, title).Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if trailing == nil {
				return layout.Dimensions{}
			}
			return trailing(gtx)
		}),
	)
}

// emptyState shows a centered icon with a title and hint.
func emptyState(gtx layout.Context, th *material.Theme, ic *widget.Icon, title, hint string) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return components.Icon(gtx, ic, 48, theme.Offline)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(components.Text(th, 16, font.Medium, theme.Text, title).Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(components.Text(th, 13, font.Normal, theme.TextMuted, hint).Layout),
		)
	})
}
