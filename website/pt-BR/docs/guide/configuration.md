# Configuração e opções de execução

## Parâmetros de linha de comando

| Parâmetro | Descrição |
| --- | --- |
| `-c` | Diretório de configuração; também pode ser definido pela variável de ambiente `GATELY_CONFIG_DIR` |
| `-listen` | Substitui o endereço de escuta da interface de administração, por exemplo `:16680` |
| `-version` | Exibe a versão |

## Diretório de configuração

Toda a configuração fica no diretório de configuração, com um arquivo de texto simples `<name>.json` por módulo, gravado de forma atômica com permissão 0600. Isso permite:

- Inspecionar tudo com qualquer editor de texto;
- Manter o diretório no git, auditar cada alteração com diffs e fazer rollback a qualquer momento;
- Fazer backup simplesmente copiando o diretório.

::: warning Atenção
O `ssl.json` contém chaves privadas de certificados. Antes de versioná-lo, confirme que o repositório é privado ou adicione o arquivo à lista de ignorados.
:::

## Diretório de programas externos

Os módulos Tailscale, frp e cloudflared gerenciam os programas externos oficiais. O Gately procura cada executável nesta ordem:

1. O caminho definido nas configurações do módulo;
2. O `PATH`;
3. O subdiretório `bin/` do diretório de configuração.

Em implantações com Docker, coloque os binários no diretório de configuração montado, por exemplo `./gately-data/bin/frpc`. O Tailscale precisa de `tailscale` e `tailscaled`. Veja os detalhes em [Tailscale / frp / cloudflared](/pt-BR/docs/modules/tunnels).

## Backup e restauração

Em "Configurações" você encontra o backup e a restauração da configuração, que também é o ponto de entrada para importar configurações do Lucky.
