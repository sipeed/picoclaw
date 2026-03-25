import { Plus, Trash2 } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"

import { ControlItem } from "./control-item"

export interface RoutingTier {
  model: string
  threshold: number
}

interface RoutingSectionProps {
  enabled: boolean
  tiers: RoutingTier[]
  onEnabledChange: (enabled: boolean) => void
  onTiersChange: (tiers: RoutingTier[]) => void
}

export function RoutingSection({
  enabled,
  tiers,
  onEnabledChange,
  onTiersChange,
}: RoutingSectionProps) {
  const { t } = useTranslation()

  const handleAddTier = () => {
    onTiersChange([...tiers, { model: "", threshold: 0 }])
  }

  const handleRemoveTier = (index: number) => {
    onTiersChange(tiers.filter((_, i) => i !== index))
  }

  const handleTierChange = (
    index: number,
    field: keyof RoutingTier,
    value: string | number,
  ) => {
    const newTiers = [...tiers]
    newTiers[index] = { ...newTiers[index], [field]: value }
    onTiersChange(newTiers)
  }

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">
        {t("pages.config.routing.title", "Model Routing")}
      </h2>
      <div className="bg-card rounded-lg border p-6 shadow-sm">
        <ControlItem
          label={t(
            "pages.config.routing.enabled",
            "Enable Intelligent Routing",
          )}
          hint={t(
            "pages.config.routing.enabled_hint",
            "Route messages to different models based on complexity score",
          )}
          control={
            <Switch checked={enabled} onCheckedChange={onEnabledChange} />
          }
        />

        {enabled && (
          <div className="mt-6 space-y-4 border-t pt-4">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium">
                {t("pages.config.routing.tiers", "Routing Tiers")}
              </Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleAddTier}
              >
                <Plus className="mr-2 h-4 w-4" />
                {t("pages.config.routing.add_tier", "Add Tier")}
              </Button>
            </div>

            <div className="space-y-3">
              {tiers.length === 0 && (
                <div className="text-muted-foreground py-4 text-center text-sm italic">
                  {t(
                    "pages.config.routing.no_tiers",
                    "No routing tiers defined. Add a tier to configure routing.",
                  )}
                </div>
              )}
              {tiers.map((tier, index) => (
                <div
                  key={index}
                  className="bg-muted/20 flex items-end gap-3 rounded-md border p-3"
                >
                  <div className="flex-1 space-y-1">
                    <Label className="text-xs">
                      {t("pages.config.routing.model", "Model Name")}
                    </Label>
                    <Input
                      value={tier.model}
                      onChange={(e) =>
                        handleTierChange(index, "model", e.target.value)
                      }
                      placeholder="e.g. gpt-4o-mini"
                      className="h-8"
                    />
                  </div>
                  <div className="w-24 space-y-1">
                    <Label className="text-xs">
                      {t("pages.config.routing.threshold", "Threshold")}
                    </Label>
                    <Input
                      type="number"
                      step="0.01"
                      min="0"
                      max="1"
                      value={tier.threshold}
                      onChange={(e) =>
                        handleTierChange(
                          index,
                          "threshold",
                          parseFloat(e.target.value) || 0,
                        )
                      }
                      className="h-8"
                    />
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="text-destructive h-8 w-8"
                    onClick={() => handleRemoveTier(index)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
