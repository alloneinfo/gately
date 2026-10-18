# Como contribuir

O Gately é um projeto open source mantido pela comunidade. Issues e pull requests são bem-vindos. Repositório: <https://github.com/wangwanjie/gately>

## Desenvolvimento local

Requisitos: Go 1.26+, Node.js 24+, pnpm.

```bash
make web      # compila o frontend (saída em web/dist, embutido via go:embed)
make build    # compila bin/gately
make test     # go test -race ./...
make lint     # gofmt, go vet, checagem de tipos do frontend
```

Para desenvolver o frontend, execute `cd web && pnpm dev`; o `/api` é encaminhado por proxy para o `127.0.0.1:16680` local.

## Convenções

- O código de compatibilidade deve se basear apenas em comportamentos, interfaces e formatos de configuração observáveis publicamente. Não envie código copiado de binários ou artefatos de frontend de terceiros.
- Um novo módulo precisa de: estrutura de configuração com validação, início e parada em tempo de execução, endpoints REST e testes unitários. Módulos que podem ser migrados do Lucky implementam `luckyimport.Importer`, e módulos que devem mostrar o status na página de visão geral implementam `module.Summarizer`.
- Os comentários de código são escritos em chinês; instruções de controle devem usar chaves.

## Testes extras

Alguns testes são ignorados por padrão e exigem preparação adicional:

| Teste | Como ativar |
| --- | --- |
| ACME de ponta a ponta (Pebble) | Instale pebble e pebble-challtestsrv e defina `GATELY_PEBBLE_BIN` e `GATELY_PEBBLE_SRC` |
| Servidores STUN públicos reais | `GATELY_NET_TEST=1` |
| Verificação de importação com uma configuração real do Lucky | `GATELY_LUCKY_SAMPLE` e outras; veja o `TestImportRealSample` de cada módulo |

## Licença

[Apache License 2.0](https://github.com/wangwanjie/gately/blob/main/LICENSE)
