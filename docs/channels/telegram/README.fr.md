> Retour au [README](../../../README.fr.md)

# Telegram

Le canal Telegram utilise le long polling via l'API Bot Telegram pour une communication basée sur les bots. Il prend en charge les messages texte, les pièces jointes multimédias (photos, messages vocaux, audio, documents), la transcription vocale via Groq Whisper et la gestion des commandes intégrée.

## Configuration

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "allow_from": ["123456789"],
      "proxy": ""
    }
  }
}
```

| Champ      | Type   | Requis | Description                                                              |
| ---------- | ------ | ------ | ------------------------------------------------------------------------ |
| enabled    | bool   | Oui    | Activer ou non le canal Telegram                                         |
| token      | string | Oui    | Token de l'API Bot Telegram                                              |
| allow_from | array  | Non    | Liste blanche d'identifiants utilisateur ; vide signifie tous les utilisateurs |
| proxy      | string | Non    | URL du proxy pour se connecter à l'API Telegram (ex. http://127.0.0.1:7890) |

## Configuration initiale

1. Rechercher `@BotFather` dans Telegram
2. Envoyer la commande `/newbot` et suivre les instructions pour créer un nouveau bot
3. Obtenir le Token de l'API HTTP
4. Renseigner le Token dans le fichier de configuration
5. (Optionnel) Configurer `allow_from` pour restreindre les identifiants utilisateur autorisés à interagir (les IDs peuvent être obtenus via `@userinfobot`)
