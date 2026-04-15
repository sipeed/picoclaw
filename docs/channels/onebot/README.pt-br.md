> Voltar ao [README](../../../README.pt-br.md)

# OneBot

OneBot é um padrão de protocolo aberto para bots QQ, fornecendo uma interface unificada para diversas implementações de bots QQ (ex.: go-cqhttp, Mirai). Utiliza WebSocket para comunicação.

## Configuração

```json
{
  "channel_list": {
    "onebot": {
      "enabled": true,
      "type": "onebot",
      "ws_url": "ws://localhost:8080",
      "access_token": "",
      "allow_from": []
    }
  }
}
```

| Campo        | Tipo   | Obrigatório | Descrição                                                            |
| ------------ | ------ | ----------- | -------------------------------------------------------------------- |
| enabled      | bool   | Sim         | Se o canal OneBot deve ser habilitado                                |
| ws_url       | string | Sim         | URL WebSocket do servidor OneBot                                     |
| access_token | string | Não         | Token de acesso para conexão ao servidor OneBot                      |
| allow_from   | array  | Não         | Lista de permissão de IDs de usuário; vazio permite todos            |

## Configuração passo a passo

1. Implante uma implementação compatível com OneBot (ex.: napcat)
2. Configure a implementação OneBot para habilitar o serviço WebSocket e definir um token de acesso (se necessário)
3. Preencha a URL WebSocket e o token de acesso no arquivo de configuração
