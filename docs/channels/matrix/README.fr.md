> Retour au [README](../../project/README.fr.md)

# Guide de configuration du canal Matrix

## 1. Exemple de configuration

Ajoutez ceci à `config.json` :

```json
{
  "channel_list": {
    "matrix": {
      "enabled": true,
      "type": "matrix",
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "device_id": "",
      "join_on_invite": true,
      "allow_from": [],
      "group_trigger": {
        "mention_only": true
      },
      "placeholder": {
        "enabled": true,
        "text": "Thinking..."
      },
      "reasoning_channel_id": "",
      "message_format": "richtext"
    }
  }
}
```

## 2. Référence des champs

| Champ                | Type     | Requis | Description |
|----------------------|----------|--------|-------------|
| enabled              | bool     | Oui    | Activer ou désactiver le canal Matrix |
| homeserver           | string   | Oui    | URL du homeserver Matrix (par exemple `https://matrix.org`) |
| user_id              | string   | Oui    | ID utilisateur Matrix du bot (par exemple `@bot:matrix.org`) |
| access_token         | string   | Oui    | Jeton d'accès du bot |
| device_id            | string   | Non    | ID d'appareil Matrix optionnel |
| join_on_invite       | bool     | Non    | Rejoindre automatiquement les salons invités |
| allow_from           | []string | Non    | Liste blanche d'utilisateurs (IDs Matrix) |
| group_trigger        | object   | Non    | Stratégie de déclenchement de groupe (`mention_only` / `prefixes`) |
| placeholder          | object   | Non    | Configuration du message de remplacement |
| reasoning_channel_id | string   | Non    | Canal cible pour la sortie de raisonnement |
| message_format       | string   | Non    | Format de sortie : `"richtext"` (défaut) rend le markdown en HTML ; `"plain"` envoie du texte brut uniquement |

## 3. Fonctionnalités actuellement supportées

- Envoi/réception de messages texte avec rendu markdown (gras, italique, titres, blocs de code, etc.)
- Format de message configurable (`richtext` / `plain`)
- Téléchargement d'images/audio/vidéo/fichiers entrants (MediaStore en priorité, chemin local en secours)
- Normalisation de l'audio entrant dans le flux de transcription existant (`[audio: ...]`)
- Upload et envoi d'images/audio/vidéo/fichiers sortants
- Règles de déclenchement de groupe (y compris le mode mention uniquement)
- État de frappe (`m.typing`)
- Message de remplacement + remplacement de la réponse finale
- Rejoindre automatiquement les salons invités (peut être désactivé)

## 4. TODO

- Améliorations des métadonnées des médias riches (par exemple taille et miniatures des images/vidéos)
