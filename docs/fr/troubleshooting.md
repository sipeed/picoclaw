# 🐛 Dépannage

> Retour au [README](../../README.fr.md)

## "model ... not found in model_list" ou OpenRouter "free is not a valid model ID"

**Symptôme :** Vous voyez l'une des erreurs suivantes :

- `Error creating provider: model "openrouter/free" not found in model_list`
- OpenRouter retourne 400 : `"free is not a valid model ID"`

**Cause :** Le champ `model` dans votre entrée `model_list` est ce qui est envoyé à l'API. Pour OpenRouter, vous devez utiliser l'identifiant de modèle **complet**, pas un raccourci.

- **Incorrect :** `"model": "free"` → OpenRouter reçoit `free` et le rejette.
- **Correct :** `"model": "openrouter/free"` → OpenRouter reçoit `openrouter/free` (routage automatique du niveau gratuit).

**Correction :** Dans `~/.picoclaw/config.json` (ou votre chemin de configuration) :

1. **agents.defaults.model** doit correspondre à un `model_name` dans `model_list` (par ex. `"openrouter-free"`).
2. Le **model** de cette entrée doit être un identifiant de modèle OpenRouter valide, par exemple :
   - `"openrouter/free"` – niveau gratuit automatique
   - `"google/gemini-2.0-flash-exp:free"`
   - `"meta-llama/llama-3.1-8b-instruct:free"`

Exemple :

```json
{
  "agents": {
    "defaults": {
      "model": "openrouter-free"
    }
  },
  "model_list": [
    {
      "model_name": "openrouter-free",
      "model": "openrouter/free",
      "api_key": "sk-or-v1-YOUR_OPENROUTER_KEY",
      "api_base": "https://openrouter.ai/api/v1"
    }
  ]
}
```

Obtenez votre clé sur [OpenRouter Keys](https://openrouter.ai/keys).
