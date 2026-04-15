> Retour au [README](../../../README.fr.md)

# QQ

PicoClaw prend en charge QQ via l'API Bot officielle de la plateforme ouverte QQ.

## Configuration

```json
{
  "channel_list": {
    "qq": {
      "enabled": true,
      "type": "qq",
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

| Champ      | Type   | Requis | Description                                                                 |
| ---------- | ------ | ------ | --------------------------------------------------------------------------- |
| enabled    | bool   | Oui    | Activer ou non le canal QQ                                                  |
| app_id     | string | Oui    | App ID de l'application bot QQ                                              |
| app_secret | string | Oui    | App Secret de l'application bot QQ                                          |
| allow_from | array  | Non    | Liste blanche d'identifiants utilisateur ; vide signifie tous les utilisateurs |

## Configuration initiale

### Configuration rapide (recommandée)

La plateforme ouverte QQ propose une entrée de création en un clic :

1. Ouvrir [QQ Bot Quick Create](https://q.qq.com/qqbot/openclaw/index.html) et se connecter en scannant le QR code
2. Le système crée automatiquement un bot — copier l'**App ID** et l'**App Secret**
3. Renseigner les identifiants dans le fichier de configuration PicoClaw
4. Exécuter `picoclaw gateway` pour démarrer le service
5. Ouvrir QQ et commencer à discuter avec le bot

> L'App Secret n'est affiché qu'une seule fois — sauvegardez-le immédiatement. Le consulter à nouveau forcera une réinitialisation.
>
> Les bots créés via l'entrée rapide sont réservés à l'usage personnel du créateur et ne prennent pas en charge les discussions de groupe. Pour la prise en charge des groupes, configurez le mode sandbox sur la [plateforme ouverte QQ](https://q.qq.com/).

### Configuration manuelle

1. Se connecter à la [plateforme ouverte QQ](https://q.qq.com/) avec son compte QQ et s'inscrire en tant que développeur
2. Créer un bot QQ et personnaliser son avatar et son nom
3. Obtenir l'**App ID** et l'**App Secret** dans les paramètres du bot
4. Renseigner les identifiants dans le fichier de configuration PicoClaw
5. Exécuter `picoclaw gateway` pour démarrer le service
6. Rechercher votre bot dans QQ et commencer à discuter

> Pendant le développement, il est recommandé d'activer le mode sandbox et d'y ajouter les utilisateurs et groupes de test pour le débogage.
