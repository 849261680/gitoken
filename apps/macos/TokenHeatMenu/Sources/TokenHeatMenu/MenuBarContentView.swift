import SwiftUI

struct MenuBarContentView: View {
    @EnvironmentObject private var viewModel: MenuBarViewModel

    private let levelColors: [Color] = [
        Color(red: 0.84, green: 0.87, blue: 0.90), // 0 empty
        Color(red: 0.75, green: 0.85, blue: 0.93), // 1
        Color(red: 0.52, green: 0.71, blue: 0.87), // 2
        Color(red: 0.24, green: 0.51, blue: 0.76), // 3
        Color(red: 0.05, green: 0.32, blue: 0.62), // 4 dark
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {

            // ── Header label
            Text("TOKEN HEATMAP")
                .font(.system(size: 15, weight: .medium))
                .foregroundStyle(.secondary)
                .padding(.horizontal, 16)
                .padding(.top, 16)
                .padding(.bottom, 14)

            Divider()

            // ── 今日用量
            statRow(label: "今日用量", value: viewModel.todaySummary, valueSize: 20)

            Divider()

            // ── 本周合计
            statRow(label: "本周合计", value: viewModel.weeklySummary, valueSize: 20)

            Divider()

            // ── 来源
            statRow(label: "来源", value: viewModel.primaryProvider, valueSize: 13, valueBold: false)

            // ── Heatmap
            heatmapGrid
                .frame(maxWidth: .infinity, alignment: .center)
                .padding(.horizontal, 0)
                .padding(.top, 16)
                .padding(.bottom, 14)

            Divider()

            // ── Footer
            ZStack {
                HStack(spacing: 4) {
                    Button {
                        viewModel.openSettings()
                    } label: {
                        Image(systemName: "gearshape")
                            .font(.system(size: 12))
                            .foregroundStyle(.secondary)
                    }
                    .buttonStyle(.plain)
                    Button {
                        viewModel.syncNow()
                    } label: {
                        GitHubUploadIcon()
                    }
                    .buttonStyle(.plain)
                    .disabled(viewModel.isRefreshing)
                    .help("同步到 GitHub")
                    Spacer()
                    if viewModel.isRefreshing {
                        ProgressView().controlSize(.mini)
                    }
                    Button {
                        viewModel.quit()
                    } label: {
                        Image(systemName: "power")
                            .font(.system(size: 12))
                            .foregroundStyle(.secondary)
                    }
                    .buttonStyle(.plain)
                }

                if let nextSync = viewModel.nextSyncSummary {
                    Text(nextSync)
                        .font(.system(size: 11))
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
        }
        .frame(width: 300)
        .background(Color(nsColor: .windowBackgroundColor))
        .task { viewModel.start() }
        .contextMenu {
            Button("刷新", action: viewModel.refresh)
            Button("同步", action: viewModel.syncNow)
            Divider()
            Toggle("每日自动同步到 GitHub", isOn: Binding(
                get: { viewModel.scheduleInstalled },
                set: { viewModel.setScheduleEnabled($0) }
            ))
            Divider()
            Button("设置...") { viewModel.openSettings() }
            Button("查看热力图", action: viewModel.openHeatmap)
            Button("退出", action: viewModel.quit)
        }
    }

    // MARK: - Subviews

    private func statRow(label: String, value: String, valueSize: CGFloat, valueBold: Bool = true) -> some View {
        HStack(alignment: .center) {
            Text(label)
                .font(.system(size: 13))
                .foregroundStyle(.primary)
            Spacer()
            Text(value)
                .font(.system(size: valueSize, weight: valueBold ? .semibold : .regular))
                .foregroundStyle(.primary)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
    }

    private var heatmapGrid: some View {
        let cols = 14, rows = 7
        let days = viewModel.heatmapDays

        return VStack(spacing: 3) {
            ForEach(0..<rows, id: \.self) { row in
                HStack(spacing: 3) {
                    ForEach(0..<cols, id: \.self) { col in
                        let idx = col * rows + row
                        let level = idx < days.count ? days[idx].level : 0
                        RoundedRectangle(cornerRadius: 2)
                            .fill(levelColors[min(level, 4)])
                            .overlay(
                                RoundedRectangle(cornerRadius: 2)
                                    .stroke(Color.black.opacity(0.05), lineWidth: 0.5)
                            )
                            .frame(width: 15, height: 15)
                    }
                }
            }
        }
    }
}

private struct GitHubUploadIcon: View {
    var body: some View {
        ZStack {
            GitHubMark()
                .fill(.secondary)
                .frame(width: 16, height: 16)

            Image(systemName: "arrow.up")
                .font(.system(size: 6.5, weight: .bold))
                .foregroundStyle(.primary)
        }
        .frame(width: 18, height: 18)
    }
}

private struct GitHubMark: Shape {
    func path(in rect: CGRect) -> Path {
        var path = Path()
        let scale = min(rect.width, rect.height) / 16
        let offset = CGPoint(
            x: rect.midX - 8 * scale,
            y: rect.midY - 8 * scale
        )

        func p(_ x: CGFloat, _ y: CGFloat) -> CGPoint {
            CGPoint(x: offset.x + x * scale, y: offset.y + y * scale)
        }

        path.move(to: p(8, 0.2))
        path.addCurve(to: p(0.6, 7.6), control1: p(3.9, 0.2), control2: p(0.6, 3.5))
        path.addCurve(to: p(5.6, 14.6), control1: p(0.6, 10.9), control2: p(2.7, 13.7))
        path.addCurve(to: p(6.1, 14.2), control1: p(6.0, 14.7), control2: p(6.1, 14.5))
        path.addLine(to: p(6.1, 12.9))
        path.addCurve(to: p(3.4, 11.5), control1: p(4.0, 13.4), control2: p(3.4, 12.0))
        path.addCurve(to: p(2.8, 10.5), control1: p(3.1, 11.0), control2: p(2.8, 10.8))
        path.addCurve(to: p(3.8, 10.6), control1: p(2.8, 10.3), control2: p(3.1, 10.3))
        path.addCurve(to: p(5.2, 11.6), control1: p(4.5, 11.4), control2: p(4.8, 11.6))
        path.addCurve(to: p(6.2, 11.5), control1: p(5.6, 11.6), control2: p(5.9, 11.5))
        path.addCurve(to: p(6.7, 10.6), control1: p(6.2, 11.1), control2: p(6.4, 10.8))
        path.addCurve(to: p(4.0, 7.0), control1: p(4.6, 10.4), control2: p(4.0, 9.6))
        path.addCurve(to: p(4.8, 4.9), control1: p(4.0, 6.1), control2: p(4.3, 5.4))
        path.addCurve(to: p(4.9, 2.8), control1: p(4.7, 4.7), control2: p(4.4, 3.7))
        path.addCurve(to: p(7.1, 3.6), control1: p(4.9, 2.8), control2: p(5.6, 2.6))
        path.addCurve(to: p(8, 3.5), control1: p(7.4, 3.5), control2: p(7.7, 3.5))
        path.addCurve(to: p(8.9, 3.6), control1: p(8.3, 3.5), control2: p(8.6, 3.5))
        path.addCurve(to: p(11.1, 2.8), control1: p(10.4, 2.6), control2: p(11.1, 2.8))
        path.addCurve(to: p(11.2, 4.9), control1: p(11.6, 3.7), control2: p(11.3, 4.7))
        path.addCurve(to: p(12.0, 7.0), control1: p(11.7, 5.4), control2: p(12.0, 6.1))
        path.addCurve(to: p(9.3, 10.6), control1: p(12.0, 9.6), control2: p(11.4, 10.4))
        path.addCurve(to: p(9.9, 11.8), control1: p(9.8, 11.0), control2: p(9.9, 11.5))
        path.addLine(to: p(9.9, 14.2))
        path.addCurve(to: p(10.4, 14.6), control1: p(9.9, 14.5), control2: p(10.1, 14.7))
        path.addCurve(to: p(15.4, 7.6), control1: p(13.3, 13.7), control2: p(15.4, 10.9))
        path.addCurve(to: p(8, 0.2), control1: p(15.4, 3.5), control2: p(12.1, 0.2))
        path.closeSubpath()
        return path
    }
}

// MARK: - Settings

struct SettingsView: View {
    @EnvironmentObject var viewModel: MenuBarViewModel
    @State private var selectedInterval: Int = 120
    @State private var launchAtLogin: Bool = false
    @State private var autoSync: Bool = false
    @State private var syncInterval: Int = 86_400

    private let intervals: [(label: String, seconds: Int)] = [
        ("1 分钟", 60),
        ("2 分钟", 120),
        ("5 分钟", 300),
        ("10 分钟", 600),
    ]

    private let syncIntervals: [(label: String, seconds: Int)] = [
        ("每 1 小时", 3_600),
        ("每 6 小时", 21_600),
        ("每 12 小时", 43_200),
        ("每天", 86_400),
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            HStack {
                Text("开机自启")
                    .font(.system(size: 13))
                Spacer()
                Toggle("", isOn: $launchAtLogin)
                    .toggleStyle(.switch)
                    .onChange(of: launchAtLogin) { viewModel.settings.updateLaunchAtLogin($0) }
            }

            HStack {
                Text("刷新频率")
                    .font(.system(size: 13))
                Spacer()
                Picker("", selection: $selectedInterval) {
                    ForEach(intervals, id: \.seconds) { item in
                        Text(item.label).tag(item.seconds)
                    }
                }
                .pickerStyle(.menu)
                .onChange(of: selectedInterval) {
                    viewModel.settings.refreshInterval = $0
                    viewModel.restartRefreshLoop()
                }
            }

            HStack {
                Text("自动同步到 GitHub")
                    .font(.system(size: 13))
                Spacer()
                Toggle("", isOn: $autoSync)
                    .toggleStyle(.switch)
                    .disabled(viewModel.isRefreshing)
                    .onChange(of: autoSync) { viewModel.setScheduleEnabled($0) }
            }

            HStack {
                Text("同步频率")
                    .font(.system(size: 13))
                Spacer()
                Picker("", selection: $syncInterval) {
                    ForEach(syncIntervals, id: \.seconds) { item in
                        Text(item.label).tag(item.seconds)
                    }
                }
                    .pickerStyle(.menu)
                    .disabled(viewModel.isRefreshing)
                    .onChange(of: syncInterval) { viewModel.setScheduleInterval($0) }
            }
        }
        .padding(20)
        .frame(width: 260)
        .onAppear {
            selectedInterval = viewModel.settings.refreshInterval
            launchAtLogin = viewModel.settings.launchAtLogin
            autoSync = viewModel.scheduleInstalled
            syncInterval = viewModel.settings.syncInterval
        }
        .onChange(of: viewModel.scheduleInstalled) { autoSync = $0 }
    }
}
