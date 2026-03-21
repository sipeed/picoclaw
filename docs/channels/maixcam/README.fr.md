> Retour au [README](../../../README.fr.md)

# MaixCam

MaixCam est un canal dédié à la connexion aux caméras AI Sipeed MaixCAM et MaixCAM2. Il utilise des sockets TCP pour une communication bidirectionnelle et prend en charge les scénarios de déploiement d'IA en périphérie.

## Configuration

```json
{
  "channels": {
    "maixcam": {
      "enabled": true,
      "host": "0.0.0.0",
      "port": 18790,
      "allow_from": []
    }
  }
}
```

| Champ      | Type   | Requis | Description                                                                 |
| ---------- | ------ | ------ | --------------------------------------------------------------------------- |
| enabled    | bool   | Oui    | Activer ou non le canal MaixCam                                             |
| host       | string | Oui    | Adresse d'écoute du serveur TCP                                             |
| port       | int    | Oui    | Port d'écoute du serveur TCP                                                |
| allow_from | array  | Non    | Liste blanche d'identifiants d'appareils ; vide signifie tous les appareils |

## Cas d'utilisation

Le canal MaixCam permet à PicoClaw de fonctionner comme backend IA pour les appareils en périphérie :

- **Surveillance intelligente** : MaixCAM envoie des images ; PicoClaw les analyse via des modèles de vision
- **Contrôle IoT** : Les appareils envoient des données de capteurs ; PicoClaw coordonne les réponses
- **IA hors ligne** : Déployer PicoClaw sur un réseau local pour une inférence à faible latence
