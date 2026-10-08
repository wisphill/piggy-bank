package pages

import (
	"context"
	"fmt"
	"image/color"
	"math"
	"piggy-bank/ui/components"
	"piggy-bank/ui/state"
	"piggy-bank/ui/theme"
	"time"

	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const haloSpinPeriod = 3 * time.Second

// HostPage is the landing page: a large status emblem for the main server
// with its power action, modelled on LocalSend's Receive screen.
type HostPage struct {
	host *state.HostState
	nav  Navigator

	historyBtn widget.Clickable
	infoBtn    widget.Clickable
	powerBtn   widget.Clickable
}

func NewHostPage(host *state.HostState, nav Navigator) *HostPage {
	return &HostPage{host: host, nav: nav}
}

func (p *HostPage) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if p.historyBtn.Clicked(gtx) {
		p.nav(History)
	}
	if p.infoBtn.Clicked(gtx) {
		p.nav(About)
	}

	p.host.Mu.Lock()
	name, desc, address := p.host.Name, p.host.Description, p.host.Address
	online, rtt := p.host.IsOnline, p.host.PingRTT
	busy, busyAction, busyLabel := p.host.Busy, p.host.BusyAction, p.host.BusyLabel
	p.host.Mu.Unlock()

	if p.powerBtn.Clicked(gtx) && !busy {
		if online {
			go p.host.Shutdown(context.Background())
		} else {
			go p.host.PowerOn(context.Background())
		}
	}

	// red means "click to shut down", teal means "click to turn on"
	haloColor, hint := theme.PowerOn, "Click the icon to turn on the server"
	switch {
	case busy && busyAction == state.HostActionShutdown:
		haloColor, hint = theme.Danger, ""
	case busy:
		haloColor, hint = theme.PowerOn, ""
	case online:
		haloColor, hint = theme.Danger, "Click the icon to shut down the server"
	}
	if !busy && p.powerBtn.Hovered() {
		haloColor = theme.WithAlpha(haloColor, 0xD8)
	}
	var rotation float32
	if busy {
		phase := float64(gtx.Now.UnixMilli()%haloSpinPeriod.Milliseconds()) / float64(haloSpinPeriod.Milliseconds())
		rotation = float32(phase * 2 * math.Pi)
		gtx.Execute(op.InvalidateCmd{})
	}

	statusText, statusColor := "Offline", theme.Danger
	switch {
	case busy:
		statusText, statusColor = busyLabel, haloColor
	case online:
		statusText, statusColor = fmt.Sprintf("Online · %d ms", rtt.Milliseconds()), theme.Success
	}

	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return contentInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min = gtx.Constraints.Max
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if busy {
										gtx = gtx.Disabled()
									}
									return p.powerBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										if !busy {
											pointer.CursorPointer.Add(gtx.Ops)
										}
										return components.StatusHalo(gtx, 180, haloColor, rotation, components.IconPower)
									})
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(28)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := components.Text(th, 36, font.Normal, theme.Text, name)
									lbl.Alignment = text.Middle
									lbl.MaxLines = 1
									return lbl.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := components.Text(th, 14, font.Normal, theme.TextMuted, desc)
									lbl.Alignment = text.Middle
									lbl.MaxLines = 1
									return lbl.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return statusChip(gtx, th, address, statusText, statusColor)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
								layout.Rigid(components.Text(th, 12, font.Normal, theme.TextMuted, hint).Layout),
							)
						})
					}),
				)
			})
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(40), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.NE.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return components.IconButton(gtx, &p.historyBtn, components.IconHistory, theme.Text)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return components.IconButton(gtx, &p.infoBtn, components.IconInfo, theme.Text)
						}),
					)
				})
			})
		}),
	)
}

func statusChip(gtx layout.Context, th *material.Theme, address, status string, c color.NRGBA) layout.Dimensions {
	return components.Surface(gtx, 16, theme.Surface, theme.OutlineSoft, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(7), Bottom: unit.Dp(7)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(components.Text(th, 13, font.Medium, theme.TextMuted, address).Layout),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return components.Dot(gtx, 8, c)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(components.Text(th, 13, font.Medium, c, status).Layout),
			)
		})
	})
}
