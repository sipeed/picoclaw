> Voltar ao [README](../project/README.pt-br.md)

# Criptografia de Credenciais

O PicoClaw suporta a criptografia de valores `api_key` nas entradas de configuração `model_list`.
As chaves criptografadas são armazenadas como strings `enc://<base64>` e descriptografadas automaticamente na inicialização.

---

## Início Rápido

**1. Defina sua frase secreta**

```bash
export PICOCLAW_KEY_PASSPHRASE="your-passphrase"
```

**2. Criptografe uma chave de API**

Execute `picoclaw onboard` — ele solicita sua frase secreta e gera a chave SSH,
depois recriptografa automaticamente quaisquer entradas `api_key` em texto simples na sua configuração
na próxima chamada `SaveConfig`. O valor `enc://` resultante será semelhante a:

```
enc://AAAA...base64...
```

**3. Cole a saída na sua configuração**

```json
{
  "model_list": [
    {
      "model_name": "gpt-4o",
      "model": "openai/gpt-4o",
      "api_keys": ["enc://AAAA...base64..."],
      "api_base": "https://api.openai.com/v1"
    }
  ]
}
```

---

## Formatos de `api_key` Suportados

| Formato | Exemplo | Comportamento |
|---------|---------|---------------|
| Texto simples | `sk-abc123` | Usado como está |
| Referência de arquivo | `file://openai.key` | Conteúdo lido do mesmo diretório do arquivo de configuração |
| Criptografado | `enc://<base64>` | Descriptografado na inicialização usando `PICOCLAW_KEY_PASSPHRASE` |
| Vazio | `""` | Passado sem alteração (usado com `auth_method: oauth`) |

---

## Design Criptográfico

### Derivação de Chave

A criptografia utiliza **HKDF-SHA256** com uma chave privada SSH como segundo fator.

```
sshHash = SHA256(ssh_private_key_file_bytes)
ikm     = HMAC-SHA256(key=sshHash, message=passphrase)
aes_key = HKDF-SHA256(ikm, salt, info="picoclaw-credential-v1", 32 bytes)
```

### Criptografia

```
AES-256-GCM(key=aes_key, nonce=random[12], plaintext=api_key)
```

### Formato de Transmissão

```
enc://<base64( salt[16] + nonce[12] + ciphertext )>
```

| Campo | Tamanho | Descrição |
|-------|---------|-----------|
| `salt` | 16 bytes | Aleatório por criptografia; alimentado no HKDF |
| `nonce` | 12 bytes | Aleatório por criptografia; IV do AES-GCM |
| `ciphertext` | variável | Texto cifrado AES-256-GCM + tag de autenticação de 16 bytes |

O tag de autenticação GCM é anexado automaticamente ao texto cifrado. Qualquer adulteração faz com que a descriptografia falhe com um erro em vez de retornar texto simples corrompido.

### Desempenho

| Operação | Tempo (ARM Cortex-A) |
|----------|----------------------|
| Derivação de chave (HKDF) | < 1 ms |
| Descriptografia AES-256-GCM | < 1 ms |
| **Sobrecarga total na inicialização** | **< 2 ms por chave** |

---

## Segurança de Dois Fatores com Chave SSH

Quando uma chave privada SSH é fornecida, quebrar a criptografia requer **ambos**:

1. A **frase secreta** (`PICOCLAW_KEY_PASSPHRASE`)
2. O **arquivo de chave privada SSH**

Isso significa que um arquivo de configuração vazado sozinho não é suficiente para recuperar a chave de API, mesmo que a frase secreta seja fraca. A chave SSH contribui com 256 bits de entropia (Ed25519) independentemente da força da frase secreta.

### Modelo de Ameaça

| O que o atacante possui | Pode descriptografar? |
|------------------------|----------------------|
| Apenas o arquivo de configuração | Não — necessita da frase secreta + chave SSH |
| Apenas a chave SSH | Não — necessita da frase secreta |
| Apenas a frase secreta | Não — necessita da chave SSH |
| Arquivo de configuração + chave SSH + frase secreta | Sim — comprometimento total |

---

## Variáveis de Ambiente

| Variável | Obrigatório | Descrição |
|----------|-------------|-----------|
| `PICOCLAW_KEY_PASSPHRASE` | Sim (para `enc://`) | Frase secreta usada para derivação de chave |
| `PICOCLAW_SSH_KEY_PATH` | Não | Caminho para a chave privada SSH. Se não definido, detecta automaticamente em `~/.ssh/picoclaw_ed25519.key` |

### Detecção Automática da Chave SSH

Se `PICOCLAW_SSH_KEY_PATH` não estiver definido, o PicoClaw procura a chave dedicada:

```
~/.ssh/picoclaw_ed25519.key
```

Este arquivo dedicado evita conflitos com as chaves SSH existentes do usuário.
Execute `picoclaw onboard` para gerá-lo automaticamente.

`os.UserHomeDir()` é usado para resolução multiplataforma do diretório home (lê `USERPROFILE` no Windows, `HOME` no Unix/macOS).

> **Nota:** Um arquivo de chave SSH é obrigatório para a criptografia de credenciais. Se nenhuma chave for encontrada e `PICOCLAW_SSH_KEY_PATH` não estiver definido, a criptografia/descriptografia falhará. Execute `picoclaw onboard` para gerar a chave automaticamente.

---

## Migração

Como os únicos materiais secretos são `PICOCLAW_KEY_PASSPHRASE` e o arquivo de chave privada SSH, a migração é simples:

1. Copie o arquivo de configuração para a nova máquina.
2. Defina `PICOCLAW_KEY_PASSPHRASE` com o mesmo valor.
3. Copie o arquivo de chave privada SSH para o mesmo caminho (ou defina `PICOCLAW_SSH_KEY_PATH` para sua nova localização).

Nenhuma recriptografia é necessária.

---

## Considerações de Segurança

- **Tanto a frase secreta quanto a chave SSH são obrigatórias.** A chave SSH atua como um segundo fator — sem ela, a criptografia/descriptografia falhará. Execute `picoclaw onboard` para gerar a chave se ela não existir.
- **A chave SSH é somente leitura em tempo de execução.** O PicoClaw nunca escreve ou modifica o arquivo de chave SSH.
- **Chaves em texto simples continuam sendo suportadas.** Configurações existentes sem `enc://` não são afetadas.
- **O formato `enc://` é versionado** através do campo `info` do HKDF (`picoclaw-credential-v1`), permitindo futuras atualizações de algoritmo sem quebrar valores criptografados existentes.
