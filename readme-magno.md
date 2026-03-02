# Modifiche PicoClaw — sessione 27/02/2026

## Bug fix: API Anthropic

### 1. Tool use `name` vuoto (`pkg/tools/toolloop.go`)
L'API Anthropic rifiutava le richieste con errore `tooluse.name: String should have at least 1 character`.

**Causa:** quando il codice ricostruiva i messaggi per le chiamate API successive, il campo `Name` del `ToolCall` non veniva copiato — solo `Function.Name` veniva impostato.

**Fix:** aggiunto `Name: tc.Name` nella costruzione del `ToolCall` in `toolloop.go`.

### 2. Tool use `input` non valido (`pkg/tools/toolloop.go`)
Errore: `tooluse.input: Input should be a valid dictionary`.

**Causa:** il campo `Arguments` (tipo `map[string]interface{}`) non veniva copiato nel `ToolCall`. `buildClaudeParams` usava `tc.Arguments` che era `nil`.

**Fix:** aggiunto `Arguments: tc.Arguments` nella costruzione del `ToolCall` in `toolloop.go`.

---

## Ottimizzazione token / costi

### 3. Rimossa duplicazione tool nel system prompt (`pkg/agent/context.go`)
I tool venivano elencati **due volte** ad ogni richiesta:
- Come testo nel system prompt (`buildToolsSection()`)
- Come tool definitions formali nell'API (`translateToolsForClaude()`)

**Fix:** rimossa la chiamata a `buildToolsSection()` da `getIdentity()`. I tool sono già visibili al modello tramite le definizioni API.

**Risparmio:** ~300-500 token per richiesta.

### 4. Fix `contextWindow` errato (`pkg/agent/loop.go`)
`contextWindow` era impostato a `MaxTokens` (8192), che e' il limite di **output**, non la context window del modello.

Questo causava summarization premature: la soglia era `8192 * 75% = 6144` token, raggiunta quasi subito, generando chiamate API extra inutili per riassumere la conversazione.

**Fix:** aggiunta funzione `estimateContextWindow()` che ritorna la vera context window basata sul modello (es. 200K per Claude, 128K per GPT-4, 1M per Gemini).

### 5. Prompt caching Anthropic (`pkg/providers/claude_provider.go`)
Il system prompt e le tool definitions sono identici tra richieste successive, ma venivano riprocessati (e pagati) ogni volta.

**Fix:** aggiunto `CacheControl: ephemeral` sul primo blocco system (statico) e sull'ultima tool definition. Dopo la prima richiesta, vengono cachati per 5 minuti con sconto del 90% sui token di input.

### 6. Separazione system prompt statico/dinamico (`pkg/agent/context.go`)
Il system prompt era un unico blocco di testo che includeva parti dinamiche (summary sessione, info canale). Qualsiasi cambiamento invalidava tutta la cache.

**Fix:** il system prompt e' ora diviso in due blocchi `TextBlockParam`:
- **Blocco 1 (statico, cacheable):** identity, bootstrap files, skills, memory
- **Blocco 2 (dinamico):** session info, conversation summary

### 7. Ordine deterministico dei tool (`pkg/tools/registry.go`)
I tool venivano iterati da una Go `map`, che non garantisce ordine. Ad ogni richiesta i tool potevano uscire in ordine diverso, invalidando la cache dei tool definitions.

**Fix:** aggiunto `sort.Strings(names)` prima di iterare i tool in `ToProviderDefs()`.

### 8. Timestamp system prompt solo giornaliero (`pkg/agent/context.go`)
Il formato data nel system prompt includeva ore e minuti (`2006-01-02 15:04`), cambiando ogni minuto e invalidando la cache.

**Fix:** formato cambiato a solo giorno: `2006-01-02 (Monday)`.

### 9. Bootstrap files ridotti (`~/.picoclaw/workspace/`)
- `USER.md`: da 365 bytes di placeholder generici a 73 bytes con info reali
- `IDENTITY.md`: da 1273 bytes di testo ridondante a 138 bytes essenziali

---

## Logging token usage (`pkg/agent/loop.go`, `pkg/providers/`)

Aggiunto logging del consumo token per ogni chiamata API:
- `input_tokens` — token di input
- `output_tokens` — token generati
- `cache_created` — token cachati per la prima volta
- `cache_read` — token letti dalla cache (90% sconto)

Aggiunto campo `CacheCreatedTokens` e `CacheReadTokens` in `UsageInfo` (`pkg/providers/types.go`) ed estratti dalla response Anthropic in `parseClaudeResponse()`.

---

## Supporto allegati file Telegram

### Modifiche (parzialmente implementate dall'utente)
- `pkg/tools/message.go` — aggiunto parametro `file_path` al tool `message` e aggiornata firma `SendCallback`
- `pkg/bus/types.go` — aggiunto campo `FilePath` a `OutboundMessage`
- `pkg/agent/loop.go` — callback aggiornata per passare `filePath`
- `pkg/channels/telegram.go` — `Send()` gia' implementato con `bot.SendDocument()` quando `FilePath` e' presente

**Flusso:** `message(content="Ecco il file", file_path="/path/to/file.md")` → tool estrae file_path → callback pubblica su bus → Telegram `Send()` invia come documento con caption.

---

## Configurazione

### Heartbeat disabilitato (`~/.picoclaw/config.json`)
L'heartbeat (ogni 30 min) consumava token dal budget Claude Code senza necessita'.

```json
"heartbeat": { "enabled": false }
```

---

## Riepilogo impatto

| Metrica | Prima | Dopo |
|---------|-------|------|
| Token input per richiesta semplice | ~7400 (tutti pagati pieni) | ~550 pagati + ~4200 da cache (90% sconto) |
| Chiamate API heartbeat/giorno | ~48 | 0 |
| System prompt size | ~5000 chars | ~3500 chars |
| Tool definitions cacheable | No (ordine random) | Si (ordine fisso) |

## File modificati

```
pkg/agent/context.go          — system prompt statico/dinamico, rimossi tool duplicati, fix timestamp
pkg/agent/loop.go             — fix contextWindow, logging token usage, estimateContextWindow()
pkg/providers/claude_provider.go — prompt caching (cache_control ephemeral)
pkg/providers/types.go         — campi CacheCreatedTokens, CacheReadTokens in UsageInfo
pkg/tools/registry.go          — ordine deterministico tool (sort)
pkg/tools/toolloop.go          — fix Name e Arguments mancanti nel ToolCall
pkg/tools/message.go           — parametro file_path, nuova firma SendCallback
pkg/bus/types.go               — campo FilePath in OutboundMessage
~/.picoclaw/config.json        — heartbeat disabilitato
~/.picoclaw/workspace/USER.md      — ridotto
~/.picoclaw/workspace/IDENTITY.md  — ridotto
```
