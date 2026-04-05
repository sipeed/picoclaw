import AppKit
import Foundation

// MARK: - Event types

struct SkillEvent: Codable, Identifiable {
    let id: UUID
    let timestamp: Date
    let operation: String  // create, update, patch, delete, block, tool_call
    let skillName: String
    let detail: String

    init(operation: String, skillName: String, detail: String = "") {
        self.id = UUID()
        self.timestamp = Date()
        self.operation = operation
        self.skillName = skillName
        self.detail = detail
    }
}

struct DailyStats: Codable, Identifiable {
    let date: String  // yyyy-MM-dd
    var skillsCreated: Int = 0
    var skillsPatched: Int = 0
    var skillsDeleted: Int = 0
    var blockedAttempts: Int = 0
    var totalConversations: Int = 0
    var toolCalls: Int = 0
    var skillManageCalls: Int = 0
    var statusViews: Int = 0
    var trialStarts: Int = 0

    var id: String { date }

    var totalSkillOps: Int { skillsCreated + skillsPatched + skillsDeleted }
}

struct StatusInteraction: Codable, Identifiable {
    let id: UUID
    let timestamp: Date
    let phone: String
    let content: String
    let crmStatus: String   // active, inactive, trial, trial_expired, ignored, unknown
    let contactName: String

    init(phone: String, content: String, crmStatus: String = "unknown", contactName: String = "") {
        self.id = UUID()
        self.timestamp = Date()
        self.phone = phone
        self.content = content
        self.crmStatus = crmStatus
        self.contactName = contactName
    }
}

// MARK: - Engine

@MainActor
final class MonitorEngine: ObservableObject {
    // Live state
    @Published var gatewayUp = false
    @Published var gatewayUptime = ""
    @Published var totalSkills = 0
    @Published var skillsList: [SkillInfo] = []
    @Published var recentEvents: [SkillEvent] = []
    @Published var recentSkillActivity = false
    @Published var weeklyStats: [DailyStats] = []
    @Published var todayStats = DailyStats(date: MonitorEngine.todayString())
    @Published var lastMessage = ""
    @Published var activeSessionCount = 0
    @Published var statusInteractions: [StatusInteraction] = []
    @Published var trialInteractions: [StatusInteraction] = []

    struct SkillInfo: Identifiable {
        let name: String
        let description: String
        let modified: Date
        var id: String { name }
    }

    private let picoHome = NSHomeDirectory() + "/.picoclaw"
    private let skillsDir: String
    private let sessionsDir: String
    private let logPath: String
    private let statsPath: String

    // Session file tracking
    private struct TrackedSession {
        let path: String
        var offset: UInt64
        let sessionId: String
    }
    private var trackedSessions: [String: TrackedSession] = [:]

    // Status broadcast parsing state
    private var pendingStatusContent: String = ""
    private var pendingStatusPhone: String = ""

    // Trial detection: sessionId -> phone from check-access.sh
    private var pendingSessionPhone: [String: String] = [:]
    // Sessions already counted as trial (dedup)
    private var countedTrialSessions: Set<String> = []

    private struct CRMContact {
        let name: String
        let status: String
    }
    private var crmCache: [String: CRMContact] = [:]

    // Log file tracking
    private var logOffset: UInt64 = 0

    private var healthTimer: Timer?
    private var sessionTimer: Timer?
    private var sessionScanTimer: Timer?
    private var skillsTimer: Timer?
    private var statsTimer: Timer?
    private var dirWatcher: DispatchSourceFileSystemObject?

    init() {
        skillsDir = picoHome + "/workspace/skills"
        sessionsDir = picoHome + "/workspace/sessions"
        logPath = picoHome + "/logs/gateway.log"
        statsPath = picoHome + "/logs/picowatch_stats.json"
    }

    func start() {
        loadStats()
        loadCRM()
        scanSkills()
        discoverSessions()
        seekLogEnd()

        processAllSessions(initialScan: true)

        healthTimer = Timer.scheduledTimer(withTimeInterval: 10, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.checkHealth() }
        }
        Task { checkHealth() }

        sessionTimer = Timer.scheduledTimer(withTimeInterval: 2, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.processAllSessions(initialScan: false) }
        }

        sessionScanTimer = Timer.scheduledTimer(withTimeInterval: 10, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.discoverSessions() }
        }

        skillsTimer = Timer.scheduledTimer(withTimeInterval: 15, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.scanSkills() }
        }

        Timer.scheduledTimer(withTimeInterval: 3, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.tailLog() }
        }

        statsTimer = Timer.scheduledTimer(withTimeInterval: 60, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.saveStats() }
        }

        Timer.scheduledTimer(withTimeInterval: 30, repeats: true) { [weak self] _ in
            Task { @MainActor in self?.loadCRM() }
        }

        watchSkillsDir()
    }

    func stop() {
        healthTimer?.invalidate()
        sessionTimer?.invalidate()
        sessionScanTimer?.invalidate()
        skillsTimer?.invalidate()
        statsTimer?.invalidate()
        dirWatcher?.cancel()
        saveStats()
    }

    // MARK: - Health

    private func checkHealth() {
        let url = URL(string: "http://127.0.0.1:18790/health")!
        let task = URLSession.shared.dataTask(with: url) { [weak self] data, response, error in
            Task { @MainActor in
                guard let self else { return }
                if let data,
                   let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                   let status = json["status"] as? String, status == "ok" {
                    self.gatewayUp = true
                    self.gatewayUptime = (json["uptime"] as? String) ?? ""
                } else {
                    self.gatewayUp = false
                    self.gatewayUptime = ""
                }
            }
        }
        task.resume()
    }

    // MARK: - Session files (JSONL)

    private func discoverSessions() {
        let fm = FileManager.default
        guard let files = try? fm.contentsOfDirectory(atPath: sessionsDir) else { return }

        for file in files where file.hasSuffix(".jsonl") {
            let path = sessionsDir + "/" + file
            guard trackedSessions[path] == nil else { continue }
            let sessionId = file.replacingOccurrences(of: ".jsonl", with: "")
            trackedSessions[path] = TrackedSession(path: path, offset: 0, sessionId: sessionId)
        }

        activeSessionCount = trackedSessions.count
    }

    private func processAllSessions(initialScan: Bool) {
        let fm = FileManager.default

        for (path, session) in trackedSessions {
            guard let attrs = try? fm.attributesOfItem(atPath: path),
                  let size = (attrs[.size] as? NSNumber)?.uint64Value else { continue }

            if size <= session.offset { continue }

            var offset = session.offset
            if initialScan && offset == 0 && size > 20_000 {
                offset = size - 20_000
            }

            guard let handle = try? FileHandle(forReadingFrom: URL(fileURLWithPath: path)) else { continue }
            defer { try? handle.close() }

            do { try handle.seek(toOffset: offset) } catch { continue }

            let data = handle.readDataToEndOfFile()
            guard let text = String(data: data, encoding: .utf8) else { continue }

            let lines = text.split(separator: "\n", omittingEmptySubsequences: false).map(String.init)
            let startIdx = (initialScan && offset > 0) ? 1 : 0

            for i in startIdx..<lines.count {
                let line = lines[i].trimmingCharacters(in: .whitespaces)
                if line.isEmpty { continue }
                processSessionLine(line, sessionId: session.sessionId, isInitial: initialScan)
            }

            var updated = session
            updated.offset = size
            trackedSessions[path] = updated
        }
    }

    private func processSessionLine(_ line: String, sessionId: String, isInitial: Bool) {
        guard let data = line.data(using: .utf8),
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { return }

        let role = json["role"] as? String ?? ""

        // Route status@broadcast to dedicated handler
        if sessionId.contains("status@broadcast") {
            processStatusLine(json, role: role, isInitial: isInitial)
            return
        }

        if role == "user" {
            todayStats.totalConversations += 1
        }

        if role == "assistant" {
            if let content = json["content"] as? String, !content.isEmpty {
                let clean = content.replacingOccurrences(of: "<|[SPLIT]|>", with: " ")
                lastMessage = String(clean.prefix(100))
            }
        }

        // Detect tool_calls
        if let toolCalls = json["tool_calls"] as? [[String: Any]] {
            for tc in toolCalls {
                guard let function = tc["function"] as? [String: Any],
                      let fnName = function["name"] as? String else { continue }

                todayStats.toolCalls += 1

                // Track check-access.sh for trial detection
                if fnName == "exec",
                   let argsStr = function["arguments"] as? String,
                   argsStr.contains("check-access.sh") {
                    if let range = argsStr.range(of: "check-access.sh ") {
                        let raw = String(argsStr[range.upperBound...])
                            .replacingOccurrences(of: "\"", with: "")
                            .replacingOccurrences(of: "}", with: "")
                            .trimmingCharacters(in: .whitespacesAndNewlines)
                        let digits = raw.unicodeScalars
                            .filter { CharacterSet.decimalDigits.contains($0) }
                            .map { String($0) }.joined()
                        if digits.count >= 10 {
                            pendingSessionPhone[sessionId] = "+" + digits
                        }
                    }
                }

                if fnName == "skill_manage" {
                    todayStats.skillManageCalls += 1

                    if let argsStr = function["arguments"] as? String,
                       let argsData = argsStr.data(using: .utf8),
                       let args = try? JSONSerialization.jsonObject(with: argsData) as? [String: Any] {

                        let op = args["operation"] as? String ?? "unknown"
                        let name = args["name"] as? String ?? "?"

                        recentSkillActivity = true
                        switch op {
                        case "create":
                            todayStats.skillsCreated += 1
                            addEvent(SkillEvent(operation: "create", skillName: name))
                        case "patch":
                            todayStats.skillsPatched += 1
                            addEvent(SkillEvent(operation: "patch", skillName: name))
                        case "update":
                            todayStats.skillsPatched += 1
                            addEvent(SkillEvent(operation: "update", skillName: name))
                        case "delete":
                            todayStats.skillsDeleted += 1
                            addEvent(SkillEvent(operation: "delete", skillName: name))
                        case "list":
                            addEvent(SkillEvent(operation: "list", skillName: "all",
                                                detail: "Agente listou skills"))
                        default:
                            break
                        }
                    }
                } else if !isInitial {
                    addEvent(SkillEvent(operation: "tool_call", skillName: fnName,
                                        detail: "via \(sessionId.prefix(20))"))
                }
            }
        }

        // Detect trial starts from check-access.sh tool responses
        if role == "tool", let phone = pendingSessionPhone[sessionId] {
            let output = (json["content"] as? String) ?? ""
            if output.contains("STATUS: TRIAL") && !output.contains("TRIAL_EXPIRED") {
                // Dedup: only count once per session
                if !countedTrialSessions.contains(sessionId) {
                    countedTrialSessions.insert(sessionId)
                    // Extract real phone from TELEFONE: +XXXX in tool output
                    var realPhone = phone
                    if let telRange = output.range(of: "TELEFONE: ") {
                        let after = String(output[telRange.upperBound...])
                        let phoneLine = after.components(separatedBy: .newlines).first ?? ""
                        let cleaned = phoneLine.trimmingCharacters(in: .whitespaces)
                        if !cleaned.isEmpty { realPhone = cleaned }
                    }
                    let contact = crmCache[realPhone]
                    let name = contact?.name ?? ""
                    let interaction = StatusInteraction(
                        phone: realPhone,
                        content: "Conversa de teste iniciada",
                        crmStatus: "trial",
                        contactName: name
                    )
                    trialInteractions.insert(interaction, at: 0)
                    if trialInteractions.count > 30 {
                        trialInteractions = Array(trialInteractions.prefix(30))
                    }
                    todayStats.trialStarts += 1
                    if !isInitial {
                        sendTrialNotification(phone: realPhone, name: name)
                    }
                }
                pendingSessionPhone.removeValue(forKey: sessionId)
            } else if output.contains("STATUS:") {
                pendingSessionPhone.removeValue(forKey: sessionId)
            }
        }
    }

    // MARK: - Status broadcast

    private func processStatusLine(_ json: [String: Any], role: String, isInitial: Bool) {
        if role == "user" {
            pendingStatusContent = (json["content"] as? String) ?? ""
            pendingStatusPhone = ""
        }

        if role == "assistant", let toolCalls = json["tool_calls"] as? [[String: Any]] {
            for tc in toolCalls {
                guard let function = tc["function"] as? [String: Any],
                      let argsStr = function["arguments"] as? String,
                      argsStr.contains("check-access.sh") else { continue }

                if let range = argsStr.range(of: "check-access.sh ") {
                    let raw = String(argsStr[range.upperBound...])
                        .replacingOccurrences(of: "\"", with: "")
                        .replacingOccurrences(of: "}", with: "")
                        .trimmingCharacters(in: .whitespacesAndNewlines)
                    let digits = raw.unicodeScalars
                        .filter { CharacterSet.decimalDigits.contains($0) }
                        .map { String($0) }.joined()
                    if digits.count >= 10 {
                        pendingStatusPhone = "+" + digits
                    }
                }
            }
        }

        if role == "tool", !pendingStatusPhone.isEmpty {
            let output = (json["content"] as? String) ?? ""

            var crmStatus = "unknown"
            if output.contains("STATUS: ACTIVE") { crmStatus = "active" }
            else if output.contains("STATUS: TRIAL_EXPIRED") { crmStatus = "trial_expired" }
            else if output.contains("STATUS: TRIAL") { crmStatus = "trial" }
            else if output.contains("STATUS: INACTIVE") { crmStatus = "inactive" }
            else if output.contains("STATUS: IGNORED") { crmStatus = "ignored" }

            let contact = crmCache[pendingStatusPhone]
            let name = contact?.name ?? ""
            let finalStatus = contact?.status ?? crmStatus

            let interaction = StatusInteraction(
                phone: pendingStatusPhone,
                content: pendingStatusContent,
                crmStatus: finalStatus,
                contactName: name
            )
            statusInteractions.insert(interaction, at: 0)
            if statusInteractions.count > 30 {
                statusInteractions = Array(statusInteractions.prefix(30))
            }
            if !isInitial {
                todayStats.statusViews += 1
                recentSkillActivity = true
            }

            pendingStatusContent = ""
            pendingStatusPhone = ""
        }
    }

    // MARK: - CRM cache

    private func loadCRM() {
        let subscribersPath = picoHome + "/workspace/scripts/.subscribers.json"
        let trialsPath = picoHome + "/workspace/scripts/.trials.json"
        var cache: [String: CRMContact] = [:]

        if let data = try? Data(contentsOf: URL(fileURLWithPath: subscribersPath)),
           let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
           let subs = json["subscribers"] as? [String: [String: Any]] {
            for (phone, info) in subs {
                let name = info["name"] as? String ?? ""
                let status = info["status"] as? String ?? "unknown"
                cache[phone] = CRMContact(name: name, status: status)
            }
        }

        if let data = try? Data(contentsOf: URL(fileURLWithPath: trialsPath)),
           let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
           let trials = json["trials"] as? [String: [String: Any]] {
            for (phone, info) in trials {
                if cache[phone] != nil { continue }
                let name = info["name"] as? String ?? ""
                let step = info["step"] as? String ?? ""
                let status = step == "connected" ? "trial_expired" : "trial"
                cache[phone] = CRMContact(name: name, status: status)
            }
        }

        crmCache = cache
    }

    // MARK: - Gateway log (for security blocks)

    private func seekLogEnd() {
        guard let attrs = try? FileManager.default.attributesOfItem(atPath: logPath),
              let size = (attrs[.size] as? NSNumber)?.uint64Value else { return }
        logOffset = size
    }

    private func tailLog() {
        guard let attrs = try? FileManager.default.attributesOfItem(atPath: logPath),
              let size = (attrs[.size] as? NSNumber)?.uint64Value,
              size > logOffset else { return }

        guard let handle = try? FileHandle(forReadingFrom: URL(fileURLWithPath: logPath)) else { return }
        defer { try? handle.close() }

        try? handle.seek(toOffset: logOffset)
        let data = handle.readDataToEndOfFile()
        logOffset = size

        guard let text = String(data: data, encoding: .utf8) else { return }

        for line in text.split(separator: "\n") {
            let s = String(line)
            guard s.hasPrefix("{"),
                  let d = s.data(using: .utf8),
                  let json = try? JSONSerialization.jsonObject(with: d) as? [String: Any] else { continue }

            let message = json["message"] as? String ?? ""
            if message.contains("security scan blocked") || message.contains("guard blocked") {
                todayStats.blockedAttempts += 1
                let name = json["skill_name"] as? String ?? "unknown"
                addEvent(SkillEvent(operation: "block", skillName: name, detail: message))
                recentSkillActivity = true
            }
        }
    }

    // MARK: - Skills directory

    private func scanSkills() {
        let fm = FileManager.default
        guard let dirs = try? fm.contentsOfDirectory(atPath: skillsDir) else {
            totalSkills = 0
            skillsList = []
            return
        }

        var skills: [SkillInfo] = []
        for dir in dirs {
            let skillMd = skillsDir + "/\(dir)/SKILL.md"
            guard fm.fileExists(atPath: skillMd) else { continue }
            let attrs = try? fm.attributesOfItem(atPath: skillMd)
            let modified = (attrs?[.modificationDate] as? Date) ?? Date.distantPast
            let desc = extractDescription(from: skillMd)
            skills.append(SkillInfo(name: dir, description: desc, modified: modified))
        }

        skills.sort { $0.modified > $1.modified }
        totalSkills = skills.count
        skillsList = skills
    }

    private func extractDescription(from path: String) -> String {
        guard let content = try? String(contentsOfFile: path, encoding: .utf8) else { return "" }
        let lines = content.components(separatedBy: .newlines)
        var inFrontmatter = false
        for line in lines {
            if line.trimmingCharacters(in: .whitespaces) == "---" {
                if inFrontmatter { break }
                inFrontmatter = true
                continue
            }
            if inFrontmatter && line.hasPrefix("description:") {
                return String(line.dropFirst("description:".count)).trimmingCharacters(in: .whitespaces)
            }
        }
        return ""
    }

    private func watchSkillsDir() {
        let fd = open(skillsDir, O_EVTONLY)
        guard fd >= 0 else { return }
        let source = DispatchSource.makeFileSystemObjectSource(
            fileDescriptor: fd,
            eventMask: [.write, .rename, .delete],
            queue: .main
        )
        source.setEventHandler { [weak self] in
            Task { @MainActor in self?.scanSkills() }
        }
        source.setCancelHandler { close(fd) }
        source.resume()
        dirWatcher = source
    }

    // MARK: - Notifications

    private func sendTrialNotification(phone: String, name: String) {
        let notification = NSUserNotification()
        notification.title = "Novo Trial no WhatsApp"
        notification.subtitle = name.isEmpty ? phone : "\(name) (\(phone))"
        notification.informativeText = "Usuario em fase trial iniciou conversa"
        notification.soundName = NSUserNotificationDefaultSoundName
        NSUserNotificationCenter.default.deliver(notification)
        updateWhatsAppStatus()
    }

    private func updateWhatsAppStatus() {
        let count = todayStats.trialStarts
        let f = DateFormatter()
        f.dateFormat = "dd/MM"
        let text = "\(f.string(from: Date())): \(count)"
        let url = URL(string: "http://127.0.0.1:18790/api/send-status")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try? JSONSerialization.data(withJSONObject: ["text": text])
        URLSession.shared.dataTask(with: request) { _, _, _ in }.resume()
    }

    // MARK: - Events

    private func addEvent(_ event: SkillEvent) {
        recentEvents.insert(event, at: 0)
        if recentEvents.count > 50 {
            recentEvents = Array(recentEvents.prefix(50))
        }

        DispatchQueue.main.asyncAfter(deadline: .now() + 30) { [weak self] in
            self?.recentSkillActivity = false
        }
    }

    // MARK: - Stats persistence

    static func todayString() -> String {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        return f.string(from: Date())
    }

    private func loadStats() {
        guard let data = try? Data(contentsOf: URL(fileURLWithPath: statsPath)),
              var stats = try? JSONDecoder().decode([DailyStats].self, from: data) else {
            weeklyStats = []
            return
        }

        let cutoff = Calendar.current.date(byAdding: .day, value: -7, to: Date())!
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        stats = stats.filter { stat in
            guard let d = f.date(from: stat.date) else { return false }
            return d >= cutoff
        }

        weeklyStats = stats

        let today = Self.todayString()
        if let existing = stats.first(where: { $0.date == today }) {
            todayStats = existing
        }
    }

    private func saveStats() {
        let today = Self.todayString()
        todayStats = DailyStats(
            date: today,
            skillsCreated: todayStats.skillsCreated,
            skillsPatched: todayStats.skillsPatched,
            skillsDeleted: todayStats.skillsDeleted,
            blockedAttempts: todayStats.blockedAttempts,
            totalConversations: todayStats.totalConversations,
            toolCalls: todayStats.toolCalls,
            skillManageCalls: todayStats.skillManageCalls,
            statusViews: todayStats.statusViews,
            trialStarts: todayStats.trialStarts
        )

        var stats = weeklyStats.filter { $0.date != today }
        stats.append(todayStats)
        stats.sort { $0.date < $1.date }

        if stats.count > 7 {
            stats = Array(stats.suffix(7))
        }

        weeklyStats = stats

        if let data = try? JSONEncoder().encode(stats) {
            try? data.write(to: URL(fileURLWithPath: statsPath))
        }
    }

    // MARK: - Weekly summary

    var weeklySummary: (skills: Int, patches: Int, blocks: Int, conversations: Int, toolCalls: Int, statusViews: Int, trialStarts: Int) {
        let all = weeklyStats + [todayStats]
        return (
            skills: all.reduce(0) { $0 + $1.skillsCreated },
            patches: all.reduce(0) { $0 + $1.skillsPatched },
            blocks: all.reduce(0) { $0 + $1.blockedAttempts },
            conversations: all.reduce(0) { $0 + $1.totalConversations },
            toolCalls: all.reduce(0) { $0 + $1.toolCalls },
            statusViews: all.reduce(0) { $0 + $1.statusViews },
            trialStarts: all.reduce(0) { $0 + $1.trialStarts }
        )
    }
}
