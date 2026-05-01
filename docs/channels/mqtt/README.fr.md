# 📡 Canal MQTT

PicoClaw prend en charge n'importe quel client MQTT comme canal de messagerie. Les appareils ou services publient des requêtes vers un broker ; PicoClaw s'abonne, les traite et publie les réponses en retour.

## 🚀 Démarrage rapide

**1. Ajouter le canal dans `~/.picoclaw/config.json` :**

```json
{
  "channel_list": {
    "mqtt": {
      "enabled": true,
      "type": "mqtt",
      "settings": {
        "broker": "tcp://localhost:1883",
        "agent_id": "assistant"
      }
    }
  }
}
```

**2. Démarrer la passerelle :**

```bash
picoclaw gateway
```

**3. Envoyer un message depuis n'importe quel client MQTT :**

```bash
mosquitto_pub -t "/picoclaw/assistant/device1/request" \
  -m '{"text": "Quel est l'\''usage CPU ?"}'
```

**4. S'abonner pour recevoir la réponse :**

```bash
mosquitto_sub -t "/picoclaw/assistant/device1/response"
```

---

## 📨 Structure des topics

```
{prefix}/{agent_id}/{client_id}/request    # Client → PicoClaw
{prefix}/{agent_id}/{client_id}/response   # PicoClaw → Client
```

| Segment | Description |
|---------|-------------|
| `prefix` | Préfixe de topic configuré côté serveur. Défaut : `/picoclaw` |
| `agent_id` | Identifiant de l'instance PicoClaw, défini dans le champ `agent_id` |
| `client_id` | Identifiant de session défini par le client — utiliser un ID stable par appareil pour maintenir le contexte |

### Payload du message (JSON)

```json
{ "text": "votre message ici" }
```

---

## ⚙️ Configuration

### config.json

```json
{
  "channel_list": {
    "mqtt": {
      "enabled": true,
      "type": "mqtt",
      "settings": {
        "broker": "ssl://votre-broker:8883",
        "agent_id": "assistant",
        "topic_prefix": "/picoclaw",
        "client_id": "",
        "keep_alive": 60,
        "qos": 0
      }
    }
  }
}
```

### .security.yml (identifiants)

Le nom d'utilisateur et le mot de passe sont stockés dans `~/.picoclaw/.security.yml`, pas dans `config.json` :

```yaml
channel_list:
  mqtt:
    settings:
      username: votre_utilisateur
      password: votre_mot_de_passe
```

### Champs de configuration

| Champ | Emplacement | Requis | Défaut | Description |
|-------|-------------|--------|--------|-------------|
| `broker` | `settings` | Oui | — | URL du broker MQTT, ex. `tcp://host:1883`, `ssl://host:8883` |
| `agent_id` | `settings` | Oui | — | Identifiant de l'agent, utilisé dans le chemin du topic |
| `topic_prefix` | `settings` | Non | `/picoclaw` | Préfixe de l'espace de noms des topics |
| `username` | `.security.yml` | Non | — | Nom d'utilisateur pour l'authentification au broker |
| `password` | `.security.yml` | Non | — | Mot de passe pour l'authentification au broker |
| `client_id` | `settings` | Non | auto-généré | ID client paho envoyé au broker. Auto-généré sous la forme `picoclaw-mqtt-{agent_id}-{8 hex}` ; fixe pour la durée du processus, réutilisé à la reconnexion |
| `keep_alive` | `settings` | Non | `60` | Intervalle keepalive MQTT en secondes |
| `qos` | `settings` | Non | `0` | Niveau QoS pour la publication et l'abonnement : `0`, `1` ou `2` |

### Variables d'environnement

| Variable | Champ |
|----------|-------|
| `PICOCLAW_CHANNELS_MQTT_BROKER` | `broker` |
| `PICOCLAW_CHANNELS_MQTT_AGENT_ID` | `agent_id` |
| `PICOCLAW_CHANNELS_MQTT_TOPIC_PREFIX` | `topic_prefix` |
| `PICOCLAW_CHANNELS_MQTT_USERNAME` | `username` |
| `PICOCLAW_CHANNELS_MQTT_PASSWORD` | `password` |
| `PICOCLAW_CHANNELS_MQTT_CLIENT_ID` | `client_id` |
| `PICOCLAW_CHANNELS_MQTT_KEEP_ALIVE` | `keep_alive` |
| `PICOCLAW_CHANNELS_MQTT_QOS` | `qos` |

---

## 🔄 Reconnexion

PicoClaw se reconnecte automatiquement au broker en cas de perte de connexion, avec un intervalle de 5 secondes. L'abonnement est rétabli automatiquement. L'ID client côté broker reste identique à chaque reconnexion.

---

## ⚠️ Remarques

- **TLS** : SSL/TLS est supporté (URL broker en `ssl://`). La vérification du certificat est désactivée par défaut.
- **Réponses en streaming** : Les réponses en streaming envoient plusieurs messages vers le topic de réponse ; les concaténer dans l'ordre pour obtenir la réponse complète.
- **client_id vs ID de session** : Le `client_id` dans le chemin du topic est défini par votre application cliente. Il est distinct de l'ID client paho utilisé par PicoClaw pour se connecter au broker.
- **Instances multiples** : Si plusieurs instances PicoClaw utilisent le même `agent_id` sur le même broker, définir des `client_id` distincts pour éviter les conflits.
