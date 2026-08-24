package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

// Con trỏ lưu lại hàm gốc của macOS
static IMP original_setActivationPolicy;

static void InitNSApp() {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}

// HÀM GIẢ MẠO: Bất kể ai (kể cả Gio UI) gọi đổi Policy, ta đều ép nó về Accessory (1)
BOOL hook_setActivationPolicy(id self, SEL _cmd, NSApplicationActivationPolicy policy) {
    BOOL (*original)(id, SEL, NSApplicationActivationPolicy) = (void *)original_setActivationPolicy;
    // Bỏ qua biến policy được truyền vào, luôn bắt macOS chạy Accessory
    return original(self, _cmd, NSApplicationActivationPolicyAccessory);
}

// Hàm này sẽ đánh tráo hàm gốc của hệ điều hành
void forceAccessoryForever() {
    Method method = class_getInstanceMethod([NSApplication class], @selector(setActivationPolicy:));
    original_setActivationPolicy = method_getImplementation(method);
    method_setImplementation(method, (IMP)hook_setActivationPolicy);

    // Ép policy ngay lúc này luôn để chắc chắn
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    });
}

// Ép macOS focus vào cửa sổ (vì app ẩn Dock sẽ bị mất khả năng tự focus)
void forceActivateApp() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp activateIgnoringOtherApps:YES];
    });
}

#include "center_mac.h"
*/
import "C"
import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"
	"piggy-bank/assets/fonts"
	"piggy-bank/config"
	"piggy-bank/ui/layouts"
	"piggy-bank/ui/state"
	uitray "piggy-bank/ui/tray"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/gogpu/systray"
)

var (
	winMutex  sync.Mutex
	activeWin *app.Window
)

const (
	APP_TITLE = "Piggy Bank"
)

func init() {
	// force MacOS to start AppKit/CoreGraphics context
	// before connecting to the window server
	C.InitNSApp()
}

func main() {
	C.forceAccessoryForever()
	C.installWindowCentering()

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	host := &state.HostState{
		Name:         "Main Server (Yuu, Kubernetes, WSL, Window Server)",
		Address:      "Yuu.local",
		ServerSignal: make(chan bool),
	}

	go host.HandleServerSignal(appCtx)

	tray := systray.New()
	uitray.SetupTray(appCtx, host, tray, func() {
		openGioWindow(host)
	})

	// Chạy vòng lặp systray bên trong goroutine này
	if err := tray.Run(); err != nil {
		fmt.Println("Error of systray:", err)
	}
}

// openGioWindow bring the Gio window to the front
func openGioWindow(host *state.HostState) {
	winMutex.Lock()

	// if the window is create, bring it to the front
	if activeWin != nil {
		activeWin.Perform(system.ActionRaise)
		winMutex.Unlock()

		// focus on the application, because this is the tray application
		C.forceActivateApp()
		return
	}

	// if not, create a new window
	w := new(app.Window)
	w.Option(
		app.Decorated(false),
		app.Size(unit.Dp(820), unit.Dp(404)),
		app.MinSize(unit.Dp(820), unit.Dp(404)),
		app.MaxSize(unit.Dp(820), unit.Dp(404)),
	)

	activeWin = w
	winMutex.Unlock()

	// focus
	C.forceActivateApp()

	// loop to render window
	go func() {
		if err := run(w, host); err != nil {
			log.Println("Window closed with error:", err)
		}

		winMutex.Lock()
		activeWin = nil
		winMutex.Unlock()
	}()
}

func run(w *app.Window, host *state.HostState) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var quitButton widget.Clickable

	th := material.NewTheme()
	th.Shaper = fonts.NewShaper()
	var ops op.Ops

	if err := config.EnsureConfig(); err != nil {
		log.Fatal(err)
	}

	if err := config.Load(); err != nil {
		log.Fatal(err)
	}

	singlePageApp := layouts.NewSinglePageApp(host)

	// background worker to check the hosts
	go fetchServerUI(ctx, host, w)
	go fetchUI(ctx, w)
	go startLogListener(w, singlePageApp)

	// handle frame events and other events
	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			cancel()
			return nil
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// set the layout
			layout.Inset{
				Top: unit.Dp(32),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return singlePageApp.Layout(gtx, th)
			})

			if quitButton.Clicked(gtx) {
				w.Perform(system.ActionClose)
			}

			// 3. Draw the custom black titlebar
			drawCustomTitleBar(gtx, th, &quitButton)

			// draw frame to the gpu
			e.Frame(gtx.Ops)
		}
	}
}

func drawCustomTitleBar(gtx layout.Context, th *material.Theme, quitBtn *widget.Clickable) {
	height := gtx.Dp(unit.Dp(32))
	bounds := image.Rect(0, 0, gtx.Constraints.Max.X, height)

	// 1. Vẽ nền đen cho title bar
	area := clip.Rect(bounds).Push(gtx.Ops)
	darkGray := color.NRGBA{R: 44, G: 47, B: 51, A: 255}
	paint.ColorOp{Color: darkGray}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	// Cho phép kéo cửa sổ từ phần nền trống
	system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
	area.Pop()

	// 2. Dùng Stack để quản lý vị trí các phần tử
	layout.Stack{}.Layout(gtx,
		// Tiêu đề căn giữa tuyệt đối
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			// Ép vùng chứa tiêu đề rộng bằng toàn bộ màn hình để layout.Center tính đúng điểm giữa
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			gtx.Constraints.Min.Y = height
			gtx.Constraints.Max.Y = height

			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Body1(th, APP_TITLE)
				label.TextSize = unit.Sp(14)
				label.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
				return label.Layout(gtx)
			})
		}),
		// Red quit button
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top:  unit.Dp(10),
				Left: unit.Dp(10),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				size := gtx.Dp(unit.Dp(12))
				gtx.Constraints.Min = image.Point{X: size, Y: size}
				gtx.Constraints.Max = gtx.Constraints.Min

				btn := material.Button(th, quitBtn, "")
				btn.Background = color.NRGBA{R: 255, G: 95, B: 86, A: 255} // Màu đỏ macOS
				btn.CornerRadius = unit.Dp(6)                              // Bo tròn hoàn toàn
				return btn.Layout(gtx)
			})
		}),
	)
}

func fetchServerUI(ctx context.Context, host *state.HostState, w *app.Window) {
	host.Mu.Lock()
	currentHostStatus := host.IsOnline
	host.Mu.Unlock()
	for {
		select {
		case <-ctx.Done():
			return

		default:
			time.Sleep(3 * time.Second)
			host.Mu.Lock()
			if currentHostStatus == host.IsOnline {
				host.Mu.Unlock()
				continue
			}

			currentHostStatus = host.IsOnline
			host.Mu.Unlock()
			fmt.Println("Invalidate UI")
			w.Invalidate()
		}
	}
}

func fetchUI(ctx context.Context, w *app.Window) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(2 * time.Second)
			w.Invalidate()
		}
	}
}

// a listener channel to handle the logs on the UI
func startLogListener(window *app.Window, pageApp *layouts.SinglePageApp) {
	go func() {
		for msg := range pageApp.LogChan {
			if msg == "" {
				pageApp.ShowLogBar = false
			} else {
				pageApp.DisplayedLogMsg = msg
				pageApp.ShowLogBar = true
			}

			window.Invalidate()
		}

		pageApp.ShowLogBar = false
		if window != nil {
			window.Invalidate()
		}
	}()
}
