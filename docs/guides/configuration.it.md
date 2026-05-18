# ⚙️ Guida alla Configurazione

> Torna al [README](../../README.md)

## ⚙️ Configurazione

File di configurazione: `~/.picoclaw/config.json`

### Variabili d'Ambiente

Puoi sovrascrivere i percorsi predefiniti usando variabili d'ambiente. Questo è utile per installazioni portatili, distribuzioni containerizzate, o per eseguire picoclaw come servizio di sistema. Queste variabili sono indipendenti e controllano percorsi diversi.

| Variabile         | Descrizione                                                                                                                             | Percorso Predefinito      |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICOCLAW_CONFIG` | Sovrascrive il percorso al file di configurazione. Indica direttamente a picoclaw quale `config.json` caricare, ignorando tutte le altre posizioni. | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | Sovrascrive la directory radice per i dati di picoclaw. Modifica la posizione predefinita del `workspace` e delle altre directory dati.  | `~/.picoclaw`             |

**Esempi:**

```bash
# Esegui picoclaw usando un file di configurazione specifico
# Il percorso del workspace verrà letto da quel file di configurazione
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# Esegui picoclaw con tutti i dati salvati in /opt/picoclaw
# La configurazione verrà caricata dal percorso predefinito ~/.picoclaw/config.json
# Il workspace verrà creato in /opt/picoclaw/workspace
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# Usa entrambi per un setup completamente personalizzato
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Struttura del Workspace

PicoClaw salva i dati nel workspace configurato (predefinito: `~/.picoclaw/workspace`):

```
~/.picoclaw/workspace/
├── sessions/          # Sessioni di conversazione e cronologia
├── memory/           # Memoria a lungo termine (MEMORY.md)
├── state/            # Stato persistente (ultimo canale, ecc.)
├── cron/             # Database dei job pianificati
├── skills/           # Skill personalizzate
├── AGENT.md          # Guida al comportamento dell'agent
├── HEARTBEAT.md      # Prompt per task periodici (controllato ogni 30 min)
├── SOUL.md           # Anima dell'agent
└── USER.md           # Preferenze dell'utente
```

> **Nota:** Le modifiche a `AGENT.md`, `SOUL.md`, `USER.md` e `memory/MEMORY.md` vengono rilevate automaticamente a runtime tramite il tracciamento della data di modifica (mtime). **Non è necessario riavviare il gateway** dopo aver modificato questi file — l'agent caricherà il nuovo contenuto alla prossima richiesta.

### Sorgenti delle Skill

Per impostazione predefinita, le skill vengono caricate da:

1. `~/.picoclaw/workspace/skills` (workspace)
2. `~/.picoclaw/skills` (globale)
3. `<current-working-directory>/skills` (builtin)

Per configurazioni avanzate/di test, puoi sovrascrivere la directory radice delle skill builtin con:

```bash
export PICOCLAW_BUILTIN_SKILLS=/path/to/skills
```

### Politica Unificata di Esecuzione dei Comandi

- I comandi slash generici vengono eseguiti tramite un unico percorso in `pkg/agent/loop.go` via `commands.Executor`.
- Gli adattatori dei canali non consumano più localmente i comandi generici; inoltrano il testo in entrata al percorso bus/agent. Telegram registra ancora automaticamente i comandi supportati all'avvio.
- Un comando slash sconosciuto (ad esempio `/foo`) viene passato all'elaborazione LLM come se fosse un messaggio dell'utente.
- Un comando registrato ma non supportato sul canale corrente (ad esempio `/show` su WhatsApp) restituisce un errore esplicito all'utente e interrompe l'elaborazione.

### Allowlist dei Tool per Agent

La dichiarazione dei tool per-agent vive nel frontmatter di `AGENT.md`, non in `config.json`.

Se `tools` è omesso nel frontmatter, l'agent riceve il normale set globale dei tool abilitati. Se `tools` è presente, PicoClaw registra per quell'agent solo i tool runtime elencati.

```md
---
name: Research Agent
description: Specialista per ricerca web e analisi approfondita.
tools: [read_file, write_file, web_search, web_fetch, message]
skills: [deep-research]
mcpServers: [web-index]
---

Sei l'agent di ricerca.
```

Note:

- È una allowlist reale, non un suggerimento per l'LLM.
- I nomi dei tool fanno match 1:1 con il nome runtime del tool.
- Se ti serve controllo preciso, usa i nomi runtime effettivi come `web_search`, `web_fetch`, `spawn`, `subagent`, `send_file`.
- Le dichiarazioni dei tool in `AGENT.md` sono usate dal runtime e dai tool, ma non vengono iniettate nel prompt di discovery.

### Discovery Multi-Agent (Automatica)

Quando un agent ha peer spawnabili, PicoClaw inietta automaticamente nel suo system prompt un registry strutturato dei peer. Non serve una chiamata aggiuntiva a un tool `list_agents`.

Questa discovery serve soprattutto a rendere affidabile la delega tramite `spawn` con `agent_id` esplicito.

Ogni entry include:

| Campo | Significato |
|-------|-------------|
| `id` | ID stabile dell'agent |
| `name` | Nome identitario da `AGENT.md` frontmatter |
| `description` | Descrizione identitaria da `AGENT.md` frontmatter |

Dettagli importanti:

- La sezione include solo i peer che l'agent corrente può spawnare tramite `subagents.allow_agents`.
- L'agent corrente e i peer non spawnabili vengono omessi, così il modello non pianifica contro agent non disponibili.
- La discovery è volutamente leggera. Fornisce al modello solo l'identità necessaria per scegliere un peer: `id`, `name`, `description`.
- `config.json` resta il layer infrastrutturale: workspace, agent di default, routing e permessi di subagent. Questi permessi controllano anche la visibilità nella discovery.
- `AGENT.md` resta il layer di identità. Il codice runtime e i tool possono comunque usare `tools`, `skills`, `mcpServers` e `model` quando avviene la delega.

Forma dell'oggetto iniettato:

```json
{
  "agents": [
    {
      "id": "research",
      "name": "Research Agent",
      "description": "Specialista per investigazioni e lavoro web."
    }
  ]
}
```

In pratica, un agent generalista sceglie un peer in base alla descrizione del suo ruolo, poi chiama `spawn` con l'`agent_id` del peer. Il runtime risolve il resto.

### 🔒 Sandbox di Sicurezza

PicoClaw esegue in un ambiente sandboxed per impostazione predefinita. L'agent può accedere solo ai file ed eseguire comandi all'interno del workspace configurato.

#### Configurazione Predefinita

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true
    }
  }
}
```

| Opzione                 | Predefinito             | Descrizione                                          |
| ----------------------- | ----------------------- | ---------------------------------------------------- |
| `workspace`             | `~/.picoclaw/workspace` | Directory di lavoro dell'agent                       |
| `restrict_to_workspace` | `true`                  | Limita l'accesso a file/comandi al workspace         |

#### Strumenti Protetti

Quando `restrict_to_workspace: true`, i seguenti strumenti sono in sandbox:

| Strumento     | Funzione                  | Restrizione                                          |
| ------------- | ------------------------- | ---------------------------------------------------- |
| `read_file`   | Legge file                | Solo file all'interno del workspace                  |
| `write_file`  | Scrive file               | Solo file all'interno del workspace                  |
| `list_dir`    | Elenca directory          | Solo directory all'interno del workspace             |
| `edit_file`   | Modifica file             | Solo file all'interno del workspace                  |
| `append_file` | Aggiunge ai file          | Solo file all'interno del workspace                  |
| `exec`        | Esegue comandi            | I percorsi dei comandi devono essere nel workspace   |

#### Protezione Exec Aggiuntiva

Anche con `restrict_to_workspace: false`, lo strumento `exec` blocca questi comandi pericolosi:

* `rm -rf`, `del /f`, `rmdir /s` — Cancellazione di massa
* `format`, `mkfs`, `diskpart` — Formattazione del disco
* `dd if=` — Imaging del disco
* Scrittura su `/dev/sd[a-z]` — Scritture dirette su disco
* `shutdown`, `reboot`, `poweroff` — Spegnimento del sistema
* Fork bomb `:(){ :|:& };:`

### Controllo Accesso ai File

| Chiave di configurazione | Tipo | Predefinito | Descrizione |
|--------------------------|------|-------------|-------------|
| `tools.allow_read_paths` | string[] | `[]` | Percorsi aggiuntivi consentiti per la lettura al di fuori del workspace |
| `tools.allow_write_paths` | string[] | `[]` | Percorsi aggiuntivi consentiti per la scrittura al di fuori del workspace |

### Sicurezza Exec

| Chiave di configurazione | Tipo | Predefinito | Descrizione |
|--------------------------|------|-------------|-------------|
| `tools.exec.allow_remote` | bool | `false` | Consente lo strumento exec da canali remoti (Telegram/Discord ecc.) |
| `tools.exec.enable_deny_patterns` | bool | `true` | Abilita l'intercettazione dei comandi pericolosi |
| `tools.exec.custom_deny_patterns` | string[] | `[]` | Pattern regex personalizzati da bloccare |
| `tools.exec.custom_allow_patterns` | string[] | `[]` | Pattern regex personalizzati da consentire |

> **Nota di sicurezza:** La protezione dei symlink è abilitata per impostazione predefinita — tutti i percorsi file vengono risolti tramite `filepath.EvalSymlinks` prima del confronto con la whitelist, prevenendo attacchi di escape tramite symlink.

#### Limitazione Nota: Processi Figlio degli Strumenti di Build

Il controllo di sicurezza exec ispeziona solo la riga di comando avviata direttamente da PicoClaw. Non ispeziona ricorsivamente i processi figlio generati da strumenti di sviluppo consentiti come `make`, `go run`, `cargo`, `npm run` o script di build personalizzati.

Ciò significa che un comando di primo livello può comunque compilare o avviare altri binari dopo aver superato il controllo iniziale. In pratica, tratta gli script di build, i Makefile, gli script di pacchetti e i binari generati come codice eseguibile che richiede lo stesso livello di revisione di un comando shell diretto.

Per ambienti ad alto rischio:

* Esamina gli script di build prima dell'esecuzione.
* Preferisci l'approvazione/revisione manuale per i workflow di compilazione ed esecuzione.
* Esegui PicoClaw in un container o VM se hai bisogno di un isolamento più forte di quello fornito dal controllo integrato.

#### Esempi di Errore

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Disabilitare le Restrizioni (Rischio di Sicurezza)

Se hai bisogno che l'agent acceda a percorsi al di fuori del workspace:

**Metodo 1: File di configurazione**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Metodo 2: Variabile d'ambiente**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **Attenzione**: Disabilitare questa restrizione consente all'agent di accedere a qualsiasi percorso sul tuo sistema. Usare con cautela solo in ambienti controllati.

#### Coerenza dei Confini di Sicurezza

L'impostazione `restrict_to_workspace` si applica in modo coerente a tutti i percorsi di esecuzione:

| Percorso di esecuzione | Confine di sicurezza              |
| ---------------------- | --------------------------------- |
| Main Agent             | `restrict_to_workspace` ✅        |
| Subagent / Spawn       | Eredita la stessa restrizione ✅  |
| Heartbeat tasks        | Eredita la stessa restrizione ✅  |

Tutti i percorsi condividono la stessa restrizione del workspace — non è possibile aggirare il confine di sicurezza tramite subagent o task pianificati.

### Heartbeat (Task Periodici)

PicoClaw può eseguire task periodici automaticamente. Crea un file `HEARTBEAT.md` nel tuo workspace:

```markdown
# Periodic Tasks

- Check my email for important messages
- Review my calendar for upcoming events
- Check the weather forecast
```

L'agent leggerà questo file ogni 30 minuti (configurabile) ed eseguirà tutti i task usando gli strumenti disponibili.

#### Task Asincroni con Spawn

Per task di lunga durata (ricerca web, chiamate API), usa lo strumento `spawn` per creare un **subagent**:

```markdown
# Periodic Tasks
```
