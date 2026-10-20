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

## Aplicativo nativo para fnOS {#fnos}

Cada versão inclui pacotes offline para o fnOS: `gately_<versão>_fnos_x86.fpk` (Intel / AMD) e `gately_<versão>_fnos_arm.fpk` (ARM). Eles não estão na Central de aplicativos oficial, então instale-os manualmente:

1. Baixe o `.fpk` correspondente à arquitetura do seu dispositivo na [página de download](/pt-BR/download#fnos);
2. No fnOS, abra "Central de aplicativos → Instalação manual" (o botão no canto inferior esquerdo), escolha o arquivo e conclua o assistente;
3. Clique no ícone do Gately na área de trabalho do fnOS para abrir a interface de administração.

### Integrado ao gateway unificado do fnOS

O ícone da área de trabalho abre a interface de administração pelo gateway unificado do fnOS, em `/app/gately/` no endereço que você usa para o fnOS, compartilhando o mesmo ponto de entrada. Tanto o endereço da rede local quanto o seu domínio remoto do fnOS funcionam, sem precisar abrir ou redirecionar a porta 16680.

O acesso pelo gateway é controlado pelo fnOS, e somente administradores do fnOS podem abri-lo. A entrada segura e o "Acesso somente pela rede local" valem apenas para o acesso direto pela porta 16680. Na rede local, `http://<endereço do NAS>:16680` continua funcionando.

### Entrar com a sessão de administrador do fnOS

Ative "Entrar com o fnOS" em "Configurações → Controle de acesso" e os administradores que já estão conectados ao fnOS entram no Gately direto pelo ícone da área de trabalho, sem senha nem código de verificação em dois fatores:

- Somente administradores do fnOS são conectados, como administrador do Gately; os usuários comuns do fnOS continuam informando usuário e senha;
- Vale apenas para o acesso pelo gateway unificado; o acesso direto pela porta 16680 continua exigindo o login normal;
- Depois que você sai, o Gately não conecta você de novo silenciosamente; a página de login mostra um botão "Entrar como usuário do fnOS &lt;nome&gt;";
- A opção vem desativada por padrão e só aparece no pacote do fnOS.

### Observações

- O aplicativo roda como `root`, então pode escutar nas portas 80/443, gerenciar o Docker pelo `docker.sock` e abrir um terminal Web de verdade. Instale-o somente em um NAS que você controla.
- Na primeira inicialização, a senha inicial do `admin` é gravada em `/var/apps/gately/var/gately.log`: `grep -i "initial password" /var/apps/gately/var/gately.log`.
- A configuração fica no diretório `var` do aplicativo e é mantida na desinstalação, a menos que você escolha "excluir tudo" no assistente de desinstalação.
- O pacote também adiciona o comando `/usr/local/bin/gately` para usar `gately ctl` e `gately mcp`.
- A porta 16680 é a mesma da imagem Docker. Pare o contêiner `gately` antes de iniciar o aplicativo nativo.

## Primeiro acesso

A interface de administração escuta por padrão na porta `16680`. O usuário é `admin`, e a senha inicial é gravada no log na primeira inicialização. Em implantações automatizadas, você pode definir a senha inicial com a variável de ambiente `GATELY_INITIAL_PASSWORD`; nesse caso nenhuma senha aleatória é gerada.

No primeiro acesso aparece um assistente de configuração inicial que sugere, em sequência, alterar a senha, ativar a autenticação em dois fatores ou uma passkey, executar o autodiagnóstico de rede, importar do Lucky e configurar notificações.

## Atualização

O Gately não tem autoatualização online embutida; as atualizações dependem do seu método de implantação:

- Docker: execute `docker pull` da nova imagem e recrie o contêiner (mantenha o diretório de configuração montado);
- Binário: substitua o executável e reinicie o serviço;
- fnOS: instale manualmente o `.fpk` mais novo pela Central de aplicativos; a sua configuração é mantida.

Antes de atualizar, exporte um backup da configuração ou mantenha o diretório de configuração no git. As assinaturas dos arquivos e imagens baixados podem ser conferidas conforme [Verificar versões](/en/docs/guide/verify).
