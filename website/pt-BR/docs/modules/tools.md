# Arquivos e ferramentas de operação

## WebDAV

Configure em "WebDAV", no grupo "Armazenamento em rede" da barra lateral.

| Campo | Padrão | Descrição |
| --- | --- | --- |
| `enable` | Desativado | Ativa o serviço; ao ativar, pelo menos um usuário é necessário |
| `listenIp` | Vazio | Endereço de escuta; deixe em branco para escutar em todos os endereços |
| `port` | 18800 | Porta de escuta |
| `tls` | Desativado | Ativa o HTTPS; o certificado é fornecido por SNI pelo módulo de [Certificados SSL](/pt-BR/docs/modules/ssl) |
| `users` | Nenhum | Vários usuários. Cada um tem `name`, `password`, `root` (um caminho absoluto de diretório) e `readOnly`, ou usa `mounts` para montar vários diretórios |

Cada usuário pode ser mapeado para um diretório ou montar vários diretórios:

- **Diretório único**: defina `root` e `readOnly`. Após o login, a raiz é esse diretório.
- **Vários diretórios**: clique em "Montar vários diretórios" na interface ou preencha várias entradas em `mounts`. Cada entrada tem `name` (nome de exibição, não pode conter `/` e é único para cada usuário), `path` (um caminho absoluto) e `readOnly`. Após o login, a raiz lista os diretórios pelo nome. A própria raiz não é gravável, e arquivos não podem ser movidos entre diretórios diferentes.

Na migração do Lucky, a lista de montagens de um usuário é convertida em vários diretórios, e usuários que aparecem mais de uma vez no Lucky com o mesmo nome e a mesma senha são mesclados em um só.

## FTP

Configure em "Serviço FTP", no grupo "Armazenamento em rede". FTP e FTPS são compatíveis.

| Campo | Padrão | Descrição |
| --- | --- | --- |
| `enable` | Desativado | Ativa o serviço; ao ativar, pelo menos um usuário é necessário |
| `listenIp` | Vazio | Endereço de escuta; deixe em branco para escutar em todos os endereços |
| `port` | 2121 | Porta de escuta |
| `passiveStart` / `passiveEnd` | 0 | Intervalo de portas de dados no modo passivo. Quando ambos são 0, o sistema atribui portas aleatoriamente; quando definidos, deve valer 1 ≤ start ≤ end ≤ 65535, e o intervalo precisa ser liberado no firewall e nos mapeamentos de porta |
| `publicHost` | Vazio | IP anunciado externamente no modo passivo. Atrás de um NAT, informe o IP público ou encaminhado. Somente endereços IP são aceitos; deixe em branco para usar automaticamente o endereço da conexão |
| `tlsMode` | 0 | 0 permite texto simples e FTPS, 1 exige FTPS explícito (`AUTH TLS`), 2 é FTPS implícito (criptografado desde a conexão). Quando diferente de zero, o certificado é fornecido por SNI pelo módulo de [Certificados SSL](/pt-BR/docs/modules/ssl) |
| `idleTimeout` | 900 | Segundos de inatividade antes de a conexão ser fechada |
| `connectionTimeout` | 30 | Timeout em segundos para estabelecer uma conexão de dados |
| `disableActiveMode` | Desativado | Usa apenas o modo passivo, mais adequado a ambientes com NAT |
| `users` | Nenhum | Vários usuários, com os mesmos campos do WebDAV: `name`, `password`, `root`, `readOnly` ou `mounts`; os nomes de usuário devem ser únicos |

Os parâmetros do serviço e os usuários podem ser importados da configuração de FTP do Lucky (veja [Migrar do Lucky](/pt-BR/docs/guide/migration)).

## Armazenamento

Adicione montagens em "Armazenamento", no grupo "Armazenamento em rede", e gerencie os arquivos online na própria página.

| Campo | Descrição |
| --- | --- |
| `name`, `remark`, `enable` | Nome, observação e chave de ativação |
| `type` | `local` (diretório local) ou `webdav` (WebDAV remoto) |
| `path` | Diretório local, deve ser um caminho absoluto existente (usado por `local`) |
| `url`, `username`, `password` | Endereço do WebDAV remoto (começando com `http://` ou `https://`) e conta; a senha é mascarada na interface (usados por `webdav`) |
| `skipVerify` | Ignora a validação do certificado HTTPS (usado por `webdav`) |
| `readOnly` | Monta como somente leitura; o gerenciador de arquivos rejeita qualquer operação de escrita |

Diretórios locais e armazenamentos WebDAV podem ser importados do gerenciamento de armazenamento do Lucky. Aliyun Drive e montagens do sistema não são migrados.

O gerenciador de arquivos online permite navegar por diretórios, baixar arquivos, enviar em streaming (o arquivo inteiro não é carregado na memória), criar diretórios, renomear e excluir. Uma montagem pode servir como local de destino de uma tarefa de backup: os arquivos de backup são gravados no diretório indicado na montagem, e os arquivos antigos são removidos de acordo com a política de retenção.

## Wake-on-LAN

Gerencie dispositivos em "Wake-on-LAN" e envie pacotes mágicos a eles (6 bytes `0xFF` seguidos do MAC repetido 16 vezes).

| Campo | Padrão | Descrição |
| --- | --- | --- |
| `name` | Nenhum | Nome do dispositivo, obrigatório |
| `macs` | Nenhum | Endereços MAC, um por linha, pelo menos um; é possível informar vários para um mesmo dispositivo |
| `broadcastIps` | Vazio | Endereços de broadcast IPv4. Quando vazio, os pacotes são enviados a `255.255.255.255` e ao endereço de broadcast de cada interface de rede ativada |
| `port` | 9 | Porta UDP de destino |
| `repeat` | 3 | Número de vezes que cada MAC é enviado, até 20 |
| `checkAddr` | Vazio | Endereço de verificação de online no formato `host:port`, por exemplo `192.168.1.10:3389`; deixe em branco para não verificar |
| `remark` | Vazio | Observação |

Clique em Ligar na lista de dispositivos para enviar o pacote mágico, e a página mostra se o dispositivo está online de acordo com o `checkAddr`.

## Tarefas agendadas

Execute comandos ou envie requisições HTTP em um agendamento cron.

- `spec` aceita expressões padrão de 5 campos, um campo opcional de segundos (6 campos, como `*/30 * * * * *`) e descritores como `@every 10m` e `@daily`;
- Quando `type` é `command`, o `command` é executado por `sh -c` (um `.bat` temporário no Windows), e o ambiente recebe `taskName` mais uma variável em maiúsculas de mesmo nome com o prefixo `GATELY_`;
- Quando `type` é `http`, uma requisição é enviada de acordo com a configuração `http`, com os campos `url`, `method` (GET, POST, PUT, PATCH, DELETE), `headers` (cada um como `Key: Value`), `body`, `successContent` (o corpo da resposta precisa conter qualquer uma de suas entradas para contar como sucesso; quando vazio, apenas o status 2xx é verificado), `timeout` e `insecureSkipVerify`;
- As tarefas podem ser ativadas ou desativadas, a página mostra a hora da próxima execução, e uma tarefa pode ser executada imediatamente de forma manual;
- Se a execução anterior da mesma tarefa ainda não terminou, o novo disparo é ignorado;
- Cada tarefa guarda os seus últimos 20 registros de execução, incluindo hora de início, duração, sucesso ou falha, se foi disparada manualmente e a saída (um resumo de até 4000 caracteres);
- O formulário de criação valida e interpreta a expressão.

## Visão geral e logs do sistema

- A página "Visão geral" mostra informações do host: modelo, uso e carga da CPU, memória, disco, endereços das interfaces de rede e taxas em tempo real, IPv4 e IPv6 públicos (clique para copiar), o estado do serviço de cada módulo e, para o processo do Gately, versão, commit, data de compilação, hora de início, PID, CPU, memória residente, heap do Go e número de goroutines;
- A página "Logs do sistema" reúne os logs de todos os módulos, incluindo os registros de execução de módulos como Docker, FTP e Tarefas agendadas.

## Temas

A interface oferece vários temas: Esmeralda, Oceano, Fresco, Sakura, Pôr do sol, Escuro, Meia-noite, Punk e seguir o sistema (Escuro quando o sistema é escuro, caso contrário Esmeralda). A escolha do tema é armazenada localmente no navegador.
