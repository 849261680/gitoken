import Foundation
import ServiceManagement
import SwiftUI

@MainActor
final class SettingsStore: ObservableObject {
    @AppStorage("refreshInterval") var refreshInterval: Int = 120
    @AppStorage("launchAtLogin") var launchAtLogin: Bool = false
    @AppStorage("syncInterval") var syncInterval: Int = 86_400
    @AppStorage("syncScheduledAt") var syncScheduledAt: Double = 0

    func updateLaunchAtLogin(_ enabled: Bool) {
        do {
            if enabled {
                try SMAppService.mainApp.register()
            } else {
                try SMAppService.mainApp.unregister()
            }
            launchAtLogin = enabled
        } catch {
            launchAtLogin = false
        }
    }
}
