> Voltar ao [README](../../../README.pt-br.md)

# Discord

Discord é um aplicativo gratuito de chat de voz, vídeo e texto projetado para comunidades. O PicoClaw se conecta a servidores Discord via Discord Bot API, com suporte para receber e enviar mensagens.

## Configuração

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "group_trigger": {
        "mention_only": false
      }
    }
  }
}
```

| Campo         | Tipo   | Obrigatório | Descrição                                                                   |
| ------------- | ------ | ----------- | --------------------------------------------------------------------------- |
| enabled       | bool   | Sim         | Se o canal Discord deve ser habilitado                                      |
| token         | string | Sim         | Token do Bot Discord                                                        |
| allow_from    | array  | Não         | Lista de IDs de usuários permitidos; vazio significa todos os usuários      |
| group_trigger | object | Não         | Configurações de gatilho de grupo (exemplo: { "mention_only": false })      |

## Configuração inicial

1. Acesse o [Portal de Desenvolvedores do Discord](https://discord.com/developers/applications) e crie uma nova aplicação
2. Habilite os Intents:
   - Message Content Intent
   - Server Members Intent
3. Obtenha o Token do Bot
4. Preencha o Token do Bot no arquivo de configuração
5. Convide o bot para o servidor e conceda as permissões necessárias (ex. enviar mensagens, ler histórico de mensagens)
