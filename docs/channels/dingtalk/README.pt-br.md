> Voltar ao [README](../../../README.pt-br.md)

# DingTalk

DingTalk é a plataforma de comunicação empresarial da Alibaba, amplamente utilizada no ambiente corporativo chinês. Ela usa um SDK de streaming para manter conexões persistentes.

## Configuração

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

| Campo         | Tipo   | Obrigatório | Descrição                                                        |
| ------------- | ------ | ----------- | ---------------------------------------------------------------- |
| enabled       | bool   | Sim         | Se o canal DingTalk deve ser habilitado                          |
| client_id     | string | Sim         | Client ID do aplicativo DingTalk                                 |
| client_secret | string | Sim         | Client Secret do aplicativo DingTalk                             |
| allow_from    | array  | Não         | Lista de permissão de IDs de usuário; vazio permite todos        |

## Configuração passo a passo

1. Acesse a [Plataforma Aberta DingTalk](https://open.dingtalk.com/)
2. Crie um aplicativo interno corporativo
3. Obtenha o Client ID e o Client Secret nas configurações do aplicativo
4. Configure OAuth e assinaturas de eventos (se necessário)
5. Preencha o Client ID e o Client Secret no arquivo de configuração
