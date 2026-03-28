> Retour au [README](../../../README.fr.md)

# WeCom

PicoClaw expose WeCom en tant que canal unique `channels.wecom`, basé sur l'API WebSocket officielle WeCom AI Bot.
Ce canal remplace l'ancienne séparation `wecom`, `wecom_app` et `wecom_aibot` par un modèle de configuration unifié.

> Aucune URL de callback webhook publique n'est requise. PicoClaw établit une connexion WebSocket sortante vers WeCom.

## Fonctionnalités prises en charge

- Chat privé et chat de groupe
- Réponses en streaming côté canal via le protocole WeCom AI Bot
- Messages entrants : texte, voix, image, fichier, vidéo et messages mixtes
- Réponses sortantes : texte et médias (`image`, `file`, `voice`, `video`)
- Onboarding par QR code via l'interface Web ou le CLI
- Liste blanche partagée et routage `reasoning_channel_id`

---

## Démarrage rapide

### Option 1 : Liaison QR via l'interface Web (recommandé)

Ouvrez l'interface Web, accédez à **Channels → WeCom** et cliquez sur le bouton de liaison QR. Scannez le QR code avec WeCom et confirmez dans l'application — les identifiants sont enregistrés automatiquement.

<p align="center">
<img src="../../../assets/wecom-qr-binding.jpg" alt="Liaison QR WeCom dans l'interface Web" width="600">
</p>

### Option 2 : Connexion QR via le CLI

Exécutez :

```bash
picoclaw auth wecom
```

La commande :
1. Demande un QR code à WeCom et l'affiche dans le terminal
2. Affiche également un **lien QR code** que vous pouvez ouvrir dans un navigateur si le QR du terminal est difficile à scanner
3. Attend la confirmation — après le scan, vous devez également **confirmer la connexion dans l'application WeCom**
4. En cas de succès, écrit `bot_id` et `secret` dans `channels.wecom` et sauvegarde la configuration

Le délai d'expiration par défaut est de **5 minutes**. Utilisez `--timeout` pour l'étendre :

```bash
picoclaw auth wecom --timeout 10m
```

> ⚠️ Scanner le QR code ne suffit pas — vous devez également appuyer sur **Confirmer** dans l'application WeCom, sinon la commande expirera.

### Option 3 : Configuration manuelle

Si vous disposez déjà d'un `bot_id` et d'un `secret` depuis la plateforme WeCom AI Bot, configurez directement :

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "bot_id": "YOUR_BOT_ID",
      "secret": "YOUR_SECRET",
      "websocket_url": "wss://openws.work.weixin.qq.com",
      "send_thinking_message": true,
      "allow_from": [],
      "reasoning_channel_id": ""
    }
  }
}
```

---

## Configuration

| Champ | Type | Défaut | Description |
| ----- | ---- | ------ | ----------- |
| `enabled` | bool | `false` | Activer le canal WeCom. |
| `bot_id` | string | — | Identifiant WeCom AI Bot. Requis lorsque le canal est activé. |
| `secret` | string | — | Secret WeCom AI Bot. Stocké chiffré dans `.security.yml`. Requis lorsque le canal est activé. |
| `websocket_url` | string | `wss://openws.work.weixin.qq.com` | Point de terminaison WebSocket WeCom. |
| `send_thinking_message` | bool | `true` | Envoyer un message `Processing...` avant le début de la réponse en streaming. |
| `allow_from` | array | `[]` | Liste blanche des expéditeurs. Vide signifie autoriser tous les expéditeurs. |
| `reasoning_channel_id` | string | `""` | ID de chat optionnel pour router la sortie de raisonnement vers une conversation séparée. |

### Variables d'environnement

Tous les champs peuvent être remplacés par des variables d'environnement avec le préfixe `PICOCLAW_CHANNELS_WECOM_` :

| Variable d'environnement | Champ correspondant |
| ------------------------ | ------------------- |
| `PICOCLAW_CHANNELS_WECOM_ENABLED` | `enabled` |
| `PICOCLAW_CHANNELS_WECOM_BOT_ID` | `bot_id` |
| `PICOCLAW_CHANNELS_WECOM_SECRET` | `secret` |
| `PICOCLAW_CHANNELS_WECOM_WEBSOCKET_URL` | `websocket_url` |
| `PICOCLAW_CHANNELS_WECOM_SEND_THINKING_MESSAGE` | `send_thinking_message` |
| `PICOCLAW_CHANNELS_WECOM_ALLOW_FROM` | `allow_from` |
| `PICOCLAW_CHANNELS_WECOM_REASONING_CHANNEL_ID` | `reasoning_channel_id` |

---

## Comportement à l'exécution

- PicoClaw maintient un tour WeCom actif pour que les réponses en streaming puissent continuer sur le même flux lorsque c'est possible.
- Les réponses en streaming ont une durée maximale de **5,5 minutes** et un intervalle d'envoi minimum de **500 ms**.
- Si le streaming n'est plus disponible, les réponses basculent vers la livraison par push actif.
- Les associations de routes de chat expirent après **30 minutes** d'inactivité.
- Les médias entrants sont téléchargés dans le stockage média local avant d'être transmis à l'agent.
- Les médias sortants sont uploadés vers WeCom en tant que fichier temporaire, puis envoyés comme message média.
- Les messages en double sont détectés et supprimés (tampon circulaire des 1000 derniers identifiants de messages).

---

## Migration depuis l'ancienne configuration WeCom

| Configuration précédente | Migration |
| ------------------------ | --------- |
| `channels.wecom` (bot webhook) | Remplacer par `channels.wecom` avec `bot_id` + `secret`. |
| `channels.wecom_app` | Supprimer. Utiliser `channels.wecom` à la place. |
| `channels.wecom_aibot` | Déplacer `bot_id` et `secret` vers `channels.wecom`. |
| `token`, `encoding_aes_key`, `webhook_url`, `webhook_path` | Plus utilisés. Supprimer de la configuration. |
| `corp_id`, `corp_secret`, `agent_id` | Plus utilisés. Supprimer de la configuration. |
| `welcome_message`, `processing_message`, `max_steps` | Ne font plus partie de la configuration du canal WeCom. |

---

## Dépannage

### La liaison QR expire

- Après avoir scanné le QR code, vous devez également **confirmer la connexion dans l'application WeCom**. Le scan seul ne suffit pas.
- Relancez avec un `--timeout` plus long : `picoclaw auth wecom --timeout 10m`
- Si le QR code dans le terminal est difficile à scanner, utilisez le **lien QR code** affiché en dessous pour l'ouvrir dans un navigateur.

### QR code expiré

- Le QR code a une durée de validité limitée. Relancez `picoclaw auth wecom` pour en obtenir un nouveau.

### Échec de la connexion WebSocket

- Vérifiez que `bot_id` et `secret` sont corrects.
- Confirmez que l'hôte peut atteindre `wss://openws.work.weixin.qq.com` (WebSocket sortant, aucun port entrant nécessaire).

### Les réponses n'arrivent pas

- Vérifiez si `allow_from` bloque l'expéditeur.
- Vérifiez que `channels.wecom.bot_id` et `channels.wecom.secret` sont définis et non vides.
