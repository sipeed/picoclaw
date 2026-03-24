> Voltar ao [README](../../../../README.pt-br.md)

# App Interno WeCom

Um App Interno WeCom é um aplicativo criado por uma empresa dentro do WeCom, destinado principalmente ao uso interno. Por meio dos Apps Internos WeCom, as empresas podem alcançar comunicação e colaboração eficientes com os funcionários, melhorando a produtividade.

## Configuração

```json
{
  "channels": {
    "wecom_app": {
      "enabled": true,
      "corp_id": "wwxxxxxxxxxxxxxxxx",
      "corp_secret": "YOUR_CORP_SECRET",
      "agent_id": 1000002,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-app",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

| Campo | Tipo | Obrigatório | Descrição |
| ---------------- | ------ | ----------- | ---------------------------------------- |
| corp_id | string | Sim | ID da empresa |
| corp_secret | string | Sim | Segredo da aplicação |
| agent_id | int | Sim | ID do agente da aplicação |
| token | string | Sim | Token de verificação de callback |
| encoding_aes_key | string | Sim | Chave AES de 43 caracteres |
| webhook_path | string | Não | Caminho do webhook (padrão: /webhook/wecom-app) |
| allow_from | array | Não | Lista de permissão de IDs de usuários |
| reply_timeout | int | Não | Timeout de resposta em segundos |

## Configuração passo a passo

1. Faça login no [Console de Administração do WeCom](https://work.weixin.qq.com/)
2. Acesse "Gerenciamento de Apps" -> "Criar App"
3. Obtenha o ID da Empresa (CorpID) e o Secret do App
4. Configure "Receber Mensagens" nas configurações do app para obter o Token e o EncodingAESKey
5. Defina a URL de callback como `http://<your-server-ip>:<port>/webhook/wecom-app`
6. Insira o CorpID, Secret, AgentID e outras informações no arquivo de configuração

   Nota: O PicoClaw agora usa um servidor HTTP Gateway compartilhado para receber callbacks de webhook de todos os canais. O endereço de escuta padrão é 127.0.0.1:18790. Para receber callbacks da internet pública, configure um reverse proxy do seu domínio externo para o Gateway (porta padrão 18790).
