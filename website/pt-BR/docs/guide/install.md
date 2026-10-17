# Instalação

## Docker

Imagem: `ghcr.io/wangwanjie/gately`, disponível para amd64, arm64 e arm/v7.

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
```

### Docker Compose

```yaml
services:
  gately:
    image: ghcr.io/wangwanjie/gately:latest
    container_name: gately
    restart: unless-stopped
    # Redirecionamento de portas, travessia STUN e serviços Web escutam diretamente nas portas do host
    network_mode: host
    volumes:
      # Diretório de configuração (JSON em texto simples, pode ficar no git; o ssl.json contém chaves privadas de certificados)
      - ./data:/etc/gately
      # Gerenciamento do Docker (remova se não precisar)
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Asia/Shanghai
```

A imagem inclui `sh`, `bash` e `curl`, então os scripts de notificação mais comuns funcionam sem ajustes.

## Binário

1. Obtenha o pacote da sua plataforma na [página de download](/pt-BR/download): Linux (amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows ou FreeBSD.
2. Extraia e execute:

```bash
./gately -c /etc/gately
```

### Exemplo de systemd

```ini
[Unit]
Description=Gately
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/gately -c /etc/gately
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Primeiro acesso

A interface de administração escuta por padrão na porta `16680`. O usuário é `admin`, e a senha inicial é gravada no log na primeira inicialização. Depois do primeiro acesso, altere a senha e ative a autenticação em dois fatores.

## Atualização

O Gately não tem autoatualização online embutida; as atualizações dependem do seu método de implantação:

- Docker: execute `docker pull` da nova imagem e recrie o contêiner (mantenha o diretório de configuração montado);
- Binário: substitua o executável e reinicie o serviço.

Antes de atualizar, exporte um backup da configuração ou mantenha o diretório de configuração no git.
