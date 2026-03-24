> Retour au [README](../../../README.fr.md)

# Discord

Discord est une application gratuite de chat vocal, vidéo et textuel conçue pour les communautés. PicoClaw se connecte aux serveurs Discord via l'API Bot Discord, avec prise en charge de la réception et de l'envoi de messages.

## Configuration

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

| Champ         | Type   | Requis | Description                                                                 |
| ------------- | ------ | ------ | --------------------------------------------------------------------------- |
| enabled       | bool   | Oui    | Activer ou non le canal Discord                                             |
| token         | string | Oui    | Token du bot Discord                                                        |
| allow_from    | array  | Non    | Liste blanche d'identifiants utilisateur ; vide signifie tous les utilisateurs |
| group_trigger | object | Non    | Paramètres de déclenchement de groupe (exemple : { "mention_only": false }) |

## Configuration initiale

1. Accéder au [Portail des développeurs Discord](https://discord.com/developers/applications) et créer une nouvelle application
2. Activer les Intents :
   - Message Content Intent
   - Server Members Intent
3. Obtenir le Token du bot
4. Renseigner le Token du bot dans le fichier de configuration
5. Inviter le bot sur le serveur et lui accorder les permissions nécessaires (ex. envoyer des messages, lire l'historique des messages)
