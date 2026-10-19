# Gately

[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md) | [한국어](README.ko.md) | [Español](README.es.md) | [Français](README.fr.md) | [Deutsch](README.de.md) | [Русский](README.ru.md) | **Português (Brasil)**

Um conjunto de ferramentas de gateway de rede, de código aberto e auditável, para redes domésticas, NAS, hosts em nuvem e servidores em geral. DDNS, travessia STUN, redirecionamento de portas, proxy reverso, certificados, WAF e outros recursos comuns reunidos em um único arquivo de cerca de 22 MB, com importação de configurações existentes do Lucky.

![Visão geral do Gately](docs/images/overview.webp)

![DNS dinâmico](docs/images/ddns.webp)

![Docker](docs/images/docker.webp)

## Por que o Gately

O Gately é totalmente aberto, para que usuários e comunidade possam examinar diretamente o código-fonte e a configuração. Para um software que roda na borda da rede e guarda chaves de DNS e chaves privadas de certificados, o código aberto permite ver o que ele faz. Os compromissos do Gately:

- Código-fonte aberto sob Apache-2.0, sujeito à revisão da comunidade
- Por padrão, nenhuma conexão com servidores do projeto e nenhum envio de informações do dispositivo; o login de terceiros usa seus próprios aplicativos OAuth, sem intermediários
- Todos os endereços externos (consulta de IP público, servidores STUN etc.) são visíveis, editáveis e desativáveis
- A configuração é JSON em texto simples, que pode ser versionada no git para auditoria e rollback

## Recursos

| Módulo | Recursos |
| --- | --- |
| DNS dinâmico | Alibaba Cloud, Cloudflare, Tencent Cloud, DNSPod, Huawei Cloud, GoDaddy, Porkbun, dynv6, DuckDNS, Amazon Route 53, OVHcloud, Gandi, NS1, Bunny DNS, Namecheap, DynDNS2 (No-IP, Dynu, …), callback personalizado; obtenção do IP por URL/interface de rede/comando; IPv4/IPv6; Webhook |
| Travessia STUN | TCP/UDP, keepalive, UPnP/NAT-PMP, encaminhamento integrado, Webhook e script acionados quando o endereço muda |
| Redirecionamento de portas | TCP/UDP, IPv4/IPv6, intervalos de portas, failover entre vários destinos, limite de conexões, estatísticas de tráfego, protocolo PROXY v1/v2 |
| Certificados SSL | Emissão e renovação automáticas por ACME via DNS-01 (curinga) ou HTTP-01 (sem provedor de DNS), EAB, upload, leitura de arquivos, mapeamento para diretório e execução de script |
| Tailscale | Gerencia o tailscaled (rede em espaço de usuário, compatível com contêineres) ou usa um daemon já existente no sistema; chave de autenticação/link de login, Headscale, rotas de sub-rede, nó de saída, lista de nós e proxy SOCKS5, porta fixa, argumentos extras e variáveis de ambiente do tailscaled |
| frp | Cliente frpc (proxies tcp/udp/http/https/stcp/xtcp, alterações aplicadas na hora) e servidor frps (lista de portas permitidas, subdomínios, painel); importação em um clique de configurações no formato ini antigo e toml novo |
| Cloudflare Tunnel | Túneis cloudflared com token e túneis temporários trycloudflare, vários túneis, reinício automático, exibição do endereço público |
| Serviços Web | Proxy reverso/redirecionamento/arquivos estáticos/texto fixo por domínio; HTTPS e HTTP/3 (QUIC); WebSocket; envio e recebimento do protocolo PROXY; autenticação básica, autenticação web, login via GitHub/Google/QQ/OIDC; lista negra e lista branca de IPs, regras por caminho, links de compartilhamento temporários, bloqueio automático após falhas de autenticação, filtro de países por GeoIP; peso dos backends/backup/estratégias de balanceamento de carga, verificações de integridade ativas e passivas, compressão, limitação de taxa, cache de respostas, reescrita de caminhos; estatísticas de acesso por minuto e métricas do Prometheus |
| Firewall WAF | Coraza + OWASP CRS v4, regras ativáveis por arquivo, SecLang personalizado, bloquear/apenas detectar |
| Gerenciamento do Docker | Criação e edição de contêineres, logs e terminal, Compose, verificação de atualização de imagens e atualização em um clique (rollback automático em caso de falha), redes e volumes |
| Terminal Web | SSH (confiança na chave do host no primeiro acesso), Telnet, shell local (desativado por padrão); proxy, sessões persistentes, gerenciamento de arquivos por SFTP, 2FA e confirmação de protocolo, comandos rápidos, divisão de tela |
| Diagnóstico de rede | Ping, traceroute, teste de porta, consulta DNS, teste de velocidade, varredura da rede local |
| Central de notificações | Webhook, Telegram, Bark, ntfy, Gotify, Pushover, Discord, Slack, ServerChan, PushPlus, WeCom, DingTalk, Feishu, e-mail; níveis, regras com curingas e de silenciamento/desduplicação; modelos de regras integrados (falhas críticas, expiração de certificados, mudanças de endereço, logins, alterações de configuração) |
| Backup da configuração | Backups criptografados (scrypt + AES-GCM), política de retenção, backup agendado para um diretório local ou ponto de montagem de armazenamento |
| Linha de comando e MCP | `gately ctl` controla a instância em execução, `gately mcp` fornece um serviço MCP via stdio (somente leitura por padrão) |
| Wake-on-LAN | Broadcast do magic packet, detecção de status online |
| WebDAV | Vários usuários, diretórios independentes, somente leitura |
| FTP | Vários usuários, diretórios independentes, somente leitura, com suporte a FTPS e intervalo de portas passivas |
| Gerenciamento de armazenamento | Diretórios locais e montagens WebDAV remotas, gerenciamento de arquivos online |
| Tarefas agendadas | Expressões cron, comandos ou requisições HTTP, histórico de execuções |
| Segurança e configurações | Login com passkey (WebAuthn), autenticação em dois fatores (TOTP), contas somente leitura, limitação de falhas de login, entrada de segurança, acesso somente pela rede interna, HTTPS na interface de administração, vários tokens de API (autorização de leitura e escrita por módulo), DNS personalizado (com suporte a DoH), fuso horário, backup e restauração da configuração, endereço real do cliente vindo de proxies reversos locais confiáveis |
| Auditoria e histórico | Log de auditoria de logins, alterações de configuração e acessos a conteúdo sensível (nunca registra corpos de requisição nem segredos); cada arquivo de configuração sobrescrito mantém as versões anteriores, com diff e rollback em um clique; diretório de configuração declarativa (`-apply`) para GitOps |
| Monitoramento e autodiagnóstico | Endpoint global do Prometheus `/api/metrics` (certificados, DDNS, STUN, redirecionamento de portas, serviços Web) com painel do Grafana pronto para uso; autodiagnóstico de rede em um clique (IP público, CGNAT, relógio, portas 80/443, resolução DDNS, expiração de certificados) |
| Migração | Importa arquivos `.lkcf` do Lucky ou o zip de configuração de todo o site: redirecionamento de portas, DDNS, STUN, certificados, proxy reverso, WAF, Wake-on-LAN, terminal Web, WebDAV, configurações básicas |

## Início rápido

### Docker

A imagem suporta amd64, arm64 e arm/v7:

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep -i "initial password"
```

Abra `http://<IP do dispositivo>:16680`, entre com `admin` e a senha inicial exibida no log e, em seguida, altere a senha em "Configurações" e ative a autenticação em dois fatores. Um exemplo completo de Compose está em [deploy/docker-compose.yml](deploy/docker-compose.yml).

Redirecionamento de portas, STUN e serviços Web precisam escutar portas do host, por isso recomenda-se `--network host`. Se você não precisa do gerenciamento do Docker, pode remover a montagem de `docker.sock`.

### Binário

Baixe o pacote da sua plataforma em [Releases](https://github.com/wangwanjie/gately/releases) (Linux amd64/arm64/arm/mips/mipsle/riscv64, macOS, Windows, FreeBSD), extraia e execute:

```bash
./gately -c /etc/gately
```

Parâmetros: `-c` diretório de configuração (também pode ser definido pela variável de ambiente `GATELY_CONFIG_DIR`), `-listen` substitui o endereço de escuta da interface de administração, `-apply` aplica a cada inicialização os arquivos `<nome>.json` de um diretório ao diretório de configuração (`GATELY_APPLY_DIR`, útil para GitOps e Kubernetes; os arquivos de contas e tokens não podem ser sobrescritos), `-version` exibe a versão. Em instalações automatizadas, defina `GATELY_INITIAL_PASSWORD` para escolher a senha inicial do administrador em vez de ler uma senha aleatória no log.

Modelos de serviço: [systemd](deploy/systemd/gately.service) e [OpenWrt procd](deploy/openwrt/gately.init). Para verificar os pacotes e imagens baixados (assinaturas cosign, proveniência do build, SBOM), consulte [Verificação dos artefatos de release](docs/en/VERIFY.md).

### Monitoramento

Crie um token de API apenas com a permissão `metrics:read` e aponte o Prometheus para `/api/metrics`. As métricas, os exemplos de alertas e o painel do Grafana estão descritos em [Monitoramento](docs/en/MONITORING.md).

### Chamadas de API

Na página "Tokens de API" do painel de administração, crie um token (é possível criar vários, concedendo somente leitura ou leitura e escrita por módulo, com prazo de validade opcional; o valor em texto simples é exibido apenas uma vez após a criação) e seus scripts poderão chamar a API de administração em lote:

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks
```

Conta, autenticação em dois fatores, gerenciamento de tokens e as interfaces de backup, restauração, reinício e configurações do sistema só podem ser usadas com uma sessão de login. Todas as interfaces, permissões necessárias e estruturas de dados estão na página "Documentação da API" do painel de administração; também é possível exportá-las com `make api-docs` para [docs/api](docs/api/pt-BR) (`openapi.json` e `API.md`), que o site https://gately.cn referencia diretamente na publicação.

Ao atualizar de uma versão antiga, o token de API global anterior se torna automaticamente um token "Legado" (o valor não muda e tem permissão em todos os módulos). Recomenda-se restringir suas permissões na página "Tokens de API" ou substituí-lo por um novo token.

### Tailscale, frp, cloudflared

Esses três módulos gerenciam programas externos oficiais (reinício automático, logs reunidos na interface). O Gately não os inclui, o que permite usar sempre a versão mais recente do upstream. Os programas são procurados nesta ordem: o caminho informado nas configurações do módulo, `PATH` e o subdiretório `bin/` do diretório de configuração. Em implantações com Docker, basta colocar o binário no diretório de configuração montado, por exemplo `./gately-data/bin/frpc`; para o Tailscale, é preciso fornecer `tailscale` e `tailscaled`. Tokens e chaves nos arquivos de configuração são salvos com permissão 0600; o token do cloudflared é passado por variável de ambiente e não aparece na lista de processos.

### Migrar do Lucky

Consulte o [guia de migração](https://gately.cn/pt-BR/docs/guide/migration). Resumo dos passos: em "Configurações → Importar do Lucky", escolha o arquivo `lucky_*.lkcf` do diretório de configuração do Lucky, confira a prévia e importe. As regras importadas ficam desativadas por padrão; desative cada módulo no Lucky e ative-o no Gately, um de cada vez.

## Desenvolvimento

Dependências: Go 1.26+, Node.js 24+, pnpm.

```bash
make web      # compila o frontend (saída em web/dist, embutida via go:embed)
make build    # compila bin/gately
make test     # go test -race ./...
make lint     # gofmt, go vet, verificação de tipos do frontend
```

Para desenvolver o frontend, execute `cd web && pnpm dev`; `/api` é encaminhado por proxy para o `127.0.0.1:16680` local.

Alguns testes são ignorados por padrão e exigem condições adicionais:

| Teste | Como habilitar |
| --- | --- |
| ACME de ponta a ponta (Pebble) | Instale pebble e pebble-challtestsrv e defina `GATELY_PEBBLE_BIN` e `GATELY_PEBBLE_SRC` (executa automaticamente na CI) |
| Servidores STUN públicos reais | `GATELY_NET_TEST=1` |
| Validação da importação com configuração real do Lucky | `GATELY_LUCKY_SAMPLE` etc., veja `TestImportRealSample` em cada módulo |

A estrutura de diretórios e as escolhas tecnológicas estão em [docs/ROADMAP.md](docs/ROADMAP.md).

O código do site ([gately.cn](https://gately.cn)) fica em `website/` e usa VitePress: `cd website && pnpm install && pnpm dev`; `pnpm deploy:site` compila e sincroniza com o servidor.

## Diretrizes de contribuição

- A compatibilidade com o Lucky se baseia em comportamento, interfaces e formatos de configuração observáveis publicamente. Não envie código copiado de binários ou artefatos de frontend do Lucky.
- Novos módulos precisam de: estrutura de configuração e validação, início/parada em tempo de execução, interface REST, testes unitários; módulos migráveis do Lucky implementam `luckyimport.Importer`, e módulos que querem exibir seu status na página de visão geral implementam `module.Summarizer`.
- Os comentários de código são escritos em chinês; instruções de controle devem usar chaves.

## Comunidade

Dúvidas, feedback e dicas de uso são bem-vindos no grupo do Telegram: [t.me/gately_chat](https://t.me/gately_chat). Relate bugs no [GitHub Issues](https://github.com/wangwanjie/gately/issues).

## Histórico de estrelas

[![Star History Chart](https://api.star-history.com/svg?repos=wangwanjie/gately&type=Date)](https://star-history.com/#wangwanjie/gately&Date)

## Licença

[Apache License 2.0](LICENSE)
