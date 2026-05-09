import { IconCheck, IconClock, IconDownload, IconFileTypePdf } from "@tabler/icons-react"
import { cn } from "@/lib/utils"
import type { ResearchReport } from "@/api/research"
import { downloadReport } from "@/api/research"

interface ResearchReportsProps {
  reports: ResearchReport[]
}

export function ResearchReports({ reports }: ResearchReportsProps) {
  const handleExport = async (reportId: string, title: string, format: "markdown" | "pdf") => {
    try {
      await downloadReport(reportId, title, format)
    } catch (error) {
      console.error("Export failed:", error)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between border-b border-white/10 pb-2">
        <span className="text-[10px] uppercase tracking-[0.22em] font-bold text-[#F27D26]">
          Recent Reports
        </span>
        <span className="text-[10px] text-white/40 font-mono">
          {reports.length} total
        </span>
      </div>

      <div className="space-y-2">
        {reports.map((report) => (
          <div
            key={report.id}
            className="rounded-lg border border-white/10 bg-[#0A0A0A] p-3 hover:border-white/20 transition-colors cursor-pointer group"
          >
            <div className="flex items-start gap-2">
              <div className={cn(
                "w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0",
                report.status === "complete"
                  ? "bg-green-500/20"
                  : "bg-[#F27D26]/20"
              )}>
                {report.status === "complete" ? (
                  <IconCheck className="w-4 h-4 text-green-400" />
                ) : (
                  <IconClock className="w-4 h-4 text-[#F27D26]" />
                )}
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-xs font-medium text-[#F2F2F2] truncate">
                  {report.title}
                </div>
                <div className="flex items-center gap-2 mt-1">
                  {report.pages && (
                    <>
                      <span className="text-white/20">•</span>
                      <span className="text-[10px] text-white/40">
                        {report.pages} pages
                      </span>
                    </>
                  )}
                </div>
              </div>
            </div>

            {/* Export buttons - visible on hover for completed reports */}
            {report.status === "complete" && (
              <div className="flex gap-1 mt-2 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    handleExport(report.id, report.title, "markdown")
                  }}
                  className="flex items-center gap-1 px-2 py-1 rounded bg-white/5 hover:bg-white/10 text-[10px] text-white/60 hover:text-white transition-colors"
                  title="Export as Markdown"
                >
                  <IconDownload className="w-3 h-3" />
                  MD
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    handleExport(report.id, report.title, "pdf")
                  }}
                  className="flex items-center gap-1 px-2 py-1 rounded bg-white/5 hover:bg-white/10 text-[10px] text-white/60 hover:text-white transition-colors"
                  title="Export as PDF"
                >
                  <IconFileTypePdf className="w-3 h-3" />
                  PDF
                </button>
              </div>
            )}
          </div>
        ))}
      </div>

      <button className="w-full px-3 py-2 rounded-lg bg-[#050505] border border-white/10 text-[10px] text-white/60 font-medium hover:border-white/30 hover:text-white transition-all">
        View All Reports
      </button>
    </div>
  )
}