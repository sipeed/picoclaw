import AppKit
import SwiftUI

@MainActor
class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusItem: NSStatusItem!
    private var popover: NSPopover!
    private var engine: MonitorEngine!
    private var eventMonitor: Any?
    private var pulseTimer: Timer?

    func applicationDidFinishLaunching(_ notification: Notification) {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)

        if let button = statusItem.button {
            button.image = NSImage(systemSymbolName: "brain.head.profile",
                                   accessibilityDescription: "PicoWatch")
            button.action = #selector(togglePopover)
            button.target = self
        }

        engine = MonitorEngine()

        popover = NSPopover()
        popover.contentSize = NSSize(width: 380, height: 720)
        popover.behavior = .transient
        popover.contentViewController = NSHostingController(
            rootView: PopoverView(engine: engine)
        )

        NSUserNotificationCenter.default.delegate = self

        engine.start()

        // Pulse the status bar icon to show activity
        pulseTimer = Timer.scheduledTimer(withTimeInterval: 5.0, repeats: true) { [weak self] _ in
            Task { @MainActor in
                self?.updateStatusIcon()
            }
        }

        eventMonitor = NSEvent.addGlobalMonitorForEvents(matching: [.leftMouseDown, .rightMouseDown]) { [weak self] _ in
            if let popover = self?.popover, popover.isShown {
                popover.performClose(nil)
            }
        }
    }

    private func updateStatusIcon() {
        guard let button = statusItem.button else { return }
        let symbolName: String
        if engine.gatewayUp {
            symbolName = engine.recentSkillActivity ? "brain.head.profile.fill" : "brain.head.profile"
        } else {
            symbolName = "brain.head.profile"
        }
        button.image = NSImage(systemSymbolName: symbolName,
                               accessibilityDescription: "PicoWatch")
        // Show skill count as badge
        let skillCount = engine.totalSkills
        button.title = skillCount > 0 ? " \(skillCount)" : ""
    }

    @objc private func togglePopover() {
        if let button = statusItem.button {
            if popover.isShown {
                popover.performClose(nil)
            } else {
                popover.show(relativeTo: button.bounds, of: button, preferredEdge: .minY)
                NSApp.activate(ignoringOtherApps: true)
            }
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        pulseTimer?.invalidate()
        engine.stop()
        if let monitor = eventMonitor {
            NSEvent.removeMonitor(monitor)
        }
    }
}

extension AppDelegate: NSUserNotificationCenterDelegate {
    func userNotificationCenter(_ center: NSUserNotificationCenter,
                                shouldPresent notification: NSUserNotification) -> Bool {
        return true
    }
}
