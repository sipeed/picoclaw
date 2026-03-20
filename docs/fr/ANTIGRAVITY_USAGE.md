> Retour au [README](../../README.fr.md)

# Utiliser le fournisseur Antigravity dans PicoClaw

Ce guide explique comment configurer et utiliser le fournisseur **Antigravity** (Google Cloud Code Assist) dans PicoClaw.

## Prérequis

1.  Un compte Google.
2.  Google Cloud Code Assist activé (généralement disponible via l'intégration « Gemini for Google Cloud »).

## 1. Authentification

Pour vous authentifier avec Antigravity, exécutez la commande suivante :

```bash
picoclaw auth login --provider antigravity
```

### Authentification manuelle (Headless/VPS)
Si vous exécutez PicoClaw sur un serveur (Coolify/Docker) et ne pouvez pas accéder à `localhost`, suivez ces étapes :
1.  Exécutez la commande ci-dessus.
2.  Copiez l'URL fournie et ouvrez-la dans votre navigateur local.
3.  Complétez la connexion.
4.  Votre navigateur sera redirigé vers une URL `localhost:51121` (qui ne se chargera pas).
5.  **Copiez cette URL finale** depuis la barre d'adresse de votre navigateur.
6.  **Collez-la dans le terminal** où PicoClaw attend.

PicoClaw extraira automatiquement le code d'autorisation et terminera le processus.

## 2. Gestion des modèles

### Lister les modèles disponibles
Pour voir quels modèles sont accessibles à votre projet et vérifier leurs quotas :

```bash
picoclaw auth models
```

### Changer de modèle
Vous pouvez modifier le modèle par défaut dans `~/.picoclaw/config.json` ou le remplacer via le CLI :

```bash
# Remplacer pour une seule commande
picoclaw agent -m "Hello" --model claude-opus-4-6-thinking
```

## 3. Utilisation en production (Coolify/Docker)

Si vous déployez via Coolify ou Docker, suivez ces étapes pour tester :

1.  **Variables d'environnement** :
    *   `PICOCLAW_AGENTS_DEFAULTS_MODEL=gemini-flash`
2.  **Persistance de l'authentification** :
    Si vous vous êtes connecté localement, vous pouvez copier vos identifiants vers le serveur :
    ```bash
    scp ~/.picoclaw/auth.json user@your-server:~/.picoclaw/
    ```
    *Alternativement*, exécutez la commande `auth login` une fois sur le serveur si vous avez un accès terminal.

## 4. Dépannage

*   **Réponse vide** : Si un modèle renvoie une réponse vide, il peut être restreint pour votre projet. Essayez `gemini-3-flash` ou `claude-opus-4-6-thinking`.
*   **429 Limite de débit** : Antigravity a des quotas stricts. PicoClaw affichera le « temps de réinitialisation » dans le message d'erreur si vous atteignez une limite.
*   **404 Non trouvé** : Assurez-vous d'utiliser un ID de modèle provenant de la liste `picoclaw auth models`. Utilisez l'ID court (par ex. `gemini-3-flash`) et non le chemin complet.

## 5. Résumé des modèles fonctionnels

D'après les tests, les modèles suivants sont les plus fiables :
*   `gemini-3-flash` (Rapide, haute disponibilité)
*   `gemini-2.5-flash-lite` (Léger)
*   `claude-opus-4-6-thinking` (Puissant, inclut le raisonnement)
