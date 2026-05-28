> Retour au [README](../../project/README.fr.md)

# Slack

Slack est l'une des principales plateformes de messagerie instantanée pour les entreprises. PicoClaw utilise le Socket Mode de Slack pour une communication bidirectionnelle en temps réel, sans nécessiter la configuration d'un endpoint webhook public.

## Configuration

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

| Champ      | Type   | Requis | Description                                                                  |
| ---------- | ------ | ------ | ---------------------------------------------------------------------------- |
| enabled    | bool   | Oui    | Activer ou non le canal Slack                                                |
| bot_token  | string | Oui    | Bot User OAuth Token du bot Slack (commence par xoxb-)                       |
| app_token  | string | Oui    | App Level Token Socket Mode de l'application Slack (commence par xapp-)      |
| allow_from | array  | Non    | Liste blanche d'ID utilisateurs ; vide signifie tous les utilisateurs        |

## Procédure de configuration

1. Rendez-vous sur [Slack API](https://api.slack.com/) et créez une nouvelle application Slack
2. Activez le Socket Mode et obtenez l'App Level Token
3. Ajoutez des Bot Token Scopes (par exemple `chat:write`, `im:history`, etc.)
4. Installez l'application dans votre espace de travail et obtenez le Bot User OAuth Token
5. Renseignez le Bot Token et l'App Token dans le fichier de configuration
