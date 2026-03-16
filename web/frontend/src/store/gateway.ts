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
  startReason: string
  passphraseState: "" | "pending" | "failed"
}

type GatewayStorePatch = Partial<GatewayStoreState>

// Global atom for gateway state
export const gatewayAtom = atom<GatewayStoreState>({
  status: "unknown",
  canStart: true,
  startReason: "",
  passphraseState: "",
})

export function updateGatewayStore(
  patch:
    | GatewayStorePatch
    | ((prev: GatewayStoreState) => GatewayStorePatch | GatewayStoreState),
) {
  getDefaultStore().set(gatewayAtom, (prev) => {
    const nextPatch = typeof patch === "function" ? patch(prev) : patch
    return { ...prev, ...nextPatch }
  })
}

function applyGatewayStatusToStore(data: GatewayStatusResponse) {
  getDefaultStore().set(gatewayAtom, (prev) => ({
    ...prev,
    status: data.gateway_status ?? "unknown",
    canStart: data.gateway_start_allowed ?? true,
    startReason: data.gateway_start_reason ?? "",
    passphraseState: data.passphrase_state ?? "",
  }))
}

export async function refreshGatewayState() {
  try {
    const status = await getGatewayStatus()
    applyGatewayStatusToStore(status)
  } catch {
    updateGatewayStore({ status: "unknown", canStart: true, startReason: "", passphraseState: "" })
  }
}
