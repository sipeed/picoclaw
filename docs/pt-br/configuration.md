# ⚙️ Guia de Configuração

> Voltar ao [README](../../README.pt-br.md)

## ⚙️ Configuração

Arquivo de configuração: `~/.picoclaw/config.json`

### Variáveis de Ambiente

Você pode substituir os caminhos padrão usando variáveis de ambiente. Isso é útil para instalações portáteis, implantações em contêineres ou execução do picoclaw como serviço do sistema. Essas variáveis são independentes e controlam caminhos diferentes.

| Variável          | Descrição                                                                                                                             | Caminho Padrão              |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICOCLAW_CONFIG` | Substitui o caminho para o arquivo de configuração. Isso indica diretamente ao picoclaw qual `config.json` carregar, ignorando todos os outros locais. | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | Substitui o diretório raiz para dados do picoclaw. Isso altera o local padrão do `workspace` e outros diretórios de dados.          | `~/.picoclaw`             |

**Exemplos:**

```bash
# Executar picoclaw usando um arquivo de configuração específico
# O caminho do workspace será lido de dentro desse arquivo de configuração
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# Executar picoclaw com todos os dados armazenados em /opt/picoclaw
# A configuração será carregada do padrão ~/.picoclaw/config.json
# O workspace será criado em /opt/picoclaw/workspace
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# Usar ambos para uma configuração totalmente personalizada
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Layout do Workspace

O PicoClaw armazena dados no seu workspace configurado (padrão: `~/.picoclaw/workspace`):

```
~/.picoclaw/workspace/
├── sessions/          # Sessões de conversa e histórico
├── memory/           # Memória de longo prazo (MEMORY.md)
├── state/            # Estado persistente (último canal, etc.)
├── cron/             # Banco de dados de tarefas agendadas
├── skills/           # Skills personalizadas
├── AGENT.md          # Guia de comportamento do agente
├── HEARTBEAT.md      # Prompts de tarefas periódicas (verificados a cada 30 min)
├── IDENTITY.md       # Identidade do agente
├── SOUL.md           # Alma do agente
└── USER.md           # Preferências do usuário
```

> **Nota:** Alterações em `AGENT.md`, `SOUL.md`, `USER.md` e `memory/MEMORY.md` são detectadas automaticamente em tempo de execução via rastreamento de data de modificação (mtime). **Não é necessário reiniciar o gateway** após editar esses arquivos — o agente carrega o novo conteúdo na próxima requisição.

### Fontes de Skills

Por padrão, as skills são carregadas de:

1. `~/.picoclaw/workspace/skills` (workspace)
2. `~/.picoclaw/skills` (global)
3. `<current-working-directory>/skills` (builtin)

Para configurações avançadas/de teste, você pode substituir o diretório raiz de skills builtin com:

```bash
export PICOCLAW_BUILTIN_SKILLS=/path/to/skills
```

### Política Unificada de Execução de Comandos

- Comandos slash genéricos são executados através de um único caminho em `pkg/agent/loop.go` via `commands.Executor`.
- Os adaptadores de canal não consomem mais comandos genéricos localmente; eles encaminham o texto de entrada para o caminho bus/agent. O Telegram ainda registra automaticamente os comandos suportados na inicialização.
- Comando slash desconhecido (por exemplo `/foo`) passa para o processamento normal do LLM.
- Comando registrado mas não suportado no canal atual (por exemplo `/show` no WhatsApp) retorna um erro explícito ao usuário e interrompe o processamento.

### 🔒 Sandbox de Segurança

O PicoClaw é executado em um ambiente sandbox por padrão. O agente só pode acessar arquivos e executar comandos dentro do workspace configurado.

#### Configuração Padrão

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

| Opção                   | Padrão                  | Descrição                                 |
| ----------------------- | ----------------------- | ----------------------------------------- |
| `workspace`             | `~/.picoclaw/workspace` | Diretório de trabalho do agente           |
| `restrict_to_workspace` | `true`                  | Restringir acesso a arquivos/comandos ao workspace |

#### Ferramentas Protegidas

Quando `restrict_to_workspace: true`, as seguintes ferramentas são isoladas:

| Ferramenta    | Função           | Restrição                              |
| ------------- | ---------------- | -------------------------------------- |
| `read_file`   | Ler arquivos     | Apenas arquivos dentro do workspace    |
| `write_file`  | Escrever arquivos| Apenas arquivos dentro do workspace    |
| `list_dir`    | Listar diretórios| Apenas diretórios dentro do workspace  |
| `edit_file`   | Editar arquivos  | Apenas arquivos dentro do workspace    |
| `append_file` | Anexar a arquivos| Apenas arquivos dentro do workspace    |
| `exec`        | Executar comandos| Caminhos de comando devem estar dentro do workspace |

#### Proteção Adicional do Exec

Mesmo com `restrict_to_workspace: false`, a ferramenta `exec` bloqueia estes comandos perigosos:

* `rm -rf`, `del /f`, `rmdir /s` — Exclusão em massa
* `format`, `mkfs`, `diskpart` — Formatação de disco
* `dd if=` — Imagem de disco
* Escrita em `/dev/sd[a-z]` — Escritas diretas em disco
* `shutdown`, `reboot`, `poweroff` — Desligamento do sistema
* Fork bomb `:(){ :|:& };:`

### Controle de Acesso a Arquivos

| Config Key | Type | Default | Description |
|------------|------|---------|-------------|
| `tools.allow_read_paths` | string[] | `[]` | Additional paths allowed for reading outside workspace |
| `tools.allow_write_paths` | string[] | `[]` | Additional paths allowed for writing outside workspace |

### Segurança do Exec

| Config Key | Type | Default | Description |
|------------|------|---------|-------------|
| `tools.exec.allow_remote` | bool | `false` | Allow exec tool from remote channels (Telegram/Discord etc.) |
| `tools.exec.enable_deny_patterns` | bool | `true` | Enable dangerous command interception |
| `tools.exec.custom_deny_patterns` | string[] | `[]` | Custom regex patterns to block |
| `tools.exec.custom_allow_patterns` | string[] | `[]` | Custom regex patterns to allow |

> **Nota de Segurança:** A proteção contra symlinks é habilitada por padrão — todos os caminhos de arquivo são resolvidos através de `filepath.EvalSymlinks` antes da correspondência com a whitelist, prevenindo ataques de escape via symlink.

#### Limitação Conhecida: Processos Filhos de Ferramentas de Build

O guard de segurança do exec inspeciona apenas a linha de comando que o PicoClaw executa diretamente. Ele não inspeciona recursivamente processos filhos gerados por ferramentas de desenvolvimento permitidas como `make`, `go run`, `cargo`, `npm run` ou scripts de build personalizados.

Isso significa que um comando de nível superior ainda pode compilar ou executar outros binários após passar pela verificação inicial do guard. Na prática, trate scripts de build, Makefiles, scripts de pacotes e binários gerados como código executável que precisa do mesmo nível de revisão que um comando shell direto.

Para ambientes de maior risco:

* Revise scripts de build antes da execução.
* Prefira aprovação/revisão manual para fluxos de trabalho de compilação e execução.
* Execute o PicoClaw dentro de um contêiner ou VM se precisar de isolamento mais forte do que o guard integrado oferece.

#### Exemplos de Erro

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### Desabilitando Restrições (Risco de Segurança)

Se você precisar que o agente acesse caminhos fora do workspace:

**Método 1: Arquivo de configuração**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Método 2: Variável de ambiente**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **Aviso**: Desabilitar esta restrição permite que o agente acesse qualquer caminho no seu sistema. Use com cautela apenas em ambientes controlados.

#### Consistência do Limite de Segurança

A configuração `restrict_to_workspace` se aplica consistentemente em todos os caminhos de execução:

| Caminho de Execução | Limite de Segurança          |
| -------------------- | ---------------------------- |
| Main Agent           | `restrict_to_workspace` ✅   |
| Subagent / Spawn     | Herda a mesma restrição ✅   |
| Heartbeat tasks      | Herda a mesma restrição ✅   |

Todos os caminhos compartilham a mesma restrição de workspace — não há como contornar o limite de segurança através de subagentes ou tarefas agendadas.

### Heartbeat (Tarefas Periódicas)

O PicoClaw pode executar tarefas periódicas automaticamente. Crie um arquivo `HEARTBEAT.md` no seu workspace:

```markdown
# Tarefas Periódicas

- Verificar meu e-mail para mensagens importantes
- Revisar meu calendário para eventos próximos
- Verificar a previsão do tempo
```

O agente lerá este arquivo a cada 30 minutos (configurável) e executará quaisquer tarefas usando as ferramentas disponíveis.

#### Tarefas Assíncronas com Spawn

Para tarefas de longa duração (busca na web, chamadas de API), use a ferramenta `spawn` para criar um **subagente**:

```markdown
# Tarefas Periódicas
```
