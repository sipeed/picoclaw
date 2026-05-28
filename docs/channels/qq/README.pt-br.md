> Voltar ao [README](../../project/README.pt-br.md)

# QQ

O PicoClaw oferece suporte ao QQ via API Bot oficial da Plataforma Aberta QQ.

## Configuração

```json
{
  "channel_list": {
    "qq": {
      "enabled": true,
      "type": "qq",
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

| Campo      | Tipo   | Obrigatório | Descrição                                                                  |
| ---------- | ------ | ----------- | -------------------------------------------------------------------------- |
| enabled    | bool   | Sim         | Se o canal QQ deve ser habilitado                                          |
| app_id     | string | Sim         | App ID da aplicação bot QQ                                                 |
| app_secret | string | Sim         | App Secret da aplicação bot QQ                                             |
| allow_from | array  | Não         | Lista de IDs de usuários permitidos; vazio significa todos os usuários     |

## Configuração inicial

### Configuração rápida (recomendada)

A Plataforma Aberta QQ oferece uma entrada de criação com um clique:

1. Abra o [QQ Bot Quick Create](https://q.qq.com/qqbot/openclaw/index.html) e faça login escaneando o QR code
2. O sistema cria o bot automaticamente — copie o **App ID** e o **App Secret**
3. Preencha as credenciais no arquivo de configuração do PicoClaw
4. Execute `picoclaw gateway` para iniciar o serviço
5. Abra o QQ e comece a conversar com o bot

> O App Secret é exibido apenas uma vez — salve-o imediatamente. Visualizá-lo novamente forçará uma redefinição.
>
> Bots criados pela entrada rápida são apenas para uso pessoal do criador e não suportam chats em grupo. Para suporte a grupos, configure o modo sandbox na [Plataforma Aberta QQ](https://q.qq.com/).

### Configuração manual

1. Faça login na [Plataforma Aberta QQ](https://q.qq.com/) com sua conta QQ e registre-se como desenvolvedor
2. Crie um bot QQ e personalize seu avatar e nome
3. Obtenha o **App ID** e o **App Secret** nas configurações do bot
4. Preencha as credenciais no arquivo de configuração do PicoClaw
5. Execute `picoclaw gateway` para iniciar o serviço
6. Pesquise seu bot no QQ e comece a conversar

> Durante o desenvolvimento, recomenda-se habilitar o modo sandbox e adicionar usuários e grupos de teste ao sandbox para depuração.
