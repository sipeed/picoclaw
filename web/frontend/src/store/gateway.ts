import { atom, getDefaultStore } from "jotai"

import { type GatewayStatusResponse, getGatewayStatus } from "@/api/gateway"

export type GatewayState =
  | "running"
  | "starting"
  | "restarting"
  | "stopped"
  | "error"
  | "unknown"

export interface GatewayStoreState {
  status: GatewayState
  canStart: boolean
  restartRequired: boolean
}

type GatewayStorePatch = Partial<GatewayStoreState>

const DEFAULT_GATEWAY_STATE: GatewayStoreState = {
  status: "unknown",
  canStart: true,
  restartRequired: false,
}

// Global atom for gateway state
export const gatewayAtom = atom<GatewayStoreState>(DEFAULT_GATEWAY_STATE)

function normalizeGatewayStoreState(
  prev: GatewayStoreState,
  patch: GatewayStorePatch,
) {
  return { ...prev, ...patch }
}

export function updateGatewayStore(
  patch:
    | GatewayStorePatch
    | ((prev: GatewayStoreState) => GatewayStorePatch | GatewayStoreState),
) {
  getDefaultStore().set(gatewayAtom, (prev) => {
    const nextPatch = typeof patch === "function" ? patch(prev) : patch
    return normalizeGatewayStoreState(prev, nextPatch)
  })
}

export function applyGatewayStatusToStore(
  data: Partial<
    Pick<
      GatewayStatusResponse,
      "gateway_status" | "gateway_start_allowed" | "gateway_restart_required"
    >
  >,
) {
  updateGatewayStore((prev) => ({
    status: data.gateway_status ?? prev.status,
    canStart: data.gateway_start_allowed ?? prev.canStart,
    restartRequired:
      data.gateway_restart_required ??
      (data.gateway_status && data.gateway_status !== "running"
        ? false
        : prev.restartRequired),
  }))
}

export async function refreshGatewayState() {
  try {
    const status = await getGatewayStatus()
    applyGatewayStatusToStore(status)
  } catch {
    updateGatewayStore(DEFAULT_GATEWAY_STATE)
  }
}
