package pages

import (
	"piggy-bank/ui/components"
	"piggy-bank/ui/theme"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const (
	appName    = "Piggy Bank"
	appVersion = "1.0"
	appSummary = "A macOS menubar utility to monitor and power-control your lab server and its WSL instances."
)

type AboutPage struct {
	nav     Navigator
	backBtn widget.Clickable
}

func NewAboutPage(nav Navigator) *AboutPage {
	return &AboutPage{nav: nav}
}

func (p *AboutPage) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if p.backBtn.Clicked(gtx) {
		p.nav(Host)
	}

	return contentInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return header(gtx, th, &p.backBtn, "About", nil)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min = gtx.Constraints.Max
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(420)
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return components.StatusHalo(gtx, 110, theme.Primary, 0, nil)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
						layout.Rigid(components.Text(th, 28, font.Bold, theme.Text, appName).Layout),
						layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
						layout.Rigid(components.Text(th, 13, font.Normal, theme.TextMuted, "Version "+appVersion).Layout),
						layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := components.Text(th, 14, font.Normal, theme.Text, appSummary)
							lbl.Alignment = text.Middle
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
						layout.Rigid(components.Text(th, 12, font.Normal, theme.TextMuted, "Released under the MIT License").Layout),
					)
				})
			}),
		)
	})
}
