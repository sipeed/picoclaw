# 🌐 Gateway HTTP API Reference

The PicoClaw gateway provides several HTTP endpoints for health monitoring, management, and direct chat interaction.

By default, the gateway listens on `127.0.0.1:18790`.

## 💬 Chat API

The `/chat` endpoint allows you to interact with the PicoClaw agent via a simple HTTP interface. This API is designed to be **asynchronous** to avoid timeouts during long-running LLM tasks or tool executions.

### 1. Initiate a Chat Session (POST)

Start a new chat request.

**Endpoint:** `POST /chat`
**Content-Type:** `application/json`

**Request Body:**
```json
{
  "message": "What is the capital of France?",
  "session_id": "optional-custom-id"
}
```

**Response (202 Accepted):**
```json
{
  "session_id": "chat-1711352400000",
  "status": "pending"
}
```

### 2. Poll for Results (GET)

Retrieve the status and response of a previously initiated session.

**Endpoint:** `GET /chat?session_id=<ID>`

**Possible Responses:**

*   **Still processing (200 OK):**
    ```json
    {
      "session_id": "chat-123",
      "status": "pending"
    }
    ```

*   **Completed (200 OK):**
    ```json
    {
      "session_id": "chat-123",
      "status": "completed",
      "response": "The capital of France is Paris."
    }
    ```

*   **Error (500 Internal Server Error):**
    ```json
    {
      "session_id": "chat-123",
      "status": "error",
      "error": "LLM call failed: context deadline exceeded"
    }
    ```

### 💾 Data Persistence & Cleanup
- **Expiry:** Completed or failed results are kept for **1 hour**. Pending sessions are kept for **2 hours**.
- **In-Memory:** Results are stored in memory and are lost if the gateway process is restarted.

---

## 🛠️ Management Endpoints

### Health Check
`GET /health`  
Returns `OK` (200) if the server is running. Used for basic uptime monitoring.

### Readiness Check
`GET /ready`  
Returns `OK` (200) once the gateway and all enabled channels have successfully initialized.

### Configuration Reload
`POST /reload`  
Triggers a hot-reload of the `.picoclaw/config.json` file without restarting the process.
