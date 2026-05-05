import Foundation
import ServiceManagement
import SwiftUI

@MainActor
final class SettingsStore: ObservableObject {
    @AppStorage("refreshInterval") var refreshInterval: Int = 120
    @AppStorage("launchAtLogin") var launchAtLogin: Bool = false
    @AppStorage("syncHour") var syncHour: Int = 0
    @AppStorage("syncMinute") var syncMinute: Int = 5

    var syncTimeString: String {
        String(format: "%02d:%02d", syncHour, syncMinute)
    }

    var syncTimeDate: Date {
        var components = DateComponents()
        components.hour = syncHour
        components.minute = syncMinute
        return Calendar.current.date(from: components) ?? Date()
    }

    func updateSyncTime(_ date: Date) {
        let components = Calendar.current.dateComponents([.hour, .minute], from: date)
        syncHour = components.hour ?? 0
        syncMinute = components.minute ?? 5
    }

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
