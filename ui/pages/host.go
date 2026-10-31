package pages

import (
	"context"
	"fmt"
	"image"
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

	// ring spin state: the ring speeds up while active and coasts to a stop
	// otherwise
	spinAngle float32
	spinSpeed float32
	lastFrame time.Time
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
	busy, busyLabel := p.host.Busy, p.host.BusyLabel
	tailscaleChecked, tailscaleReady := p.host.TailscaleChecked, p.host.TailscaleReady
	p.host.Mu.Unlock()

	if p.powerBtn.Clicked(gtx) && !busy {
		if online {
			go p.host.Shutdown(context.Background())
		} else {
			go p.host.PowerOn(context.Background())
		}
	}

	// teal means online or busy, gray means offline
	haloColor := theme.Offline
	if busy || online {
		haloColor = theme.PowerOn
	}
	if !busy && online && p.powerBtn.Hovered() {
		haloColor = theme.WithAlpha(haloColor, 0xD8)
	}
	// the ring spins while online and coasts to a stop on hover; offline it
	// stays still, and it spins up as the server comes online
	rotation := p.spin(gtx, busy || (online && !p.powerBtn.Hovered()))

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
					// halo centre sits ~36% down the window, like LocalSend
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						top := int(float32(gtx.Constraints.Max.Y)*0.34) - gtx.Dp(75)
						return layout.Dimensions{Size: image.Pt(0, max(top, 0))}
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min = image.Point{}
						return func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if busy {
										gtx = gtx.Disabled()
									}
									return p.powerBtn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										if !busy {
											pointer.CursorPointer.Add(gtx.Ops)
										}
										return components.StatusHaloSVG(gtx, 150, haloColor, rotation, nil)
									})
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(28)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := components.Text(th, 44, font.Normal, theme.Text, name)
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
							)
						}(gtx)
					}),
					layout.Flexed(1, layout.Spacer{}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !tailscaleChecked || tailscaleReady {
							return layout.Dimensions{}
						}
						return components.Text(th, 13, font.Medium, theme.Danger, "Tailscale unavailable — connect it to reach this host").Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !tailscaleChecked || tailscaleReady {
							return layout.Dimensions{}
						}
						return layout.Spacer{Height: unit.Dp(12)}.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return statusChip(gtx, th, address, statusText, statusColor)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(44)}.Layout),
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

const (
	spinAccel = 1.0 // fraction of top speed gained per second while active
	spinDecel = 1.0 // fraction of top speed lost per second otherwise
)

// spin advances the ring rotation and returns the current angle in radians.
func (p *HostPage) spin(gtx layout.Context, active bool) float32 {
	now := gtx.Now
	dt := float32(0)
	if !p.lastFrame.IsZero() {
		dt = min(float32(now.Sub(p.lastFrame).Seconds()), 0.1)
	}
	p.lastFrame = now

	top := float32(2 * math.Pi / haloSpinPeriod.Seconds())
	if active {
		p.spinSpeed = min(p.spinSpeed+top*spinAccel*dt, top)
	} else {
		p.spinSpeed = max(p.spinSpeed-top*spinDecel*dt, 0)
	}
	p.spinAngle = float32(math.Mod(float64(p.spinAngle+p.spinSpeed*dt), 2*math.Pi))

	if active || p.spinSpeed > 0 {
		gtx.Execute(op.InvalidateCmd{})
	} else {
		p.lastFrame = time.Time{}
	}
	return p.spinAngle
}

func statusChip(gtx layout.Context, th *material.Theme, address, status string, c color.NRGBA) layout.Dimensions {
	gtx.Constraints.Min = image.Point{}
	return components.Surface(gtx, 24, color.NRGBA{}, theme.Outline, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(18), Right: unit.Dp(18), Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(components.Text(th, 14, font.Medium, theme.Primary, address).Layout),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return components.Dot(gtx, 8, c)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(components.Text(th, 14, font.Medium, c, status).Layout),
			)
		})
	})
}
