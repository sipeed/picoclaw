> Voltar ao [README](../../project/README.pt-br.md)

# Telegram

O canal Telegram utiliza long polling via a API de Bot do Telegram para comunicação baseada em bots. Suporta mensagens de texto, anexos de mídia (fotos, voz, áudio, documentos), transcrição de voz via Groq Whisper e tratamento de comandos integrado.

## Configuração

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "proxy": "",
        "use_markdown_v2": false,
        "business_mode": false,
        "business_owner": "123456789",
        "business_commands_enable": false,
        "guest_mode": false
      }
    }
  }
}
```

| Campo           | Tipo   | Obrigatório | Descrição                                                                  |
| --------------- | ------ | ----------- | -------------------------------------------------------------------------- |
| enabled         | bool   | Sim         | Se o canal Telegram deve ser habilitado                                    |
| allow_from      | array  | Não         | Lista de IDs de usuários permitidos; vazio significa todos os usuários     |
| settings.proxy  | string | Não         | URL do proxy para conexão com a API do Telegram (ex. http://127.0.0.1:7890) |
| settings.use_markdown_v2 | bool | Não | Habilitar formatação Telegram MarkdownV2                                   |
| settings.business_mode | bool | Não | Habilitar tratamento de mensagens Telegram Business                        |
| settings.business_owner | string | Não | ID de usuário Telegram do proprietário Business a ser ignorado             |
| settings.business_commands_enable | bool | Não | Permitir comandos do bot em chats Telegram Business                        |
| settings.guest_mode | bool | Não | Habilitar tratamento e respostas de mensagens Telegram Guest               |

## Configuração inicial

1. Pesquise por `@BotFather` no Telegram
2. Envie o comando `/newbot` e siga as instruções para criar um novo bot
3. Obtenha o Token da API HTTP
4. Preencha o Token no arquivo de configuração
5. (Opcional) Configure `allow_from` para restringir quais IDs de usuário podem interagir (os IDs podem ser obtidos via `@userinfobot`)

## Modo Telegram Business

Defina `settings.business_mode: true` para receber e responder mensagens Telegram Business de contas comerciais conectadas. As respostas usam o `business_connection_id` recebido, e as mensagens Business são marcadas como lidas quando o bot tem o direito `can_read_messages`.

Use `settings.business_owner` para informar o ID de usuário Telegram do proprietário da conta Business. Mensagens Business desse usuário são ignoradas, evitando respostas automáticas a mensagens enviadas manualmente pela conta conectada.

Por padrão, comandos do bot em chats Business são ignorados. Defina `settings.business_commands_enable: true` para processar `/new`, `/help`, `/show`, `/list` e `/use`.

Exemplo :

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "business_mode": true,
        "business_owner": "123456789",
        "business_commands_enable": true
      }
    }
  }
}
```

## Modo Telegram Guest

Defina `settings.guest_mode: true` para receber atualizações `guest_message` e responder com o método Telegram `answerGuestQuery`. Mensagens Guest vêm de chats onde o bot não é membro, então o PicoClaw mantém sessões separadas usando o `guest_query_id` recebido.

Quando `settings.guest_mode` é `false`, atualizações Guest não são solicitadas e qualquer mensagem Guest decodificada é ignorada. Indicadores de digitação e placeholders são ignorados para respostas Guest porque o Telegram requer uma única resposta `answerGuestQuery`.

Exemplo :

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "guest_mode": true
      }
    }
  }
}
```

## Formatação Avançada

Você pode definir `use_markdown_v2: true` para habilitar opções de formatação aprimoradas. Isso permite que o bot utilize todos os recursos do Telegram MarkdownV2, incluindo estilos aninhados, spoilers e blocos de largura fixa personalizados.

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["YOUR_USER_ID"],
      "settings": {
        "use_markdown_v2": true
      }
    }
  }
}
```
