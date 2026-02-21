# PicoClaw - Notes de projet

## Contexte

PicoClaw est un assistant IA ultra-léger en Go (<10MB RAM) du repo `sipeed/picoclaw`.
Fork de travail : `edouard-claude/picoclaw`.
Pseudo GitHub : `edouard-claude`
Pseudo Discord : `Bedoskil`

## PRs

- **#408** — `docs/add-french-readme` : README français (README.fr.md) — **OPEN**
- **#409** — `feat/multi-agent-framework` : Fondations multi-agent — **CLOSED** en faveur de #423
- **#469** — `feat/aieos-profile` : AIEOS v1.1 profile loading (#296) — **OPEN**
  - Branche : `feat/aieos-profile` sur `fork`
  - Commentaire posté sur issue #296
- **Leeaandrob/picoclaw#1** — Routage par capabilities sur fork Leandrob — **OPEN**, en attente review

## Collaboration multi-agent (issue #294)

- **PR de référence** : **#423** par @Leandrob (`pkg/multiagent/`) — c'est la PR sur laquelle contribuer
  - Blackboard, HandoffTool, ListAgentsTool, BlackboardTool (LLM-callable)
  - AgentResolver interface (2 méthodes, idiomatique Go)
  - Intégré dans AgentLoop, basé sur #213 + #131 (déjà mergés)
- **@yk** travaille sur le Swarm Mode (lié)
- Le travail officiel du groupe commence le **24 février 2026**
- Contributions possibles : event log/audit, agent state tracking (lifecycle management)

## Discord — PicoClaw Dev Group

- Serveur : `PicoClaw Dev Group` (ID: `1471858454992781563`)
- Canal principal : `#general-dev` (ID: `1471858456788205723`)
- Canal sécurité : `#security`
- Membres clés :
  - **zepan** — mainteneur principal du projet
  - **Leandrob** (gh: Leeaandrob) — collaborateur sur #294 multi-agent
  - **yk** — travaille sur le Swarm Mode
  - **Harsh** (gh: harshbansal7) — contributeur actif, fait des hotfix/merge
  - **Huaaudio** — contributeur, peut merger des PRs
  - **mymmrac** — contribue sur le linting/formatting
  - **alexhoshina** — OneBot channel (#192)

### Workflow Discord (IMPORTANT)

A chaque début de session, si le travail concerne PicoClaw :

1. **Lire les messages Discord** via Chrome Claude (`claude-in-chrome`) :
   - Naviguer vers `https://discord.com/channels/1471858454992781563/1471858456788205723`
   - Prendre un screenshot pour lire les messages récents
   - Scroller vers le haut si nécessaire pour voir plus de contexte

2. **Traduire en français** tous les messages pertinents pour l'utilisateur

3. **Proposer des réponses** en deux versions :
   - **Anglais** : le message à copier-coller sur Discord
   - **Français** : la traduction pour que l'utilisateur comprenne ce qu'il envoie

4. L'utilisateur ne parle pas très bien anglais — toujours fournir les traductions françaises

## Architecture du code

- `pkg/agent/loop.go` — AgentLoop (boucle principale)
- `pkg/agent/context.go` — ContextBuilder : system prompt (AIEOS-first, fallback .md)
- `pkg/agent/instance.go` — AgentInstance : crée agent depuis config
- `pkg/aieos/` — Package AIEOS v1.1 (notre contribution, PR #469)
  - `profile.go` — Structs (Profile, Identity, Psychology OCEAN, Linguistics, etc.)
  - `loader.go` — LoadProfile, ProfileExists, validation
  - `render.go` — RenderToPrompt (OCEAN → langage naturel, linguistics → ton)
- `pkg/multiagent/` — Framework multi-agent par Leandrob (PR #423)
  - `blackboard.go` — Blackboard shared context
  - `handoff.go` — AgentResolver interface + ExecuteHandoff()
  - `handoff_tool.go` — Tool LLM : délégation de tâche
  - `list_agents_tool.go` — Tool LLM : discovery des agents
- `pkg/tools/subagent.go` — SubagentManager existant
- `pkg/tools/base.go` — Interface Tool, AsyncTool, ContextualTool

## CI/CD

- Workflow PR : `.github/workflows/pr.yml` — 4 checks obligatoires :
  1. **Formatting** : `make fmt` + `git diff --exit-code` (gofmt strict)
  2. **Linter** : `golangci-lint v2.10.1` avec `.golangci.yaml` (config v2)
     - Formatter activé : `goimports` (vérifie imports ET formatage gofmt)
     - Beaucoup de linters désactivés pour l'instant (TODO progressif)
  3. **Vet** : `go vet ./...` (dépend de Formatting)
  4. **Tests** : `go test ./...` (dépend de Formatting)
- Les PRs de forks first-time nécessitent `action_required` (approbation mainteneur)

### Checklist OBLIGATOIRE avant push

```bash
gofmt -w ./pkg/...          # Formater TOUS les fichiers modifiés
gofmt -l ./pkg/...          # Vérifier qu'il n'y a plus rien à formater (doit être vide)
go vet ./...                # Pas d'erreurs vet
go test ./...               # Tous les tests passent
```

### Pièges connus (formatting)

- **NE PAS aligner les commentaires manuellement** avec des espaces supplémentaires.
  `gofmt` normalise l'alignement et la CI rejettera tout écart.
  Exemple : `// comment` doit suivre immédiatement la fin du code, pas être aligné en colonne.
- Toujours exécuter `gofmt -w` sur les fichiers modifiés avant de commiter.

## Conventions du projet

- PR template : `.github/pull_request_template.md`
- Style de commit : `type: description` (feat, fix, docs, style)
- Suivre le pattern des PRs existantes (ex: #385 pour les docs)
- Le code existant utilise `interface{}` (pas `any`) — rester cohérent
