package uitray

import (
	"context"
	"log"
	"os"
	"piggy-bank/platform/darwin"
	"piggy-bank/ui/state"
	"time"

	"github.com/gogpu/systray"

	_ "embed"
)

// Embed the icon.ico (in the same folder with main.go)
//
//go:embed icon.ico
var iconBytes []byte

func SetupTray(ctx context.Context, host *state.HostState, tray *systray.SystemTray, onClickAdmin func()) {
	menu := systray.NewMenu()
	// This is a read-only status indicator. Its native checkbox mark shows a
	// connected tailnet without needing a separate icon.
	tailscaleItem := menu.AddCheckbox("Tailscale: Checking…", false, func() {})
	menu.AddSeparator()
	menu.Add("Open", onClickAdmin)

	isAutoStart := darwin.IsStartAtLoginEnabled()
	var autoStartItem *systray.MenuItem

	// 2. Tạo menu item checkbox
	autoStartItem = menu.AddCheckbox("Start application on login", isAutoStart, func() {
		go func() {
			if isAutoStart {
				if err := darwin.DisableStartAtLogin(); err != nil {
					log.Printf("[AutoStart] Disable error: %v", err)
					return
				}

				autoStartItem.SetChecked(false)
				isAutoStart = false
				log.Println("[AutoStart] Disabled")
			} else {
				if err := darwin.EnableStartAtLogin(); err != nil {
					log.Printf("[AutoStart] Enable error: %v", err)
					return
				}
				autoStartItem.SetChecked(true)
				isAutoStart = true
				log.Println("[AutoStart] Enabled")
			}
		}()
	})
	menu.Add("Quit", func() {
		os.Exit(0)
	})

	tray.
		SetTemplateIcon(iconBytes).
		SetTooltip("Piggy bank").
		SetMenu(menu).
		Show()

	go watchTailscaleStatus(ctx, host, tray, tailscaleItem)
}

func watchTailscaleStatus(ctx context.Context, host *state.HostState, tray *systray.SystemTray, item *systray.MenuItem) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		host.Mu.Lock()
		checked, cliReady, ready, message := host.TailscaleChecked, host.TailscaleCLIReady, host.TailscaleReady, host.TailscaleError
		host.Mu.Unlock()

		label, tooltip := "Tailscale: Checking…", "Piggy Bank\nTailscale: checking…"
		if checked && ready {
			label, tooltip = "Tailscale: Connected", "Piggy Bank\nTailscale: connected"
		} else if checked && !cliReady {
			label = "Tailscale CLI: Unavailable"
			tooltip = "Piggy Bank\nTailscale CLI unavailable: " + message
		} else if checked {
			label = "Tailscale: Not connected"
			tooltip = "Piggy Bank\nTailscale unavailable: " + message
		}
		item.SetLabel(label)
		item.SetChecked(checked && ready)
		tray.SetTooltip(tooltip)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
