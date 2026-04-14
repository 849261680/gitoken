import SwiftUI

struct MenuBarContentView: View {
    @EnvironmentObject private var viewModel: MenuBarViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            headerSection
            Divider()
            providerSection
            Divider()
            scheduleSection
            Divider()
            actionSection
        }
        .frame(width: 332)
        .background(Color(nsColor: .windowBackgroundColor))
        .task {
            viewModel.start()
        }
    }

    private var headerSection: some View {
        HStack(alignment: .top, spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                Text("今日")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(.secondary)
                Text(viewModel.totalSummary)
                    .font(.system(size: 24, weight: .semibold))
                    .lineLimit(1)
                Text("Tokens")
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
            }

            Spacer(minLength: 0)

            if viewModel.isRefreshing {
                ProgressView()
                    .controlSize(.small)
                    .padding(.top, 4)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
    }

    private var providerSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            if viewModel.providerSummaries.isEmpty {
                Text("今天暂无数据")
                    .font(.system(size: 13))
                    .foregroundStyle(.secondary)
            } else {
                ForEach(viewModel.providerSummaries) { summary in
                    providerRow(summary)
                }
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
    }

    private var scheduleSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            Toggle(isOn: Binding(
                get: { viewModel.scheduleInstalled },
                set: { viewModel.setScheduleEnabled($0) }
            )) {
                Text("每日同步")
                    .font(.system(size: 13, weight: .medium))
            }
            .toggleStyle(.switch)
            .disabled(viewModel.isRefreshing)

            if let error = viewModel.lastError {
                Text(error)
                    .font(.system(size: 12))
                    .foregroundStyle(.red)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
    }

    private var actionSection: some View {
        VStack(spacing: 8) {
            HStack(spacing: 8) {
                actionButton("刷新", action: viewModel.refresh)
                actionButton("同步", action: viewModel.syncNow)
            }

            HStack(spacing: 8) {
                actionButton("热力图", action: viewModel.openHeatmap)
                actionButton("退出", action: viewModel.quit)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 14)
    }

    private func providerRow(_ summary: MenuBarViewModel.ProviderSummary) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 10) {
                Text(summary.name)
                    .font(.system(size: 13, weight: .medium))
                Spacer(minLength: 0)
                Text(summary.tokensText)
                    .font(.system(size: 13))
                    .foregroundStyle(.secondary)
            }

            GeometryReader { proxy in
                let width = max(proxy.size.width * viewModel.share(for: summary), 6)
                ZStack(alignment: .leading) {
                    Capsule()
                        .fill(Color(nsColor: .separatorColor).opacity(0.18))
                    Capsule()
                        .fill(summary.accentColor.opacity(0.9))
                        .frame(width: width)
                }
            }
            .frame(height: 5)
        }
    }

    private func actionButton(_ title: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(.system(size: 13, weight: .medium))
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
        }
        .buttonStyle(.bordered)
        .controlSize(.regular)
    }
}
