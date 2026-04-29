import type { ChannelConfig } from "@/api/channels"
import { getSecretInputPlaceholder } from "@/components/channels/channel-config-fields"
import { Field, KeyInput } from "@/components/shared-form"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { useTranslation } from "react-i18next"

interface MqttFormProps {
  config: ChannelConfig
  onChange: (key: string, value: unknown) => void
  configuredSecrets: string[]
  fieldErrors?: Record<string, string>
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asNumber(value: unknown): string {
  if (typeof value === "number") return String(value)
  if (typeof value === "string" && value !== "") return value
  return ""
}

function CodeLine({ children }: { children: string }) {
  return (
    <code className="bg-muted text-foreground block rounded px-3 py-1 font-mono text-xs">
      {children}
    </code>
  )
}

export function MqttForm({
  config,
  onChange,
  configuredSecrets,
  fieldErrors = {},
}: MqttFormProps) {
  const { t } = useTranslation()
  const prefix = asString(config.topic_prefix) || "/picoclaw"
  const agentID = asString(config.agent_id) || "{agent_id}"
  const topicBase = `${prefix}/${agentID}/{client_id}`

  return (
    <div className="space-y-6">
      <Card className="shadow-sm">
        <CardContent className="divide-border/60 divide-y px-6 py-0 [&>div]:py-5">
          <Field
            label="Broker"
            required
            hint={t("channels.form.desc.broker")}
            error={fieldErrors.broker}
          >
            <Input
              value={asString(config.broker)}
              onChange={(e) => onChange("broker", e.target.value)}
              placeholder="mqtt://broker.example.com:1883"
            />
          </Field>

          <Field
            label="Agent ID"
            required
            hint={t("channels.form.desc.mqttAgentId")}
            error={fieldErrors.agent_id}
          >
            <Input
              value={asString(config.agent_id)}
              onChange={(e) => onChange("agent_id", e.target.value)}
              placeholder="my-agent"
            />
          </Field>

          <Field
            label="Topic Prefix"
            hint={t("channels.form.desc.topicPrefix")}
          >
            <Input
              value={asString(config.topic_prefix)}
              onChange={(e) => onChange("topic_prefix", e.target.value)}
              placeholder="/picoclaw"
            />
          </Field>
        </CardContent>
      </Card>

      <Card className="shadow-sm">
        <CardContent className="divide-border/60 divide-y px-6 py-0 [&>div]:py-5">
          <Field
            label="Username"
            hint={t("channels.form.desc.mqttUsername")}
          >
            <KeyInput
              value={asString(config._username)}
              onChange={(v) => onChange("_username", v)}
              placeholder={getSecretInputPlaceholder(
                configuredSecrets,
                "username",
                t("channels.mqtt.secretSet"),
                t("channels.mqtt.secretEmpty"),
              )}
            />
          </Field>

          <Field
            label="Password"
            hint={t("channels.form.desc.mqttPassword")}
          >
            <KeyInput
              value={asString(config._password)}
              onChange={(v) => onChange("_password", v)}
              placeholder={getSecretInputPlaceholder(
                configuredSecrets,
                "password",
                t("channels.mqtt.secretSet"),
                t("channels.mqtt.secretEmpty"),
              )}
            />
          </Field>
        </CardContent>
      </Card>

      <Card className="shadow-sm">
        <CardContent className="divide-border/60 divide-y px-6 py-0 [&>div]:py-5">
          <Field
            label="Client ID"
            hint={t("channels.form.desc.mqttClientId")}
          >
            <Input
              value={asString(config.client_id)}
              onChange={(e) => onChange("client_id", e.target.value)}
              placeholder={t("channels.mqtt.clientIdPlaceholder")}
            />
          </Field>

          <Field
            label="Keep Alive"
            hint={t("channels.form.desc.keepAlive")}
          >
            <Input
              type="number"
              value={asNumber(config.keep_alive)}
              onChange={(e) => onChange("keep_alive", Number(e.target.value))}
              placeholder="60"
            />
          </Field>

          <Field
            label="QoS"
            hint={t("channels.form.desc.qos")}
          >
            <Input
              type="number"
              value={asNumber(config.qos)}
              onChange={(e) => onChange("qos", Number(e.target.value))}
              placeholder="0"
            />
          </Field>
        </CardContent>
      </Card>

      <Card className="shadow-sm">
        <CardHeader className="border-border/60 border-b px-6 py-4">
          <CardTitle className="text-foreground text-sm font-medium">
            {t("channels.mqtt.protocolTitle")}
          </CardTitle>
          <CardDescription>
            {t("channels.mqtt.protocolDesc")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5 px-6 py-5">
          <div className="space-y-2">
            <p className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
              {t("channels.mqtt.uplink")}
            </p>
            <CodeLine>{`${topicBase}/request`}</CodeLine>
            <pre className="bg-muted text-foreground rounded px-3 py-2 font-mono text-xs leading-relaxed">
              {`{\n  "text": "your message"\n}`}
            </pre>
            <div className="text-muted-foreground space-y-1 text-xs">
              <p>
                <span className="text-foreground font-medium">
                  {t("channels.mqtt.fieldText")}
                </span>
                {" — "}
                {t("channels.mqtt.uplinkTextDesc")}
              </p>
            </div>
          </div>

          <div className="space-y-2">
            <p className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
              {t("channels.mqtt.downlink")}
            </p>
            <CodeLine>{`${topicBase}/response`}</CodeLine>
            <pre className="bg-muted text-foreground rounded px-3 py-2 font-mono text-xs leading-relaxed">
              {`{\n  "text": "agent response"\n}`}
            </pre>
            <div className="text-muted-foreground space-y-1 text-xs">
              <p>
                <span className="text-foreground font-medium">
                  {t("channels.mqtt.fieldText")}
                </span>
                {" — "}
                {t("channels.mqtt.downlinkTextDesc")}
              </p>
            </div>
          </div>

          <div className="space-y-2">
            <p className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
              {t("channels.mqtt.topicParams")}
            </p>
            <div className="text-muted-foreground space-y-1 text-xs">
              <p>
                <span className="text-foreground font-mono font-medium">
                  {prefix}
                </span>
                {" — "}
                {t("channels.mqtt.topicPrefixDesc")}
              </p>
              <p>
                <span className="text-foreground font-mono font-medium">
                  {agentID}
                </span>
                {" — "}
                {t("channels.mqtt.agentIdDesc")}
              </p>
              <p>
                <span className="text-foreground font-mono font-medium">
                  {"{client_id}"}
                </span>
                {" — "}
                {t("channels.mqtt.clientIdDesc")}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
