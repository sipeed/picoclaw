> Voltar ao [README](../../project/README.pt-br.md)

# Slack

O Slack é uma das principais plataformas de mensagens instantâneas para empresas. O PicoClaw usa o Socket Mode do Slack para comunicação bidirecional em tempo real, sem necessidade de configurar um endpoint de webhook público.

## Configuração

```json
{
  "channel_list": {
    "slack": {
      "enabled": true,
      "type": "slack",
      "bot_token": "xoxb-...",
      "app_token": "xapp-...",
      "allow_from": []
    }
  }
}
```

| Campo      | Tipo   | Obrigatório | Descrição                                                                    |
| ---------- | ------ | ----------- | ---------------------------------------------------------------------------- |
| enabled    | bool   | Sim         | Se o canal Slack deve ser habilitado                                         |
| bot_token  | string | Sim         | Bot User OAuth Token do bot Slack (começa com xoxb-)                         |
| app_token  | string | Sim         | App Level Token do Socket Mode do aplicativo Slack (começa com xapp-)        |
| allow_from | array  | Não         | Lista de permissão de IDs de usuário; vazio permite todos                    |

## Configuração passo a passo

1. Acesse o [Slack API](https://api.slack.com/) e crie um novo aplicativo Slack
2. Ative o Socket Mode e obtenha o App Level Token
3. Adicione Bot Token Scopes (ex.: `chat:write`, `im:history`, etc.)
4. Instale o aplicativo no seu workspace e obtenha o Bot User OAuth Token
5. Preencha o Bot Token e o App Token no arquivo de configuração
