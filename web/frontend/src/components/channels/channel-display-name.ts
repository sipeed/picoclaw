import type { TFunction } from "i18next"

import type { SupportedChannel } from "@/api/channels"

export function getChannelDisplayName(
  channel: Pick<SupportedChannel, "name" | "display_name">,
  t: TFunction,
): string {
  const key = `channels.name.${channel.name}`
  const translated = t(key)
  if (translated !== key) {
    return translated
  }

  if (channel.display_name && channel.display_name.trim() !== "") {
    return channel.display_name
  }

  return channel.name
    .split("_")
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(" ")
}
