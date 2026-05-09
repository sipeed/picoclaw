import { useDeferredValue, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { motion } from "motion/react"
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
import { IconPlus } from "@tabler/icons-react"

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
        <div className="flex items-center gap-2 text-cyan-100/50">
          <motion.div
            className="w-2 h-2 rounded-full bg-[#00bcff]"
            animate={{ opacity: [0.3, 1, 0.3] }}
            transition={{ duration: 1.5, repeat: Infinity }}
          />
          Loading agents...
        </div>
      </div>
    )
  }

  if (agentsQuery.isError) {
    return (
      <div className="flex h-48 items-center justify-center">
        <p className="text-red-400">Failed to load agents. Please try again.</p>
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
    <div className="w-full">
      {/* Header Controls */}
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="max-w-md flex-1">
          <Input
            placeholder={t("common.search", "Search agents...")}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="bg-black/40 border-[#00bcff]/20 text-cyan-100 placeholder:text-cyan-100/30 focus:border-[#00bcff] focus:ring-[#00bcff]/20 backdrop-blur-sm"
          />
        </div>
        <Button
          onClick={() => setIsCreateModalOpen(true)}
          className="bg-[#00bcff]/15 hover:bg-[#00bcff]/25 text-[#00bcff] border border-[#00bcff]/30 hover:border-[#00bcff] shadow-[0_0_15px_rgba(0,188,255,0.1)] transition-all"
        >
          <IconPlus className="w-4 h-4 mr-1" />
          {t("pages.agent.agents.create_agent", "Create Agent")}
        </Button>
      </div>

      {/* Agent Grid */}
      <div className="grid gap-4 grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
        {filteredAgents.length === 0 ? (
          <div className="col-span-full rounded-xl border border-dashed border-[#00bcff]/15 p-12 text-center bg-black/20 backdrop-blur-sm">
            <p className="text-cyan-100/50">
              {deferredSearchQuery
                ? t("pages.agent.agents.no_results", "No agents found")
                : t("pages.agent.agents.no_agents", "No agents yet")}
            </p>
            <p className="mt-2 text-[11px] text-cyan-100/30">
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
        <AlertDialogContent className="bg-[#0a0e27] border-[#00bcff]/20 text-cyan-100 shadow-[0_0_40px_#00bcff1a]">
          <AlertDialogTitle>
            {t("pages.agent.agents.confirm_delete", "Delete Agent?")}
          </AlertDialogTitle>
          <p className="text-cyan-100/50">
            {t(
              "pages.agent.agents.confirm_delete_message",
              `Are you sure you want to delete "${agentToDelete?.name}"? This action cannot be undone.`,
            )}
          </p>
          <AlertDialogFooter>
            <AlertDialogCancel className="border-[#00bcff]/20 text-cyan-100/60 hover:text-cyan-100 hover:bg-[#00bcff]/5">
              {t("common.cancel", "Cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/30"
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
    <div className="bg-[#0a0e27] flex h-full flex-col">
      <PageHeader title={t("navigation.agents", "Agents")} />
      <div className="flex-1 overflow-auto px-6 py-6 pb-20">
        {mainContent}
      </div>
      {modals}
    </div>
  )
}