import AppKit
import Foundation
import SwiftUI

@MainActor
final class MenuBarViewModel: ObservableObject {
    struct ProviderSummary: Identifiable {
        let id: String
        let name: String
        let totalTokens: Int

        var tokensText: String {
            compactTokenString(totalTokens)
        }

        var accentColor: Color {
            switch id {
            case "codex":
                return Color(red: 0.12, green: 0.48, blue: 0.98)
            case "claude":
                return Color(red: 0.09, green: 0.67, blue: 0.56)
            case "opencode":
                return Color(red: 0.42, green: 0.39, blue: 0.95)
            default:
                return .accentColor
            }
        }
    }

    @Published private(set) var providerSummaries: [ProviderSummary] = []
    @Published private(set) var totalTokens: Int = 0
    @Published private(set) var lastUpdated: Date?
    @Published private(set) var isRefreshing = false
    @Published private(set) var scheduleInstalled = false
    @Published private(set) var lastError: String?

    private let cli = TokenHeatCLI()
    private var didStart = false
    private var refreshTask: Task<Void, Never>?

    var menuTitle: String {
        if totalTokens == 0 {
            return "热图"
        }
        return "热图 \(compactTokenString(totalTokens))"
    }

    var totalSummary: String {
        if totalTokens == 0 {
            return "暂无数据"
        }
        return compactTokenString(totalTokens)
    }

    var lastUpdatedSummary: String {
        guard let lastUpdated else { return "尚未刷新" }
        let formatter = RelativeDateTimeFormatter()
        formatter.locale = Locale(identifier: "zh_CN")
        return formatter.localizedString(for: lastUpdated, relativeTo: Date())
    }

    var scheduleStatusText: String {
        scheduleInstalled ? "已开启" : "未开启"
    }

    var scheduleDetailText: String {
        "00:05 自动同步"
    }

    func start() {
        guard !didStart else { return }
        didStart = true
        refresh()
        refreshTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(120))
                self?.refresh()
            }
        }
    }

    func refresh() {
        guard !isRefreshing else { return }
        isRefreshing = true
        lastError = nil

        Task {
            do {
                async let report = cli.todayReport()
                async let scheduleInstalled = cli.scheduleInstalled()
                let (rows, schedule) = try await (report, scheduleInstalled)

                providerSummaries = rows
                    .map { row in
                        ProviderSummary(
                            id: row.provider,
                            name: row.providerDisplayName,
                            totalTokens: row.totalTokens
                        )
                    }
                    .sorted { $0.name < $1.name }
                totalTokens = rows.reduce(0) { $0 + $1.totalTokens }
                self.scheduleInstalled = schedule
                lastUpdated = Date()
            } catch {
                lastError = error.localizedDescription
            }
            isRefreshing = false
        }
    }

    func syncNow() {
        runAction {
            try await self.cli.runDaily()
        }
    }

    func installSchedule() {
        runAction {
            try await self.cli.installSchedule()
        }
    }

    func removeSchedule() {
        runAction {
            try await self.cli.removeSchedule()
        }
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
                    try await self.cli.installSchedule()
                } else {
                    try await self.cli.removeSchedule()
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

    func openHeatmap() {
        open(urlString: cli.profileURLString)
    }

    func quit() {
        NSApplication.shared.terminate(nil)
    }

    func share(for summary: ProviderSummary) -> Double {
        guard totalTokens > 0 else { return 0 }
        return Double(summary.totalTokens) / Double(totalTokens)
    }

    private func runAction(_ operation: @escaping () async throws -> Void) {
        guard !isRefreshing else { return }
        isRefreshing = true
        lastError = nil
        Task {
            do {
                try await operation()
                isRefreshing = false
                refresh()
            } catch {
                lastError = error.localizedDescription
                isRefreshing = false
            }
        }
    }

    private func open(urlString: String?) {
        guard let urlString, let url = URL(string: urlString) else { return }
        NSWorkspace.shared.open(url)
    }

}

private func compactTokenString(_ value: Int) -> String {
    let number = Double(value)
    if number >= 1_000_000_000 {
        return String(format: "%.1fB", number / 1_000_000_000)
    }
    if number >= 1_000_000 {
        return String(format: "%.1fM", number / 1_000_000)
    }
    if number >= 1_000 {
        return String(format: "%.1fK", number / 1_000)
    }
    return "\(value)"
}
