import EventKit
import Foundation

// apple-calendar-helper [start] [end] - prints Apple Calendar events between the
// two dates (YYYY-MM-DD, local time, end exclusive) as JSON to stdout.
// Defaults to today. Compiled by scripts/build-apple-calendar-helper.sh into
// bin/apple-calendar-helper.

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

func parseDay(_ argument: String) -> Date {
    let dayFormatter = DateFormatter()
    dayFormatter.dateFormat = "yyyy-MM-dd"
    dayFormatter.timeZone = TimeZone.current
    guard let date = dayFormatter.date(from: argument) else {
        FileHandle.standardError.write(Data("Invalid date \"\(argument)\", expected YYYY-MM-DD.\n".utf8))
        exit(1)
    }
    return calendar.startOfDay(for: date)
}

let arguments = CommandLine.arguments
let start = arguments.count > 1 ? parseDay(arguments[1]) : calendar.startOfDay(for: Date())
let end = arguments.count > 2 ? parseDay(arguments[2]) : calendar.date(byAdding: .day, value: 1, to: start)!
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
