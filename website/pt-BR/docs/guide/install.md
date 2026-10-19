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

### systemd

O repositório traz um arquivo de serviço completo, [deploy/systemd/gately.service](https://github.com/wangwanjie/gately/blob/main/deploy/systemd/gately.service), que já concede as capabilities necessárias para escutar em portas baixas:

```bash
sudo install -m 0755 gately /usr/local/bin/gately
sudo install -m 0644 gately.service /etc/systemd/system/gately.service
sudo systemctl daemon-reload && sudo systemctl enable --now gately
journalctl -u gately | grep -i "initial password"
```

### OpenWrt

Use o script de inicialização procd [deploy/openwrt/gately.init](https://github.com/wangwanjie/gately/blob/main/deploy/openwrt/gately.init):

```bash
cp gately /usr/bin/gately && chmod +x /usr/bin/gately
cp gately.init /etc/init.d/gately && chmod +x /etc/init.d/gately
/etc/init.d/gately enable && /etc/init.d/gately start
```

## Primeiro acesso

A interface de administração escuta por padrão na porta `16680`. O usuário é `admin`, e a senha inicial é gravada no log na primeira inicialização. Em implantações automatizadas, você pode definir a senha inicial com a variável de ambiente `GATELY_INITIAL_PASSWORD`; nesse caso nenhuma senha aleatória é gerada.

No primeiro acesso aparece um assistente de configuração inicial que sugere, em sequência, alterar a senha, ativar a autenticação em dois fatores ou uma passkey, executar o autodiagnóstico de rede, importar do Lucky e configurar notificações.

## Atualização

O Gately não tem autoatualização online embutida; as atualizações dependem do seu método de implantação:

- Docker: execute `docker pull` da nova imagem e recrie o contêiner (mantenha o diretório de configuração montado);
- Binário: substitua o executável e reinicie o serviço.

Antes de atualizar, exporte um backup da configuração ou mantenha o diretório de configuração no git. As assinaturas dos arquivos e imagens baixados podem ser conferidas conforme [Verificar versões](/en/docs/guide/verify).
