import { getRepositoryReviewAutomationRunFinding } from "@/api/repository-reviews"

export async function resolveRepositoryFindingRouteID(
  automationID: string,
  findingID: string,
): Promise<string | undefined> {
  if (!findingID.startsWith("rfn_")) return findingID
  try {
    const detail = await getRepositoryReviewAutomationRunFinding(
      automationID,
      findingID,
    )
    return detail.repository_finding?.id
  } catch {
    return undefined
  }
}
