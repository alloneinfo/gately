# Convenções da API do Terminal Web

Todos os endpoints ficam em `/api/webterminal/` e exigem login. Os erros são sempre retornados como `{"error": "..."}`.

## 1. Estrutura da configuração (o arquivo webterminal em `conf`)

```jsonc
{
  "settings": {
    "allowLocalShell": false,
    "maxSessions": 20,
    "idleTimeout": 0,          // minutos; uma sessão sem entrada ou saída por esse tempo é encerrada automaticamente; 0 significa sem limite
    "bufferSize": 256,         // KB; buffer de saída mantido por sessão e reproduzido na reconexão
    "heartbeatInterval": 30,   // segundos; heartbeat do WebSocket (ping do servidor / ping do frontend)
    "sessionKeepAlive": 0      // minutos; por quanto tempo uma sessão é mantida após o navegador desconectar; 0 encerra imediatamente na desconexão
  },
  "connections": [Connection],
  "shortcuts": [{ "name": "Show disk usage", "command": "df -h", "sendEnter": true }],
  "splitLayout": null,         // layout de painéis divididos personalizado pelo frontend; o backend armazena e devolve como está (json.RawMessage)
  "security": {
    "agreementAccepted": false,
    "twoFAEnable": false,
    "twoFAKey": ""             // segredo TOTP; mascarado na saída da API
  },
  "credentials": [Credential]  // predefinições de credenciais SSH
}
```

Connection (`type` é `ssh` / `telnet` / `local`):

```jsonc
{
  "key": "", "name": "", "type": "ssh", "remark": "", "enable": true,
  "quickAccessDirs": ["/etc", "/var/log"],      // diretórios de acesso rápido do SFTP
  "ssh": { "host", "port", "username", "authType": "password|key|credential",
           "credentialKey": "",  // chave da predefinição de credencial quando authType=credential; username pode ficar vazio e recorre ao padrão da predefinição
           "password", "privateKey",
           "keyFile": "",        // caminho de um arquivo de chave privada no próprio servidor; usado quando authType=key e privateKey está vazio
           "passphrase", "hostKey", "keepAlive", "workDir": "", "timeout": 15,
           "proxy": Proxy },
  "telnet": { "host", "port": 23, "username", "password", "timeout": 15, "proxy": Proxy },
  "local": { "shell", "workDir", "env": {"K":"V"} }
}
Proxy = { "enabled": false, "type": "socks5|http", "host", "port", "username", "password" }
```

Credential (predefinição de credencial, pode ser referenciada por várias conexões SSH):

```jsonc
{ "key": "", "name": "", "authType": "password|key", "username": "", // opcional, usado quando a conexão não tem username
  "password", "privateKey", "keyFile", "passphrase", "remark": "", "enable": true }
```

Senhas, chaves privadas, frases secretas de chaves privadas e senhas de proxy são mascaradas com `crud.Mask` na saída e preservadas com `crud.KeepSecret` na atualização.

## 2. Gerenciamento de conexões

| Método | Caminho | Descrição |
|---|---|---|
| GET/POST | `/connections` | Lista / cria |
| PUT/DELETE | `/connections/{key}` | Atualiza / exclui |
| PUT | `/connections-order` | Reordena `{"keys":[...]}` |
| POST | `/connections/{key}/test` | Testa a conectividade SSH / Telnet |
| POST | `/connections/test` | Testa uma conexão não salva (o corpo é uma Connection; os segredos salvos são mesclados apontando o campo `key` para a conexão original) |
| DELETE | `/connections/{key}/hostkey` | Redefine a impressão digital do host |
| PUT | `/connections/{key}/quickaccess` | Corpo `{"dirs":["/a","/b"]}` |
| GET/POST | `/credentials` | Lista / cria predefinições de credenciais |
| PUT/DELETE | `/credentials/{key}` | Atualiza / exclui (retorna 409 com os nomes das conexões enquanto ainda houver referências) |
| PUT | `/credentials-order` | Reordena as predefinições `{"keys":[...]}` |
| GET | `/shells` | Lista de shells disponíveis nesta máquina `["/bin/bash", ...]` (lê /etc/shells e filtra os que existem) |

Observações adicionais (a implementação é a referência):
- `POST /connections/test` e `POST /connections/{key}/test` retornam 200 com `{ok, error?, serverVersion?, newHostKey?, latencyMs}`; o tipo `local` retorna 400. No Telnet, só a acessibilidade TCP (inclusive pelo proxy) é verificada.
- `PUT /connections/{key}/quickaccess` retorna `{ok, dirs}` (espaços em branco e duplicatas removidos). Ao editar uma conexão sem `quickAccessDirs`, o valor antigo é mantido.
- `ssh.timeout` / `telnet.timeout` aceitam de 1 a 120 segundos, com padrão 15. `ssh.workDir` é aplicado com `cd` após o login.
- Conexões com `authType=credential` têm a senha, a chave privada, o arquivo de chave e a frase secreta inline apagados ao salvar; as credenciais são obtidas da predefinição no momento da conexão. Atualizar uma predefinição descarta as conexões SSH do pool que a referenciam.
- O preenchimento automático do login Telnet tenta uma vez cada, e só quando prompts como `login:` / `username:` / `password:` são detectados.

## 3. Sessões de terminal (sessões persistentes que podem reconectar após uma desconexão)

- `GET /connections/{key}/ws?cols=&rows=`: cria uma nova sessão e faz o upgrade para WebSocket.
- `GET /sessions/{id}/ws?cols=&rows=`: anexa a uma sessão existente, reproduz primeiro o buffer e depois encaminha em tempo real. Vários clientes podem se anexar à mesma sessão ao mesmo tempo (a entrada é compartilhada, a saída é transmitida a todos).

Protocolo WebSocket:
- Servidor → navegador: **frames binários** levam a saída do terminal; **frames de texto** levam JSON:
  `{"t":"s","d":"<sessionId>"}` sessão estabelecida (enviado uma vez tanto após criar quanto após anexar);
  `{"t":"n","d":"notice"}` aviso; `{"t":"x","d":"reason"}` sessão encerrada; `{"t":"p"}` resposta de heartbeat.
- Navegador → servidor: JSON em texto: `{"t":"i","d":"input"}`, `{"t":"r","c":cols,"r":rows}`, `{"t":"p"}` heartbeat.
- Quando o WebSocket do navegador desconecta: se `sessionKeepAlive > 0`, a sessão é mantida (desanexada) e fechada após o tempo limite se ninguém se anexar; se for 0, é fechada imediatamente.
- Quando o processo da sessão termina sozinho, uma mensagem `x` é enviada, o WebSocket é fechado e a sessão é removida da tabela de sessões.

- Criar uma sessão além de `maxSessions` retorna HTTP 429; uma falha de conexão é informada por um frame de aviso `n`, e depois o WebSocket é fechado.
- `bytesIn` é o número de bytes de entrada do navegador, `bytesOut` é o número de bytes de saída do terminal e `attached` é o número de clientes anexados no momento (um número); os campos de tempo são RFC3339.

REST:

| Método | Caminho | Descrição |
|---|---|---|
| GET | `/sessions` | `[{id, connKey, connName, type, remark, createdAt, lastActive, attached, cols, rows, bytesIn, bytesOut}]` |
| GET | `/sessions/{id}` | Uma única sessão |
| PUT | `/sessions/{id}/remark` | `{"remark":""}` |
| GET | `/sessions/{id}/stats` | `{bytesIn, bytesOut, durationSec, idleSec, attached, cols, rows}` |
| DELETE | `/sessions/{id}` | Encerra a sessão |

## 4. Configurações, atalhos e layout dividido

| Método | Caminho | Descrição |
|---|---|---|
| GET/PUT | `/settings` | `{settings, activeSessions}` / o corpo é Settings |
| GET/PUT | `/globalshortcuts` | Array de comandos de atalho |
| GET/PUT/DELETE | `/splitlayout` | Layout em JSON arbitrário |

## 5. Segurança (termo de uso + verificação em dois fatores)

| Método | Caminho | Descrição |
|---|---|---|
| GET | `/security` | `{agreementAccepted, twoFAEnable, verified}`; `verified` indica que o token de login atual passou pela verificação em dois fatores |
| POST | `/security/agreement` | Aceita o termo |
| POST | `/security/2fa/setup` | Gera um segredo pendente `{secret, url}` (não persistido) |
| POST | `/security/2fa/enable` | `{secret, code}` ativa o 2FA quando a verificação passa |
| POST | `/security/2fa/disable` | `{code}` desativa o 2FA quando a verificação passa |
| POST | `/security/verify2fa` | `{code}`; quando passa, o token de login atual conta como verificado por 30 minutos |
| GET | `/security/check2fa` | `{need: bool}`: se a verificação ainda é necessária |

Após 5 códigos 2FA errados seguidos, o token de login fica bloqueado por 1 minuto (HTTP 429). `2fa/setup` retorna 400 quando o 2FA já está ativado (desative-o antes). Após um `2fa/enable` bem-sucedido, o token atual conta como verificado.
Bloqueio: ativar o shell local, criar uma sessão, anexar a uma sessão e todos os endpoints SFTP retornam 403 `{"error":"...","needAgreement":true}` quando o termo não foi aceito,
e retornam 403 `{"error":"...","need2fa":true}` quando o 2FA está ativado e o token atual não foi verificado. "Ativar o shell local" aqui significa um `PUT /settings` que muda `allowLocalShell` de false para true. No lado Go, todos esses casos chamam `m.requireAccess(w, r) bool`.

## 6. Gerenciamento de arquivos via SFTP

O parâmetro de caminho `{key}` é a **chave da conexão**. Conexões SSH usam SFTP (cada requisição abre um canal do subsistema SFTP em uma conexão SSH reutilizada); conexões `local` operam diretamente no sistema de arquivos do próprio servidor (pelos mesmos endpoints).
Telnet e outros tipos retornam 400; uma conexão inexistente retorna 404. O backend reutiliza uma conexão SSH por conexão e a fecha após 5 minutos ociosa (`m.acquireSSH`).
Uma conexão `local` exige que "Permitir shell local" esteja ativado nas configurações, caso contrário retorna 403 (as permissões de arquivos locais equivalem a um shell local). Uma falha de conexão SSH retorna 502.

Convenções gerais:
- Todo `path` / `from` / `to` / `dest` precisa ser um caminho absoluto começando com `/` (quando o `path` de `list` está vazio, usa-se o diretório padrão: o diretório de login no SSH e o workDir da conexão ou o diretório home do usuário no local). Caminhos relativos retornam 400.
- Os erros são sempre `{"error":"message"}`: não encontrado 404, permissão negada 403, grande demais 413, tipo não suportado 415, destino já existe 409 (adicionado nesta seção), parâmetros inválidos 400, todo o resto 500.
- Endpoints de escrita bem-sucedidos retornam `{"ok":true, ...}`.

| Método | Caminho | Descrição |
|---|---|---|
| GET | `/sftp/{key}/list?path=` | `{path, parent, entries:[{name, path, isDir, isLink, size, mode, modeStr, modTime, owner?}]}`; diretórios primeiro, depois ordenados por nome sem diferenciar maiúsculas e minúsculas. `mode` é uma string de permissão octal como `"755"`, `modeStr` segue a forma `drwxr-xr-x`, `modTime` é RFC3339 e `owner` é o nome do usuário no local e o uid numérico no SSH. Links simbólicos têm `isLink=true`, e `isDir=true` quando apontam para um diretório |
| GET | `/sftp/{key}/read?path=&download=1` | Sem download: pré-visualização de texto, retorna `{path, name, size, modTime, content}`; acima de 1MB retorna 413, e conteúdo binário (com NUL ou fora do UTF-8) retorna 415. Com download=1: download em streaming (`application/octet-stream` + `Content-Disposition: attachment`), com suporte a `Range` e `If-Modified-Since` |
| POST | `/sftp/{key}/write` | `{path, content}` salva texto (sobrescreve; um destino que é diretório retorna 400) → `{ok, path, size}` |
| POST | `/sftp/{key}/upload` | Gravação multipart em streaming: `path` (o diretório de destino, que já deve existir e **deve vir antes dos arquivos**), `files` (pode se repetir) e o opcional `relativePaths` (um por arquivo; associado na ordem de aparição aos arquivos seguintes, e deve vir antes dos arquivos correspondentes; uma string com array JSON também é aceita). Sem `relativePaths`, o nome do arquivo é usado como está (pode trazer um subcaminho quando o navegador envia um diretório). Arquivos com o mesmo nome são sobrescritos e diretórios intermediários ausentes são criados automaticamente; caminhos com `..` retornam 400, garantindo que nada seja gravado fora do diretório de destino → `{ok, count, files:[caminhos absolutos]}` |
| POST | `/sftp/{key}/mkdir` | `{path}` cria um diretório (incluindo os pais ausentes); 409 se já existir |
| POST | `/sftp/{key}/touch` | `{path}` cria um arquivo vazio (incluindo os pais ausentes); 409 se já existir |
| POST | `/sftp/{key}/rename` | `{from, to}`; 409 se `to` já existir, 400 ao mover um diretório para dentro dele mesmo, 403 ao renomear o diretório raiz; no local entre dispositivos diferentes, recorre automaticamente a copiar e depois excluir |
| POST | `/sftp/{key}/copy` | `{from:[...], to}`; `to` é o **diretório** de destino, e cada item é copiado para `to/nome-original` (recursivo; links simbólicos são copiados como links; arquivos especiais como dispositivos e pipes são ignorados). Em caso de conflito de nome, a cópia é **renomeada automaticamente** para `name copy.ext`, `name copy 2.ext` e assim por diante, sem erro; copiar um diretório para dentro dele mesmo retorna 400 → `{ok, paths:[novos caminhos]}` |
| POST | `/sftp/{key}/chmod` | `{path, mode:"755", recursive}`; `mode` é uma string octal de 0 a 7777; a recursão não segue links simbólicos, e arquivos e diretórios recebem a mesma permissão |
| DELETE | `/sftp/{key}/remove` | `{paths:[...]}` (corpo JSON; exclusão recursiva; em links simbólicos só o link é removido); se algum caminho for `/`, retorna 403 e nada é excluído → `{ok, count}` |
| POST | `/sftp/{key}/compress` | `{paths:[...], dest, format:"zip|tar.gz"}` (format tem padrão zip; `tgz` é tratado como tar.gz). `dest` é o caminho completo do arquivo compactado de saída; se `dest` for um diretório existente ou terminar com `/`, o arquivo recebe um nome automático dentro dele (o nome do próprio item para um único item, ou o nome do diretório pai para vários itens, mais `.zip` / `.tar.gz`). Cada item vira uma entrada de nível superior do arquivo compactado com o próprio nome; itens com nomes duplicados retornam 400, um arquivo de saída existente retorna 409, e o arquivo de saída nunca é incluído nele mesmo → `{ok, path}` |
| POST | `/sftp/{key}/decompress` | `{path, dest}` suporta zip / tar / tar.gz / tgz (detectado pela extensão; qualquer outro retorna 415). Quando `dest` está vazio, extrai no diretório que contém o arquivo compactado, criando-o se necessário; arquivos com o mesmo nome são sobrescritos. Um arquivo compactado com qualquer entrada `..` é rejeitado por inteiro (400, proteção contra zip-slip); entradas que não são arquivos comuns, como links simbólicos e hard links, são ignoradas; as permissões dos arquivos mantêm só os bits rwx → `{ok, dest, count}` |
| GET | `/sftp/{key}/preview-archive?path=` | Lista de entradas do arquivo compactado `[{name,size,isDir}]` (no máximo 10000 entradas; `name` não tem `/` no final) |

Compressão e descompressão são feitas inteiramente em Go pela interface do sistema de arquivos (os no local, leituras e gravações SFTP no SSH), sem depender de comandos remotos.

## 7. Outros

- Logs: os logs do módulo já aparecem na página global de logs, então não é preciso um endpoint separado.
- Importação do Lucky: preenche telnet, proxy, keyFile, workDir, timeout, env, quickAccessDirs, sessionConfig, globalShortcuts
  e securityConfig (o termo de uso e o segredo 2FA são migrados juntos).
