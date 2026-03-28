# PicoClaw K3s Deployment

This directory contains the Kubernetes manifests for deploying the PicoClaw agent on a K3s cluster. The deployment is hardened with workspace isolation and secure secret management.

## 📁 Manifests

- **[deployment.yaml](deployment.yaml)**: Defines the PicoClaw agent deployment, including an init container for configuration syncing and volume mounts for secrets and persistent storage.
- **[configmap.yaml](configmap.yaml)**: The main agent configuration (Syncs to `config.json`).
- **[secrets.yaml](secrets.yaml)**: Template for sensitive API keys (Telegram, NVIDIA, Azure, etc.).
- **[pvc.yaml](pvc.yaml)**: Persistent Volume Claim for agent workspaces and chat history.
- **[service.yaml](service.yaml)**: Internal service for MCP server communication.

## 🚀 Deployment Steps

### 1. Configure Secrets
Open **[secrets.yaml](secrets.yaml)** and replace the placeholders with your actual API keys. Then apply it to your cluster:

```bash
kubectl apply -f secrets.yaml
```

### 2. Prepare Storage
Ensure your K3s cluster has a default storage class or configure the **[pvc.yaml](pvc.yaml)** to match your storage provider:

```bash
kubectl apply -f pvc.yaml
```

### 3. Deploy the Agent
Apply the configuration and the deployment:

```bash
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

## 🔒 Security Features

### Workspace Isolation
The agent is configured to restrict all filesystem tools to its respective workspace. The `deployment.yaml` ensures the correct directory structure is initialized before the agent starts.

### Secret Management
API keys are never stored in the `ConfigMap`. Instead, they are mounted as files from a Kubernetes Secret into `/etc/picoclaw/secrets/`. The agent reads these using the `file://` scheme:

```json
"token": "file:///etc/picoclaw/secrets/telegram-token"
```

### Safe Command Execution
Standard high-risk shell commands are blocked by the `exec` tool's safety guard. Targeted relaxations (e.g., for `git push`) are explicitly added to `custom_allow_patterns` in `configmap.yaml`.

## 🛠️ Management

### Logs
To view the agent logs:
```bash
kubectl logs -f deployment/picoclaw-agent
```

### Updating Configuration
1. Modify **[configmap.yaml](configmap.yaml)**.
2. Apply the change: `kubectl apply -f configmap.yaml`.
3. Restart the pod: `kubectl rollout restart deployment/picoclaw-agent`.
