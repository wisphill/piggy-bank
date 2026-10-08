package pages

import (
	"piggy-bank/ui/components"
	"piggy-bank/ui/state"
	"piggy-bank/ui/theme"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type HistoryPage struct {
	activity *state.ActivityLog
	nav      Navigator
	list     widget.List

	backBtn  widget.Clickable
	clearBtn widget.Clickable
}

func NewHistoryPage(activity *state.ActivityLog, nav Navigator) *HistoryPage {
	return &HistoryPage{
		activity: activity,
		nav:      nav,
		list:     widget.List{List: layout.List{Axis: layout.Vertical}},
	}
}

func (p *HistoryPage) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if p.backBtn.Clicked(gtx) {
		p.nav(Host)
	}
	if p.clearBtn.Clicked(gtx) {
		p.activity.Clear()
	}
	entries := p.activity.Entries()

	return contentInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return header(gtx, th, &p.backBtn, "History", func(gtx layout.Context) layout.Dimensions {
					if len(entries) == 0 {
						return layout.Dimensions{}
					}
					return components.IconButton(gtx, &p.clearBtn, components.IconDelete, theme.Text)
				})
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(entries) == 0 {
					return emptyState(gtx, th, components.IconHistory, "No activity yet", "Power actions and their results will show up here.")
				}
				return material.List(th, &p.list).Layout(gtx, len(entries), func(gtx layout.Context, i int) layout.Dimensions {
					e := entries[i]
					return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return components.Card(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(12), Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, components.Text(th, 14, font.Normal, theme.Text, e.Message).Layout),
									layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
									layout.Rigid(components.Text(th, 12, font.Normal, theme.TextMuted, e.At.Format("Jan 2, 15:04:05")).Layout),
								)
							})
						})
					})
				})
			}),
		)
	})
}
