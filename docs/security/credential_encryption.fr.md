> Retour au [README](../project/README.fr.md)

# Chiffrement des identifiants

PicoClaw prend en charge le chiffrement des valeurs `api_key` dans les entrées de configuration `model_list`.
Les clés chiffrées sont stockées sous forme de chaînes `enc://<base64>` et déchiffrées automatiquement au démarrage.

---

## Démarrage rapide

**1. Définir votre phrase secrète**

```bash
export PICOCLAW_KEY_PASSPHRASE="your-passphrase"
```

**2. Chiffrer une clé API**

Exécutez `picoclaw onboard` — il vous demande votre phrase secrète et génère la clé SSH,
puis re-chiffre automatiquement toutes les entrées `api_key` en clair dans votre configuration
lors du prochain appel à `SaveConfig`. La valeur `enc://` résultante ressemblera à :

```
enc://AAAA...base64...
```

**3. Coller la sortie dans votre configuration**

```json
{
  "model_list": [
    {
      "model_name": "gpt-4o",
      "model": "openai/gpt-4o",
      "api_key": "enc://AAAA...base64...",
      "api_base": "https://api.openai.com/v1"
    }
  ]
}
```

---

## Formats `api_key` pris en charge

| Format | Exemple | Comportement |
|--------|---------|--------------|
| Texte clair | `sk-abc123` | Utilisé tel quel |
| Référence fichier | `file://openai.key` | Contenu lu depuis le même répertoire que le fichier de configuration |
| Chiffré | `enc://<base64>` | Déchiffré au démarrage avec `PICOCLAW_KEY_PASSPHRASE` |
| Vide | `""` | Transmis tel quel (utilisé avec `auth_method: oauth`) |

---

## Conception cryptographique

### Dérivation de clé

Le chiffrement utilise **HKDF-SHA256** avec une clé privée SSH comme second facteur.

```
sshHash = SHA256(ssh_private_key_file_bytes)
ikm     = HMAC-SHA256(key=sshHash, message=passphrase)
aes_key = HKDF-SHA256(ikm, salt, info="picoclaw-credential-v1", 32 bytes)
```

### Chiffrement

```
AES-256-GCM(key=aes_key, nonce=random[12], plaintext=api_key)
```

### Format de transmission

```
enc://<base64( salt[16] + nonce[12] + ciphertext )>
```

| Champ | Taille | Description |
|-------|--------|-------------|
| `salt` | 16 octets | Aléatoire par chiffrement ; fourni à HKDF |
| `nonce` | 12 octets | Aléatoire par chiffrement ; IV AES-GCM |
| `ciphertext` | variable | Texte chiffré AES-256-GCM + tag d'authentification de 16 octets |

Le tag d'authentification GCM est automatiquement ajouté au texte chiffré. Toute altération provoque l'échec du déchiffrement avec une erreur plutôt que de retourner un texte clair corrompu.

### Performance

| Opération | Durée (ARM Cortex-A) |
|-----------|----------------------|
| Dérivation de clé (HKDF) | < 1 ms |
| Déchiffrement AES-256-GCM | < 1 ms |
| **Surcoût total au démarrage** | **< 2 ms par clé** |

---

## Sécurité à deux facteurs avec clé SSH

Lorsqu'une clé privée SSH est fournie, casser le chiffrement nécessite **les deux** :

1. La **phrase secrète** (`PICOCLAW_KEY_PASSPHRASE`)
2. Le **fichier de clé privée SSH**

Cela signifie qu'un fichier de configuration divulgué seul ne suffit pas pour récupérer la clé API, même si la phrase secrète est faible. La clé SSH apporte 256 bits d'entropie (Ed25519) indépendamment de la force de la phrase secrète.

### Modèle de menace

| Ce que l'attaquant possède | Peut-il déchiffrer ? |
|---------------------------|---------------------|
| Fichier de configuration uniquement | Non — nécessite la phrase secrète + la clé SSH |
| Clé SSH uniquement | Non — nécessite la phrase secrète |
| Phrase secrète uniquement | Non — nécessite la clé SSH |
| Fichier de configuration + clé SSH + phrase secrète | Oui — compromission totale |

---

## Variables d'environnement

| Variable | Requis | Description |
|----------|--------|-------------|
| `PICOCLAW_KEY_PASSPHRASE` | Oui (pour `enc://`) | Phrase secrète utilisée pour la dérivation de clé |
| `PICOCLAW_SSH_KEY_PATH` | Non | Chemin vers la clé privée SSH. Si non défini, détection automatique depuis `~/.ssh/picoclaw_ed25519.key` |

### Détection automatique de la clé SSH

Si `PICOCLAW_SSH_KEY_PATH` n'est pas défini, PicoClaw recherche la clé dédiée :

```
~/.ssh/picoclaw_ed25519.key
```

Ce fichier dédié évite les conflits avec les clés SSH existantes de l'utilisateur.
Exécutez `picoclaw onboard` pour le générer automatiquement.

`os.UserHomeDir()` est utilisé pour la résolution multiplateforme du répertoire personnel (lit `USERPROFILE` sous Windows, `HOME` sous Unix/macOS).

> **Remarque :** Un fichier de clé SSH est requis pour le chiffrement des identifiants. Si aucune clé n'est trouvée et que `PICOCLAW_SSH_KEY_PATH` n'est pas défini, le chiffrement/déchiffrement échouera. Exécutez `picoclaw onboard` pour générer la clé automatiquement.

---

## Migration

Étant donné que les seuls éléments secrets sont `PICOCLAW_KEY_PASSPHRASE` et le fichier de clé privée SSH, la migration est simple :

1. Copiez le fichier de configuration sur la nouvelle machine.
2. Définissez `PICOCLAW_KEY_PASSPHRASE` avec la même valeur.
3. Copiez le fichier de clé privée SSH au même chemin (ou définissez `PICOCLAW_SSH_KEY_PATH` vers son nouvel emplacement).

Aucun re-chiffrement n'est nécessaire.

---

## Considérations de sécurité

- **La phrase secrète et la clé SSH sont toutes deux requises.** La clé SSH agit comme un second facteur — sans elle, le chiffrement/déchiffrement échouera. Exécutez `picoclaw onboard` pour générer la clé si elle n'existe pas.
- **La clé SSH est en lecture seule à l'exécution.** PicoClaw n'écrit ni ne modifie jamais le fichier de clé SSH.
- **Les clés en texte clair restent prises en charge.** Les configurations existantes sans `enc://` ne sont pas affectées.
- **Le format `enc://` est versionné** via le champ `info` de HKDF (`picoclaw-credential-v1`), permettant de futures mises à niveau d'algorithme sans casser les valeurs chiffrées existantes.
