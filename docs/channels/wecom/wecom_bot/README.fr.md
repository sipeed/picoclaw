> Retour au [README](../../../../README.fr.md)

# WeCom Bot

Le WeCom Bot est une méthode d'intégration rapide fournie par WeCom, permettant de recevoir des messages via une URL Webhook.

## Configuration

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_path": "/webhook/wecom",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

| Champ | Type | Requis | Description |
| ---------------- | ------ | ------ | -------------------------------------------- |
| token | string | Oui | Jeton de vérification de signature |
| encoding_aes_key | string | Oui | Clé AES de 43 caractères utilisée pour le déchiffrement |
| webhook_url | string | Oui | URL Webhook du bot de groupe WeCom utilisée pour envoyer les réponses |
| webhook_path | string | Non | Chemin de l'endpoint webhook (par défaut : /webhook/wecom) |
| allow_from | array | Non | Liste blanche d'ID utilisateurs (vide = autoriser tous les utilisateurs) |
| reply_timeout | int | Non | Délai de réponse en secondes (par défaut : 5) |

## Procédure de configuration

1. Ajouter un bot à un groupe WeCom
2. Obtenir l'URL Webhook
3. (Pour recevoir des messages) Configurer l'adresse API de réception des messages (URL de callback), le Token et l'EncodingAESKey sur la page de configuration du bot
4. Saisir les informations pertinentes dans le fichier de configuration

   Remarque : PicoClaw utilise désormais un serveur HTTP Gateway partagé pour recevoir les callbacks webhook de tous les canaux. L'adresse d'écoute par défaut est 127.0.0.1:18790. Pour recevoir des callbacks depuis l'internet public, configurez un reverse proxy de votre domaine externe vers le Gateway (port par défaut 18790).
