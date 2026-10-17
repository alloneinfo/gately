# Serviços Web

Ofereça proxy reverso, redirecionamentos, arquivos estáticos e texto fixo por domínio. Este é o ponto de entrada web unificado do Gately. Local no menu: barra lateral "Web → Serviços Web".

## Estrutura: listeners, sites e o site padrão

A configuração tem três camadas:

1. Listener: uma porta de escuta. Os campos incluem a rede (`tcp` dual-stack, `tcp4`, `tcp6`), o endereço de escuta (vazio para todos os endereços), a porta, a chave de HTTPS, a versão mínima do TLS (1.0 / 1.1 / 1.2 / 1.3) e se o protocolo PROXY é aceito.
2. Site: um site dentro de um listener, correspondido por domínio. Os domínios podem ser um domínio exato, `*.example.com` (corresponde a subdomínios em qualquer profundidade) ou `*` (corresponde a tudo). A prioridade de correspondência é: domínio exato, curinga e depois `*`.
3. Site padrão: usado quando nenhum site corresponde. Quando não está ativado, um 404 é retornado.

## Tipos de site

| Tipo | Descrição |
| --- | --- |
| Proxy reverso | Encaminha para um ou mais backends. Os endereços dos backends usam o formato `http(s)://host:port`, e o WebSocket é compatível |
| Redirecionamento | Redireciona para uma URL completa com código de status 301 / 302 / 307 / 308 (padrão 302), opcionalmente mantendo o caminho e a query string originais |
| Arquivos estáticos | O diretório deve ser um caminho absoluto. Com a "Listagem de diretório" desativada, uma requisição a um diretório só retorna resposta se existir um `index.html` |
| Texto fixo | Conteúdo personalizado, `Content-Type` (padrão `text/plain; charset=utf-8`) e código de status (100-599, padrão 200) |

Outras opções dos sites de proxy reverso:

- Usar o Host do backend: o cabeçalho Host é alterado para o endereço do backend; quando desativado, o Host da requisição do cliente é mantido;
- Ignorar a validação do certificado do backend, desativar o Keep-Alive;
- Enviar o protocolo PROXY ao backend (cabeçalho v2, repassa o IP real do cliente, exige suporte do backend);
- Adicionar `X-Forwarded-For` / `X-Forwarded-Proto` / `X-Forwarded-Host`.

Sites de qualquer tipo podem definir um log de acesso, cross-origin (CORS), cabeçalhos personalizados de requisição e resposta (um `Key: Value` por linha; um Value vazio remove aquele cabeçalho) e podem referenciar uma instância do [Firewall WAF](/pt-BR/docs/modules/waf) (se a instância não existir ou estiver desativada, as requisições passam direto).

## Transporte e protocolos

- HTTPS: usa um certificado do módulo de [Certificados SSL](/pt-BR/docs/modules/ssl) que cubra o domínio;
- WebSocket: compatível automaticamente com o proxy reverso;
- Protocolo PROXY: um listener pode aceitar cabeçalhos PROXY enviados por componentes upstream (como haproxy e frp). Ele confia apenas nos IPs ou sub-redes listados em "Configurações globais → Origens PROXY confiáveis"; deixar vazio não confia em nenhuma origem. O proxy reverso pode enviar cabeçalhos PROXY ao backend.

## Autenticação

Configure na página "Segurança" do editor do site:

- Autenticação web: o navegador mostra uma página de login;
- Autenticação básica: HTTP Basic;
- Usuários: um `usuario:senha` por linha, e a senha pode ser um hash bcrypt. A autenticação básica exige pelo menos um usuário quando ativada; a autenticação web exige pelo menos um usuário ou identidade de login de terceiros quando ativada;
- Login de terceiros: GitHub, Google, QQ e OIDC são compatíveis, usando os seus próprios aplicativos OAuth. As identidades permitidas vão uma por linha no formato `provider:conta`, por exemplo `github:octocat` e `google:me@gmail.com`. Um `#` no fim de uma linha adiciona um comentário, e `github:*` significa qualquer conta daquele provedor.

Os provedores de login de terceiros são configurados em "Configurações globais": informe o seu próprio Client ID / Secret e ative o provedor, e defina também um endereço de callback (ele precisa ser atendido por um dos listeners do Gately, por exemplo `https://auth.example.com`). A URL de callback a registrar no GitHub / Google / QQ é:

```text
https://auth.example.com/__gately_auth/oauth/callback
```

O login de terceiros da própria interface de administração é configurado separadamente em "Configurações → Conta e segurança".

## Controle de acesso

### Lista de IPs permitidos e bloqueados

O "Controle de acesso por IP" na página "Segurança" do site pode ficar desativado, como lista de permissão ou como lista de bloqueio. A lista recebe um IP ou sub-rede por linha (como `192.168.1.0/24`).

### Regras de caminho

Em "Regras de caminho", na página "Segurança", substitua a autenticação e as configurações de IP do site por prefixo de caminho:

| Campo | Valores |
| --- | --- |
| `prefix` | Um prefixo de caminho começando com `/`, como `/admin` |
| `auth` | Herdar do site (vazio), `off` (sem autenticação), `on` (autenticação obrigatória) |
| `ipMode` / `ipList` | Herdar do site (vazio), `whitelist` ou `blacklist`, com uma lista de IPs ou sub-redes |

Regras de correspondência:

- O prefixo mais longo vence;
- A correspondência respeita os limites de segmentos do caminho: `/admin` corresponde a `/admin` e `/admin/x`, mas não a `/administrator`;
- O caminho da requisição é normalizado primeiro (`.`, `..` e barras duplicadas são removidos) e só depois correspondido, então `/public/../admin` é tratado como `/admin`;
- Só é possível configurar uma regra para o mesmo prefixo.

A opção "Escopo da autenticação" (`authOptional`): quando marcada, a autenticação se aplica apenas aos caminhos marcados como "Obrigatória" nas regras, e todos os outros caminhos são públicos. Essa opção exige que a autenticação básica ou web esteja ativada antes.

### Links de compartilhamento temporários

Depois que a autenticação básica ou web é ativada, você pode criar links de compartilhamento para um site, para que os visitantes o acessem sem conta:

| Campo | Descrição |
| --- | --- |
| Nome | Um rótulo, como "Para a Alice" |
| Expiração | Deixe em branco para não expirar |
| Usos `maxUses` | Número máximo de vezes que o link pode ser resgatado; 0 significa ilimitado. A contagem é zerada após uma reinicialização |
| Limite de caminho `pathPrefix` | Quando não vazio, o link só acessa caminhos sob esse prefixo; deve começar com `/` |

O visitante abre `https://domain/any-path?_gs=token` para resgatar uma sessão. Após o resgate bem-sucedido, o servidor define um cookie que dispensa login e redireciona para o endereço sem o parâmetro `_gs`. A sessão dura no máximo 24 horas e nunca além da expiração do link. Um token inválido é tratado como uma visita normal e não revela o motivo da falha. As sessões já emitidas deixam de valer imediatamente depois que o link é excluído. O token é gerado após salvar (com no mínimo 16 caracteres), e o link completo pode ser copiado na interface.

### Limitação de taxa de falhas de autenticação e banimento automático

- Após 10 autenticações com falha do mesmo IP de origem em 15 minutos, novas tentativas são rejeitadas por um tempo;
- Em "Configurações globais → Banimento automático → Banir por falha de autenticação", informe um número de minutos (0 a 43200); o IP de origem é banido ao atingir 10 falhas. 0 significa sem banimento, apenas limitação de taxa;
- Um evento `AuthBanned` é publicado quando um banimento automático é disparado, e pode ser enviado pela [Central de notificações](/pt-BR/docs/modules/notify);
- Endereços IPv6 são contados e banidos por sub-rede /64.

### Lista de banimentos

A lista de banimentos em "Configurações globais" mostra os IPs atualmente banidos, o motivo (falhas de autenticação repetidas ou banimento manual) e a hora de expiração, e cada entrada pode ser removida com "Desbanir". Você também pode banir manualmente: informe um IP e um número de horas (1 a 720, ou seja, no máximo 30 dias); a API usa 24 horas por padrão.

### Filtro de países por GeoIP

O "Filtro de países" de um site tem três modos: desativado, permitir apenas e negar. Informe códigos de país ISO de duas letras separados por vírgula, como `CN, HK, US`. Pelo menos um país é obrigatório quando ativado.

- A filtragem depende de um banco de dados GeoIP, que você precisa importar antes em "Configurações globais → Banco de dados GeoIP": envie um arquivo `.mmdb` ou informe uma URL de download. MaxMind GeoLite2, DB-IP e outros bancos de países ou cidades são compatíveis; o limite de envio é 129 MB;
- Quando nenhum banco de dados está carregado, a filtragem não tem efeito (todos os visitantes passam) e a interface mostra um aviso;
- Endereços de redes privadas são sempre permitidos;
- No modo "Negar", endereços cujo país não é encontrado no banco de dados são permitidos.

## Melhorias do proxy reverso

Essas opções ficam na página "Tráfego" do editor do site.

### Backends, peso e backup

A lista de backends recebe um endereço por linha, opcionalmente seguido de parâmetros:

```text
http://10.0.0.2:8080 weight=3
http://10.0.0.3:8080 backup
```

- `weight=N`: peso, um inteiro de 1 a 100, padrão 1;
- `backup`: um backend em espera, usado somente quando todos os backends que não são backup estão indisponíveis. Quando todos os backends estão indisponíveis, todos os que não são backup são tentados de novo.

### Política de balanceamento de carga

| Opção na interface | `lbPolicy` | Descrição |
| --- | --- | --- |
| Round robin (ponderado) | Vazio | Padrão |
| Aleatório (ponderado) | `random` | |
| Hash de IP (sessões fixas) | `ip_hash` | O mesmo IP de cliente sempre vai para o mesmo backend |
| Menos conexões | `least_conn` | Escolhe o menor valor de conexões / peso |

### Health checks

A chave de verificação ativa vem desativada por padrão. Campos e valores padrão depois de ativada:

| Campo | Padrão | Intervalo |
| --- | --- | --- |
| Modo `mode` | `http` | `http` (requisita o caminho de verificação; um código de status abaixo de 400 conta como saudável) ou `tcp` (apenas testa a conexão) |
| Caminho `path` | `/` | Começa com `/`, apenas no modo http |
| Intervalo `interval` | 10 segundos | 1 a 3600 |
| Timeout `timeout` | 3 segundos | 1 a 60 |
| Falhas consecutivas `fails` | 3 falhas o marcam como indisponível | 1 a 20 |
| Sucessos consecutivos `passes` | 2 sucessos o recuperam | 1 a 20 |

Quando o envio do protocolo PROXY ao backend está ativado, as verificações HTTP mudam automaticamente para TCP.

As verificações passivas estão sempre em vigor: entre vários backends, um backend com 3 falhas de conexão consecutivas é pausado por 30 segundos antes de ser sondado de novo (um backend único sem verificação ativa nunca é removido). Quando as verificações ativas estão ativadas, a recuperação é decidida inteiramente pelos resultados das verificações. Quando o estado de um backend muda, eventos de upstream fora do ar e upstream recuperado são publicados e podem ser enviados pela [Central de notificações](/pt-BR/docs/modules/notify).

### Compressão

A compressão pode ser ativada em sites de proxy reverso, de arquivos estáticos e de texto fixo (não em sites de redirecionamento). Com base no `Accept-Encoding`, `br` tem preferência, depois `gzip`.

- Somente conteúdo do tipo texto é comprimido: `text/*`, JSON, JavaScript, XML, XHTML, RSS / Atom, WebAssembly e SVG; `text/event-stream` é excluído;
- Respostas cujo `Content-Length` declarado é menor que 256 bytes ficam como estão;
- Respostas que já têm um `Content-Encoding` ficam como estão; requisições de upgrade para WebSocket passam direto;
- Após a compressão, ETags fortes são convertidos em ETags fracos.

### Limitação de taxa por IP

Limitação de taxa por token bucket para cada IP de cliente. Campos:

| Campo | Descrição |
| --- | --- |
| `rate` | Requisições permitidas por segundo |
| `burst` | Capacidade de rajada. Quando `rate` está definido e `burst` é 0, o padrão é `rate` arredondado para baixo mais 1 |
| `maxConns` | Máximo de requisições simultâneas por IP; 0 significa ilimitado |

Quando a limitação de taxa está ativada, pelo menos um entre `rate` e `maxConns` deve ser definido.

### Cache de respostas em memória

Aplica-se apenas a sites de proxy reverso. O cache fica em memória e usa remoção LRU.

| Campo | Padrão | Intervalo |
| --- | --- | --- |
| `ttl`, duração padrão | 0 | 0 a 30 dias (em segundos). Quando o upstream não declara nenhuma diretiva de cache, respostas 200 são mantidas em cache por esse tempo; 0 significa que somente respostas que o upstream permite explicitamente entram no cache |
| `maxMb`, capacidade do cache | 64 | 1 a 4096 MB |

Regras de cache:

- Somente requisições GET entram no cache, e a requisição não pode conter os cabeçalhos `Authorization`, `Cookie` ou `Range`, nem `Cache-Control: no-cache`;
- Somente respostas com status 200 entram no cache; uma resposta é ignorada se contiver `Set-Cookie`, `Vary: *` ou um `Cache-Control` com `no-store` / `private` / `no-cache`;
- O `s-maxage` é respeitado primeiro, depois o `max-age`;
- Uma única entrada é limitada a 4 MB e também a no máximo um quarto da capacidade total;
- Um acerto retorna os cabeçalhos `X-Cache: HIT` e `Age`, e uma falha retorna `X-Cache: MISS`;
- A chave do cache inclui o Host, a URI completa da requisição e o `Accept-Encoding`.

### Reescrita e redirecionamento de caminho

As regras são correspondidas em ordem, e a correspondência para na primeira ocorrência.

| Campo | Descrição |
| --- | --- |
| `pattern` | Uma expressão regular aplicada ao caminho da requisição, como `^/old/(.*)$` |
| `to` | O resultado da substituição; use `$1` para referenciar grupos |
| `redirect` | Desativado para reescrita interna (o caminho reescrito é enviado ao backend, e o destino deve começar com `/`); ativado para redirecionamento |
| `code` | Código de status do redirecionamento 301 / 302 / 307 / 308, padrão 302 |

Os redirecionamentos mantêm os parâmetros de query originais.

## Estatísticas de acesso e monitoramento

### Estatísticas do site

Para um listener em execução, clique em "Estatísticas" na linha de um site para abrir a janela de estatísticas, onde você pode escolher a última 1 hora, 6 horas ou 24 horas:

- Estatísticas de acesso por minuto: um ponto por minuto, mantidos pelas últimas 24 horas, incluindo uma curva de contagem de requisições e curvas de erros 4xx / 5xx;
- Distribuição de códigos de status (2xx / 3xx / 4xx / 5xx) e tráfego acumulado de respostas;
- Top 10 de IPs de origem e top 10 de países (as estatísticas de países exigem um banco de dados GeoIP importado);
- Sites de proxy reverso mostram o estado de saúde, o peso, a marca de backup e o número atual de conexões de cada backend.

As estatísticas ficam em memória e voltam a contar do zero depois de uma reinicialização ou de uma modificação do listener. Os rankings acompanham no máximo 5000 origens distintas.

### Métricas Prometheus

O endpoint de métricas é `GET /api/webservice/metrics`, em formato de texto Prometheus, e exige credenciais de login ou um [token de API](/en/docs/reference/api-reference).

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/webservice/metrics
```

| Métrica | Tipo | Descrição |
| --- | --- | --- |
| `gately_site_requests_total` | counter | Requisições acumuladas de um site |
| `gately_site_errors_total` | counter | Erros de backend de um site |
| `gately_site_active_requests` | gauge | Requisições em andamento |
| `gately_site_response_bytes_total` | counter | Bytes de resposta acumulados |
| `gately_site_responses_total` | counter | Respostas contadas por classe de status (label `class`: `2xx` a `5xx`) |
| `gately_upstream_up` | gauge | Se um backend está saudável: 1 saudável, 0 indisponível (com a label `upstream`) |

Todas as métricas têm as labels `listener` (nome do listener#chave do listener) e `site` (nome do site, ou o primeiro domínio quando não há nome).

## Dica de implantação

Os Serviços Web precisam escutar em portas do host como 80 / 443. Em implantações com Docker, recomendamos `--network host`.
