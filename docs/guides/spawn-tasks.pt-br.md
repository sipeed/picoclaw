# 🔄 Tarefas Assíncronas e Spawn

> Voltar ao [README](../project/README.pt-br.md)

## Tarefas Rápidas (resposta direta)

- Informar a hora atual

## Tarefas Longas (usar spawn para assíncrono)

- Pesquisar na web notícias sobre IA e resumir
- Verificar e-mail e relatar mensagens importantes
```

**Comportamentos principais:**

| Feature                 | Description                                               |
| ----------------------- | --------------------------------------------------------- |
| **spawn**               | Creates async subagent, doesn't block heartbeat           |
| **Independent context** | Subagent has its own context, no session history          |
| **message tool**        | Subagent communicates with user directly via message tool |
| **Non-blocking**        | After spawning, heartbeat continues to next task          |

#### Como Funciona a Comunicação do Subagente

```
Heartbeat é acionado
    ↓
Agente lê HEARTBEAT.md
    ↓
Para tarefa longa: spawn subagente
    ↓                           ↓
Continua para próxima tarefa  Subagente trabalha independentemente
    ↓                           ↓
Todas as tarefas concluídas   Subagente usa ferramenta "message"
    ↓                           ↓
Responde HEARTBEAT_OK         Usuário recebe resultado diretamente
```

O subagente tem acesso a ferramentas (message, web_search, etc.) e pode se comunicar com o usuário independentemente sem passar pelo agente principal.

**Configuração:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Option     | Default | Description                        |
| ---------- | ------- | ---------------------------------- |
| `enabled`  | `true`  | Enable/disable heartbeat           |
| `interval` | `30`    | Check interval in minutes (min: 5) |

**Variáveis de ambiente:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` para desabilitar
* `PICOCLAW_HEARTBEAT_INTERVAL=60` para alterar o intervalo
