package uitray

import (
	"context"
	"log"
	"os"
	"piggy-bank/platform/darwin"
	"piggy-bank/ui/state"

	"github.com/gogpu/systray"

	_ "embed"
)

// Embed the icon.ico (in the same folder with main.go)
//
//go:embed icon.ico
var iconBytes []byte

func SetupTray(ctx context.Context, host *state.HostState, tray *systray.SystemTray, onClickAdmin func()) {
	menu := systray.NewMenu()
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
}
