import SwiftUI

struct PopoverView: View {
    @ObservedObject var engine: MonitorEngine

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            headerView
            Divider()
            gatewayStatus
            Divider()
            trialActivity
            Divider()
            weeklyReport
            Divider()
            recentActivity
            Divider()
            footerView
        }
        .frame(width: 380)
    }

    // MARK: - Header

    private var headerView: some View {
        HStack(spacing: 8) {
            Image(systemName: "brain.head.profile")
                .font(.title3)
                .foregroundStyle(.purple)
            Text("PicoWatch")
                .font(.headline)
            Spacer()
            HStack(spacing: 5) {
                Circle()
                    .fill(engine.gatewayUp ? Color.green : Color.red)
                    .frame(width: 7, height: 7)
                Text(engine.gatewayUp ? "Online" : "Offline")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    // MARK: - Gateway

    private var gatewayStatus: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Gateway")
                        .font(.system(.caption, weight: .semibold))
                        .foregroundStyle(.secondary)
                        .textCase(.uppercase)
                    if engine.gatewayUp {
                        Text("Uptime: \(engine.gatewayUptime)")
                            .font(.system(.caption, design: .monospaced))
                    } else {
                        Text("Not running")
                            .font(.caption)
                            .foregroundStyle(.red)
                    }
                }
                Spacer()
                VStack(alignment: .trailing, spacing: 2) {
                    Text("Skills")
                        .font(.system(.caption, weight: .semibold))
                        .foregroundStyle(.secondary)
                        .textCase(.uppercase)
                    Text("\(engine.totalSkills)")
                        .font(.system(.title2, design: .rounded, weight: .bold))
                        .foregroundStyle(.purple)
                }
            }
            if !engine.lastMessage.isEmpty {
                HStack(spacing: 5) {
                    Image(systemName: "bubble.left.fill")
                        .font(.system(size: 8))
                        .foregroundStyle(.green)
                    Text(engine.lastMessage)
                        .font(.system(size: 9))
                        .foregroundStyle(.secondary)
                        .lineLimit(2)
                }
            }
            HStack(spacing: 4) {
                Image(systemName: "antenna.radiowaves.left.and.right")
                    .font(.system(size: 8))
                    .foregroundStyle(.tertiary)
                Text("\(engine.activeSessionCount) sessoes ativas")
                    .font(.system(size: 9))
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }

    // MARK: - Trial Activity

    private var trialActivity: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: "person.badge.clock")
                    .font(.system(size: 10))
                    .foregroundStyle(.orange)
                Text("Conversas de Teste")
                    .font(.system(.caption, weight: .semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)
                Spacer()
                Text("\(engine.todayStats.trialStarts) hoje")
                    .font(.system(size: 9, weight: .semibold, design: .rounded))
                    .foregroundStyle(.orange)
            }

            if engine.trialInteractions.isEmpty {
                Text("Nenhuma conversa de teste ainda")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .padding(.vertical, 2)
            } else {
                ScrollView {
                    VStack(spacing: 4) {
                        ForEach(engine.trialInteractions.prefix(6)) { trial in
                            HStack(spacing: 8) {
                                Circle()
                                    .fill(Color.orange)
                                    .frame(width: 7, height: 7)
                                VStack(alignment: .leading, spacing: 1) {
                                    Text(trial.contactName.isEmpty ? trial.phone : trial.contactName)
                                        .font(.system(size: 10, weight: .semibold))
                                        .lineLimit(1)
                                    Text(trial.content)
                                        .font(.system(size: 9))
                                        .foregroundStyle(.tertiary)
                                        .lineLimit(1)
                                }
                                Spacer()
                                Text(relativeDate(trial.timestamp))
                                    .font(.system(size: 8, design: .monospaced))
                                    .foregroundStyle(.tertiary)
                            }
                            .padding(.horizontal, 6)
                            .padding(.vertical, 4)
                            .background(RoundedRectangle(cornerRadius: 5).fill(Color.orange.opacity(0.06)))
                        }
                    }
                }
                .frame(maxHeight: 110)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }

    // MARK: - Status Broadcast

    private var statusActivity: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: "dot.radiowaves.left.and.right")
                    .font(.system(size: 10))
                    .foregroundStyle(.cyan)
                Text("Status Broadcast")
                    .font(.system(.caption, weight: .semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)
                Spacer()
                Text("\(engine.todayStats.statusViews) hoje")
                    .font(.system(size: 9, weight: .semibold, design: .rounded))
                    .foregroundStyle(.cyan)
            }

            if engine.statusInteractions.isEmpty {
                Text("Nenhuma interacao de status ainda")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .padding(.vertical, 2)
            } else {
                ScrollView {
                    VStack(spacing: 4) {
                        ForEach(engine.statusInteractions.prefix(6)) { interaction in
                            statusRow(interaction)
                        }
                    }
                }
                .frame(maxHeight: 110)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }

    private func statusRow(_ interaction: StatusInteraction) -> some View {
        HStack(spacing: 8) {
            Circle()
                .fill(crmBadgeColor(interaction.crmStatus))
                .frame(width: 7, height: 7)
            VStack(alignment: .leading, spacing: 1) {
                HStack(spacing: 4) {
                    Text(interaction.contactName.isEmpty ? interaction.phone : interaction.contactName)
                        .font(.system(size: 10, weight: .semibold))
                        .lineLimit(1)
                    Text(crmBadgeLabel(interaction.crmStatus))
                        .font(.system(size: 8, weight: .medium))
                        .foregroundStyle(crmBadgeColor(interaction.crmStatus))
                        .padding(.horizontal, 4)
                        .padding(.vertical, 1)
                        .background(
                            RoundedRectangle(cornerRadius: 3)
                                .fill(crmBadgeColor(interaction.crmStatus).opacity(0.12))
                        )
                }
                if !interaction.content.isEmpty {
                    Text(interaction.content)
                        .font(.system(size: 9))
                        .foregroundStyle(.tertiary)
                        .lineLimit(1)
                }
            }
            Spacer()
            Text(relativeDate(interaction.timestamp))
                .font(.system(size: 8, design: .monospaced))
                .foregroundStyle(.tertiary)
        }
        .padding(.horizontal, 6)
        .padding(.vertical, 4)
        .background(RoundedRectangle(cornerRadius: 5).fill(Color.cyan.opacity(0.04)))
    }

    private func crmBadgeColor(_ status: String) -> Color {
        switch status {
        case "active": return .green
        case "trial": return .orange
        case "trial_expired": return .red
        case "inactive": return .red
        case "ignored": return .gray
        default: return .secondary
        }
    }

    private func crmBadgeLabel(_ status: String) -> String {
        switch status {
        case "active": return "assinante"
        case "trial": return "trial"
        case "trial_expired": return "expirado"
        case "inactive": return "inativo"
        case "ignored": return "ignorado"
        default: return "novo"
        }
    }

    // MARK: - Skills

    private var skillsSummary: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Installed Skills")
                .font(.system(.caption, weight: .semibold))
                .foregroundStyle(.secondary)
                .textCase(.uppercase)

            if engine.skillsList.isEmpty {
                Text("Nenhuma skill ainda — o agente cria quando completa tarefas complexas")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .padding(.vertical, 4)
            } else {
                ForEach(engine.skillsList) { skill in
                    HStack(spacing: 8) {
                        Image(systemName: "doc.text")
                            .font(.system(size: 10))
                            .foregroundStyle(.purple)
                        VStack(alignment: .leading, spacing: 1) {
                            Text(skill.name)
                                .font(.system(.caption, design: .monospaced, weight: .semibold))
                            if !skill.description.isEmpty {
                                Text(skill.description)
                                    .font(.system(size: 9))
                                    .foregroundStyle(.secondary)
                                    .lineLimit(1)
                            }
                        }
                        Spacer()
                        Text(relativeDate(skill.modified))
                            .font(.system(size: 9, design: .monospaced))
                            .foregroundStyle(.tertiary)
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 5)
                    .background(RoundedRectangle(cornerRadius: 5).fill(Color.purple.opacity(0.06)))
                }
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }

    // MARK: - Weekly Report

    private var weeklyReport: some View {
        let summary = engine.weeklySummary
        return VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("Relatorio Semanal")
                    .font(.system(.caption, weight: .semibold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)
                Spacer()
                Text("\(engine.weeklyStats.count + 1) dias")
                    .font(.system(size: 9))
                    .foregroundStyle(.tertiary)
            }

            HStack(spacing: 6) {
                metricCard(icon: "person.badge.clock", color: .orange,
                           label: "Trials", value: "\(summary.trialStarts)")
            }

            // Mini bar chart for daily activity
            if !engine.weeklyStats.isEmpty {
                weeklyChart
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }

    private func metricCard(icon: String, color: Color, label: String, value: String) -> some View {
        VStack(spacing: 3) {
            HStack(spacing: 3) {
                Image(systemName: icon)
                    .font(.system(size: 8))
                    .foregroundStyle(color)
                Text(value)
                    .font(.system(.callout, design: .rounded, weight: .bold))
            }
            Text(label)
                .font(.system(size: 8))
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 6)
        .background(RoundedRectangle(cornerRadius: 6).fill(color.opacity(0.08)))
    }

    private var weeklyChart: some View {
        let allDays = engine.weeklyStats + [engine.todayStats]
        let maxOps = max(allDays.map(\.totalSkillOps).max() ?? 1, 1)

        return HStack(alignment: .bottom, spacing: 3) {
            ForEach(allDays) { day in
                VStack(spacing: 2) {
                    RoundedRectangle(cornerRadius: 2)
                        .fill(Color.purple.opacity(0.6))
                        .frame(height: max(2, CGFloat(day.totalSkillOps) / CGFloat(maxOps) * 30))
                    Text(dayLabel(day.date))
                        .font(.system(size: 7, design: .monospaced))
                        .foregroundStyle(.tertiary)
                }
                .frame(maxWidth: .infinity)
            }
        }
        .frame(height: 45)
    }

    // MARK: - Recent Activity

    private var recentActivity: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Atividade Recente")
                .font(.system(.caption, weight: .semibold))
                .foregroundStyle(.secondary)
                .textCase(.uppercase)

            if engine.recentEvents.isEmpty {
                Text("Nenhum evento de skill ainda — use o agente no WhatsApp")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .padding(.vertical, 4)
            } else {
                ScrollView {
                    VStack(spacing: 4) {
                        ForEach(engine.recentEvents.prefix(8)) { event in
                            eventRow(event)
                        }
                    }
                }
                .frame(maxHeight: 120)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }

    private func eventRow(_ event: SkillEvent) -> some View {
        HStack(spacing: 8) {
            Image(systemName: eventIcon(event.operation))
                .font(.system(size: 10))
                .foregroundStyle(eventColor(event.operation))
                .frame(width: 16)
            VStack(alignment: .leading, spacing: 1) {
                HStack(spacing: 4) {
                    Text(eventLabel(event.operation))
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(eventColor(event.operation))
                    Text(event.skillName)
                        .font(.system(size: 10, design: .monospaced))
                }
                if !event.detail.isEmpty {
                    Text(event.detail)
                        .font(.system(size: 8))
                        .foregroundStyle(.tertiary)
                        .lineLimit(1)
                }
            }
            Spacer()
            Text(relativeDate(event.timestamp))
                .font(.system(size: 8, design: .monospaced))
                .foregroundStyle(.tertiary)
        }
        .padding(.horizontal, 6)
        .padding(.vertical, 3)
    }

    private func eventIcon(_ op: String) -> String {
        switch op {
        case "create": return "plus.circle.fill"
        case "patch", "update": return "wrench.fill"
        case "delete": return "trash.fill"
        case "block": return "shield.slash.fill"
        default: return "circle.fill"
        }
    }

    private func eventColor(_ op: String) -> Color {
        switch op {
        case "create": return .green
        case "patch", "update": return .orange
        case "delete": return .red
        case "block": return .red
        default: return .gray
        }
    }

    private func eventLabel(_ op: String) -> String {
        switch op {
        case "create": return "CRIOU"
        case "patch": return "PATCH"
        case "update": return "UPDATE"
        case "delete": return "DELETOU"
        case "block": return "BLOQUEOU"
        default: return op.uppercased()
        }
    }

    // MARK: - Footer

    private var footerView: some View {
        HStack {
            Button("Quit") {
                NSApplication.shared.terminate(nil)
            }
            .buttonStyle(.plain)
            .foregroundStyle(.secondary)
            .font(.caption)
            Spacer()
            Text("~/.picoclaw/workspace/skills")
                .font(.system(size: 8, design: .monospaced))
                .foregroundStyle(.tertiary)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }

    // MARK: - Helpers

    private func relativeDate(_ date: Date) -> String {
        let elapsed = Date().timeIntervalSince(date)
        if elapsed < 60 { return "agora" }
        if elapsed < 3600 { return "\(Int(elapsed / 60))m" }
        if elapsed < 86400 { return "\(Int(elapsed / 3600))h" }
        return "\(Int(elapsed / 86400))d"
    }

    private func dayLabel(_ dateStr: String) -> String {
        String(dateStr.suffix(2))
    }
}
