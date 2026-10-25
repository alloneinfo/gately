# Tailscale / frp / cloudflared

Estes três módulos integram os programas oficiais como processos gerenciados: reinício automático, logs reunidos na interface e configuração feita na interface. O Gately não embute esses programas, então você pode sempre usar a versão mais recente do projeto original.

## Local dos programas

Os executáveis são procurados nesta ordem:

1. O caminho definido nas configurações do módulo;
2. O `PATH`;
3. O subdiretório `bin/` do diretório de configuração.

Em implantações com Docker, coloque o binário no diretório de configuração montado, por exemplo `./gately-data/bin/frpc`. O Tailscale precisa de `tailscale` e `tailscaled`.

Tokens e chaves são armazenados com permissão 0600. O token do cloudflared é passado por variável de ambiente, então não aparece na lista de processos.

## Tailscale

- Gerencia o `tailscaled` (rede em espaço de usuário, adequada a contêineres) ou se conecta a um daemon já existente no sistema;
- Chave de autenticação ou link de login, com suporte a Headscale;
- Rotas de sub-rede e exit nodes;
- Lista de nós e proxy SOCKS5;
- Porta UDP fixa do WireGuard, facilitando liberar conexões diretas no firewall;
- Argumentos extras e variáveis de ambiente do tailscaled. Se o iptables do host usa o backend nf_tables, adicione `TS_DEBUG_FIREWALL_MODE=nftables` às variáveis de ambiente; sem isso, o encaminhamento de sub-rede não funciona.

## frp

- **Cliente frpc**: proxies tcp / udp / http / https / stcp / xtcp; adicionar ou remover um tem efeito imediato;
- **Servidor frps**: lista de portas permitidas, subdomínios, painel;
- **Endereços de escuta do frps**: "Endereço de escuta" define onde a porta de conexão do frpc escuta, e "Endereço de escuta de proxies", onde escutam as portas HTTP / HTTPS e as portas remotas dos proxies tcp / udp; vazios, escutam em todos os endereços. Para colocar o frps atrás de um proxy reverso como o Gately, use `127.0.0.1` como endereço de escuta, crie no Serviço Web um site com proxy reverso para `127.0.0.1:<porta de bind>` e escolha websocket ou wss como protocolo de transporte do frpc apontando para esse site; deixe o endereço de escuta de proxies vazio para que as portas remotas dos proxies tcp continuem acessíveis de fora. Ao importar uma configuração do frps sem `proxyBindAddr`, vale o `bindAddr`, como no próprio frps;
- **Importar configuração existente**: "Importar configuração", no cabeçalho da página, aceita `frpc.ini` / `frps.ini` antigos (incluindo a seção `[common]`) e `toml` do frp 0.52+. Você pode escolher um arquivo ou colar o conteúdo. Veja primeiro a prévia do resultado e dos campos ignorados e depois confirme a importação. No cliente, é possível acrescentar (proxies com o mesmo nome permanecem inalterados) ou substituir todos os proxies. A importação não altera o estado de ativação e também pode ser chamada pela API em `POST /api/frp/import`.

## Cloudflare Tunnel

- Túneis com token e túneis temporários trycloudflare;
- Vários túneis com reinício automático;
- Exibe o endereço público.

## Como escolher

| Cenário | Recomendação |
| --- | --- |
| Criar uma rede privada entre vários dispositivos | Tailscale |
| Você tem um servidor público e quer expor serviços internos por ele | frp |
| Sem IP público e sem servidor, e você quer expor rapidamente um serviço web | Cloudflare Tunnel |
| Sua operadora oferece um NAT que permite hole punching | [Travessia STUN](/pt-BR/docs/modules/stun) |
