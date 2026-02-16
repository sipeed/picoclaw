# PicoClaw - Deploy na Hostinger VPS

Guia completo para deploy do PicoClaw em um servidor VPS da Hostinger.

## Pre-requisitos

- VPS Hostinger com Ubuntu 22.04+ ou Debian 12+ (KVM)
- Acesso SSH configurado (chave SSH recomendado)
- API keys dos provedores que deseja usar (Anthropic, OpenAI, etc.)

> **Nota:** Hospedagem compartilhada da Hostinger **nao** suporta PicoClaw.
> Voce precisa de um plano VPS ou Cloud com acesso root.

## Metodo 1: Deploy via Docker (Recomendado)

### Passo 1: Configurar acesso SSH

```bash
# Gerar chave SSH (se ainda nao tem)
ssh-keygen -t ed25519 -C "picoclaw-hostinger"

# Copiar chave para o servidor
ssh-copy-id -i ~/.ssh/id_ed25519 root@SEU_IP_VPS
```

### Passo 2: Setup inicial do servidor

Execute uma unica vez para preparar o servidor:

```bash
# Opcao A: Via make
make deploy-hostinger-setup HOSTINGER_HOST=SEU_IP_VPS

# Opcao B: Direto via SSH
ssh root@SEU_IP_VPS 'bash -s' < deploy/hostinger/setup-server.sh
```

Isso instala: Docker, firewall (ufw), fail2ban, e cria a estrutura de diretorios.

### Passo 3: Configurar API keys no servidor

```bash
ssh root@SEU_IP_VPS

# Editar chaves de API
nano /opt/picoclaw/config/.env

# Editar configuracao do PicoClaw
nano /opt/picoclaw/config/config.json
```

Exemplo minimo do `.env`:
```bash
ANTHROPIC_API_KEY=sk-ant-sua-chave-aqui
TELEGRAM_BOT_TOKEN=123456:ABC-seu-token
TZ=America/Sao_Paulo
```

### Passo 4: Deploy

```bash
# Deploy com Docker
make deploy-hostinger HOSTINGER_HOST=SEU_IP_VPS

# Ou com variaveis de ambiente
export HOSTINGER_HOST=SEU_IP_VPS
export HOSTINGER_USER=root
make deploy-hostinger
```

### Passo 5: Verificar

```bash
# Status completo
make deploy-hostinger-status HOSTINGER_HOST=SEU_IP_VPS

# Health check rapido
curl http://SEU_IP_VPS:18790/health
```

## Metodo 2: Deploy via Binario (Menor consumo)

Para VPS com pouca memoria (512MB-1GB), o deploy via binario usa menos recursos.

### Setup do servidor

```bash
ssh root@SEU_IP_VPS 'bash -s -- binary' < deploy/hostinger/setup-server.sh
```

### Deploy

```bash
# Build local e upload (mais rapido se sua maquina e potente)
make deploy-hostinger HOSTINGER_HOST=SEU_IP_VPS HOSTINGER_DEPLOY_METHOD=binary

# Ou build no servidor (nao precisa de Go local)
# O script sincroniza o codigo e compila no VPS
```

### Gerenciar servico

```bash
ssh root@SEU_IP_VPS

# Status
systemctl status picoclaw

# Logs em tempo real
tail -f /opt/picoclaw/logs/picoclaw.log

# Reiniciar
systemctl restart picoclaw

# Parar
systemctl stop picoclaw
```

## Comandos Make

| Comando | Descricao |
|---------|-----------|
| `make deploy-hostinger-setup` | Setup inicial do servidor |
| `make deploy-hostinger` | Deploy/atualizar PicoClaw |
| `make deploy-hostinger-status` | Verificar status |
| `make deploy-hostinger-rollback` | Reverter para versao anterior |

### Variaveis de ambiente

| Variavel | Padrao | Descricao |
|----------|--------|-----------|
| `HOSTINGER_HOST` | (obrigatorio) | IP ou hostname do VPS |
| `HOSTINGER_USER` | `root` | Usuario SSH |
| `HOSTINGER_SSH_KEY` | `~/.ssh/id_rsa` | Caminho da chave SSH |
| `HOSTINGER_SSH_PORT` | `22` | Porta SSH |
| `HOSTINGER_DEPLOY_METHOD` | `docker` | `docker` ou `binary` |

## Estrutura no Servidor

```
/opt/picoclaw/
├── bin/              # Binario do PicoClaw (metodo binary)
├── config/
│   ├── .env          # Chaves de API (chmod 600)
│   └── config.json   # Configuracao do PicoClaw
├── workspace/        # Workspace persistente
│   ├── memory/       # Memoria de sessoes
│   └── skills/       # Skills customizadas
├── logs/             # Logs (metodo binary)
├── backups/          # Backups de versoes anteriores
└── src/              # Codigo fonte (para build remoto)
```

## Seguranca

- Firewall (ufw) configurado: apenas SSH (22) e Gateway (18790)
- fail2ban ativo para protecao contra brute-force SSH
- Servico roda com usuario dedicado `picoclaw` (sem root)
- Hardening via systemd (ProtectSystem, NoNewPrivileges, etc.)
- Arquivos `.env` e `config.json` com permissao 600

## Troubleshooting

### Container nao inicia (Docker)
```bash
ssh root@SEU_IP_VPS
docker compose -f /opt/picoclaw/docker-compose.yml logs picoclaw-gateway
```

### Servico nao inicia (Binary)
```bash
ssh root@SEU_IP_VPS
journalctl -u picoclaw -n 50 --no-pager
cat /opt/picoclaw/logs/picoclaw-error.log
```

### Health check falha
```bash
# Verificar se a porta esta aberta
ss -tlnp | grep 18790

# Verificar firewall
ufw status

# Testar localmente no servidor
curl -v http://localhost:18790/health
```

### Memoria insuficiente
Se o VPS tem pouca RAM, use o metodo binary em vez de Docker:
```bash
make deploy-hostinger HOSTINGER_HOST=SEU_IP_VPS HOSTINGER_DEPLOY_METHOD=binary
```

## Atualizacoes

Para atualizar o PicoClaw, basta rodar novamente:
```bash
make deploy-hostinger HOSTINGER_HOST=SEU_IP_VPS
```

Para reverter:
```bash
make deploy-hostinger-rollback HOSTINGER_HOST=SEU_IP_VPS
```
