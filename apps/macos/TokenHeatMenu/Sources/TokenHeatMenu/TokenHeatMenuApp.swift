import SwiftUI

@main
struct TokenHeatMenuApp: App {
    @StateObject private var viewModel = MenuBarViewModel()

    var body: some Scene {
        MenuBarExtra(viewModel.menuTitle, systemImage: "chart.bar.xaxis") {
            MenuBarContentView()
                .environmentObject(viewModel)
                .environmentObject(viewModel.settings)
                .frame(width: 300)
        }
        .menuBarExtraStyle(.window)
    }
}
