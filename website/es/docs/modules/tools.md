# Archivos y herramientas de operación

## WebDAV

Se configura en «Servicio WebDAV», dentro del grupo «Servicios de archivos» de la barra lateral.

| Campo | Por defecto | Descripción |
| --- | --- | --- |
| `enable` | Desactivado | Activa el servicio; cuando está activado se requiere al menos un usuario |
| `listenIp` | Vacío | Dirección de escucha; déjalo vacío para escuchar en todas las direcciones |
| `port` | 18800 | Puerto de escucha |
| `tls` | Desactivado | Activa HTTPS; el certificado lo proporciona por SNI el módulo [Certificados SSL](/es/docs/modules/ssl) |
| `users` | Ninguno | Varios usuarios. Cada uno tiene `name`, `password`, `root` (una ruta de directorio absoluta) y `readOnly`, o usa `mounts` para montar varios directorios |

Cada usuario puede apuntar a un directorio o montar varios directorios:

- **Un solo directorio**: define `root` y `readOnly`. Tras iniciar sesión, la raíz es ese directorio.
- **Varios directorios**: pulsa «Montar varios directorios» en la interfaz, o rellena varias entradas en `mounts`. Cada entrada tiene `name` (nombre visible, no puede contener `/` y es único dentro de un mismo usuario), `path` (una ruta absoluta) y `readOnly`. Tras iniciar sesión, la raíz lista los directorios por nombre. La propia raíz no admite escritura y los archivos no se pueden mover entre directorios distintos.

## FTP

Se configura en «Servicio FTP», dentro del grupo «Servicios de archivos». Se admiten FTP y FTPS.

| Campo | Por defecto | Descripción |
| --- | --- | --- |
| `enable` | Desactivado | Activa el servicio; cuando está activado se requiere al menos un usuario |
| `listenIp` | Vacío | Dirección de escucha; déjalo vacío para escuchar en todas las direcciones |
| `port` | 2121 | Puerto de escucha |
| `passiveStart` / `passiveEnd` | 0 | Rango de puertos de datos del modo pasivo. Cuando ambos son 0, el sistema asigna los puertos al azar; cuando se definen, debe cumplirse 1 ≤ start ≤ end ≤ 65535, y el rango debe estar permitido en el cortafuegos y en los mapeos de puertos |
| `publicHost` | Vacío | IP que se anuncia al exterior en modo pasivo. Detrás de un NAT, introduce la IP pública o reenviada. Solo se admiten direcciones IP; déjalo vacío para usar automáticamente la dirección de la conexión |
| `tlsMode` | 0 | 0 permite tanto texto plano como FTPS, 1 exige FTPS explícito (`AUTH TLS`), 2 es FTPS implícito (cifrado desde la conexión). Cuando no es 0, el certificado lo proporciona por SNI el módulo [Certificados SSL](/es/docs/modules/ssl) |
| `idleTimeout` | 900 | Segundos de inactividad antes de cerrar una conexión |
| `connectionTimeout` | 30 | Tiempo de espera, en segundos, para establecer una conexión de datos |
| `disableActiveMode` | Desactivado | Usa solo el modo pasivo, más adecuado para entornos con NAT |
| `users` | Ninguno | Varios usuarios, con los mismos campos que WebDAV: `name`, `password`, `root`, `readOnly` o `mounts`; los nombres de usuario deben ser únicos |

## Montajes de almacenamiento

Añade montajes en «Montajes de almacenamiento», dentro del grupo «Servicios de archivos», y administra los archivos en línea desde la página.

| Campo | Descripción |
| --- | --- |
| `name`, `remark`, `enable` | Nombre, nota e interruptor de activación |
| `type` | `local` (directorio local), `webdav` (WebDAV remoto) o `smb` (recurso compartido SMB) |
| `path` | Directorio local, debe ser una ruta absoluta existente (lo usa `local`) |
| `url`, `username`, `password` | Dirección remota y cuenta; la contraseña se enmascara en la interfaz (lo usan `webdav` y `smb`). La dirección WebDAV empieza por `http://` o `https://`; la dirección SMB tiene la forma `smb://host[:puerto]/recurso[/subdirectorio]`, admite SMB2/3 y con el usuario vacío se conecta como invitado |
| `skipVerify` | Omite la verificación del certificado HTTPS (lo usa `webdav`) |
| `domain` | Dominio o grupo de trabajo para la autenticación SMB, normalmente vacío (lo usa `smb`) |
| `readOnly` | Montaje de solo lectura; el gestor de archivos rechaza cualquier operación de escritura |

El gestor de archivos en línea permite explorar directorios, descargar archivos, subir en streaming (el archivo completo no se carga en memoria), crear directorios, renombrar y eliminar. Un montaje puede servir como ubicación de destino de una tarea de copia de seguridad: los archivos de copia se escriben en el directorio indicado del montaje y los archivos antiguos se eliminan según la política de retención.

## Wake-on-LAN

Gestiona los dispositivos en «Wake-on-LAN» y envíales paquetes mágicos (6 bytes de `0xFF` seguidos de la MAC repetida 16 veces).

| Campo | Por defecto | Descripción |
| --- | --- | --- |
| `name` | Ninguno | Nombre del dispositivo, obligatorio |
| `macs` | Ninguno | Direcciones MAC, una por línea, al menos una; puedes introducir varias para un mismo dispositivo |
| `broadcastIps` | Vacío | Direcciones de difusión IPv4. Cuando está vacío, los paquetes se envían a `255.255.255.255` y a la dirección de difusión de cada interfaz de red activada |
| `port` | 9 | Puerto UDP de destino |
| `repeat` | 3 | Número de veces que se envía a cada MAC, hasta 20 |
| `checkAddr` | Vacío | Dirección de comprobación de conexión con la forma `host:port`, por ejemplo `192.168.1.10:3389`; déjalo vacío para omitir la comprobación |
| `remark` | Vacío | Nota |

Pulsa Encender en la lista de dispositivos para enviar el paquete mágico, y la página muestra si el dispositivo está conectado según `checkAddr`.

## Tareas programadas

Ejecuta comandos o envía peticiones HTTP según una programación cron.

- `spec` admite expresiones estándar de 5 campos, un campo de segundos opcional (6 campos, como `*/30 * * * * *`) y descriptores como `@every 10m` y `@daily`;
- Cuando `type` es `command`, `command` se ejecuta mediante `sh -c` (un `.bat` temporal en Windows), y el entorno recibe `taskName` más una variable en mayúsculas con el mismo nombre y el prefijo `GATELY_`;
- Cuando `type` es `http`, se envía una petición según la configuración `http`, con los campos `url`, `method` (GET, POST, PUT, PATCH, DELETE), `headers` (cada uno como `Clave: Valor`), `body`, `successContent` (el cuerpo de la respuesta debe contener cualquiera de sus entradas para contar como éxito; cuando está vacío, solo se comprueba un estado 2xx), `timeout` e `insecureSkipVerify`;
- Las tareas se pueden activar o desactivar, la página muestra la hora de la próxima ejecución y una tarea se puede ejecutar de inmediato a mano;
- Si la ejecución anterior de la misma tarea no ha terminado, se omite el nuevo disparo;
- Cada tarea conserva sus últimos 20 registros de ejecución, con la hora de inicio, la duración, éxito o fallo, si se activó manualmente y la salida (un resumen de hasta 4000 caracteres);
- El formulario de creación valida e interpreta la expresión.

## Resumen y registros del sistema

- La página «Resumen» muestra la información del host: modelo de CPU, uso y carga, memoria, disco, direcciones de las interfaces de red y velocidades en vivo, IPv4 e IPv6 públicas (pulsa para copiar), el estado del servicio de cada módulo y, para el proceso de Gately, su versión, commit, hora de compilación, hora de inicio, PID, CPU, memoria residente, heap de Go y número de goroutines;
- La página «Registros del sistema» reúne los registros de todos los módulos, incluidos los registros de ejecución de módulos como Docker, FTP y Tareas programadas.

## Temas

La interfaz ofrece 15 paletas en dos grupos, originales de Gately (Esmeralda, Océano, Fresco, Sakura, Atardecer, Medianoche, Punk) y clásicas (GitHub, Nord, Dracula, Catppuccin, Tokyo Night, Gruvbox, Rosé Pine, Solarized), y cada paleta tiene variante clara y oscura. El modo puede ser claro, oscuro o seguir el sistema, que usa la variante clara u oscura de la paleta actual. Cualquier paleta admite uno de 10 colores de acento predefinidos o uno elegido con el selector del sistema, y el logotipo y el degradado de la página de inicio de sesión lo siguen; volver a elegir una paleta quita el acento personalizado. Las esquinas pueden ser rectas, las de la paleta o redondeadas. La configuración del tema se guarda localmente en el navegador.
