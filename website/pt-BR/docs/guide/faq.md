# Perguntas frequentes

## Ele precisa se conectar aos servidores do projeto?

Não. Por padrão, o Gately não se conecta a nenhum servidor do projeto e não envia informações do dispositivo. Endereços externos, como a consulta do IP público, usam serviços públicos de terceiros e podem ser alterados na tarefa correspondente.

## Onde está a senha inicial?

Ela é gravada no log na primeira inicialização e exibida uma única vez. Em implantações com Docker, execute `docker logs gately 2>&1 | grep -i "initial password"`.

## Esqueci a senha do administrador. O que fazer?

Pare o Gately, apague o `account.json` do diretório de configuração e inicie de novo: o programa recria a conta `admin` e imprime uma nova senha inicial no log. Depois de entrar, revise as configurações de segurança, como a autenticação em dois fatores.

## Ele é compatível com o fnOS?

Sim. Cada versão inclui pacotes offline para o fnOS (`.fpk`), que você instala manualmente pela Central de aplicativos. O Gately se integra ao gateway unificado do fnOS, então abre pelo ícone na área de trabalho e também funciona pelo seu domínio remoto do fnOS. Ative "Entrar com o fnOS" em "Configurações → Controle de acesso" e os administradores que já estão conectados ao fnOS entram direto, sem senha nem código de verificação em dois fatores. Veja [Aplicativo nativo para fnOS](/pt-BR/docs/guide/install#fnos).

## Por que `--network host` é recomendado?

Redirecionamento de portas, STUN e serviços Web precisam escutar diretamente nas portas do host. Com a rede bridge, você mesmo precisa mapear cada porta usada.

## Posso executá-lo junto com o Lucky?

Sim. A porta de administração padrão é a 16680, que não conflita com a 16601 do Lucky. As regras importadas ficam desativadas por padrão, evitando disputa por portas e mapeamentos UPnP do roteador com um Lucky ainda em execução.

## Por que preciso fornecer os binários de Tailscale, frp e cloudflared?

Esses três módulos gerenciam programas externos oficiais, que o Gately não embute. Assim você sempre pode usar a versão mais recente do projeto original. Veja onde colocá-los em [Configuração e opções de execução](/pt-BR/docs/guide/configuration).

## Com a regra STUN ativa, o acesso externo continua sem funcionar?

Quando a porta WAN do roteador está atrás de um CGNAT da operadora (100.64.0.0/10), as portas abertas pela travessia não são acessíveis de fora; é uma limitação do próprio ambiente de rede. Use [Tailscale / frp / Cloudflare Tunnel](/pt-BR/docs/modules/tunnels).

## Como relatar problemas ou participar do desenvolvimento?

Abra uma issue em [GitHub Issues](https://github.com/wangwanjie/gately/issues). As convenções de desenvolvimento estão em [Como contribuir](/pt-BR/docs/guide/contributing).
