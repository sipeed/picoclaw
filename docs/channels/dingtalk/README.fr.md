> Retour au [README](../../project/README.fr.md)

# DingTalk

DingTalk est la plateforme de communication d'entreprise d'Alibaba, très populaire dans les milieux professionnels chinois. Elle utilise un SDK de streaming pour maintenir des connexions persistantes.

## Configuration

```json
{
  "channel_list": {
    "dingtalk": {
      "enabled": true,
      "type": "dingtalk",
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

| Champ         | Type   | Requis | Description                                                      |
| ------------- | ------ | ------ | ---------------------------------------------------------------- |
| enabled       | bool   | Oui    | Activer ou non le canal DingTalk                                 |
| client_id     | string | Oui    | Client ID de l'application DingTalk                              |
| client_secret | string | Oui    | Client Secret de l'application DingTalk                          |
| allow_from    | array  | Non    | Liste blanche d'ID utilisateurs ; vide signifie tous les utilisateurs |

## Procédure de configuration

1. Rendez-vous sur la [plateforme ouverte DingTalk](https://open.dingtalk.com/)
2. Créez une application interne d'entreprise
3. Obtenez le Client ID et le Client Secret depuis les paramètres de l'application
4. Configurez OAuth et les abonnements aux événements (si nécessaire)
5. Renseignez le Client ID et le Client Secret dans le fichier de configuration
