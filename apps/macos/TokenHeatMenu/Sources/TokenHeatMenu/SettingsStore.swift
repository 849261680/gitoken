import Foundation
import ServiceManagement
import SwiftUI

@MainActor
final class SettingsStore: ObservableObject {
    @AppStorage("refreshInterval") var refreshInterval: Int = 120
    @AppStorage("launchAtLogin") var launchAtLogin: Bool = false

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
