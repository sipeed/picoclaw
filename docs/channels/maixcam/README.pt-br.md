> Voltar ao [README](../../../README.pt-br.md)

# MaixCam

MaixCam é um canal dedicado para conectar dispositivos de câmera AI Sipeed MaixCAM e MaixCAM2. Utiliza sockets TCP para comunicação bidirecional e suporta cenários de implantação de IA na borda.

## Configuração

```json
{
  "channels": {
    "maixcam": {
      "enabled": true,
      "host": "0.0.0.0",
      "port": 18790,
      "allow_from": []
    }
  }
}
```

| Campo      | Tipo   | Obrigatório | Descrição                                                                  |
| ---------- | ------ | ----------- | -------------------------------------------------------------------------- |
| enabled    | bool   | Sim         | Se o canal MaixCam deve ser habilitado                                     |
| host       | string | Sim         | Endereço de escuta do servidor TCP                                         |
| port       | int    | Sim         | Porta de escuta do servidor TCP                                            |
| allow_from | array  | Não         | Lista de IDs de dispositivos permitidos; vazio significa todos os dispositivos |

## Casos de uso

O canal MaixCam permite que o Piconomous atue como backend de IA para dispositivos de borda:

- **Vigilância inteligente**: MaixCAM envia quadros de imagem; Piconomous os analisa usando modelos de visão
- **Controle IoT**: Dispositivos enviam dados de sensores; Piconomous coordena as respostas
- **IA offline**: Implante o Piconomous em uma rede local para inferência de baixa latência
