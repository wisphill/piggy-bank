package layouts

import (
	"image"
	"piggy-bank/ui/components"
	"piggy-bank/ui/pages"
	"piggy-bank/ui/state"
	"piggy-bank/ui/theme"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const sidebarWidth = unit.Dp(240)

type navEntry struct {
	id    pages.ID
	label string
	icon  *widget.Icon
	btn   widget.Clickable
}

// AppShell is the LocalSend-style frame: a navigation sidebar on the left and
// the selected page on the right, with a toast for in-progress actions.
type AppShell struct {
	host    *state.HostState
	current pages.ID
	nav     []*navEntry
	pages   map[pages.ID]pages.Page
}

func NewAppShell(host *state.HostState) *AppShell {
	s := &AppShell{
		host: host,
		nav: []*navEntry{
			{id: pages.Host, label: "Host", icon: components.IconHost},
			{id: pages.WSL, label: "WSL", icon: components.IconWSL},
			{id: pages.Settings, label: "Settings", icon: components.IconSettings},
		},
	}
	s.pages = map[pages.ID]pages.Page{
		pages.Host:     pages.NewHostPage(host, s.Navigate),
		pages.WSL:      pages.NewWSLPage(host),
		pages.Settings: pages.NewSettingsPage(host),
		pages.History:  pages.NewHistoryPage(host.Activity, s.Navigate),
		pages.About:    pages.NewAboutPage(s.Navigate),
	}
	return s
}

func (s *AppShell) Navigate(id pages.ID) {
	s.current = id
}

// navSelection maps sub-pages to the sidebar entry they belong to.
func (s *AppShell) navSelection() pages.ID {
	switch s.current {
	case pages.History, pages.About:
		return pages.Host
	}
	return s.current
}

func (s *AppShell) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	for _, e := range s.nav {
		if e.btn.Clicked(gtx) {
			s.Navigate(e.id)
		}
	}

	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return s.layoutSidebar(gtx, th)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return s.layoutContent(gtx, th)
		}),
	)
}

func (s *AppShell) layoutSidebar(gtx layout.Context, th *material.Theme) layout.Dimensions {
	size := image.Pt(gtx.Dp(sidebarWidth), gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, theme.Sidebar, clip.Rect{Max: size}.Op())
	gtx.Constraints = layout.Exact(size)

	selected := s.navSelection()
	children := []layout.FlexChild{
		layout.Rigid(layout.Spacer{Height: unit.Dp(60)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, components.Text(th, 30, font.Bold, theme.Text, "Piggy Bank").Layout)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(36)}.Layout),
	}
	for _, e := range s.nav {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return components.NavItem(gtx, th, &e.btn, e.icon, e.label, e.id == selected)
		}))
	}

	layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	return layout.Dimensions{Size: size}
}

func (s *AppShell) layoutContent(gtx layout.Context, th *material.Theme) layout.Dimensions {
	paint.FillShape(gtx.Ops, theme.Background, clip.Rect{Max: gtx.Constraints.Max}.Op())
	gtx.Constraints.Min = gtx.Constraints.Max

	toast := s.host.Activity.Toast()
	s.host.Mu.Lock()
	busy := s.host.Busy
	s.host.Mu.Unlock()

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return s.pages[s.current].Layout(gtx, th)
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			if toast == "" {
				return layout.Dimensions{}
			}
			return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(32)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(460)
					return components.Toast(gtx, th, toast, busy)
				})
			})
		}),
	)
}
