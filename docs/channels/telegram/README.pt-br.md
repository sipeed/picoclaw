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
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "allow_from": ["123456789"],
      "proxy": "",
      "use_markdown_v2": false
    }
  }
}
```

| Campo           | Tipo   | Obrigatório | Descrição                                                                  |
| --------------- | ------ | ----------- | -------------------------------------------------------------------------- |
| enabled         | bool   | Sim         | Se o canal Telegram deve ser habilitado                                    |
| token           | string | Sim         | Token da API de Bot do Telegram                                            |
| allow_from      | array  | Não         | Lista de IDs de usuários permitidos; vazio significa todos os usuários     |
| proxy           | string | Não         | URL do proxy para conexão com a API do Telegram (ex. http://127.0.0.1:7890) |
| use_markdown_v2 | bool   | Não         | Habilitar formatação Telegram MarkdownV2                                   |

## Configuração inicial

1. Pesquise por `@BotFather` no Telegram
2. Envie o comando `/newbot` e siga as instruções para criar um novo bot
3. Obtenha o Token da API HTTP
4. Preencha o Token no arquivo de configuração
5. (Opcional) Configure `allow_from` para restringir quais IDs de usuário podem interagir (os IDs podem ser obtidos via `@userinfobot`)

## Formatação Avançada

Você pode definir `use_markdown_v2: true` para habilitar opções de formatação aprimoradas. Isso permite que o bot utilize todos os recursos do Telegram MarkdownV2, incluindo estilos aninhados, spoilers e blocos de largura fixa personalizados.

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "use_markdown_v2": true
    }
  }
}
```
