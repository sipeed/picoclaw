import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { t } from "i18next"

import {
  listAgents,
  deleteAgent,
  createAgent,
  updateAgent,
  importAgent,
  type Agent,
  type AgentCreateRequest,
} from "@/api/agents"

export function useAgents() {
  const queryClient = useQueryClient()

  const listQuery = useQuery({
    queryKey: ["agents"],
    queryFn: listAgents,
  })

  const deleteMutation = useMutation({
    mutationFn: deleteAgent,
    onSuccess: () => {
      toast.success(t("pages.agent.agents.delete_success", "Agent deleted"))
      queryClient.invalidateQueries({ queryKey: ["agents"] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Failed to delete agent")
    },
  })

  const createMutation = useMutation({
    mutationFn: (data: AgentCreateRequest) => createAgent(data),
    onSuccess: () => {
      toast.success(t("pages.agent.agents.create_success", "Agent created"))
      queryClient.invalidateQueries({ queryKey: ["agents"] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Failed to create agent")
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ slug, data }: { slug: string; data: Agent }) =>
      updateAgent(slug, data),
    onSuccess: () => {
      toast.success(t("pages.agent.agents.update_success", "Agent updated"))
      queryClient.invalidateQueries({ queryKey: ["agents"] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Failed to update agent")
    },
  })

  const importMutation = useMutation({
    mutationFn: (content: string) => importAgent(content),
    onSuccess: () => {
      toast.success(t("pages.agent.agents.import_success", "Agent imported"))
      queryClient.invalidateQueries({ queryKey: ["agents"] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Failed to import agent")
    },
  })

  return {
    agents: listQuery.data?.agents ?? [],
    isLoading: listQuery.isLoading,
    isError: listQuery.isError,
    deleteAgent: deleteMutation.mutate,
    createAgent: createMutation.mutate,
    updateAgent: updateMutation.mutate,
    importAgent: importMutation.mutate,
  }
}
