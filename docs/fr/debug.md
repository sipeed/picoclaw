# Débogage de PicoClaw

> Retour au [README](../../README.fr.md)

PicoClaw effectue de multiples interactions complexes en arrière-plan pour chaque requête qu'il reçoit — du routage des messages et de l'évaluation de la complexité, à l'exécution des outils et à l'adaptation aux défaillances de modèle. Pouvoir voir exactement ce qui se passe est crucial, non seulement pour résoudre les problèmes potentiels, mais aussi pour véritablement comprendre le fonctionnement de l'agent.

## Démarrer PicoClaw en mode débogage

Pour obtenir des informations détaillées sur ce que fait l'agent (requêtes LLM, appels d'outils, routage des messages), vous pouvez démarrer la passerelle PicoClaw avec le drapeau de débogage :

```bash
picoclaw gateway --debug
# or
picoclaw gateway -d
```

Dans ce mode, le système formate les logs de manière détaillée et affiche des aperçus des prompts système et des résultats d'exécution des outils.

## Désactiver la troncature des logs (logs complets)

Par défaut, PicoClaw tronque les chaînes très longues (comme le *Prompt Système* ou les résultats JSON volumineux) dans les logs de débogage afin de garder la console lisible.

Si vous avez besoin d'inspecter la sortie complète d'une commande ou le payload exact envoyé au modèle LLM, vous pouvez utiliser le drapeau `--no-truncate`.

**Remarque :** Ce drapeau fonctionne *uniquement* en combinaison avec le mode `--debug`.

```bash
picoclaw gateway --debug --no-truncate

```

Lorsque ce drapeau est actif, la fonction de troncature globale est désactivée. Cela est extrêmement utile pour :

* Vérifier la syntaxe exacte des messages envoyés au fournisseur.
* Lire la sortie complète d'outils comme `exec`, `web_fetch` ou `read_file`.
* Déboguer l'historique de session sauvegardé en mémoire.
