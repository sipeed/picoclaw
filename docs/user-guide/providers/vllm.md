# vLLM Provider

Use vLLM for high-performance, self-hosted LLM serving.

## Why vLLM?

- **High throughput**: Optimized for production workloads
- **Memory efficient**: PagedAttention for better GPU utilization
- **OpenAI compatible**: Drop-in replacement for OpenAI API
- **Flexible**: Supports many open-source models

## Prerequisites

- NVIDIA GPU (CUDA support)
- Docker (recommended) or Python environment
- Sufficient GPU memory for your model

## Setup

### Option 1: Docker (Recommended)

```bash
# Pull vLLM image
docker pull vllm/vllm-openai:latest

# Run vLLM server
docker run --gpus all \
  -v ~/.cache/huggingface:/root/.cache/huggingface \
  -p 8000:8000 \
  --ipc=host \
  vllm/vllm-openai:latest \
  --model meta-llama/Llama-3.2-3B-Instruct
```

### Option 2: Python Installation

```bash
# Install vLLM
pip install vllm

# Start server
python -m vllm.entrypoints.openai.api_server \
  --model meta-llama/Llama-3.2-3B-Instruct \
  --host 0.0.0.0 \
  --port 8000
```

### Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model": "meta-llama/Llama-3.2-3B-Instruct"
    }
  },
  "providers": {
    "vllm": {
      "api_base": "http://localhost:8000/v1"
    }
  }
}
```

## Configuration Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `api_base` | No | `http://localhost:8000/v1` | vLLM API endpoint |
| `api_key` | No | - | API key (if configured) |

## Supported Models

vLLM supports many Hugging Face models:

| Model Family | Example Models |
|--------------|----------------|
| Llama | `meta-llama/Llama-3.2-3B-Instruct` |
| Mistral | `mistralai/Mistral-7B-Instruct-v0.3` |
| Qwen | `Qwen/Qwen2.5-7B-Instruct` |
| Phi | `microsoft/Phi-3-mini-4k-instruct` |
| Gemma | `google/gemma-2-9b-it` |
| Mixtral | `mistralai/Mixtral-8x7B-Instruct-v0.1` |

## vLLM Server Options

### GPU Selection

```bash
# Use specific GPUs
docker run --gpus '"device=0,1"' \
  vllm/vllm-openai:latest \
  --model meta-llama/Llama-3.2-3B-Instruct
```

### Memory Management

```bash
# GPU memory utilization (0-1)
--gpu-memory-utilization 0.9

# Max model length
--max-model-len 4096
```

### Performance Tuning

```bash
# Tensor parallelism (multi-GPU)
--tensor-parallel-size 2

# Batch size
--max-num-seqs 256
```

### Full Example

```bash
docker run --gpus all \
  -v ~/.cache/huggingface:/root/.cache/huggingface \
  -p 8000:8000 \
  --ipc=host \
  vllm/vllm-openai:latest \
  --model meta-llama/Llama-3.2-3B-Instruct \
  --gpu-memory-utilization 0.9 \
  --max-model-len 4096 \
  --max-num-seqs 128
```

## Model Format

Use the full Hugging Face model name:

```json
{
  "agents": {
    "defaults": {
      "model": "meta-llama/Llama-3.2-3B-Instruct"
    }
  }
}
```

Or with provider prefix:

```json
{
  "agents": {
    "defaults": {
      "model": "vllm/meta-llama/Llama-3.2-3B-Instruct"
    }
  }
}
```

## Using with Hugging Face Token

Some models require authentication:

```bash
# Login to Hugging Face
huggingface-cli login

# Or set token
export HF_TOKEN=your_token

# Docker with token
docker run --gpus all \
  -e HF_TOKEN=your_token \
  -v ~/.cache/huggingface:/root/.cache/huggingface \
  -p 8000:8000 \
  vllm/vllm-openai:latest \
  --model meta-llama/Llama-3.2-3B-Instruct
```

## Fallback Configuration

Combine vLLM with cloud providers:

```json
{
  "agents": {
    "defaults": {
      "model": "meta-llama/Llama-3.2-3B-Instruct",
      "model_fallbacks": [
        "openrouter/meta-llama/llama-3.2-3b-instruct"
      ]
    }
  },
  "providers": {
    "vllm": {
      "api_base": "http://localhost:8000/v1"
    },
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    }
  }
}
```

## GPU Requirements

### Approximate VRAM Requirements

| Model Size | Minimum VRAM | Recommended VRAM |
|------------|--------------|------------------|
| 3B | 8GB | 12GB |
| 7B | 16GB | 24GB |
| 13B | 24GB | 40GB |
| 70B | 80GB (2x40GB) | 160GB (4x40GB) |

### Multi-GPU Setup

```bash
# 2 GPUs with tensor parallelism
docker run --gpus all \
  vllm/vllm-openai:latest \
  --model meta-llama/Llama-3.1-70B-Instruct \
  --tensor-parallel-size 2
```

## Remote vLLM

Connect to vLLM running on another server:

```json
{
  "providers": {
    "vllm": {
      "api_base": "http://192.168.1.100:8000/v1"
    }
  }
}
```

## Authentication

Add API key authentication:

```bash
# Start vLLM with API key
--api-key your-secret-key
```

Configure PicoClaw:

```json
{
  "providers": {
    "vllm": {
      "api_base": "http://localhost:8000/v1",
      "api_key": "your-secret-key"
    }
  }
}
```

## Troubleshooting

### Server Not Running

```
Error: connection refused
```

Check vLLM server:

```bash
curl http://localhost:8000/v1/models
```

### Out of GPU Memory

```
Error: CUDA out of memory
```

Solutions:
1. Use smaller model
2. Reduce `--gpu-memory-utilization`
3. Reduce `--max-model-len`
4. Use multiple GPUs

### Model Not Found

```
Error: model not found
```

Ensure model name is correct and accessible:

```bash
# List available models
curl http://localhost:8000/v1/models
```

### Slow Responses

Solutions:
1. Check GPU utilization
2. Adjust batch size
3. Enable tensor parallelism
4. Check network latency

### Permission Denied (Hugging Face)

```
Error: Access to model is restricted
```

Login to Hugging Face:

```bash
huggingface-cli login
```

## Production Deployment

### Systemd Service

```ini
[Unit]
Description=vLLM Server
After=network.target

[Service]
Type=simple
User=vllm
ExecStart=/usr/bin/docker run --gpus all \
  -v /var/lib/vllm:/root/.cache/huggingface \
  -p 8000:8000 \
  vllm/vllm-openai:latest \
  --model meta-llama/Llama-3.2-3B-Instruct
Restart=always

[Install]
WantedBy=multi-user.target
```

### Kubernetes

See vLLM documentation for Kubernetes deployment with:

- GPU resource requests
- Horizontal scaling
- Load balancing

## Best Practices

1. **Use Docker** - Easier deployment and isolation
2. **Monitor GPU usage** - Watch VRAM and utilization
3. **Tune parameters** - Optimize for your workload
4. **Set up fallbacks** - Cloud providers for reliability
5. **Cache models** - Mount HuggingFace cache volume

## See Also

- [Providers Overview](README.md)
- [Configuration Reference](../../configuration/config-file.md)
- [Model Fallbacks](../advanced/model-fallbacks.md)
- [vLLM Documentation](https://docs.vllm.ai)
