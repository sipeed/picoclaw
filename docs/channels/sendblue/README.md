# Sendblue: iMessage and SMS

Connect direct phone conversations to PicoClaw using Sendblue's hosted API.
No Apple device is needed on the PicoClaw host.

Build PicoClaw from a checkout containing this channel and complete
[model setup](../../../README.md). First confirm that
`picoclaw agent -m "Hello"` works.

## Create a Sendblue account

Run the [Sendblue CLI](https://docs.sendblue.com/getting-started/credentials)
with Node.js installed:

```sh
npx -y @sendblue/cli@0.10.0 setup --phone +15550000002
```

Use your own phone in E.164 format. Complete the challenge by **sending the
requested text from that phone**. The CLI saves credentials to
`~/.sendblue/credentials.json`; `apiKey`, `apiSecret`, and `assignedNumber` are
the values needed below. If setup is pending, run
`npx -y @sendblue/cli@0.10.0 setup --check` after sending the text.

The free plan uses a shared Sendblue line and verified contacts. Additional
contacts must be added with `npx -y @sendblue/cli@0.10.0 add-contact` and verify
by texting the line before you can reply to them. A channel allowlist alone
does not complete Sendblue's contact verification. Use the **assigned** line,
not your personal phone, as the sending number. Model-provider credentials and
usage are separate from Sendblue account setup.

Generate a webhook secret, then privately load the CLI credentials into the
shell that will run the gateway (do not commit them):

```sh
export SENDBLUE_WEBHOOK_SECRET="$(openssl rand -hex 32)"
export SENDBLUE_API_KEY="$(node -p 'JSON.parse(require("fs").readFileSync(require("os").homedir()+"/.sendblue/credentials.json","utf8")).apiKey')"
export SENDBLUE_API_SECRET="$(node -p 'JSON.parse(require("fs").readFileSync(require("os").homedir()+"/.sendblue/credentials.json","utf8")).apiSecret')"
export SENDBLUE_FROM_NUMBER="$(node -p 'JSON.parse(require("fs").readFileSync(require("os").homedir()+"/.sendblue/credentials.json","utf8")).assignedNumber')"
```

Keep these environment variables in your service's secret configuration for
subsequent starts. Regenerating the webhook secret requires updating Sendblue's
subscription too.

## Configure PicoClaw

Export the channel's environment variables in the gateway shell:

```sh
export PICOCLAW_CHANNELS_SENDBLUE_API_KEY="$SENDBLUE_API_KEY"
export PICOCLAW_CHANNELS_SENDBLUE_API_SECRET="$SENDBLUE_API_SECRET"
export PICOCLAW_CHANNELS_SENDBLUE_FROM_NUMBER="$SENDBLUE_FROM_NUMBER"
export PICOCLAW_CHANNELS_SENDBLUE_WEBHOOK_SECRET="$SENDBLUE_WEBHOOK_SECRET"
```

Merge this into `~/.picoclaw/config.json`, keeping your model configuration.
Use your verified personal phone in `allow_from`:

```json
{
  "channel_list": {
    "sendblue": {
      "type": "sendblue",
      "enabled": true,
      "allow_from": ["+15550000002"],
      "settings": {}
    }
  }
}
```

Start `picoclaw gateway`. The channel registers with the shared Gateway HTTP
server (default `127.0.0.1:18790`) at `/webhook/sendblue/sendblue`. The final
path component is the channel instance name; a second instance called
`phone2` uses `/webhook/sendblue/phone2`.

Alternatively, populate `settings.api_key`, `settings.api_secret`,
`settings.from_number`, and `settings.webhook_secret` directly. Secrets use
PicoClaw's `SecureString` support, including `file://` references. For multiple
accounts, use per-instance settings and leave the shared environment overrides
unset. Credential changes participate in the channel reload hash.

## Register the receive webhook

Expose only the webhook route through an HTTPS reverse proxy or tunnel. Keep
administrative routes private. Set `PUBLIC_WEBHOOK_URL` to its complete public
URL, including the path described above, then run:

```sh
export PUBLIC_WEBHOOK_URL="https://your-public-host.example/REPLACE_WITH_WEBHOOK_PATH"
python3 - <<'PYTHON'
import json, os, urllib.request
payload = {"webhooks": {"receive": [{
    "url": os.environ["PUBLIC_WEBHOOK_URL"],
    "secret": os.environ["SENDBLUE_WEBHOOK_SECRET"],
    "sendblue_numbers": [os.environ["SENDBLUE_FROM_NUMBER"]],
}]}}
request = urllib.request.Request(
    "https://api.sendblue.com/api/account/webhooks",
    data=json.dumps(payload).encode(),
    headers={"Content-Type": "application/json",
             "sb-api-key-id": os.environ["SENDBLUE_API_KEY"],
             "sb-api-secret-key": os.environ["SENDBLUE_API_SECRET"]},
    method="POST",
)
with urllib.request.urlopen(request, timeout=30) as response:
    print("Webhook registration HTTP status:", response.status)
PYTHON
```

This [appends a subscription](https://docs.sendblue.com/api/resources/webhooks)
without replacing existing webhooks. Run it once for each new URL; remove an
old subscription when changing URLs. The per-webhook secret must match the
channel secret. Sendblue passes it in `sb-signing-secret` (a shared secret,
not an HMAC signature).

## Check the full conversation

1. Start the gateway with a configured, working model provider.
2. From your verified, allowlisted phone, send a unique text to the assigned
   Sendblue number, such as `Reply with: hello 2026-10-05`.
3. Confirm the webhook gets HTTP 204 and the agent receives the text in a
   direct-message session keyed by your phone number.
4. Confirm the agent's answer arrives on the same phone from the configured
   Sendblue line. Repeat with a follow-up to check session continuity.
5. To verify SMS as well as iMessage, use an eligible verified SMS recipient;
   an iMessage-only run does not prove SMS delivery. Sendblue handles service
   selection according to the line, recipient, and account capabilities.

401 at the webhook means the secret differs. 503 means the channel/queue is
unavailable. A successful webhook with no reply often means the sender is not
allowlisted, the line number differs, or Sendblue contact verification is
incomplete. Check the provider and gateway logs too. HTTP acceptance or a
`QUEUED` send result is not confirmation of delivery to the phone.

## Scope and delivery behavior

- Direct-message text over iMessage/SMS. Group messages and outbound status
  callbacks are ignored. Incoming attachments produce a text notice; files
  are not downloaded or sent by this channel.
- Requests require authentication, have a 64 KiB body limit, and must target
  the configured line. An explicit phone allowlist is required; `"*"` opts in
  to unrestricted senders. No unsolicited pairing replies are sent.
- The most recent 4096 accepted message handles are deduplicated in memory.
  Restarting or running multiple gateway processes resets/does not share
  this window. Run one gateway per subscription. A 204 means the message was
  accepted by the in-memory agent bus, not that the agent completed it;
  there is no crash-safe inbox or exactly-once processing guarantee.
- Outbound POSTs are not automatically retried, including ambiguous timeouts,
  because Sendblue does not offer an idempotency key for this endpoint.
  Inspect Sendblue message history before manually retrying. A long answer
  is split into 2000-character messages and may be partially delivered if a
  later chunk fails.

## Local verification

```sh
go test -race ./pkg/channels/sendblue ./pkg/config ./pkg/channels
```

Tests load the channel through the factory registry and exercise the webhook,
real agent loop, session history, message bus, and send API against local
HTTP/TLS fixtures with a deterministic model. They use
no live account or model keys. Complete the conversation checklist for a live
carrier-delivery check.
