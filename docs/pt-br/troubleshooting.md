# 🐛 Solução de Problemas

> Voltar ao [README](../../README.pt-br.md)

## "model ... not found in model_list" ou OpenRouter "free is not a valid model ID"

**Sintoma:** Você vê um dos seguintes erros:

- `Error creating provider: model "openrouter/free" not found in model_list`
- OpenRouter retorna 400: `"free is not a valid model ID"`

**Causa:** O campo `model` na sua entrada `model_list` é o que é enviado para a API. Para o OpenRouter, você deve usar o ID de modelo **completo**, não uma abreviação.

- **Errado:** `"model": "free"` → OpenRouter recebe `free` e rejeita.
- **Correto:** `"model": "openrouter/free"` → OpenRouter recebe `openrouter/free` (roteamento automático do nível gratuito).

**Correção:** Em `~/.picoclaw/config.json` (ou seu caminho de configuração):

1. **agents.defaults.model** deve corresponder a um `model_name` em `model_list` (ex.: `"openrouter-free"`).
2. O **model** dessa entrada deve ser um ID de modelo OpenRouter válido, por exemplo:
   - `"openrouter/free"` – nível gratuito automático
   - `"google/gemini-2.0-flash-exp:free"`
   - `"meta-llama/llama-3.1-8b-instruct:free"`

Exemplo:

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

Obtenha sua chave em [OpenRouter Keys](https://openrouter.ai/keys).
