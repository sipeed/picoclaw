> Voltar ao [README](../../project/README.pt-br.md)

# Line

O PicoClaw suporta o LINE por meio da LINE Messaging API com callbacks de webhook.

## Configuração

```json
{
  "channel_list": {
    "line": {
      "enabled": true,
      "type": "line",
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

| Campo                | Tipo   | Obrigatório | Descrição                                                              |
| -------------------- | ------ | ----------- | ---------------------------------------------------------------------- |
| enabled              | bool   | Sim         | Se o canal LINE deve ser habilitado                                    |
| channel_secret       | string | Sim         | Channel Secret da LINE Messaging API                                   |
| channel_access_token | string | Sim         | Channel Access Token da LINE Messaging API                             |
| webhook_path         | string | Não         | Caminho do webhook (padrão: /webhook/line)                             |
| allow_from           | array  | Não         | Lista de permissão de IDs de usuário; vazio permite todos              |

## Configuração passo a passo

1. Acesse o [LINE Developers Console](https://developers.line.biz/console/) e crie um provedor de serviços e um canal Messaging API
2. Obtenha o Channel Secret e o Channel Access Token
3. Configure o webhook:
   - O LINE exige que os webhooks usem HTTPS, portanto é necessário implantar um servidor com suporte a HTTPS ou usar uma ferramenta de proxy reverso como o ngrok para expor seu servidor local à internet
   - O PicoClaw usa um servidor HTTP Gateway compartilhado para receber callbacks de webhook de todos os canais, escutando em 127.0.0.1:18790 por padrão
   - Defina a URL do webhook como `https://your-domain.com/webhook/line` e configure um proxy reverso do seu domínio externo para o Gateway local (porta padrão 18790)
   - Ative o webhook e verifique a URL
4. Preencha o Channel Secret e o Channel Access Token no arquivo de configuração
