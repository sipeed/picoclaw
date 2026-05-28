# 🔄 Tâches Asynchrones et Spawn

> Retour au [README](../project/README.fr.md)

## Tâches Rapides (réponse directe)

- Rapporter l'heure actuelle

## Tâches Longues (utiliser spawn pour l'asynchrone)

- Rechercher sur le web des actualités IA et résumer
- Vérifier les emails et rapporter les messages importants
```

**Comportements clés :**

| Fonctionnalité          | Description                                                     |
| ----------------------- | --------------------------------------------------------------- |
| **spawn**               | Crée un subagent asynchrone, ne bloque pas le heartbeat         |
| **Independent context** | Le subagent a son propre contexte, pas d'historique de session  |
| **message tool**        | Le subagent communique directement avec l'utilisateur via l'outil message |
| **Non-blocking**        | Après le spawn, le heartbeat continue à la tâche suivante       |

#### Fonctionnement de la Communication du Subagent

```
Heartbeat se déclenche
    ↓
L'agent lit HEARTBEAT.md
    ↓
Pour une tâche longue : spawn subagent
    ↓                           ↓
Continue à la tâche suivante  Le subagent travaille indépendamment
    ↓                           ↓
Toutes les tâches terminées  Le subagent utilise l'outil "message"
    ↓                           ↓
Répond HEARTBEAT_OK          L'utilisateur reçoit le résultat directement
```

Le subagent a accès aux outils (message, web_search, etc.) et peut communiquer avec l'utilisateur indépendamment sans passer par l'agent principal.

**Configuration :**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Option     | Par défaut | Description                                    |
| ---------- | ---------- | ---------------------------------------------- |
| `enabled`  | `true`     | Activer/désactiver le heartbeat                |
| `interval` | `30`       | Intervalle de vérification en minutes (min: 5) |

**Variables d'environnement :**

* `PICOCLAW_HEARTBEAT_ENABLED=false` pour désactiver
* `PICOCLAW_HEARTBEAT_INTERVAL=60` pour changer l'intervalle
