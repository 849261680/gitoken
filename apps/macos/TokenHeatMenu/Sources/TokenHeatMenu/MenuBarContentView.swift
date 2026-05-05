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
                    Image(systemName: "icloud.and.arrow.up")
                        .font(.system(size: 12))
                        .foregroundStyle(.secondary)
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
            Toggle("每日自动同步", isOn: Binding(
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

// MARK: - Settings

struct SettingsView: View {
    @EnvironmentObject var viewModel: MenuBarViewModel
    @State private var selectedInterval: Int = 120
    @State private var launchAtLogin: Bool = false
    @State private var autoSync: Bool = false
    @State private var syncTime: Date = Date()

    private let intervals: [(label: String, seconds: Int)] = [
        ("1 分钟", 60),
        ("2 分钟", 120),
        ("5 分钟", 300),
        ("10 分钟", 600),
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
                Text("自动同步")
                    .font(.system(size: 13))
                Spacer()
                Toggle("", isOn: $autoSync)
                    .toggleStyle(.switch)
                    .disabled(viewModel.isRefreshing)
                    .onChange(of: autoSync) { viewModel.setScheduleEnabled($0) }
            }

            HStack {
                Text("同步时间")
                    .font(.system(size: 13))
                Spacer()
                DatePicker("", selection: $syncTime, displayedComponents: .hourAndMinute)
                    .labelsHidden()
                    .disabled(viewModel.isRefreshing)
                    .onChange(of: syncTime) { viewModel.setScheduleTime($0) }
            }
        }
        .padding(20)
        .frame(width: 260)
        .onAppear {
            selectedInterval = viewModel.settings.refreshInterval
            launchAtLogin = viewModel.settings.launchAtLogin
            autoSync = viewModel.scheduleInstalled
            syncTime = viewModel.settings.syncTimeDate
        }
        .onChange(of: viewModel.scheduleInstalled) { autoSync = $0 }
    }
}
