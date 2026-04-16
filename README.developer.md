# How to Run
1. Create file docker/.env with following value:
    ```
    AWS_ACCESS_KEY_ID=
    AWS_SECRET_ACCESS_KEY=
    AWS_REGION_NAME=
    ```
2. Run docker compose launcher by running:
    ```
    docker compose --profile launcher up
    ```
3. Add AWS Bedrock model via launcher or editing config.json file:
    ```
    {
      "model_name": "claude-sonnet-bedrock",
      "model": "bedrock/global.anthropic.claude-sonnet-4-6",
      "api_base": "https://bedrock-runtime.ap-southeast-1.amazonaws.com",
      "api_keys": "[NOT_HERE]"
    }
    ```