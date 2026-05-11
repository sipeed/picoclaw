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

| Champ           | Type   | Requis | Description                                                              |
| --------------- | ------ | ------ | ------------------------------------------------------------------------ |
| enabled         | bool   | Oui    | Activer ou non le canal Telegram                                         |
| allow_from      | array  | Non    | Liste blanche d'identifiants utilisateur ; vide signifie tous les utilisateurs |
| settings.proxy  | string | Non    | URL du proxy pour se connecter à l'API Telegram (ex. http://127.0.0.1:7890) |
| settings.use_markdown_v2 | bool | Non | Activer le formatage Telegram MarkdownV2                                 |
| settings.business_mode | bool | Non | Activer la gestion des messages Telegram Business                         |
| settings.business_owner | string | Non | ID utilisateur Telegram du propriétaire Business à ignorer                 |
| settings.business_commands_enable | bool | Non | Autoriser les commandes du bot dans les chats Telegram Business            |
| settings.guest_mode | bool | Non | Activer la gestion et les réponses des messages Telegram Guest             |

## Configuration initiale

1. Rechercher `@BotFather` dans Telegram
2. Envoyer la commande `/newbot` et suivre les instructions pour créer un nouveau bot
3. Obtenir le Token de l'API HTTP
4. Renseigner le Token dans le fichier de configuration
5. (Optionnel) Configurer `allow_from` pour restreindre les identifiants utilisateur autorisés à interagir (les IDs peuvent être obtenus via `@userinfobot`)

## Mode Telegram Business

Définissez `settings.business_mode: true` pour recevoir et répondre aux messages Telegram Business des comptes connectés. Les réponses utilisent le `business_connection_id` entrant, et PicoClaw marque les messages Business comme lus lorsque le bot dispose du droit `can_read_messages`.

Définissez `settings.business_owner` avec l'ID utilisateur Telegram du propriétaire du compte Business. Les messages Business envoyés par cet utilisateur sont ignorés, ce qui évite de répondre aux messages envoyés manuellement depuis le compte connecté.

Par défaut, les commandes du bot sont ignorées dans les chats Business. Définissez `settings.business_commands_enable: true` pour autoriser `/new`, `/help`, `/show`, `/list` et `/use`.

Exemple :

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

## Mode Telegram Guest

Définissez `settings.guest_mode: true` pour recevoir les mises à jour `guest_message` et répondre avec la méthode Telegram `answerGuestQuery`. Les messages Guest proviennent de chats où le bot n'est pas membre, donc PicoClaw crée des sessions séparées avec le `guest_query_id` entrant.

Lorsque `settings.guest_mode` est `false`, PicoClaw ne demande pas les mises à jour Guest et ignore tout message Guest décodé. Les indicateurs de saisie et de placeholder sont ignorés pour les réponses Guest car Telegram requiert une seule réponse `answerGuestQuery`.

Exemple :

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

## Formatage avancées

Vous pouvez définir `use_markdown_v2: true` pour activer les options de formatage améliorées. Cela permet au bot d'utiliser toutes les fonctionnalités de Telegram MarkdownV2, y compris les styles imbriqués, les spoilers et les blocs de largeur fixe personnalisés.

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
