> Retour au [README](../../../README.fr.md)

# Feishu

Feishu (nom international : Lark) est une plateforme de collaboration d'entreprise de ByteDance. Elle prend en charge les marchés chinois et mondiaux via des connexions WebSocket pilotées par événements.

## Configuration

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  }
}
```

| Champ                 | Type   | Requis | Description                                                                 |
| --------------------- | ------ | ------ | --------------------------------------------------------------------------- |
| enabled               | bool   | Oui    | Activer ou non le canal Feishu                                              |
| app_id                | string | Oui    | App ID de l'application Feishu (commence par `cli_`)                        |
| app_secret            | string | Oui    | App Secret de l'application Feishu                                          |
| encrypt_key           | string | Non    | Clé de chiffrement pour les callbacks d'événements                          |
| verification_token    | string | Non    | Token utilisé pour la vérification des événements Webhook                   |
| allow_from            | array  | Non    | Liste blanche d'identifiants utilisateur ; vide signifie tous les utilisateurs |
| random_reaction_emoji | array  | Non    | Liste d'emojis de réaction aléatoires ; vide utilise le "Pin" par défaut    |

## Configuration initiale

1. Accéder à la [plateforme ouverte Feishu](https://open.feishu.cn/) et créer une application
2. Activer la capacité **Bot** dans les paramètres de l'application
3. Créer une version et publier l'application (la configuration prend effet après la publication)
4. Obtenir l'**App ID** (commence par `cli_`) et l'**App Secret**
5. Renseigner l'App ID et l'App Secret dans le fichier de configuration PicoClaw
6. Exécuter `picoclaw gateway` pour démarrer le service
7. Rechercher le nom du bot dans Feishu et commencer une conversation

> PicoClaw se connecte à Feishu en mode WebSocket/SDK — aucune adresse de callback publique ni URL Webhook n'est requise.
>
> `encrypt_key` et `verification_token` sont optionnels ; l'activation du chiffrement des événements est recommandée pour les environnements de production.
>
> Pour les références d'emojis personnalisés, voir : [Liste des emojis Feishu](https://open.larkoffice.com/document/server-docs/im-v1/message-reaction/emojis-introduce)
