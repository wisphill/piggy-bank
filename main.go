package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa -framework QuartzCore
#import <Cocoa/Cocoa.h>
#import <QuartzCore/CAMetalLayer.h>
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

// Gio leaves the Metal layer untagged, so macOS shows its sRGB colours in
// the display's native colour space (e.g. Display P3) and they look off.
static void setLayerSRGB(uintptr_t layerRef) {
    CALayer *layer = (__bridge CALayer *)(void *)layerRef;
    if ([layer isKindOfClass:[CAMetalLayer class]]) {
        CGColorSpaceRef cs = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
        ((CAMetalLayer *)layer).colorspace = cs;
        CGColorSpaceRelease(cs);
    }
}

#include "center_mac.h"
*/
import "C"
import (
	"context"
	"fmt"
	"log"
	"piggy-bank/assets/fonts"
	"piggy-bank/config"
	"piggy-bank/ui/layouts"
	"piggy-bank/ui/state"
	"piggy-bank/ui/theme"
	uitray "piggy-bank/ui/tray"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/gogpu/systray"
)

var (
	winMutex  sync.Mutex
	activeWin *app.Window
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
		Name:         "Main Server",
		Description:  "Yuu · Kubernetes · WSL · Windows Server",
		Address:      "Yuu.local",
		Activity:     &state.ActivityLog{},
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
	if w := activeWin; w != nil {
		winMutex.Unlock()

		// Perform blocks until the main thread runs it, and tray callbacks
		// run on the main thread, so it must not be called synchronously here.
		go w.Perform(system.ActionRaise)

		// focus on the application, because this is the tray application
		C.forceActivateApp()
		return
	}

	// if not, create a new window
	w := new(app.Window)
	w.Option(
		app.Decorated(true),
		app.Title("Piggy Bank"),
		app.Size(unit.Dp(900), unit.Dp(560)),
		app.MinSize(unit.Dp(720), unit.Dp(465)),
		app.MaxSize(unit.Dp(1440), unit.Dp(930)),
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

	th := theme.New(fonts.NewShaper())
	var ops op.Ops

	if err := config.EnsureConfig(); err != nil {
		log.Fatal(err)
	}

	if err := config.Load(); err != nil {
		log.Fatal(err)
	}

	shell := layouts.NewAppShell(host)
	host.Activity.SetInvalidator(w.Invalidate)
	defer host.Activity.SetInvalidator(nil)

	// background worker to check the hosts
	go host.PingToServerLoop(ctx)
	go host.FetchWSLNodesLoop(ctx)
	go fetchServerUI(ctx, host, w)
	go fetchUI(ctx, w)

	// handle frame events and other events
	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			cancel()
			return nil
		case app.AppKitViewEvent:
			if e.View == 0 {
				// The window was closed. Gio only sends DestroyEvent once the view
				// is deallocated, which may never happen, and a closed window
				// cannot be raised again. Treat this as the end of the window and
				// keep draining its events so Gio never blocks the main thread.
				go drainEvents(w)
				return nil
			}
			if e.Layer != 0 {
				C.setLayerSRGB(C.uintptr_t(e.Layer))
			}
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			shell.Layout(gtx, th)

			// draw frame to the gpu
			e.Frame(gtx.Ops)
		}
	}
}

func drainEvents(w *app.Window) {
	for {
		if _, ok := w.Event().(app.DestroyEvent); ok {
			return
		}
	}
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
