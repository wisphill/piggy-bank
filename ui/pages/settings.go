package pages

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"piggy-bank/platform/darwin"
	"piggy-bank/ui/components"
	"piggy-bank/ui/state"
	"piggy-bank/ui/theme"
	"sync/atomic"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type SettingsPage struct {
	host *state.HostState
	list widget.List

	startAtLogin widget.Bool
	// set when a start-at-login change failed and the switch must be resynced
	resyncStartAtLogin atomic.Bool
}

func NewSettingsPage(host *state.HostState) *SettingsPage {
	p := &SettingsPage{
		host: host,
		list: widget.List{List: layout.List{Axis: layout.Vertical}},
	}
	p.startAtLogin.Value = darwin.IsStartAtLoginEnabled()
	return p
}

type settingRow struct {
	label string
	value layout.Widget
}

func (p *SettingsPage) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if p.resyncStartAtLogin.CompareAndSwap(true, false) {
		p.startAtLogin.Value = darwin.IsStartAtLoginEnabled()
	}
	if p.startAtLogin.Update(gtx) {
		go p.applyStartAtLogin(p.startAtLogin.Value)
	}

	p.host.Mu.Lock()
	name, address, online := p.host.Name, p.host.Address, p.host.IsOnline
	p.host.Mu.Unlock()

	status, statusColor := "Offline", theme.Danger
	if online {
		status, statusColor = "Online", theme.Success
	}

	token, tokenColor := "Not set", theme.Danger
	if os.Getenv("TELEGRAM_BOT_TOKEN") != "" {
		token, tokenColor = "Configured", theme.Success
	}

	configPath := "~/.piggy_bank/config"
	if home, err := os.UserHomeDir(); err == nil {
		configPath = filepath.Join(home, ".piggy_bank", "config")
	}

	text := func(s string, c color.NRGBA) layout.Widget {
		return components.Text(th, 13, font.Normal, c, s).Layout
	}

	sections := []struct {
		title string
		rows  []settingRow
	}{
		{"General", []settingRow{
			{"Start on login", func(gtx layout.Context) layout.Dimensions {
				return material.Switch(th, &p.startAtLogin, "Start on login").Layout(gtx)
			}},
		}},
		{"Host", []settingRow{
			{"Name", text(name, theme.TextMuted)},
			{"Address", text(address, theme.TextMuted)},
			{"Status", text(status, statusColor)},
		}},
		{"Configuration", []settingRow{
			{"Config file", text(configPath, theme.TextMuted)},
			{"Telegram bot token", text(token, tokenColor)},
		}},
	}

	return contentInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return header(gtx, th, nil, "Settings", nil)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return material.List(th, &p.list).Layout(gtx, len(sections), func(gtx layout.Context, i int) layout.Dimensions {
					s := sections[i]
					return layout.Inset{Bottom: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return settingsSection(gtx, th, s.title, s.rows)
					})
				})
			}),
		)
	})
}

func (p *SettingsPage) applyStartAtLogin(enable bool) {
	var err error
	if enable {
		err = darwin.EnableStartAtLogin()
	} else {
		err = darwin.DisableStartAtLogin()
	}
	if err != nil {
		p.host.Activity.Log(fmt.Sprintf("Could not update start on login: %v", err))
		p.resyncStartAtLogin.Store(true)
		return
	}
	if enable {
		p.host.Activity.Log("Start on login enabled")
	} else {
		p.host.Activity.Log("Start on login disabled")
	}
}

func settingsSection(gtx layout.Context, th *material.Theme, title string, rows []settingRow) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return components.Card(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(18), Right: unit.Dp(18), Top: unit.Dp(14), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := []layout.FlexChild{
				layout.Rigid(components.Text(th, 15, font.Bold, theme.Primary, title).Layout),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			}
			for i, row := range rows {
				if i > 0 {
					children = append(children, layout.Rigid(components.Divider))
				}
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, components.Text(th, 14, font.Normal, theme.Text, row.label).Layout),
							layout.Rigid(row.value),
						)
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}
