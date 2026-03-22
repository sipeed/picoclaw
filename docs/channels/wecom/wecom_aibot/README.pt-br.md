> Voltar ao [README](../../../../README.pt-br.md)

# WeCom AI Bot

O WeCom AI Bot é uma forma oficial de integração de conversas com IA fornecida pelo WeCom. Suporta conversas privadas e em grupo, possui um protocolo de resposta em streaming integrado e suporta o envio proativo da resposta final via `response_url` após um timeout.

## Comparação com outros canais WeCom

| Recurso | WeCom Bot | WeCom App | **WeCom AI Bot** |
|---------|-----------|-----------|-----------------|
| Chat privado | ✅ | ✅ | ✅ |
| Chat em grupo | ✅ | ❌ | ✅ |
| Saída em streaming | ❌ | ❌ | ✅ |
| Push proativo em timeout | ❌ | ✅ | ✅ |
| Complexidade de configuração | Baixa | Alta | Média |

## Configuração

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "你好！有什么可以帮助你的吗？",
      "max_steps": 10
    }
  }
}
```

| Campo | Tipo | Obrigatório | Descrição |
| ---------------- | ------ | ----------- | -------------------------------------------------- |
| token | string | Sim | Token de verificação de callback, configurado na página de gerenciamento do AI Bot |
| encoding_aes_key | string | Sim | Chave AES de 43 caracteres, gerada aleatoriamente na página de gerenciamento do AI Bot |
| webhook_path | string | Não | Caminho do webhook (padrão: /webhook/wecom-aibot) |
| allow_from | array | Não | Lista de permissão de IDs de usuários; array vazio permite todos os usuários |
| welcome_message | string | Não | Mensagem de boas-vindas enviada quando o usuário abre o chat; deixe vazio para desativar |
| reply_timeout | int | Não | Timeout de resposta em segundos (padrão: 5) |
| max_steps | int | Não | Número máximo de etapas de execução do agente (padrão: 10) |

## Configuração passo a passo

1. Faça login no [Console de Administração do WeCom](https://work.weixin.qq.com/wework_admin)
2. Acesse "Gerenciamento de Apps" → "AI Bot", depois crie ou selecione um AI Bot
3. Na página de configuração do AI Bot, preencha as informações de "Recebimento de Mensagens":
   - **URL**: `http://<your-server-ip>:18790/webhook/wecom-aibot`
   - **Token**: Gerado aleatoriamente ou personalizado
   - **EncodingAESKey**: Clique em "Gerar Aleatoriamente" para obter uma chave de 43 caracteres
4. Insira o Token e o EncodingAESKey no arquivo de configuração do PicoClaw, inicie o serviço e volte ao console de administração para salvar (o WeCom enviará uma requisição de verificação)

> [!TIP]
> O servidor precisa ser acessível pelos servidores do WeCom. Se estiver em uma intranet ou desenvolvendo localmente, use [ngrok](https://ngrok.com) ou frp para tunelamento.

## Protocolo de resposta em streaming

O WeCom AI Bot usa um protocolo de "pull em streaming", diferente da resposta única de um webhook padrão:

```
Usuário envia uma mensagem
  │
  ▼
PicoClaw retorna imediatamente {finish: false} (Agente começa a processar)
  │
  ▼
WeCom faz pull aproximadamente a cada 1 segundo com {msgtype: "stream", stream: {id: "..."}}
  │
  ├─ Agente não concluído → retorna {finish: false} (continuar aguardando)
  │
  └─ Agente concluído → retorna {finish: true, content: "conteúdo da resposta"}
```

**Tratamento de timeout** (tarefa excede 30 segundos):

Se o processamento do agente demorar mais de aproximadamente 30 segundos (a janela máxima de polling do WeCom é de 6 minutos), o PicoClaw irá:

1. Fechar imediatamente o stream e exibir ao usuário: "⏳ 正在处理中，请稍候，结果将稍后发送。"
2. O agente continua executando em segundo plano
3. Após a conclusão do agente, a resposta final é enviada proativamente ao usuário via `response_url` incluído na mensagem

> `response_url` é emitido pelo WeCom, válido por 1 hora, pode ser usado apenas uma vez, sem necessidade de criptografia — basta fazer um POST com o corpo da mensagem em markdown diretamente.

## Mensagem de boas-vindas

Quando `welcome_message` está configurado, o PicoClaw responde automaticamente com essa mensagem quando um usuário abre a janela de chat com o AI Bot (evento `enter_chat`). Deixe vazio para ignorar silenciosamente.

```json
"welcome_message": "你好！我是 PicoClaw AI 助手，有什么可以帮你？"
```

## Perguntas frequentes

### Falha na verificação da URL de callback

- Confirme que o firewall do servidor tem a porta correspondente aberta (padrão 18790)
- Confirme que `token` e `encoding_aes_key` estão preenchidos corretamente
- Verifique os logs do PicoClaw para ver se uma requisição GET do WeCom foi recebida

### Mensagens sem resposta

- Verifique se `allow_from` está restringindo acidentalmente o remetente
- Procure por `context canceled` ou erros do agente nos logs
- Confirme que a configuração do agente (ex.: `model_name`) está correta

### Nenhum push final recebido para tarefas longas

- Confirme que o callback da mensagem inclui `response_url` (suportado apenas pelo novo WeCom AI Bot)
- Confirme que o servidor consegue fazer requisições de saída (precisa fazer POST para `response_url`)
- Verifique nos logs as palavras-chave `response_url mode` e `Sending reply via response_url`

## Referências

- [Documentação de integração do WeCom AI Bot](https://developer.work.weixin.qq.com/document/path/100719)
- [Descrição do protocolo de resposta em streaming](https://developer.work.weixin.qq.com/document/path/100719)
- [Resposta proativa via response_url](https://developer.work.weixin.qq.com/document/path/101138)
