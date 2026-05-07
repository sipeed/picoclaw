> Voltar ao [README](../../project/README.pt-br.md)

# Feishu

Feishu (nome internacional: Lark) é uma plataforma de colaboração empresarial da ByteDance. Suporta os mercados chinês e global por meio de conexões WebSocket orientadas a eventos.

## Configuração

```json
{
  "channel_list": {
    "feishu": {
      "enabled": true,
      "type": "feishu",
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  }
}
```

| Campo                 | Tipo   | Obrigatório | Descrição                                                                  |
| --------------------- | ------ | ----------- | -------------------------------------------------------------------------- |
| enabled               | bool   | Sim         | Se o canal Feishu deve ser habilitado                                      |
| app_id                | string | Sim         | App ID da aplicação Feishu (começa com `cli_`)                             |
| app_secret            | string | Sim         | App Secret da aplicação Feishu                                             |
| encrypt_key           | string | Não         | Chave de criptografia para callbacks de eventos                            |
| verification_token    | string | Não         | Token usado para verificação de eventos Webhook                            |
| allow_from            | array  | Não         | Lista de IDs de usuários permitidos; vazio significa todos os usuários     |
| random_reaction_emoji | array  | Não         | Lista de emojis de reação aleatórios; vazio usa o "Pin" padrão             |

## Configuração inicial

1. Acesse a [Plataforma Aberta Feishu](https://open.feishu.cn/) e crie uma aplicação
2. Habilite a capacidade de **Bot** nas configurações da aplicação
3. Crie uma versão e publique a aplicação (a configuração entra em vigor após a publicação)
4. Obtenha o **App ID** (começa com `cli_`) e o **App Secret**
5. Preencha o App ID e o App Secret no arquivo de configuração do PicoClaw
6. Execute `picoclaw gateway` para iniciar o serviço
7. Pesquise o nome do bot no Feishu e inicie uma conversa

> O PicoClaw se conecta ao Feishu usando o modo WebSocket/SDK — nenhum endereço de callback público ou URL de Webhook é necessário.
>
> `encrypt_key` e `verification_token` são opcionais; recomenda-se habilitar a criptografia de eventos em ambientes de produção.
>
> Para referências de emojis personalizados, consulte: [Lista de Emojis do Feishu](https://open.larkoffice.com/document/server-docs/im-v1/message-reaction/emojis-introduce)

## Limitações de Plataforma

> ⚠️ **O canal Feishu não suporta dispositivos 32 bits.** O SDK do Feishu fornece apenas builds 64 bits. Arquiteturas 32 bits (armv6, armv7, mipsle, etc.) não podem usar o canal Feishu. Para mensagens em dispositivos 32 bits, use Telegram, Discord ou OneBot.
