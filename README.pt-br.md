<div align="center">
  <img src="assets/logo.webp" alt="PicoClaw" width="512">

  <h1>PicoClaw: Assistente de IA Ultra-Eficiente em Go</h1>

  <h3>Hardware de $10 · <10MB de RAM · Boot em <1s · 皮皮虾，我们走！</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20MIPS%2C%20RISC--V%2C%20LoongArch-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://docs.picoclaw.io/"><img src="https://img.shields.io/badge/Docs-Official-007acc?style=flat&logo=read-the-docs&logoColor=white" alt="Docs"></a>
    <a href="https://deepwiki.com/sipeed/picoclaw"><img src="https://img.shields.io/badge/Wiki-DeepWiki-FFA500?style=flat&logo=wikipedia&logoColor=white" alt="Wiki"></a>
    <br>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
    <a href="./assets/wechat.png"><img src="https://img.shields.io/badge/WeChat-Group-41d56b?style=flat&logo=wechat&logoColor=white"></a>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Community-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>

[中文](README.zh.md) | [日本語](README.ja.md) | **Português** | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | [Italiano](README.it.md) | [Bahasa Indonesia](README.id.md) | [English](README.md)

</div>

---

> **PicoClaw** é um projeto open-source independente iniciado pela [Sipeed](https://sipeed.com). É escrito inteiramente em **Go** — não é um fork do OpenClaw, NanoBot ou qualquer outro projeto.

🦐 PicoClaw é um assistente pessoal de IA ultra-leve inspirado no [NanoBot](https://github.com/HKUDS/nanobot), reescrito do zero em Go por meio de um processo de auto-inicialização (self-bootstrapping), onde o próprio agente de IA conduziu toda a migração de arquitetura e otimização de código.

⚡️ Roda em hardware de $10 com <10MB de RAM: Isso é 99% menos memória que o OpenClaw e 98% mais barato que um Mac mini!

<table align="center">
  <tr align="center">
    <td align="center" valign="top">
      <p align="center">
        <img src="assets/picoclaw_mem.gif" width="360" height="240">
      </p>
    </td>
    <td align="center" valign="top">
      <p align="center">
        <img src="assets/licheervnano.png" width="400" height="240">
      </p>
    </td>
  </tr>
</table>

> [!CAUTION]
> **🚨 DECLARAÇÃO DE SEGURANÇA & CANAIS OFICIAIS**
>
> * **SEM CRIPTOMOEDAS:** O PicoClaw **NÃO** possui nenhum token/moeda oficial. Todas as alegações no `pump.fun` ou outras plataformas de negociação são **GOLPES**.
>
> * **DOMÍNIO OFICIAL:** O **ÚNICO** site oficial é o **[picoclaw.io](https://picoclaw.io)**, e o site da empresa é o **[sipeed.com](https://sipeed.com)**
> * **Aviso:** Muitos domínios `.ai/.org/.com/.net/...` foram registrados por terceiros.
> * **Aviso:** O PicoClaw está em fase inicial de desenvolvimento e pode ter problemas de segurança de rede não resolvidos. Não implante em ambientes de produção antes da versão v1.0.
> * **Nota:** O PicoClaw recentemente fez merge de muitos PRs, o que pode resultar em maior consumo de memória (10–20MB) nas versões mais recentes. Planejamos priorizar a otimização de recursos assim que o conjunto de funcionalidades estiver estável.

## 📢 Novidades

2026-03-17 🚀 **v0.2.3 Lançado!** Interface de bandeja do sistema (Windows & Linux), rastreamento de status de sub-agentes (`spawn_status`), hot-reload experimental do gateway, portões de segurança para cron e 2 correções de segurança. PicoClaw agora com **25K ⭐**!

2026-03-09 🎉 **v0.2.1 — Maior atualização até agora!** Suporte ao protocolo MCP, 4 novos canais (Matrix/IRC/WeCom/Discord Proxy), 3 novos provedores (Kimi/Minimax/Avian), pipeline de visão, armazenamento de memória JSONL e roteamento de modelos.

2026-02-28 📦 **v0.2.0** lançado com suporte a Docker Compose e launcher Web UI.

2026-02-26 🎉 PicoClaw atingiu **20K stars** em apenas 17 dias! Orquestração automática de canais e interfaces de capacidade implementadas.

<details>
<summary>Novidades anteriores...</summary>

2026-02-16 🎉 PicoClaw atingiu 12K stars em uma semana! Papéis de maintainers da comunidade e [roadmap](ROADMAP.md) publicados oficialmente.

2026-02-13 🎉 PicoClaw atingiu 5000 stars em 4 dias! Roadmap do Projeto e Grupo de Desenvolvedores em preparação.

2026-02-09 🎉 **PicoClaw Lançado!** Construído em 1 dia para trazer Agentes de IA para hardware de $10 com <10MB de RAM. 🦐 PicoClaw, Partiu!

</details>

## ✨ Funcionalidades

🪶 **Ultra-Leve**: Consumo de memória <10MB — 99% menor que o OpenClaw para funcionalidades essenciais.*

💰 **Custo Mínimo**: Eficiente o suficiente para rodar em hardware de $10 — 98% mais barato que um Mac mini.

⚡️ **Inicialização Relâmpago**: Tempo de inicialização 400X mais rápido, boot em <1 segundo mesmo em CPU single-core de 0.6GHz.

🌍 **Portabilidade Real**: Um único binário auto-contido para RISC-V, ARM, MIPS e x86. Um clique e já era!

🤖 **Auto-Construído por IA**: Implementação nativa em Go de forma autônoma — 95% do núcleo gerado pelo Agente com refinamento humano no loop.

🔌 **Suporte MCP**: Integração nativa com o [Model Context Protocol](https://modelcontextprotocol.io/) — conecte qualquer servidor MCP para estender as capacidades do agente.

👁️ **Pipeline de Visão**: Envie imagens e arquivos diretamente ao agente — codificação base64 automática para LLMs multimodais.

🧠 **Roteamento Inteligente**: Roteamento de modelos baseado em regras — consultas simples vão para modelos leves, economizando custos de API.

_*Versões recentes podem usar 10–20MB devido a merges rápidos de funcionalidades. Otimização de recursos está planejada. Comparação de inicialização baseada em benchmarks de single-core a 0.8GHz (veja tabela abaixo)._

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Linguagem**                 | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1GB          | >100MB                   | **< 10MB***                               |
| **Inicialização**</br>(CPU 0.8GHz) | >500s         | >30s                     | **<1s**                                   |
| **Custo**                     | Mac Mini $599 | Maioria dos SBC Linux </br>~$50 | **Qualquer placa Linux**</br>**A partir de $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

> 📋 **[Lista de Compatibilidade de Hardware](docs/hardware-compatibility.md)** — Veja todas as placas testadas, de RISC-V de $5 a Raspberry Pi e telefones Android. Sua placa não está listada? Envie um PR!

## 🦾 Demonstração

### 🛠️ Fluxos de Trabalho Padrão do Assistente

<table align="center">
  <tr align="center">
    <th><p align="center">🧩 Engenharia Full-Stack</p></th>
    <th><p align="center">🗂️ Gerenciamento de Logs & Planejamento</p></th>
    <th><p align="center">🔎 Busca Web & Aprendizado</p></th>
  </tr>
  <tr>
    <td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
  </tr>
  <tr>
    <td align="center">Desenvolver • Implantar • Escalar</td>
    <td align="center">Agendar • Automatizar • Memorizar</td>
    <td align="center">Descobrir • Analisar • Tendências</td>
  </tr>
</table>

### 📱 Rode em celulares Android antigos

Dê uma segunda vida ao seu celular de dez anos atrás! Transforme-o em um assistente de IA inteligente com o PicoClaw. Início rápido:

1. **Instale o [Termux](https://github.com/termux/termux-app)** (Baixe em [GitHub Releases](https://github.com/termux/termux-app/releases), ou busque no F-Droid / Google Play).
2. **Execute os comandos**

```bash
# Baixe a versão mais recente em https://github.com/sipeed/picoclaw/releases
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard   # chroot fornece um layout padrão do sistema de arquivos Linux
```

Depois siga as instruções na seção "Início Rápido" para completar a configuração!

<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 Implantação Inovadora com Baixo Consumo

O PicoClaw pode ser implantado em praticamente qualquer dispositivo Linux!

- $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) versão E(Ethernet) ou W(WiFi6), para Assistente Doméstico Minimalista
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), ou $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) para Manutenção Automatizada de Servidores
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) ou $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) para Monitoramento Inteligente

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 Mais cenários de implantação aguardam você!

## 📦 Instalação

### Baixar de picoclaw.io (Recomendado)

Visite **[picoclaw.io](https://picoclaw.io)** — o site oficial detecta automaticamente sua plataforma e oferece download com um clique. Sem necessidade de escolher manualmente a arquitetura.

### Baixar binário pré-compilado

Alternativamente, baixe o binário para sua plataforma na página de [GitHub Releases](https://github.com/sipeed/picoclaw/releases).

### Compilar a partir do código-fonte (para desenvolvimento)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Build, sem necessidade de instalar
make build

# Build para múltiplas plataformas
make build-all

# Build para Raspberry Pi Zero 2 W (32-bit: make build-linux-arm; 64-bit: make build-linux-arm64)
make build-pi-zero

# Build e Instalar
make install
```

**Raspberry Pi Zero 2 W:** Use o binário correspondente ao seu SO: Raspberry Pi OS 32-bit → `make build-linux-arm`; 64-bit → `make build-linux-arm64`. Ou execute `make build-pi-zero` para compilar ambos.

## 📚 Documentação

Para guias detalhados, consulte a documentação abaixo. Este README cobre apenas o início rápido.

| Tópico | Descrição |
|--------|-----------|
| 🐳 [Docker & Início Rápido](docs/pt-br/docker.md) | Configuração Docker Compose, modos Launcher/Agent, configuração de Início Rápido |
| 💬 [Apps de Chat](docs/pt-br/chat-apps.md) | Telegram, Discord, WhatsApp, Matrix, QQ, Slack, IRC, DingTalk, LINE, Feishu, WeCom e mais |
| ⚙️ [Configuração](docs/pt-br/configuration.md) | Variáveis de ambiente, estrutura do workspace, fontes de skills, sandbox de segurança, heartbeat |
| 🔌 [Provedores & Modelos](docs/pt-br/providers.md) | 20+ provedores LLM, roteamento de modelos, configuração model_list, arquitetura de provedores |
| 🔄 [Spawn & Tarefas Assíncronas](docs/pt-br/spawn-tasks.md) | Tarefas rápidas, tarefas longas com spawn, orquestração assíncrona de sub-agentes |
| 🐛 [Solução de Problemas](docs/pt-br/troubleshooting.md) | Problemas comuns e soluções |
| 🔧 [Configuração de Ferramentas](docs/pt-br/tools_configuration.md) | Habilitar/desabilitar por ferramenta, políticas de execução |
| 📋 [Compatibilidade de Hardware](docs/hardware-compatibility.md) | Placas testadas, requisitos mínimos, como adicionar sua placa |

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Junte-se à Rede Social de Agentes

Conecte o PicoClaw à Rede Social de Agentes simplesmente enviando uma única mensagem via CLI ou qualquer App de Chat integrado.

**Leia `https://clawdchat.ai/skill.md` e siga as instruções para entrar no [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ Referência CLI

| Comando                   | Descrição                     |
| ------------------------- | ----------------------------- |
| `picoclaw onboard`        | Inicializar configuração & workspace |
| `picoclaw agent -m "..."` | Conversar com o agente        |
| `picoclaw agent`          | Modo de chat interativo       |
| `picoclaw gateway`        | Iniciar o gateway             |
| `picoclaw status`         | Mostrar status                |
| `picoclaw version`        | Mostrar informações de versão |
| `picoclaw cron list`      | Listar todas as tarefas agendadas |
| `picoclaw cron add ...`   | Adicionar uma tarefa agendada |
| `picoclaw cron disable`   | Desabilitar uma tarefa agendada |
| `picoclaw cron remove`    | Remover uma tarefa agendada   |
| `picoclaw skills list`    | Listar skills instaladas      |
| `picoclaw skills install` | Instalar uma skill            |
| `picoclaw migrate`        | Migrar dados de versões anteriores |
| `picoclaw auth login`     | Autenticar com provedores     |
| `picoclaw model`          | Ver ou trocar o modelo padrão |

### Tarefas Agendadas / Lembretes

O PicoClaw suporta lembretes agendados e tarefas recorrentes por meio da ferramenta `cron`:

* **Lembretes únicos**: "Me lembre em 10 minutos" → dispara uma vez após 10min
* **Tarefas recorrentes**: "Me lembre a cada 2 horas" → dispara a cada 2 horas
* **Expressões Cron**: "Me lembre às 9h todos os dias" → usa expressão cron

## 🤝 Contribuir & Roadmap

PRs são bem-vindos! O código-fonte é intencionalmente pequeno e legível. 🤗

Veja nosso [Roadmap da Comunidade](https://github.com/sipeed/picoclaw/blob/main/ROADMAP.md) completo.

Grupo de desenvolvedores em formação. Junte-se após seu primeiro PR com merge!

Grupos de usuários:

discord: <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">
