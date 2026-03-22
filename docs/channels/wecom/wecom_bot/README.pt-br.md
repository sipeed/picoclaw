> Voltar ao [README](../../../../README.pt-br.md)

# WeCom Bot

O WeCom Bot é um método de integração rápida fornecido pelo WeCom que pode receber mensagens via URL de Webhook.

## Configuração

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_path": "/webhook/wecom",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

| Campo | Tipo | Obrigatório | Descrição |
| ---------------- | ------ | ----------- | -------------------------------------------- |
| token | string | Sim | Token de verificação de assinatura |
| encoding_aes_key | string | Sim | Chave AES de 43 caracteres usada para descriptografia |
| webhook_url | string | Sim | URL do webhook do bot de grupo WeCom usada para enviar respostas |
| webhook_path | string | Não | Caminho do endpoint webhook (padrão: /webhook/wecom) |
| allow_from | array | Não | Lista de permissão de IDs de usuários (vazio = permitir todos) |
| reply_timeout | int | Não | Timeout de resposta em segundos (padrão: 5) |

## Configuração passo a passo

1. Adicione um bot a um grupo WeCom
2. Obtenha a URL do Webhook
3. (Para receber mensagens) Configure o endereço da API de recebimento de mensagens (URL de callback), Token e EncodingAESKey na página de configuração do bot
4. Insira as informações relevantes no arquivo de configuração

   Nota: O PicoClaw agora usa um servidor HTTP Gateway compartilhado para receber callbacks de webhook de todos os canais. O endereço de escuta padrão é 127.0.0.1:18790. Para receber callbacks da internet pública, configure um reverse proxy do seu domínio externo para o Gateway (porta padrão 18790).
