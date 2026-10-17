# Cómo contribuir

Gately es un proyecto de código abierto impulsado por la comunidad. Las incidencias y los pull requests son bienvenidos. Repositorio: <https://github.com/wangwanjie/gately>

## Desarrollo local

Requisitos: Go 1.26+, Node.js 24+, pnpm.

```bash
make web      # compila el frontend (genera web/dist, integrado con go:embed)
make build    # compila bin/gately
make test     # go test -race ./...
make lint     # gofmt, go vet, comprobación de tipos del frontend
```

Para desarrollar el frontend, ejecuta `cd web && pnpm dev`; `/api` se redirige al `127.0.0.1:16680` local.

## Convenciones

- La compatibilidad con Lucky se basa en el comportamiento, las interfaces y los formatos de configuración observables públicamente. No envíes código copiado de los binarios o del frontend de Lucky.
- Un módulo nuevo necesita: una estructura de configuración con validación, arranque y parada en ejecución, endpoints REST y pruebas unitarias. Los módulos que se pueden migrar desde Lucky implementan `luckyimport.Importer`, y los que deben mostrar su estado en la página de resumen implementan `module.Summarizer`.
- Los comentarios del código se escriben en chino; las sentencias de control deben usar llaves.

## Pruebas adicionales

Algunas pruebas se omiten por defecto y requieren una preparación adicional:

| Prueba | Cómo activarla |
| --- | --- |
| ACME de extremo a extremo (Pebble) | Instala pebble y pebble-challtestsrv, y define `GATELY_PEBBLE_BIN` y `GATELY_PEBBLE_SRC` |
| Servidores STUN públicos reales | `GATELY_NET_TEST=1` |
| Verificación de importación con una configuración real de Lucky | `GATELY_LUCKY_SAMPLE` y otras; consulta `TestImportRealSample` de cada módulo |

## Licencia

[Apache License 2.0](https://github.com/wangwanjie/gately/blob/main/LICENSE)
