> Voltar ao [README](../../project/README.pt-br.md)

# Guia de Configuração do Canal Matrix

## 1. Exemplo de Configuração

Adicione isto ao `config.json`:

```json
{
  "channel_list": {
    "matrix": {
      "enabled": true,
      "type": "matrix",
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "device_id": "",
      "join_on_invite": true,
      "allow_from": [],
      "group_trigger": {
        "mention_only": true
      },
      "placeholder": {
        "enabled": true,
        "text": "Thinking..."
      },
      "reasoning_channel_id": "",
      "message_format": "richtext"
    }
  }
}
```

## 2. Referência de Campos

| Campo                | Tipo     | Obrigatório | Descrição |
|----------------------|----------|-------------|-----------|
| enabled              | bool     | Sim         | Habilitar ou desabilitar o canal Matrix |
| homeserver           | string   | Sim         | URL do homeserver Matrix (por exemplo `https://matrix.org`) |
| user_id              | string   | Sim         | ID de usuário Matrix do bot (por exemplo `@bot:matrix.org`) |
| access_token         | string   | Sim         | Token de acesso do bot |
| device_id            | string   | Não         | ID de dispositivo Matrix opcional |
| join_on_invite       | bool     | Não         | Entrar automaticamente em salas convidadas |
| allow_from           | []string | Não         | Lista branca de usuários (IDs Matrix) |
| group_trigger        | object   | Não         | Estratégia de gatilho de grupo (`mention_only` / `prefixes`) |
| placeholder          | object   | Não         | Configuração de mensagem de espaço reservado |
| reasoning_channel_id | string   | Não         | Canal alvo para saída de raciocínio |
| message_format       | string   | Não         | Formato de saída: `"richtext"` (padrão) renderiza markdown como HTML; `"plain"` envia apenas texto simples |

## 3. Suporte Atual

- Envio/recebimento de mensagens de texto com renderização markdown (negrito, itálico, cabeçalhos, blocos de código, etc.)
- Formato de mensagem configurável (`richtext` / `plain`)
- Download de imagens/áudio/vídeo/arquivos recebidos (MediaStore primeiro, fallback para caminho local)
- Normalização de áudio recebido no fluxo de transcrição existente (`[audio: ...]`)
- Upload e envio de imagens/áudio/vídeo/arquivos de saída
- Regras de gatilho de grupo (incluindo modo somente menção)
- Estado de digitação (`m.typing`)
- Mensagem de espaço reservado + substituição de resposta final
- Entrada automática em salas convidadas (pode ser desabilitado)

## 4. TODO

- Melhorias nos metadados de mídia rica (por exemplo tamanho e miniaturas de imagens/vídeos)
