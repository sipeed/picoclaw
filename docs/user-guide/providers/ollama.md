# Ollama Provider

Run LLMs locally using Ollama for offline, private, and cost-free inference.

## Why Ollama?

- **Free**: No API costs
- **Private**: Data stays on your machine
- **Offline**: Works without internet
- **Flexible**: Run many open-source models

## Prerequisites

- Sufficient RAM (8GB+ recommended for larger models)
- Sufficient disk space (models are 2-10GB+)
- Linux, macOS, or Windows (with WSL2)

## Setup

### Step 1: Install Ollama

**Linux/macOS:**

```bash
curl -fsSL https://ollama.com/install.sh | sh
```

**Or download from [ollama.com](https://ollama.com)**

### Step 2: Pull a Model

```bash
# Pull a model (example: Llama 3.2)
ollama pull llama3.2

# List available models
ollama list
```

### Step 3: Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model": "llama3.2"
    }
  },
  "providers": {
    "ollama": {
      "api_base": "http://localhost:11434/v1"
    }
  }
}
```

## Configuration Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `api_base` | No | `http://localhost:11434/v1` | Ollama API endpoint |

No API key needed - Ollama runs locally.

## Popular Models

| Model | Command | Size | Best For |
|-------|---------|------|----------|
| Llama 3.2 3B | `ollama pull llama3.2` | 2GB | General use |
| Llama 3.2 1B | `ollama pull llama3.2:1b` | 1GB | Fast, lightweight |
| Llama 3.1 8B | `ollama pull llama3.1` | 5GB | Better quality |
| Llama 3.1 70B | `ollama pull llama3.1:70b` | 40GB | Best quality |
| Mistral | `ollama pull mistral` | 4GB | Efficient |
| CodeLlama | `ollama pull codellama` | 4GB | Code generation |
| Phi-3 | `ollama pull phi3` | 2GB | Microsoft's small model |
| Gemma 2 | `ollama pull gemma2` | 5GB | Google's open model |
| Qwen 2.5 | `ollama pull qwen2.5` | 5GB | Multilingual |

## Model Management

### Pull Models

```bash
# Pull latest version
ollama pull llama3.2

# Pull specific version
ollama pull llama3.2:3b
```

### List Models

```bash
ollama list
```

Output:

```
NAME            ID              SIZE    MODIFIED
llama3.2:latest 123abc...       2.0 GB  2 hours ago
mistral:latest  456def...       4.1 GB  3 days ago
```

### Remove Models

```bash
ollama rm old-model
```

### Update Models

```bash
ollama pull llama3.2
```

## Model Format

Use the model name as shown in `ollama list`:

```json
{
  "agents": {
    "defaults": {
      "model": "llama3.2"
    }
  }
}
```

Or with provider prefix:

```json
{
  "agents": {
    "defaults": {
      "model": "ollama/llama3.2"
    }
  }
}
```

## Fallback Configuration

Combine local and cloud models:

```json
{
  "agents": {
    "defaults": {
      "model": "llama3.2",
      "model_fallbacks": [
        "openrouter/anthropic/claude-sonnet-4"
      ]
    }
  },
  "providers": {
    "ollama": {
      "api_base": "http://localhost:11434/v1"
    },
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    }
  }
}
```

## Remote Ollama

Connect to Ollama running on another machine:

```json
{
  "providers": {
    "ollama": {
      "api_base": "http://192.168.1.100:11434/v1"
    }
  }
}
```

### Start Ollama with Network Access

```bash
# Set host to listen on all interfaces
OLLAMA_HOST=0.0.0.0 ollama serve
```

## Resource Requirements

### RAM Guidelines

| Model Size | Minimum RAM | Recommended RAM |
|------------|-------------|-----------------|
| 1B parameters | 4GB | 8GB |
| 3B parameters | 6GB | 12GB |
| 7-8B parameters | 8GB | 16GB |
| 13B parameters | 16GB | 32GB |
| 70B parameters | 48GB | 64GB+ |

### GPU Acceleration

Ollama automatically uses GPU when available:

- **NVIDIA**: CUDA support (requires nvidia drivers)
- **AMD**: ROCm support
- **Apple Silicon**: Metal support (M1/M2/M3)

Check GPU usage:

```bash
# NVIDIA
nvidia-smi

# Or during inference
ollama run llama3.2
```

## Custom Models

### Create a Modelfile

```dockerfile
FROM llama3.2

# Set parameters
PARAMETER temperature 0.7
PARAMETER num_ctx 4096

# Set system prompt
SYSTEM You are a helpful coding assistant.
```

### Build Custom Model

```bash
ollama create my-coder -f Modelfile
```

### Use Custom Model

```json
{
  "agents": {
    "defaults": {
      "model": "my-coder"
    }
  }
}
```

## Troubleshooting

### Ollama Not Running

```
Error: connection refused
```

Start Ollama:

```bash
ollama serve
```

Or run interactively:

```bash
ollama run llama3.2
```

### Model Not Found

```
Error: model 'xxx' not found
```

Pull the model first:

```bash
ollama pull xxx
```

### Out of Memory

```
Error: out of memory
```

Solutions:
1. Use a smaller model
2. Close other applications
3. Reduce context length

### Slow Responses

Solutions:
1. Use a smaller model
2. Ensure GPU acceleration
3. Check system resources

### Port Already in Use

```
Error: address already in use
```

Change Ollama port:

```bash
OLLAMA_HOST=0.0.0.0:11435 ollama serve
```

Update PicoClaw config:

```json
{
  "providers": {
    "ollama": {
      "api_base": "http://localhost:11435/v1"
    }
  }
}
```

## Best Practices

1. **Start with small models** - Test with 1B-3B models first
2. **Monitor resources** - Watch RAM and GPU usage
3. **Use quantized models** - Default models are already optimized
4. **Combine with cloud** - Use fallbacks for reliability
5. **Keep models updated** - `ollama pull` to update

## See Also

- [Providers Overview](README.md)
- [Configuration Reference](../../configuration/config-file.md)
- [Model Fallbacks](../advanced/model-fallbacks.md)
- [Ollama Documentation](https://github.com/ollama/ollama)
