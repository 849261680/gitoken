import AppKit
import Foundation
import SwiftUI

@MainActor
final class MenuBarViewModel: ObservableObject {

    struct HeatmapDay {
        let date: String
        let tokens: Int
        var level: Int {
            switch tokens {
            case 0:               return 0
            case 1..<1_000_000:   return 1
            case 1_000_000..<10_000_000:  return 2
            case 10_000_000..<100_000_000: return 3
            default:              return 4
            }
        }
    }

    struct ProviderSummary: Identifiable {
        let id: String
        let name: String
        let totalTokens: Int

        var tokensText: String { compactTokenString(totalTokens) }
    }

    @Published private(set) var todayTokens: Int = 0
    @Published private(set) var weeklyTokens: Int = 0
    @Published private(set) var primaryProvider: String = "—"
    @Published private(set) var heatmapDays: [HeatmapDay] = []
    @Published private(set) var lastUpdated: Date?
    @Published private(set) var isRefreshing = false
    @Published private(set) var scheduleInstalled = false
    @Published private(set) var lastError: String?
    @Published private var countdownNow = Date()

    private let cli = TokenHeatCLI()
    private var didStart = false
    private var refreshTask: Task<Void, Never>?
    private var countdownTask: Task<Void, Never>?
    var settings = SettingsStore()

    var menuTitle: String {
        todayTokens == 0 ? "热图" : "热图 \(compactTokenString(todayTokens))"
    }

    var todaySummary: String {
        todayTokens == 0 ? "暂无数据" : compactTokenString(todayTokens)
    }

    var weeklySummary: String {
        weeklyTokens == 0 ? "暂无数据" : compactTokenString(weeklyTokens)
    }

    var nextSyncSummary: String? {
        guard scheduleInstalled else { return nil }
        let interval = max(settings.syncInterval, 1)
        let scheduledAt = settings.syncScheduledAt > 0
            ? Date(timeIntervalSince1970: settings.syncScheduledAt)
            : Date()
        let elapsed = max(0, countdownNow.timeIntervalSince(scheduledAt))
        let remaining = interval - (Int(elapsed) % interval)
        return "下次同步：\(durationString(remaining))"
    }

    func start() {
        guard !didStart else { return }
        didStart = true
        refresh()
        startRefreshLoop()
        startCountdownLoop()
    }

    func restartRefreshLoop() {
        refreshTask?.cancel()
        startRefreshLoop()
    }

    private func startRefreshLoop() {
        let interval = settings.refreshInterval
        refreshTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(interval))
                self?.refresh()
            }
        }
    }

    private func startCountdownLoop() {
        countdownTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(1))
                self?.countdownNow = Date()
            }
        }
    }

    func refresh() {
        guard !isRefreshing else { return }
        isRefreshing = true
        lastError = nil

        Task {
            do {
                try await cli.collect()
                async let todayRows   = cli.todayReport()
                async let usageReport = cli.usageReport()
                async let scheduled   = cli.scheduleInstalled()

                let (rows, usage, sched) = try await (todayRows, usageReport, scheduled)

                todayTokens = rows.reduce(0) { $0 + $1.totalTokens }

                // primary provider (highest today)
                primaryProvider = rows.max(by: { $0.totalTokens < $1.totalTokens })
                    .map { $0.providerDisplayName } ?? "—"

                // weekly total from usage.json (last 7 days)
                let today = Calendar.current.startOfDay(for: Date())
                let sevenDaysAgo = Calendar.current.date(byAdding: .day, value: -6, to: today)!
                let fmt = DateFormatter()
                fmt.dateFormat = "yyyy-MM-dd"
                weeklyTokens = usage.rows
                    .filter { row in
                        guard let d = fmt.date(from: row.day) else { return false }
                        return d >= sevenDaysAgo
                    }
                    .reduce(0) { $0 + $1.totalTokens }

                // heatmap: last 98 days (14 weeks × 7)
                let ninetyEightDaysAgo = Calendar.current.date(byAdding: .day, value: -97, to: today)!
                let usageMap = Dictionary(uniqueKeysWithValues: usage.rows.map { ($0.day, $0.totalTokens) })
                var days: [HeatmapDay] = []
                for offset in 0..<98 {
                    let date = Calendar.current.date(byAdding: .day, value: offset, to: ninetyEightDaysAgo)!
                    let key = fmt.string(from: date)
                    days.append(HeatmapDay(date: key, tokens: usageMap[key] ?? 0))
                }
                heatmapDays = days

                scheduleInstalled = sched
                lastUpdated = Date()
            } catch {
                lastError = error.localizedDescription
            }
            isRefreshing = false
        }
    }

    func syncNow() {
        runAction { try await self.cli.runDaily() }
    }

    func setScheduleEnabled(_ enabled: Bool) {
        guard enabled != scheduleInstalled, !isRefreshing else { return }
        let previous = scheduleInstalled
        scheduleInstalled = enabled
        isRefreshing = true
        lastError = nil
        Task {
            do {
                if enabled {
                    try await cli.installSchedule(interval: settings.syncInterval)
                    settings.syncScheduledAt = Date().timeIntervalSince1970
                    countdownNow = Date()
                } else {
                    try await cli.removeSchedule()
                    settings.syncScheduledAt = 0
                }
                isRefreshing = false
                refresh()
            } catch {
                scheduleInstalled = previous
                lastError = error.localizedDescription
                isRefreshing = false
            }
        }
    }

    func setScheduleInterval(_ interval: Int) {
        settings.syncInterval = interval
        guard scheduleInstalled, !isRefreshing else { return }
        isRefreshing = true
        lastError = nil
        Task {
            do {
                try await cli.installSchedule(interval: settings.syncInterval)
                settings.syncScheduledAt = Date().timeIntervalSince1970
                countdownNow = Date()
                isRefreshing = false
                refresh()
            } catch {
                lastError = error.localizedDescription
                isRefreshing = false
            }
        }
    }

    private var settingsWindow: NSWindow?

    func openSettings() {
        if let existing = settingsWindow {
            existing.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
            return
        }
        let view = NSHostingView(rootView: SettingsView()
            .environmentObject(settings)
            .environmentObject(self))
        view.frame = NSRect(x: 0, y: 0, width: 300, height: 220)
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 300, height: 220),
            styleMask: [.titled, .closable],
            backing: .buffered,
            defer: false
        )
        window.title = "设置"
        window.contentView = view
        centerWindowOnCurrentScreen(window)
        window.isReleasedWhenClosed = false
        let delegate = SettingsWindowDelegate { [weak self] in
            self?.settingsWindow = nil
        }
        objc_setAssociatedObject(window, "delegate_owner", delegate, .OBJC_ASSOCIATION_RETAIN)
        window.delegate = delegate
        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        settingsWindow = window
    }

    func openHeatmap() {
        guard let s = cli.profileURLString, let url = URL(string: s) else { return }
        NSWorkspace.shared.open(url)
    }

    func quit() { NSApplication.shared.terminate(nil) }

    private func centerWindowOnCurrentScreen(_ window: NSWindow) {
        let mouseLocation = NSEvent.mouseLocation
        let screen = NSScreen.screens.first { NSMouseInRect(mouseLocation, $0.frame, false) } ?? NSScreen.main

        guard let visibleFrame = screen?.visibleFrame else {
            window.center()
            return
        }

        let size = window.frame.size
        let origin = NSPoint(
            x: visibleFrame.midX - size.width / 2,
            y: visibleFrame.midY - size.height / 2
        )
        window.setFrameOrigin(origin)
    }

    private func runAction(_ op: @escaping () async throws -> Void) {
        guard !isRefreshing else { return }
        isRefreshing = true
        lastError = nil
        Task {
            do {
                try await op()
                isRefreshing = false
                refresh()
            } catch {
                lastError = error.localizedDescription
                isRefreshing = false
            }
        }
    }
}

private func compactTokenString(_ value: Int) -> String {
    let n = Double(value)
    if n >= 1_000_000_000 { return String(format: "%.1fB", n / 1_000_000_000) }
    if n >= 1_000_000     { return String(format: "%.1fM", n / 1_000_000) }
    if n >= 1_000         { return String(format: "%.1fK", n / 1_000) }
    return "\(value)"
}

private func durationString(_ seconds: Int) -> String {
    let seconds = max(seconds, 0)
    let hours = seconds / 3600
    let minutes = (seconds % 3600) / 60
    let remainingSeconds = seconds % 60
    return String(format: "%02d:%02d:%02d", hours, minutes, remainingSeconds)
}

private final class SettingsWindowDelegate: NSObject, NSWindowDelegate {
    let onClose: () -> Void
    init(onClose: @escaping () -> Void) { self.onClose = onClose }
    func windowWillClose(_ notification: Notification) { onClose() }
}
