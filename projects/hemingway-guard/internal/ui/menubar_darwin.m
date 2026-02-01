#import <Cocoa/Cocoa.h>

// Callback declarations - implemented in Go
extern void goMenuItemClicked(int tag);

// Menu item callback
@interface MenuDelegate : NSObject
@end

@implementation MenuDelegate
- (void)menuItemClicked:(NSMenuItem *)sender {
    goMenuItemClicked((int)[sender tag]);
}
@end

static NSStatusItem *statusItem = nil;
static MenuDelegate *menuDelegate = nil;

void createStatusItem(const char* title) {
    if (statusItem != nil) return;

    dispatch_async(dispatch_get_main_queue(), ^{
        statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
        [statusItem.button setTitle:[NSString stringWithUTF8String:title]];

        menuDelegate = [[MenuDelegate alloc] init];

        NSMenu *menu = [[NSMenu alloc] init];

        NSMenuItem *enableItem = [[NSMenuItem alloc] initWithTitle:@"Enabled"
                                                            action:@selector(menuItemClicked:)
                                                     keyEquivalent:@""];
        [enableItem setTarget:menuDelegate];
        [enableItem setTag:1];
        [enableItem setState:NSControlStateValueOn];
        [menu addItem:enableItem];

        [menu addItem:[NSMenuItem separatorItem]];

        NSMenuItem *settingsItem = [[NSMenuItem alloc] initWithTitle:@"Settings..."
                                                              action:@selector(menuItemClicked:)
                                                       keyEquivalent:@","];
        [settingsItem setTarget:menuDelegate];
        [settingsItem setTag:2];
        [menu addItem:settingsItem];

        [menu addItem:[NSMenuItem separatorItem]];

        NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit"
                                                          action:@selector(menuItemClicked:)
                                                   keyEquivalent:@"q"];
        [quitItem setTarget:menuDelegate];
        [quitItem setTag:3];
        [menu addItem:quitItem];

        statusItem.menu = menu;
    });
}

void setStatusItemTitle(const char* title) {
    if (statusItem == nil) return;

    dispatch_async(dispatch_get_main_queue(), ^{
        [statusItem.button setTitle:[NSString stringWithUTF8String:title]];
    });
}

void setEnabledState(int enabled) {
    if (statusItem == nil) return;

    dispatch_async(dispatch_get_main_queue(), ^{
        NSMenuItem *enableItem = [statusItem.menu itemWithTag:1];
        [enableItem setState:enabled ? NSControlStateValueOn : NSControlStateValueOff];
    });
}

void removeStatusItem(void) {
    if (statusItem == nil) return;

    dispatch_async(dispatch_get_main_queue(), ^{
        [[NSStatusBar systemStatusBar] removeStatusItem:statusItem];
        statusItem = nil;
    });
}
