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

  // Dynamic weixin channels: "weixin_2" -> "WeChat 2", "weixin_account1" -> "WeChat Account1"
  if (channel.name !== "weixin" && channel.name.startsWith("weixin")) {
    const suffix = channel.name.slice("weixin_".length)
    const wechatBase = t("channels.name.weixin", "WeChat")
    if (suffix) {
      return `${wechatBase} ${suffix.split("_").map((s) => s.charAt(0).toUpperCase() + s.slice(1)).join(" ")}`
    }
  }

  return channel.name
    .split("_")
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(" ")
}
