package pages

import (
	"context"
	"fmt"
	"image"
	"piggy-bank/ui/components"
	"piggy-bank/ui/state"
	"piggy-bank/ui/theme"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type WSLPage struct {
	host *state.HostState
	list widget.List
}

func NewWSLPage(host *state.HostState) *WSLPage {
	return &WSLPage{
		host: host,
		list: widget.List{List: layout.List{Axis: layout.Vertical}},
	}
}

func (p *WSLPage) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	nodes := p.host.WSLSnapshot()
	p.host.Mu.Lock()
	online, address := p.host.IsOnline, p.host.Address
	p.host.Mu.Unlock()

	for _, n := range nodes {
		if n.Node.BtnPower.Clicked(gtx) && !n.Pending {
			go p.host.ToggleWSL(context.Background(), n.Node)
		}
	}

	running := 0
	for _, n := range nodes {
		if n.Status == "Running" {
			running++
		}
	}

	return contentInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return header(gtx, th, nil, "WSL Instances", nil)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				sub := fmt.Sprintf("%d of %d running on %s", running, len(nodes), address)
				return components.Text(th, 13, font.Normal, theme.TextMuted, sub).Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(nodes) == 0 {
					if !online {
						return emptyState(gtx, th, components.IconWSL, "Host is offline", "Turn on the host to see its WSL instances.")
					}
					return emptyState(gtx, th, components.IconWSL, "No WSL instances found", "Instances will appear here once they are detected.")
				}
				return material.List(th, &p.list).Layout(gtx, len(nodes), func(gtx layout.Context, i int) layout.Dimensions {
					return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return wslCard(gtx, th, nodes[i])
					})
				})
			}),
		)
	})
}

func wslCard(gtx layout.Context, th *material.Theme, n state.WSLSnapshot) layout.Dimensions {
	isRunning := n.Status == "Running"
	statusColor := theme.Offline
	if isRunning {
		statusColor = theme.Success
	}

	btn := components.PillButton{Compact: true, Disabled: n.Pending}
	switch {
	case n.Pending:
		btn.Label = "Working…"
	case isRunning:
		btn.Icon, btn.Label, btn.Color = components.IconStop, "Stop", theme.Danger
	case n.Status == "Stopped":
		btn.Icon, btn.Label, btn.Color = components.IconPlay, "Start", theme.Primary
	default:
		btn.Label, btn.Disabled = n.Status, true
	}

	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return components.Card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return iconTile(gtx, components.IconBoard)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(components.Text(th, 15, font.Medium, theme.Text, n.Name).Layout),
						layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return components.Dot(gtx, 8, statusColor)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
								layout.Rigid(components.Text(th, 12, font.Normal, theme.TextMuted, n.Status+" · WSL 2").Layout),
							)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return btn.Layout(gtx, th, &n.Node.BtnPower)
				}),
			)
		})
	})
}

// iconTile draws ic on a soft rounded square, like LocalSend's device avatars.
func iconTile(gtx layout.Context, ic *widget.Icon) layout.Dimensions {
	sz := gtx.Dp(44)
	paint.FillShape(gtx.Ops, theme.PrimarySoft, clip.UniformRRect(image.Rectangle{Max: image.Pt(sz, sz)}, gtx.Dp(10)).Op(gtx.Ops))
	iconSz := gtx.Dp(24)
	stack := op.Offset(image.Pt((sz-iconSz)/2, (sz-iconSz)/2)).Push(gtx.Ops)
	components.Icon(gtx, ic, 24, theme.Primary)
	stack.Pop()
	return layout.Dimensions{Size: image.Pt(sz, sz)}
}
