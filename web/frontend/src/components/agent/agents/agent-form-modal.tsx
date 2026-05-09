import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  createAgent,
  updateAgent,
  type Agent,
  type AgentCreateRequest,
} from "@/api/agents"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { useChatModels } from "@/hooks/use-chat-models"
import { useGateway } from "@/hooks/use-gateway"

interface AgentFormModalProps {
  isOpen: boolean
  onClose: (open: boolean) => void
  agent?: Agent | null
  onSave?: () => void
}

export function AgentFormModal({
  isOpen,
  onClose,
  agent,
  onSave,
}: AgentFormModalProps) {
  const { t } = useTranslation()
  const { state: gatewayState } = useGateway()
  const { localModels, oauthModels, defaultModelName } = useChatModels({
    isConnected: gatewayState === "running",
  })

  const isEdit = !!agent

  const [name, setName] = useState("")
  const [description, setDescription] = useState("")
  const [systemPrompt, setSystemPrompt] = useState("")
  const [model, setModel] = useState("")
  const [toolPermissions, setToolPermissions] = useState("")

  // Reset form when modal opens with new agent data
  useEffect(() => {
    if (isOpen) {
      if (agent) {
        setName(agent.name)
        setDescription(agent.description || "")
        setSystemPrompt(agent.system_prompt || "")
        setModel(agent.model || "")
        setToolPermissions((agent.tool_permissions || []).join(", "))
      } else {
        setName("")
        setDescription("")
        setSystemPrompt("")
        setModel(defaultModelName || "")
        setToolPermissions("")
      }
    }
  }, [agent, defaultModelName, isOpen])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!name.trim()) {
      toast.error("Name is required")
      return
    }
    if (!systemPrompt.trim()) {
      toast.error("System prompt is required")
      return
    }
    if (!model.trim()) {
      toast.error("Model is required")
      return
    }

    const request: AgentCreateRequest = {
      name: name.trim(),
      description: description.trim(),
      system_prompt: systemPrompt.trim(),
      model: model.trim(),
      tool_permissions: toolPermissions
        .split(",")
        .map((s) => s.trim())
        .filter((s) => s.length > 0),
    }

    try {
      if (isEdit && agent) {
        await updateAgent(agent.slug, request)
        toast.success(
          t("pages.agent.agents.update_success", "Agent updated successfully"),
        )
      } else {
        await createAgent(request)
        toast.success(
          t("pages.agent.agents.create_success", "Agent created successfully"),
        )
      }
      onSave?.()
      onClose(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to save agent")
    }
  }

  const availableModels = [...localModels, ...oauthModels]

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEdit
              ? t("pages.agent.agents.edit_agent", "Edit Agent")
              : t("pages.agent.agents.create_agent", "Create Agent")}
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">
              {t("pages.agent.agents.form_name", "Name")}
              <span className="text-destructive"> *</span>
            </Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., researcher, coder"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">
              {t("pages.agent.agents.form_description", "Description")}
            </Label>
            <Textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Brief description of what this agent does"
              rows={2}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="systemPrompt">
              {t("pages.agent.agents.form_system_prompt", "System Prompt")}
              <span className="text-destructive"> *</span>
            </Label>
            <Textarea
              id="systemPrompt"
              value={systemPrompt}
              onChange={(e) => setSystemPrompt(e.target.value)}
              placeholder="Define the agent's personality, capabilities, and instructions..."
              rows={6}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="model">
              {t("pages.agent.agents.form_model", "Model")}
              <span className="text-destructive"> *</span>
            </Label>
            <select
              id="model"
              value={model}
              onChange={(e) => setModel(e.target.value)}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option value="">
                {availableModels.length === 0
                  ? t(
                      "pages.agent.agents.no_models_available",
                      "No models available",
                    )
                  : t("pages.agent.agents.select_model", "Select a model")}
              </option>
              {availableModels.map((m) => (
                <option key={m.model_name} value={m.model_name}>
                  {m.model_name}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="toolPermissions">
              {t("pages.agent.agents.form_tool_permissions", "Tool Permissions")}
            </Label>
            <Input
              id="toolPermissions"
              value={toolPermissions}
              onChange={(e) => setToolPermissions(e.target.value)}
              placeholder="web_search, file_read, file_write (comma-separated)"
            />
            <p className="text-xs text-muted-foreground">
              {t(
                "pages.agent.agents.tool_permissions_hint",
                "List tools this agent can use (comma-separated)",
              )}
            </p>
          </div>
        </form>

        <DialogFooter className="flex-row justify-end gap-2">
          <Button variant="outline" onClick={() => onClose(false)}>
            {t("common.cancel", "Cancel")}
          </Button>
          <Button onClick={handleSubmit}>
            {isEdit
              ? t("common.save", "Save")
              : t("common.create", "Create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}