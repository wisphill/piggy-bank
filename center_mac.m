#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#import "center_mac.h"

static void center(NSWindow *window) {
    NSScreen *screen = window.screen ?: NSScreen.mainScreen;

    if (!screen) {
        return;
    }

    NSRect screenFrame = screen.visibleFrame;
    NSRect windowFrame = window.frame;

    CGFloat x = NSMidX(screenFrame) - NSWidth(windowFrame) / 2.0;
    CGFloat y = NSMidY(screenFrame) - NSHeight(windowFrame) / 2.0;

    [window setFrameOrigin:NSMakePoint(x, y)];
}

@implementation NSWindow (LaborCentering)

- (void)labor_orderFront:(id)sender {
    center(self);
    [self labor_orderFront:sender];
}

- (void)labor_makeKeyAndOrderFront:(id)sender {
    center(self);
    [self labor_makeKeyAndOrderFront:sender];
}

@end

void installWindowCentering(void) {
    static dispatch_once_t onceToken;

    dispatch_once(&onceToken, ^{
        Method orderFrontOriginal =
            class_getInstanceMethod(
                NSWindow.class,
                @selector(orderFront:)
            );

        Method orderFrontSwizzled =
            class_getInstanceMethod(
                NSWindow.class,
                @selector(labor_orderFront:)
            );

        method_exchangeImplementations(
            orderFrontOriginal,
            orderFrontSwizzled
        );

        Method makeKeyOriginal =
            class_getInstanceMethod(
                NSWindow.class,
                @selector(makeKeyAndOrderFront:)
            );

        Method makeKeySwizzled =
            class_getInstanceMethod(
                NSWindow.class,
                @selector(labor_makeKeyAndOrderFront:)
            );

        method_exchangeImplementations(
            makeKeyOriginal,
            makeKeySwizzled
        );
    });
}