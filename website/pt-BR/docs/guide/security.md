# Modelo de segurança e compromisso de privacidade

O Gately roda na borda da rede e lida com chaves de provedores de DNS, chaves privadas de certificados e outras credenciais, por isso segurança e transparência são os seus principais objetivos de design.

## Compromisso de privacidade

- **Zero conexões externas por padrão**: nenhuma conexão com servidores do projeto, nenhum envio de informações do dispositivo, nenhum envio remoto de configuração.
- **Endereços externos visíveis e editáveis**: a consulta do IP público, os servidores STUN e endereços semelhantes aparecem na interface e podem ser alterados ou desativados.
- **Login de terceiros sob o seu controle**: o login com GitHub, Google, QQ e OIDC usa aplicativos OAuth que você mesmo registra, sem intermediários.
- **Atualizações sob o seu controle**: o programa nunca substitui o próprio binário em segundo plano; você decide quando atualizar.

## Proteção de dados

- A configuração é JSON em texto simples com permissão 0600, podendo ser auditada e revertida.
- Campos sensíveis são mascarados na saída da API.
- O token do cloudflared é passado por variável de ambiente e não aparece na lista de processos; tokens e chaves do frp e do Tailscale são armazenados com permissão 0600.

## Autenticação e controle de acesso

- Hash de senhas com bcrypt e tokens de sessão assinados com HMAC;
- Autenticação em dois fatores (TOTP, verificada com os vetores de teste da RFC 6238);
- Limitação de tentativas de login;
- Caminho de entrada seguro, acesso apenas pela rede local e HTTPS na interface de administração;
- API com vários tokens: cada token tem acesso de leitura e escrita apenas aos módulos selecionados na criação, pode ter data de expiração e registra o IP de origem do último uso. Os endpoints de conta, autenticação em dois fatores, gerenciamento de tokens, backup, restauração, reinício e configurações do sistema aceitam apenas sessões com login. Veja [Tokens de API e OpenAPI](/pt-BR/docs/reference/api);
- Controle de acesso por site: banimento automático após falhas de autenticação, filtro de países por GeoIP e regras de caminho. Veja [Serviços Web](/pt-BR/docs/modules/web);
- Os backups da configuração aceitam criptografia (scrypt + AES-GCM). Veja [Backup da configuração](/pt-BR/docs/modules/backup).

## Padrões dos recursos de alto risco

O terminal de shell local fica desativado por padrão. O terminal SSH usa a política trust-on-first-use para registrar as impressões digitais das chaves de host.

## Relatar problemas de segurança

Relate vulnerabilidades de forma privada pela página Security do repositório e evite divulgar detalhes em issues públicas: <https://github.com/wangwanjie/gately/security>

Para mais recomendações práticas, veja o [Guia de reforço de segurança](/pt-BR/docs/guide/hardening).
