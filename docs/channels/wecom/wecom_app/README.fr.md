> Retour au [README](../../../../README.fr.md)

# Application interne WeCom

Une application interne WeCom est une application créée par une entreprise au sein de WeCom, principalement destinée à un usage interne. Grâce aux applications internes WeCom, les entreprises peuvent assurer une communication et une collaboration efficaces avec leurs employés, améliorant ainsi la productivité.

## Configuration

```json
{
  "channels": {
    "wecom_app": {
      "enabled": true,
      "corp_id": "wwxxxxxxxxxxxxxxxx",
      "corp_secret": "YOUR_CORP_SECRET",
      "agent_id": 1000002,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-app",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

| Champ | Type | Requis | Description |
| ---------------- | ------ | ------ | ---------------------------------------- |
| corp_id | string | Oui | ID de l'entreprise |
| corp_secret | string | Oui | Secret de l'application |
| agent_id | int | Oui | ID de l'agent de l'application |
| token | string | Oui | Jeton de vérification du callback |
| encoding_aes_key | string | Oui | Clé AES de 43 caractères |
| webhook_path | string | Non | Chemin du webhook (par défaut : /webhook/wecom-app) |
| allow_from | array | Non | Liste blanche d'ID utilisateurs |
| reply_timeout | int | Non | Délai de réponse en secondes |

## Procédure de configuration

1. Connectez-vous à la [console d'administration WeCom](https://work.weixin.qq.com/)
2. Accédez à « Gestion des applications » -> « Créer une application »
3. Obtenez l'ID d'entreprise (CorpID) et le Secret de l'application
4. Configurez « Réception des messages » dans les paramètres de l'application pour obtenir le Token et l'EncodingAESKey
5. Définissez l'URL de callback sur `http://<your-server-ip>:<port>/webhook/wecom-app`
6. Saisissez le CorpID, le Secret, l'AgentID et les autres informations dans le fichier de configuration

   Remarque : PicoClaw utilise désormais un serveur HTTP Gateway partagé pour recevoir les callbacks webhook de tous les canaux. L'adresse d'écoute par défaut est 127.0.0.1:18790. Pour recevoir des callbacks depuis l'internet public, configurez un reverse proxy de votre domaine externe vers le Gateway (port par défaut 18790).
