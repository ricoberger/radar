import EventKit
import Foundation

// calendar-helper - prints today's Apple Calendar events as JSON to stdout.
// Compiled by scripts/build-calendar-helper.sh into bin/calendar-helper.

struct Event: Codable {
    let title: String
    let calendar: String
    let start: String
    let end: String
    let isAllDay: Bool
    let location: String?
}

let store = EKEventStore()
let semaphore = DispatchSemaphore(value: 0)
var granted = false

if #available(macOS 14.0, *) {
    store.requestFullAccessToEvents { ok, _ in
        granted = ok
        semaphore.signal()
    }
} else {
    store.requestAccess(to: .event) { ok, _ in
        granted = ok
        semaphore.signal()
    }
}
semaphore.wait()

guard granted else {
    FileHandle.standardError.write(Data("Calendar access denied. Grant access to your terminal in System Settings > Privacy & Security > Calendars.\n".utf8))
    exit(1)
}

let calendar = Calendar.current
let start = calendar.startOfDay(for: Date())
let end = calendar.date(byAdding: .day, value: 1, to: start)!
let predicate = store.predicateForEvents(withStart: start, end: end, calendars: nil)
let events = store.events(matching: predicate).sorted { $0.startDate < $1.startDate }

let formatter = ISO8601DateFormatter()
let output = events.map { event in
    Event(
        title: event.title ?? "",
        calendar: event.calendar?.title ?? "",
        start: formatter.string(from: event.startDate),
        end: formatter.string(from: event.endDate),
        isAllDay: event.isAllDay,
        location: event.location
    )
}

let data = try JSONEncoder().encode(output)
print(String(data: data, encoding: .utf8)!)
