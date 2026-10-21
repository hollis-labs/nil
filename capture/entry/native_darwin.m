#import <Cocoa/Cocoa.h>
#import <Carbon/Carbon.h>
#import "native_darwin.h"

extern void nilEntryRequested(uintptr_t handle);

// AppKit owns the main event loop through Wails v2. Do not start another loop,
// replace NSApp's delegate, install event taps, or request accessibility access.
static void on_main(void (^work)(void)) {
    if ([NSThread isMainThread]) work();
    else dispatch_sync(dispatch_get_main_queue(), work);
}

@interface NilCaptureEntry : NSObject {
@public
    uintptr_t callback;
    EventHandlerRef handler;
    EventHotKeyRef hotkey;
    UInt32 hotkeyID;
    UInt32 keyCode;
    UInt32 modifiers;
    NSStatusItem *item;
}
- (void)openCapture:(id)sender;
@end
@implementation NilCaptureEntry
- (void)openCapture:(id)sender { nilEntryRequested(callback); }
@end

static OSStatus hotkey_pressed(EventHandlerCallRef next, EventRef event, void *data) {
    NilCaptureEntry *entry = (NilCaptureEntry *)data;
    EventHotKeyID key;
    OSStatus status = GetEventParameter(event, kEventParamDirectObject, typeEventHotKeyID,
                                       NULL, sizeof(key), NULL, &key);
    if (status != noErr || key.signature != 0x4e494c43 || key.id != entry->hotkeyID || !entry->hotkey)
        return eventNotHandledErr;
    nilEntryRequested(entry->callback);
    return noErr;
}

void *nil_entry_start(uintptr_t callback, int32_t *status) {
    __block NilCaptureEntry *entry = nil;
    on_main(^{
        entry = [[NilCaptureEntry alloc] init];
        entry->callback = callback;
        EventTypeSpec type = {kEventClassKeyboard, kEventHotKeyPressed};
        *status = InstallEventHandler(GetApplicationEventTarget(), hotkey_pressed, 1,
                                     &type, entry, &entry->handler);
        if (*status != noErr) { [entry release]; entry = nil; return; }
        entry->item = [[[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength] retain];
        entry->item.button.title = @"Nil";
        entry->item.button.toolTip = @"NIL Capture (global shortcut disabled)";
        [entry->item.button setAccessibilityLabel:@"NIL Capture"];
        NSMenu *menu = [[NSMenu alloc] initWithTitle:@"NIL Capture"];
        NSMenuItem *open = [[NSMenuItem alloc] initWithTitle:@"Open Capture" action:@selector(openCapture:) keyEquivalent:@""];
        open.target = entry;
        [menu addItem:open];
        entry->item.menu = menu;
        [open release];
        [menu release];
    });
    return entry;
}

int32_t nil_entry_configure(void *ptr, int enabled, uint32_t key, uint32_t modifiers) {
    __block OSStatus result = noErr;
    on_main(^{
        NilCaptureEntry *entry = (NilCaptureEntry *)ptr;
        if (!enabled) {
            if (entry->hotkey) {
                result = UnregisterEventHotKey(entry->hotkey);
                if (result != noErr) return;
                entry->hotkey = NULL;
            }
            entry->item.button.toolTip = @"NIL Capture (global shortcut disabled)";
            return;
        }
        if (entry->hotkey && entry->keyCode == key && entry->modifiers == modifiers) return;
        EventHotKeyRef replacement = NULL;
        UInt32 nextID = entry->hotkeyID + 1;
        EventHotKeyID identity = {0x4e494c43, nextID};
        result = RegisterEventHotKey(key, modifiers, identity, GetApplicationEventTarget(), 0, &replacement);
        if (result != noErr) return; // Existing owner and old registration untouched.
        if (entry->hotkey) {
            result = UnregisterEventHotKey(entry->hotkey);
            if (result != noErr) { UnregisterEventHotKey(replacement); return; }
        }
        entry->hotkey = replacement;
        entry->hotkeyID = nextID;
        entry->keyCode = key;
        entry->modifiers = modifiers;
        entry->item.button.toolTip = @"NIL Capture (global shortcut enabled)";
    });
    return result;
}

void nil_entry_close(void *ptr) {
    on_main(^{
        NilCaptureEntry *entry = (NilCaptureEntry *)ptr;
        if (entry->hotkey) { UnregisterEventHotKey(entry->hotkey); entry->hotkey = NULL; }
        if (entry->handler) { RemoveEventHandler(entry->handler); entry->handler = NULL; }
        entry->item.menu = nil;
        [[NSStatusBar systemStatusBar] removeStatusItem:entry->item];
        [entry->item release];
        [entry release];
    });
}
