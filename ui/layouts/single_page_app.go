package layouts

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"piggy-bank/assets"
	server "piggy-bank/cmd/host"
	"piggy-bank/ui/components"
	"piggy-bank/ui/state"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	_ "embed"
)

type SinglePageApp struct {
	host              *state.HostState
	list              widget.List
	shutdownIcon      *components.SVGRenderer
	redShutdownIcon   *components.SVGRenderer
	greenShutdownIcon *components.SVGRenderer
	serverIcon        *components.SVGRenderer

	LogChan         chan string
	DisplayedLogMsg string
	ShowLogBar      bool
	wslList         layout.List
}

func NewSinglePageApp(host *state.HostState) *SinglePageApp {
	shutdownIcon, err := components.LoadSVG(assets.ShutdownSVG, 24, 24, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	if err != nil {
		log.Fatalf("Error while loading SVG: %v", err)
	}

	redShutdownIcon, err := components.LoadSVG(assets.ShutdownSVG, 24, 24, color.NRGBA{R: 231, G: 76, B: 60, A: 255})
	if err != nil {
		log.Fatalf("Error while loading SVG: %v", err)
	}

	greenShutdownIcon, err := components.LoadSVG(assets.ShutdownSVG, 24, 24, color.NRGBA{R: 40, G: 160, B: 60, A: 255})
	if err != nil {
		log.Fatalf("Error while loading SVG: %v", err)
	}

	serverIcon, err := components.LoadSVG(assets.ServerSVG, 40, 40, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	if err != nil {
		log.Fatalf("Error while loading SVG: %v", err)
	}

	return &SinglePageApp{
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		host:              host,
		shutdownIcon:      shutdownIcon,
		redShutdownIcon:   redShutdownIcon,
		greenShutdownIcon: greenShutdownIcon,
		serverIcon:        serverIcon,
		ShowLogBar:        false,
		LogChan:           make(chan string),
	}
}

func (app *SinglePageApp) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// Ví dụ: Đặt nút Shutdown ở góc dưới bên phải màn hình mà không ảnh hưởng tới danh sách
	return layout.Stack{}.Layout(gtx,
		// Lớp 1: Danh sách hiện tại của bạn
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			return app.LayoutMainContent(gtx, th)
		}),

		// Lớp 2: Thành phần nổi đặt ở góc dưới bên phải (Expanded / SE - South East)
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			if !app.ShowLogBar {
				return layout.Dimensions{}
			}
			// layout.S giúp neo component ở sát viền đáy (South)
			return layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return app.layoutBottomLogBar(gtx, th)
			})
		}),
	)
}

// Bảng Log đè lên dưới đáy
func (app *SinglePageApp) layoutBottomLogBar(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// Cách lề xung quanh bảng log
	return layout.Inset{
		Bottom: unit.Dp(10),
		Left:   unit.Dp(300),
		Right:  unit.Dp(300),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Nền đen tuyệt đối cho Terminal
		bgColor := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
		macro := op.Record(gtx.Ops)

		dims := widget.Border{
			Color:        color.NRGBA{R: 50, G: 50, B: 50, A: 255}, // Viền xám tối
			Width:        unit.Dp(1),
			CornerRadius: unit.Dp(8),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {

			return layout.Inset{
				Top:    unit.Dp(10),
				Bottom: unit.Dp(10),
				Left:   unit.Dp(14),
				Right:  unit.Dp(14),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {

				// Bố cục ngang cho dòng log terminal
				return layout.Flex{
					Axis:      layout.Horizontal,
					Alignment: layout.Middle,
				}.Layout(gtx,
					// Dấu nhắc (Prompt) của terminal
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(th, "$ ")
						lbl.Color = color.NRGBA{R: 0, G: 255, B: 0, A: 255} // Xanh Terminal
						lbl.Font.Typeface = "IBM Plex Mono"                 // Ưu tiên font này, thiếu sẽ fallback
						lbl.Font.Weight = font.Bold
						return lbl.Layout(gtx)
					}),
					// Nội dung log
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(th, app.DisplayedLogMsg)
						lbl.Color = color.NRGBA{R: 0, G: 255, B: 0, A: 255} // Xanh Terminal
						lbl.Font.Typeface = "IBM Plex Mono"                 // Ưu tiên font này, thiếu sẽ fallback
						return lbl.Layout(gtx)
					}),
				)
			})
		})

		// Vẽ nền đen bo góc phía dưới
		call := macro.Stop()
		rr := gtx.Dp(unit.Dp(8))
		paint.FillShape(
			gtx.Ops,
			bgColor,
			clip.RRect{
				Rect: image.Rectangle{Max: dims.Size},
				NE:   rr, NW: rr, SE: rr, SW: rr,
			}.Op(gtx.Ops),
		)
		call.Add(gtx.Ops)

		return dims
	})
}

func (app *SinglePageApp) LayoutMainContent(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// setting global configs for the layout
	app.wslList.Axis = layout.Horizontal

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.List(th, &app.list).Layout(
				gtx,
				1,
				func(gtx layout.Context, i int) layout.Dimensions {
					return app.layoutHostRow(gtx, th, app.host)
				},
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			wslNodesLen := len(app.host.Wsls)
			if wslNodesLen == 0 {
				return layout.Dimensions{}
			}

			return layout.Inset{
				Top:  unit.Dp(16),
				Left: unit.Dp(16),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(th, "WSL Instances")
				lbl.TextSize = 16
				lbl.Font.Weight = font.Bold
				return lbl.Layout(gtx)
			})
		}),
		// WSL nodes - horizontal
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return app.layoutWSLNodes(gtx, th)
		}),
	)
}

// Layout vẽ từng dòng Server
func (app *SinglePageApp) layoutHostRow(gtx layout.Context, th *material.Theme, host *state.HostState) layout.Dimensions {
	host.Mu.Lock()
	isOnline := host.IsOnline
	rtt := host.PingRTT
	name := host.Name
	address := host.Address
	host.Mu.Unlock()

	isPowerButtonClicked := host.BtnPower.Clicked(gtx) // consume the event
	if isPowerButtonClicked {
		if isOnline {
			go app.WaitForServerShutdown()
		} else if !isOnline {
			go app.WaitForServerStart()
		}
	}

	layoutMainRowContent := func(gtx layout.Context, th *material.Theme, host *state.HostState) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return components.DrawStatusBadge(gtx, isOnline)
			}),

			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Right: unit.Dp(10),
					Left:  unit.Dp(10),
				}.Layout(gtx, app.serverIcon.Layout)
			}),

			// Name + IP
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{
							Axis:      layout.Horizontal,
							Alignment: layout.Middle,
						}.Layout(gtx,
							// Server name
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body2(th, name)
								lbl.Font.Weight = font.Bold
								lbl.Font.Typeface = "Google Sans"
								lbl.TextSize = unit.Sp(14)
								lbl.LineHeight = unit.Sp(15)
								return lbl.Layout(gtx)
							}),
						)
					}),

					// IP address
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Top: unit.Dp(0),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Caption(th, address)
							lbl.Color = color.NRGBA{
								R: 130,
								G: 130,
								B: 130,
								A: 255,
							}
							lbl.LineHeight = unit.Sp(14)

							return lbl.Layout(gtx)
						})
					}),
				)
			}),

			// C. Ping RTT
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				pingStr := "Timeout"
				pingColor := color.NRGBA{R: 220, G: 50, B: 50, A: 255} // Red

				if isOnline {
					pingStr = fmt.Sprintf("%d ms", rtt.Milliseconds())
					pingColor = color.NRGBA{R: 40, G: 160, B: 60, A: 255} // Green
				}

				lbl := material.Body2(th, pingStr)
				lbl.Color = pingColor
				lbl.TextSize = unit.Sp(12)
				return layout.Inset{Right: unit.Dp(16)}.Layout(gtx, lbl.Layout)
			}),

			// D. power button
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.ButtonLayout(th, &host.BtnPower)

				if isOnline {
					btn.Background = color.NRGBA{R: 231, G: 76, B: 60, A: 255}
				} else {
					btn.Background = color.NRGBA{R: 46, G: 204, B: 113, A: 255}
				}

				circuitText := "Shutdown"
				if !isOnline {
					circuitText = "Turn on"
				}

				return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{
							Alignment: layout.Middle,
						}.Layout(gtx,
							layout.Rigid(layout.Spacer{
								Width: unit.Dp(6),
							}.Layout),
							// Icon
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{
									Right: unit.Dp(4),
								}.Layout(gtx, app.shutdownIcon.Layout)
							}),

							// Text
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{
									Top: unit.Dp(0),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									lbl := material.Body2(th, circuitText)
									lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
									lbl.Font.Weight = font.Medium

									return lbl.Layout(gtx)
								})
							}),
							layout.Rigid(layout.Spacer{
								Width: unit.Dp(6),
							}.Layout),
						)
					})
				})
			}),
		)
	}

	return layout.Inset{
		Top:   unit.Dp(16),
		Right: unit.Dp(8),
		Left:  unit.Dp(16),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return widget.Border{
			Color: color.NRGBA{
				R: 220,
				G: 220,
				B: 220,
				A: 255,
			},
			Width:        unit.Dp(1),
			CornerRadius: unit.Dp(6),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top:    unit.Dp(16),
				Right:  unit.Dp(16),
				Left:   unit.Dp(16),
				Bottom: unit.Dp(16),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layoutMainRowContent(gtx, th, host)
			})
		})
	})
}

func (app *SinglePageApp) layoutWSLNodes(gtx layout.Context, th *material.Theme) layout.Dimensions {
	allWslNodes := make([]*state.WSLState, 0)
	allWslNodes = append(allWslNodes, app.host.Wsls...)

	return layout.Inset{
		Left: unit.Dp(16),
		Top:  unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return app.wslList.Layout(
			gtx,
			len(allWslNodes),
			func(gtx layout.Context, i int) layout.Dimensions {
				return app.layoutWSLNode(gtx, th, allWslNodes[i])
			},
		)
	})
}

func (app *SinglePageApp) layoutWSLNode(
	gtx layout.Context,
	th *material.Theme,
	wslNodeState *state.WSLState,
) layout.Dimensions {
	cardWidth := gtx.Dp(unit.Dp(170))
	gtx.Constraints.Min.X = cardWidth
	gtx.Constraints.Max.X = cardWidth

	isPowerButtonClicked := wslNodeState.BtnPower.Clicked(gtx) // consume the event
	if isPowerButtonClicked {
		fmt.Println("Status of the node ", wslNodeState.Status)
		if wslNodeState.Status == "Running" {
			go server.TurnOffWSLNode(wslNodeState.Name)
		} else if wslNodeState.Status == "Stopped" {
			go server.TurnOnWSLNode(wslNodeState.Name)
		}
	}

	return layout.Inset{
		Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return widget.Border{
			Color: color.NRGBA{
				R: 220,
				G: 220,
				B: 220,
				A: 255,
			},
			Width:        unit.Dp(1),
			CornerRadius: unit.Dp(6),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {

			return layout.Inset{
				Top:    unit.Dp(8),
				Bottom: unit.Dp(8),
				Left:   unit.Dp(12),
				Right:  unit.Dp(12),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {

				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(
					gtx,

					// Name
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(
							gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body2(th, wslNodeState.Name)
								lbl.TextSize = unit.Sp(14)
								lbl.Font.Weight = font.Medium

								return lbl.Layout(gtx)
							}),
						)
					}),

					// Status
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Caption(th, fmt.Sprintf(`%s • WSL 2`, wslNodeState.Status))
						lbl.TextSize = unit.Sp(12)

						return layout.Inset{
							Top: unit.Dp(0),
						}.Layout(gtx, lbl.Layout)
					}),

					// D. Power Button
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Top: unit.Dp(6),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								btn := material.ButtonLayout(
									th,
									&wslNodeState.BtnPower,
								)

								// white
								btn.Background = color.NRGBA{R: 231, G: 76, B: 60, A: 0}
								buttonLayout := app.redShutdownIcon.Layout
								if wslNodeState.Status == "Stopped" {
									buttonLayout = app.greenShutdownIcon.Layout
								}

								return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.UniformInset(unit.Dp(5)).Layout(
										gtx,
										buttonLayout,
									)
								})
							})
						})
					}),
				)
			})
		})
	})
}

func (app *SinglePageApp) WaitForServerShutdown() {
	app.LogChan <- "Server is shutting down!"
	server.TurnOffServer()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	pingTimeout := 1 * time.Second

	for {
		select {
		case <-ticker.C:
			app.LogChan <- "Server is shutting down! Checking the server"
			isOnline, _ := server.PingOS("yuu", time.Duration(pingTimeout))
			if !isOnline {
				ticker.Stop()

				app.LogChan <- "Server is shutted down!"
				time.Sleep(time.Duration(2 * time.Second))
				app.LogChan <- ""
				return
			}
		}
	}
}

func (app *SinglePageApp) WaitForServerStart() {
	app.LogChan <- "Server is starting!"
	err := server.TurnOnServer()
	if err != nil {
		app.LogChan <- fmt.Sprintf("Error while starting the server %v", err)
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	pingTimeout := 1 * time.Second

	for {
		select {
		case <-ticker.C:
			app.LogChan <- "Server is turning on! Checking the server"
			isOnline, _ := server.PingOS("yuu", time.Duration(pingTimeout))
			if isOnline {
				ticker.Stop()
				app.LogChan <- "Server is turned on!"
				time.Sleep(time.Duration(2 * time.Second))
				app.LogChan <- ""
				return
			}
		}
	}
}
