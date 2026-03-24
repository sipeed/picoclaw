> Retour au [README](../../../../README.fr.md)

# WeCom AI Bot

Le WeCom AI Bot est une méthode d'intégration de conversation IA officiellement fournie par WeCom. Il prend en charge les conversations privées et de groupe, intègre un protocole de réponse en streaming et supporte l'envoi proactif de la réponse finale via `response_url` en cas de dépassement de délai.

## Comparaison avec les autres canaux WeCom

| Fonctionnalité | WeCom Bot | WeCom App | **WeCom AI Bot** |
|----------------|-----------|-----------|-----------------|
| Chat privé | ✅ | ✅ | ✅ |
| Chat de groupe | ✅ | ❌ | ✅ |
| Sortie en streaming | ❌ | ❌ | ✅ |
| Push proactif en cas de timeout | ❌ | ✅ | ✅ |
| Complexité de configuration | Faible | Élevée | Moyenne |

## Configuration

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "你好！有什么可以帮助你的吗？",
      "max_steps": 10
    }
  }
}
```

| Champ | Type | Requis | Description |
| ---------------- | ------ | ------ | -------------------------------------------------- |
| token | string | Oui | Jeton de vérification du callback, configuré sur la page de gestion de l'AI Bot |
| encoding_aes_key | string | Oui | Clé AES de 43 caractères, générée aléatoirement sur la page de gestion de l'AI Bot |
| webhook_path | string | Non | Chemin du webhook (par défaut : /webhook/wecom-aibot) |
| allow_from | array | Non | Liste blanche d'ID utilisateurs ; un tableau vide autorise tous les utilisateurs |
| welcome_message | string | Non | Message de bienvenue envoyé à l'ouverture du chat ; laisser vide pour désactiver |
| reply_timeout | int | Non | Délai de réponse en secondes (par défaut : 5) |
| max_steps | int | Non | Nombre maximum d'étapes d'exécution de l'agent (par défaut : 10) |

## Procédure de configuration

1. Connectez-vous à la [console d'administration WeCom](https://work.weixin.qq.com/wework_admin)
2. Accédez à « Gestion des applications » → « AI Bot », puis créez ou sélectionnez un AI Bot
3. Sur la page de configuration de l'AI Bot, renseignez les informations de « Réception des messages » :
   - **URL** : `http://<your-server-ip>:18790/webhook/wecom-aibot`
   - **Token** : Généré aléatoirement ou personnalisé
   - **EncodingAESKey** : Cliquez sur « Générer aléatoirement » pour obtenir une clé de 43 caractères
4. Saisissez le Token et l'EncodingAESKey dans le fichier de configuration PicoClaw, démarrez le service, puis revenez à la console d'administration pour enregistrer (WeCom enverra une requête de vérification)

> [!TIP]
> Le serveur doit être accessible par les serveurs WeCom. Si vous êtes sur un intranet ou en développement local, utilisez [ngrok](https://ngrok.com) ou frp pour le tunneling.

## Protocole de réponse en streaming

Le WeCom AI Bot utilise un protocole de « pull en streaming », différent de la réponse unique d'un webhook standard :

```
L'utilisateur envoie un message
  │
  ▼
PicoClaw retourne immédiatement {finish: false} (l'agent commence le traitement)
  │
  ▼
WeCom effectue un pull environ toutes les 1 seconde avec {msgtype: "stream", stream: {id: "..."}}
  │
  ├─ Agent non terminé → retourne {finish: false} (continuer à attendre)
  │
  └─ Agent terminé → retourne {finish: true, content: "contenu de la réponse"}
```

**Gestion du timeout** (tâche dépassant 30 secondes) :

Si le traitement de l'agent dépasse environ 30 secondes (la fenêtre de polling maximale de WeCom est de 6 minutes), PicoClaw va :

1. Fermer immédiatement le stream et afficher à l'utilisateur : « ⏳ 正在处理中，请稍候，结果将稍后发送。 »
2. L'agent continue de s'exécuter en arrière-plan
3. Une fois l'agent terminé, la réponse finale est envoyée proactivement à l'utilisateur via le `response_url` inclus dans le message

> `response_url` est émis par WeCom, valable 1 heure, utilisable une seule fois, sans chiffrement requis — il suffit de POSTer directement le corps du message markdown.

## Message de bienvenue

Lorsque `welcome_message` est configuré, PicoClaw répond automatiquement avec ce message lorsqu'un utilisateur ouvre la fenêtre de chat avec l'AI Bot (événement `enter_chat`). Laisser vide pour ignorer silencieusement.

```json
"welcome_message": "你好！我是 PicoClaw AI 助手，有什么可以帮你？"
```

## FAQ

### Échec de la vérification de l'URL de callback

- Vérifiez que le pare-feu du serveur autorise le port concerné (par défaut 18790)
- Vérifiez que `token` et `encoding_aes_key` sont correctement renseignés
- Consultez les logs PicoClaw pour voir si une requête GET de WeCom a été reçue

### Les messages ne reçoivent pas de réponse

- Vérifiez que `allow_from` ne restreint pas accidentellement l'expéditeur
- Recherchez `context canceled` ou des erreurs d'agent dans les logs
- Vérifiez que la configuration de l'agent (ex. `model_name`) est correcte

### Pas de push final reçu pour les tâches longues

- Vérifiez que le callback du message inclut `response_url` (uniquement supporté par la nouvelle version du WeCom AI Bot)
- Vérifiez que le serveur peut effectuer des requêtes sortantes (nécessite un POST vers `response_url`)
- Consultez les logs pour les mots-clés `response_url mode` et `Sending reply via response_url`

## Références

- [Documentation d'intégration WeCom AI Bot](https://developer.work.weixin.qq.com/document/path/100719)
- [Description du protocole de réponse en streaming](https://developer.work.weixin.qq.com/document/path/100719)
- [Réponse proactive via response_url](https://developer.work.weixin.qq.com/document/path/101138)
