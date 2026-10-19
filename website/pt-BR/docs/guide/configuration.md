# Configuração e opções de execução

## Parâmetros de linha de comando

| Parâmetro | Descrição |
| --- | --- |
| `-c` | Diretório de configuração; também pode ser definido pela variável de ambiente `GATELY_CONFIG_DIR` |
| `-listen` | Substitui o endereço de escuta da interface de administração, por exemplo `:16680` |
| `-apply` | Diretório de configuração declarativa; também pode ser definido pela variável de ambiente `GATELY_APPLY_DIR`, veja abaixo |
| `-version` | Exibe a versão |

| Variável de ambiente | Descrição |
| --- | --- |
| `GATELY_INITIAL_PASSWORD` | Senha inicial do administrador na primeira inicialização (no mínimo 8 caracteres); sem efeito se a conta já existir |

## Diretório de configuração

Toda a configuração fica no diretório de configuração, com um arquivo de texto simples `<name>.json` por módulo, gravado de forma atômica com permissão 0600. Isso permite:

- Inspecionar tudo com qualquer editor de texto;
- Manter o diretório no git, auditar cada alteração com diffs e fazer rollback a qualquer momento;
- Fazer backup simplesmente copiando o diretório.

::: warning Atenção
O `ssl.json` contém chaves privadas de certificados. Antes de versioná-lo, confirme que o repositório é privado ou adicione o arquivo à lista de ignorados.
:::

## Configuração declarativa

Com `-apply <diretório>`, a cada inicialização os arquivos `<nome>.json` desse diretório são copiados por cima dos do diretório de configuração; arquivos com conteúdo idêntico não são reescritos. Isso é útil para distribuir a configuração de forma centralizada a partir de um repositório git, de um ConfigMap do Kubernetes ou de scripts de implantação.

- Antes de gravar, verifica-se se cada arquivo é JSON válido; se algum for inválido, a inicialização é recusada;
- `account.json` e `tokens.json` não podem ser sobrescritos: contas e tokens só são gerenciados na interface de administração;
- O conteúdo antigo sobrescrito fica salvo no [histórico de configuração](/pt-BR/docs/guide/audit) e pode ser revertido a qualquer momento.

## Diretório de programas externos

Os módulos Tailscale, frp e cloudflared gerenciam os programas externos oficiais. O Gately procura cada executável nesta ordem:

1. O caminho definido nas configurações do módulo;
2. O `PATH`;
3. O subdiretório `bin/` do diretório de configuração.

Em implantações com Docker, coloque os binários no diretório de configuração montado, por exemplo `./gately-data/bin/frpc`. O Tailscale precisa de `tailscale` e `tailscaled`. Veja os detalhes em [Tailscale / frp / cloudflared](/pt-BR/docs/modules/tunnels).

## Backup e restauração

Em "Configurações" você encontra o backup e a restauração da configuração, que também é o ponto de entrada para importar configurações do Lucky.
