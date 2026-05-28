> Retour au [README](../../project/README.fr.md)

# Telegram

Le canal Telegram utilise le long polling via l'API Bot Telegram pour une communication basée sur les bots. Il prend en charge les messages texte, les pièces jointes multimédias (photos, messages vocaux, audio, documents), la transcription vocale via Groq Whisper et la gestion des commandes intégrée.

## Configuration

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

| Champ           | Type   | Requis | Description                                                              |
| --------------- | ------ | ------ | ------------------------------------------------------------------------ |
| enabled         | bool   | Oui    | Activer ou non le canal Telegram                                         |
| token           | string | Oui    | Token de l'API Bot Telegram                                              |
| allow_from      | array  | Non    | Liste blanche d'identifiants utilisateur ; vide signifie tous les utilisateurs |
| proxy           | string | Non    | URL du proxy pour se connecter à l'API Telegram (ex. http://127.0.0.1:7890) |
| use_markdown_v2 | bool   | Non    | Activer le formatage Telegram MarkdownV2                                 |

## Configuration initiale

1. Rechercher `@BotFather` dans Telegram
2. Envoyer la commande `/newbot` et suivre les instructions pour créer un nouveau bot
3. Obtenir le Token de l'API HTTP
4. Renseigner le Token dans le fichier de configuration
5. (Optionnel) Configurer `allow_from` pour restreindre les identifiants utilisateur autorisés à interagir (les IDs peuvent être obtenus via `@userinfobot`)

## Formatage avancées

Vous pouvez définir `use_markdown_v2: true` pour activer les options de formatage améliorées. Cela permet au bot d'utiliser toutes les fonctionnalités de Telegram MarkdownV2, y compris les styles imbriqués, les spoilers et les blocs de largeur fixe personnalisés.

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
