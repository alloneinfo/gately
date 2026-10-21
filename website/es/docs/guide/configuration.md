# Configuración y parámetros de ejecución

## Opciones de línea de comandos

| Opción | Descripción |
| --- | --- |
| `-c` | Directorio de configuración; también se puede definir con la variable de entorno `GATELY_CONFIG_DIR` |
| `-listen` | Sobrescribe la dirección de escucha de la interfaz de administración, p. ej. `:16680` |
| `-apply` | Directorio de configuración declarativa; también se puede definir con la variable de entorno `GATELY_APPLY_DIR`. Consulta más abajo |
| `-version` | Muestra la información de versión |

| Variable de entorno | Descripción |
| --- | --- |
| `GATELY_INITIAL_PASSWORD` | Contraseña inicial del administrador en el primer arranque (8 caracteres como mínimo); si no se define, se usa la contraseña predeterminada `gately.cn`. No tiene efecto si la cuenta ya existe |

## Directorio de configuración

Toda la configuración vive en el directorio de configuración, con un archivo de texto plano `<nombre>.json` por módulo, escrito de forma atómica con permisos 0600. Esto te permite:

- Inspeccionarlo con cualquier editor de texto;
- Guardar el directorio en git, auditar cada cambio con diferencias y revertir en cualquier momento;
- Hacer copias de seguridad simplemente copiando el directorio.

::: warning Nota
`ssl.json` contiene claves privadas de certificados. Antes de subirlo al control de versiones, asegúrate de que el repositorio sea privado o añade el archivo a tu lista de ignorados.
:::

## Configuración declarativa

Con `-apply <directorio>`, en cada arranque los archivos `<nombre>.json` de ese directorio se copian sobre el directorio de configuración; los archivos con contenido idéntico no se reescriben. Es útil para distribuir la configuración desde un repositorio git, un ConfigMap de Kubernetes o un script de despliegue.

- Antes de escribir se comprueba que cada archivo sea JSON válido; si alguno no lo es, Gately se niega a arrancar;
- `account.json` y `tokens.json` no se pueden sobrescribir: la cuenta y los tokens solo se gestionan desde la interfaz de administración;
- El contenido sobrescrito se conserva en el [historial de configuración](/es/docs/guide/audit) y se puede revertir en cualquier momento.

## Directorio de programas externos

Los módulos de Tailscale, frp y cloudflared supervisan los programas oficiales externos. Gately busca cada ejecutable en este orden:

1. La ruta definida en los ajustes del módulo;
2. `PATH`;
3. El subdirectorio `bin/` del directorio de configuración.

En despliegues con Docker, coloca los binarios en el directorio de configuración montado, por ejemplo `./gately-data/bin/frpc`. Tailscale necesita tanto `tailscale` como `tailscaled`. Consulta [Tailscale / frp / cloudflared](/es/docs/modules/tunnels) para más detalles.

## Copia de seguridad y restauración

«Ajustes» ofrece la copia de seguridad y la restauración de la configuración, y es también el punto de entrada para importar configuraciones de Lucky.
