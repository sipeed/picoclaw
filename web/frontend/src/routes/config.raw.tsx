import { IconAdjustments } from "@tabler/icons-react"
import { Link, createFileRoute } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"

import { RawJsonPanel } from "@/components/config/raw-json-panel"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"

export const Route = createFileRoute("/config/raw")({
  component: RawConfigPage,
})

function RawConfigPage() {
  const { t } = useTranslation()

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("pages.config.raw_json_title")}>
        <Button variant="outline" asChild>
          <Link to="/config">
            <IconAdjustments className="size-4" />
            {t("pages.config.back_to_visual")}
          </Link>
        </Button>
      </PageHeader>

      <div className="flex-1 overflow-auto p-3 lg:p-6">
        <div className="mx-auto max-w-4xl">
          <RawJsonPanel />
        </div>
      </div>
    </div>
  )
}
