# Google Chat

The Google Chat channel allows PicoClaw to receive messages from Google Chat via Cloud Pub/Sub and send replies using the Google Chat API.

## Configuration

```json
{
  "channels": {
    "googlechat": {
      "enabled": true,
      "subscription_id": "projects/YOUR_PROJECT_ID/subscriptions/YOUR_SUBSCRIPTION_ID",
      "project_id": "YOUR_PROJECT_ID",
      "allow_from": [],
      "debug": false
    }
  }
}
```

| Field           | Type   | Required | Description                                                                 |
| --------------- | ------ | -------- | --------------------------------------------------------------------------- |
| enabled         | bool   | Yes      | Enable Google Chat channel                                                  |
| subscription_id | string | Yes      | The full Pub/Sub subscription ID (e.g., `projects/.../subscriptions/...`)   |
| project_id      | string | No       | The Google Cloud Project ID. If omitted, it will be inferred from ADC or ID |
| allow_from      | array  | No       | List of allowed users (names or emails)                                     |
| debug           | bool   | No       | Enable debug logging for the channel                                        |

## Setup Guide

### 1. Google Cloud Setup
1. Create a Google Cloud Project.
2. Enable the **Google Chat API** and **Cloud Pub/Sub API**.
3. Create a Pub/Sub **Topic**.
4. Create a Pub/Sub **Subscription** for that topic.
5. Create a Service Account (or use your user account) and ensure it has:
   - `Pub/Sub Subscriber` role on the subscription.
   - `Pub/Sub Viewer` (optional but recommended).

### 2. Google Chat App Configuration
1. Go to [Google Chat API Configuration](https://console.cloud.google.com/apis/api/chat.googleapis.com/hangouts-chat) in Google Cloud Console.
2. Under **Connection settings**, select **Cloud Pub/Sub**.
3. Enter the **Topic Name** you created earlier.
4. Grant the Google Chat API service agent permission to publish to your topic (the console often provides a button to do this, or grant `roles/pubsub.publisher` to `chat-api-push@system.gserviceaccount.com`).

### 3. Authentication (Important)
This channel uses the `https://www.googleapis.com/auth/chat.bot` scope, which **requires a Service Account**.
You **cannot** use your personal user credentials (via `gcloud auth application-default login`) because user accounts cannot act as a Bot.

1.  Go to IAM & Admin > Service Accounts.
2.  Select the Service Account you created (which has Pub/Sub permissions).
3.  Go to **Keys** > **Add Key** > **Create new key** > **JSON**.
4.  Download the JSON key file.
5.  Set the environment variable when running PicoClaw:
    ```bash
    export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/service-account-key.json"
    ./picoclaw gateway
    ```
