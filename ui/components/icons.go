package components

import (
	"log"

	"gioui.org/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

var (
	IconHost     = mustIcon(icons.HardwareComputer)
	IconWSL      = mustIcon(icons.ActionDNS)
	IconSettings = mustIcon(icons.ActionSettings)
	IconHistory  = mustIcon(icons.ActionHistory)
	IconInfo     = mustIcon(icons.ActionInfo)
	IconPower    = mustIcon(icons.ActionPowerSettingsNew)
	IconBack     = mustIcon(icons.NavigationArrowBack)
	IconBoard    = mustIcon(icons.HardwareDeveloperBoard)
	IconDelete   = mustIcon(icons.ActionDelete)
	IconPlay     = mustIcon(icons.AVPlayArrow)
	IconStop     = mustIcon(icons.AVStop)
)

func mustIcon(data []byte) *widget.Icon {
	ic, err := widget.NewIcon(data)
	if err != nil {
		log.Fatalf("Error while loading icon: %v", err)
	}
	return ic
}
