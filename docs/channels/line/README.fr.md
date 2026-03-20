> Retour au [README](../../../README.fr.md)

# Line

PicoClaw prend en charge LINE via l'API LINE Messaging avec des callbacks webhook.

## Configuration

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

| Champ                | Type   | Requis | Description                                                              |
| -------------------- | ------ | ------ | ------------------------------------------------------------------------ |
| enabled              | bool   | Oui    | Activer ou non le canal LINE                                             |
| channel_secret       | string | Oui    | Channel Secret de l'API LINE Messaging                                   |
| channel_access_token | string | Oui    | Channel Access Token de l'API LINE Messaging                             |
| webhook_path         | string | Non    | Chemin du webhook (par défaut : /webhook/line)                           |
| allow_from           | array  | Non    | Liste blanche d'ID utilisateurs ; vide signifie tous les utilisateurs    |

## Procédure de configuration

1. Rendez-vous sur la [LINE Developers Console](https://developers.line.biz/console/) et créez un fournisseur de services ainsi qu'un canal Messaging API
2. Obtenez le Channel Secret et le Channel Access Token
3. Configurez le webhook :
   - LINE exige que les webhooks utilisent HTTPS. Vous devez donc déployer un serveur compatible HTTPS ou utiliser un outil de proxy inverse comme ngrok pour exposer votre serveur local sur Internet
   - PicoClaw utilise un serveur HTTP Gateway partagé pour recevoir les callbacks webhook de tous les canaux, écoutant par défaut sur 127.0.0.1:18790
   - Définissez l'URL du webhook sur `https://your-domain.com/webhook/line`, puis configurez un proxy inverse de votre domaine externe vers le Gateway local (port par défaut 18790)
   - Activez le webhook et vérifiez l'URL
4. Renseignez le Channel Secret et le Channel Access Token dans le fichier de configuration
