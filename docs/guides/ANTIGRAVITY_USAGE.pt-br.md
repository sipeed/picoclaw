> Voltar ao [README](../project/README.pt-br.md)

# Usando o provedor Antigravity no PicoClaw

Este guia explica como configurar e usar o provedor **Antigravity** (Google Cloud Code Assist) no PicoClaw.

## Pré-requisitos

1.  Uma conta Google.
2.  Google Cloud Code Assist habilitado (geralmente disponível através da integração "Gemini for Google Cloud").

## 1. Autenticação

Para se autenticar com o Antigravity, execute o seguinte comando:

```bash
picoclaw auth login --provider antigravity
```

### Autenticação manual (Headless/VPS)
Se você está executando em um servidor (Coolify/Docker) e não consegue acessar `localhost`, siga estas etapas:
1.  Execute o comando acima.
2.  Copie a URL fornecida e abra-a no seu navegador local.
3.  Complete o login.
4.  Seu navegador será redirecionado para uma URL `localhost:51121` (que não carregará).
5.  **Copie essa URL final** da barra de endereços do seu navegador.
6.  **Cole-a de volta no terminal** onde o PicoClaw está aguardando.

O PicoClaw extrairá automaticamente o código de autorização e completará o processo.

## 2. Gerenciando modelos

### Listar modelos disponíveis
Para ver quais modelos seu projeto tem acesso e verificar suas cotas:

```bash
picoclaw auth models
```

### Trocar de modelo
Você pode alterar o modelo padrão em `~/.picoclaw/config.json` ou substituí-lo via CLI:

```bash
# Substituir para um único comando
picoclaw agent -m "Hello" --model claude-opus-4-6-thinking
```

## 3. Uso em produção (Coolify/Docker)

Se você está implantando via Coolify ou Docker, siga estas etapas para testar:

1.  **Variáveis de ambiente**:
    *   `PICOCLAW_AGENTS_DEFAULTS_MODEL=gemini-flash`
2.  **Persistência da autenticação**:
    Se você já fez login localmente, pode copiar suas credenciais para o servidor:
    ```bash
    scp ~/.picoclaw/auth.json user@your-server:~/.picoclaw/
    ```
    *Alternativamente*, execute o comando `auth login` uma vez no servidor se você tiver acesso ao terminal.

## 4. Solução de problemas

*   **Resposta vazia**: Se um modelo retorna uma resposta vazia, ele pode estar restrito para o seu projeto. Tente `gemini-3-flash` ou `claude-opus-4-6-thinking`.
*   **429 Limite de taxa**: O Antigravity possui cotas rigorosas. O PicoClaw exibirá o "tempo de redefinição" na mensagem de erro se você atingir um limite.
*   **404 Não encontrado**: Certifique-se de que está usando um ID de modelo da lista `picoclaw auth models`. Use o ID curto (ex.: `gemini-3-flash`) e não o caminho completo.

## 5. Resumo dos modelos funcionais

Com base nos testes, os seguintes modelos são os mais confiáveis:
*   `gemini-3-flash` (Rápido, alta disponibilidade)
*   `gemini-2.5-flash-lite` (Leve)
*   `claude-opus-4-6-thinking` (Poderoso, inclui raciocínio)
