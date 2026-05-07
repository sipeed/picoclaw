import { useDeferredValue, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  deleteAgent,
  listAgents,
  type Agent,
  updateAgent,
} from "@/api/agents"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { useQuery, useQueryClient } from "@tanstack/react-query"

import { AgentCard } from "./agent-card"
import { AgentFormModal } from "./agent-form-modal"

interface AgentsPageProps {
  embedded?: boolean
}

export function AgentsPage({ embedded = false }: AgentsPageProps = {}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const [searchQuery, setSearchQuery] = useState("")
  const deferredSearchQuery = useDeferredValue(searchQuery)
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null)
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [isEditModalOpen, setIsEditModalOpen] = useState(false)
  const [agentToDelete, setAgentToDelete] = useState<Agent | null>(null)

  const agentsQuery = useQuery({
    queryKey: ["agents"],
    queryFn: listAgents,
  })

  const filteredAgents = useMemo(() => {
    const agents = agentsQuery.data?.agents ?? []
    const query = deferredSearchQuery.trim().toLowerCase()
    if (!query) return agents

    return agents.filter(
      (agent) =>
        agent.name.toLowerCase().includes(query) ||
        agent.description.toLowerCase().includes(query) ||
        agent.slug.toLowerCase().includes(query),
    )
  }, [agentsQuery.data?.agents, deferredSearchQuery])

  if (agentsQuery.isLoading) {
    return (
      <div className="flex h-48 items-center justify-center">
        <p className="text-muted-foreground">Loading agents...</p>
      </div>
    )
  }

  if (agentsQuery.isError) {
    return (
      <div className="flex h-48 items-center justify-center">
        <p className="text-destructive">Failed to load agents. Please try again.</p>
      </div>
    )
  }

  const handleEdit = (agent: Agent) => {
    setSelectedAgent(agent)
    setIsEditModalOpen(true)
  }

  const handleDelete = async () => {
    if (!agentToDelete) return

    try {
      await deleteAgent(agentToDelete.slug)
      toast.success(t("pages.agent.agents.delete_success", "Agent deleted"))
      void queryClient.invalidateQueries({ queryKey: ["agents"] })
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Failed to delete agent",
      )
    } finally {
      setAgentToDelete(null)
    }
  }

  const handleToggle = async (agent: Agent, enabled: boolean) => {
    try {
      await updateAgent(agent.slug, { status: enabled ? "enabled" : "disabled" })
      toast.success(
        enabled
          ? t("pages.agent.agents.enable_success", "Agent enabled")
          : t("pages.agent.agents.disable_success", "Agent disabled"),
      )
      void queryClient.invalidateQueries({ queryKey: ["agents"] })
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "Failed to update agent",
      )
      // Refresh to revert the UI state
      void queryClient.invalidateQueries({ queryKey: ["agents"] })
    }
  }

  const handleModalClose = (open: boolean) => {
    if (!open) {
      setIsCreateModalOpen(false)
      setIsEditModalOpen(false)
      setSelectedAgent(null)
    }
  }

  const mainContent = (
    <div className="mx-auto w-full max-w-6xl">
      {/* Header Controls */}
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="max-w-md flex-1">
          <Input
            placeholder={t("common.search", "Search agents...")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="bg-background"
          />
        </div>
        <Button
          onClick={() => setIsCreateModalOpen(true)}
          className="bg-[#F27D26] hover:bg-[#F27D26]/90 text-black"
        >
          {t("pages.agent.agents.create_agent", "+ Create Agent")}
        </Button>
      </div>

      {/* Agent Grid */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {filteredAgents.length === 0 ? (
          <div className="col-span-full rounded-lg border border-dashed border-white/10 p-12 text-center">
            <p className="text-muted-foreground">
              {deferredSearchQuery
                ? t("pages.agent.agents.no_results", "No agents found")
                : t("pages.agent.agents.no_agents", "No agents yet")}
            </p>
            <p className="mt-2 text-sm text-muted-foreground/60">
              {deferredSearchQuery
                ? t("pages.agent.agents.no_results_hint", "Try a different search")
                : t("pages.agent.agents.no_agents_hint", "Create an agent to get started")}
            </p>
          </div>
        ) : (
          filteredAgents.map((agent) => (
            <AgentCard
              key={agent.slug}
              agent={agent}
              onEdit={() => handleEdit(agent)}
              onDelete={() => setAgentToDelete(agent)}
              onToggle={(enabled) => handleToggle(agent, enabled)}
            />
          ))
        )}
      </div>
    </div>
  );

  const modals = (
    <>
      {/* Create Modal */}
      <AgentFormModal
        isOpen={isCreateModalOpen}
        onClose={() => handleModalClose(false)}
        onSave={() =>
          void queryClient.invalidateQueries({ queryKey: ["agents"] })
        }
      />

      {/* Edit Modal */}
      <AgentFormModal
        isOpen={isEditModalOpen}
        onClose={() => handleModalClose(false)}
        agent={selectedAgent}
        onSave={() =>
          void queryClient.invalidateQueries({ queryKey: ["agents"] })
        }
      />

      {/* Delete Confirmation */}
      <AlertDialog
        open={agentToDelete !== null}
        onOpenChange={(open) => {
          if (!open) setAgentToDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("pages.agent.agents.confirm_delete", "Delete Agent?")}
          </AlertDialogTitle>
          <p className="text-muted-foreground">
            {t(
              "pages.agent.agents.confirm_delete_message",
              `Are you sure you want to delete "${agentToDelete?.name}"? This action cannot be undone.`,
            )}
          </p>
          <AlertDialogFooter>
            <AlertDialogCancel>
              {t("common.cancel", "Cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t("common.delete", "Delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );

  if (embedded) {
    return (
      <>
        {mainContent}
        {modals}
      </>
    );
  }

  return (
    <div className="bg-background flex h-full flex-col">
      <PageHeader title={t("navigation.agents", "Agents")} />
      <div className="flex-1 overflow-auto px-6 py-6 pb-20">
        {mainContent}
      </div>
      {modals}
    </div>
  )
}