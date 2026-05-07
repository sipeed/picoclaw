> Retour au [README](../../project/README.fr.md)

# OneBot

OneBot est un standard de protocole ouvert pour les bots QQ, fournissant une interface unifiée pour diverses implémentations de bots QQ (par exemple go-cqhttp, Mirai). Il utilise WebSocket pour la communication.

## Configuration

```json
{
  "channel_list": {
    "onebot": {
      "enabled": true,
      "type": "onebot",
      "ws_url": "ws://localhost:8080",
      "access_token": "",
      "allow_from": []
    }
  }
}
```

| Champ        | Type   | Requis | Description                                                          |
| ------------ | ------ | ------ | -------------------------------------------------------------------- |
| enabled      | bool   | Oui    | Activer ou non le canal OneBot                                       |
| ws_url       | string | Oui    | URL WebSocket du serveur OneBot                                      |
| access_token | string | Non    | Jeton d'accès pour la connexion au serveur OneBot                    |
| allow_from   | array  | Non    | Liste blanche d'ID utilisateurs ; vide signifie tous les utilisateurs |

## Procédure de configuration

1. Déployez une implémentation compatible OneBot (par exemple napcat)
2. Configurez l'implémentation OneBot pour activer le service WebSocket et définir un jeton d'accès (si nécessaire)
3. Renseignez l'URL WebSocket et le jeton d'accès dans le fichier de configuration
