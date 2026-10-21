# Convenciones de la API del terminal web

Todos los endpoints se montan bajo `/api/webterminal/` y requieren inicio de sesión. Los errores siempre se devuelven como `{"error": "..."}`.

## 1. Estructura de la configuración (el archivo webterminal en `conf`)

```jsonc
{
  "settings": {
    "allowLocalShell": false,
    "maxSessions": 20,
    "idleTimeout": 0,          // minutos; una sesión sin entrada ni salida durante este tiempo se cierra automáticamente; 0 significa sin límite
    "bufferSize": 256,         // KB; búfer de reproducción de la salida que se guarda por sesión y se reproduce al volver a conectar
    "heartbeatInterval": 30,   // segundos; latido del WebSocket (ping del servidor / ping del frontend)
    "sessionKeepAlive": 0      // minutos; cuánto tiempo se conserva una sesión tras desconectarse el navegador; 0 la cierra en cuanto se desconecta
  },
  "connections": [Connection],
  "shortcuts": [{ "name": "Show disk usage", "command": "df -h", "sendEnter": true }],
  "splitLayout": null,         // disposición dividida personalizada por el frontend; el backend la guarda y la devuelve tal cual (json.RawMessage)
  "security": {
    "agreementAccepted": false,
    "twoFAEnable": false,
    "twoFAKey": ""             // secreto TOTP; enmascarado en la salida de la API
  },
  "credentials": [Credential]  // preajustes de credenciales SSH
}
```

Connection (`type` es `ssh` / `telnet` / `local`):

```jsonc
{
  "key": "", "name": "", "type": "ssh", "remark": "", "enable": true,
  "quickAccessDirs": ["/etc", "/var/log"],      // directorios de acceso rápido de SFTP
  "ssh": { "host", "port", "username", "authType": "password|key|credential",
           "credentialKey": "",  // clave del preajuste de credenciales cuando authType=credential; username puede quedar vacío y entonces se usa el predeterminado del preajuste
           "password", "privateKey",
           "keyFile": "",        // ruta a un archivo de clave privada en el propio servidor; se usa cuando authType=key y privateKey está vacío
           "passphrase", "hostKey", "keepAlive", "workDir": "", "timeout": 15,
           "proxy": Proxy },
  "telnet": { "host", "port": 23, "username", "password", "timeout": 15, "proxy": Proxy },
  "local": { "shell", "workDir", "env": {"K":"V"} }
}
Proxy = { "enabled": false, "type": "socks5|http", "host", "port", "username", "password" }
```

Credential (preajuste de credenciales; lo pueden usar varias conexiones SSH):

```jsonc
{ "key": "", "name": "", "authType": "password|key", "username": "", // opcional; se usa cuando la conexión no tiene nombre de usuario
  "password", "privateKey", "keyFile", "passphrase", "remark": "", "enable": true }
```

Las contraseñas, las claves privadas, las frases de contraseña de las claves y las contraseñas de proxy se enmascaran con `crud.Mask` en la salida y se conservan con `crud.KeepSecret` al actualizar.

## 2. Gestión de conexiones

| Método | Ruta | Descripción |
|---|---|---|
| GET/POST | `/connections` | Lista / crea |
| PUT/DELETE | `/connections/{key}` | Actualiza / elimina |
| PUT | `/connections-order` | Reordena `{"keys":[...]}` |
| POST | `/connections/{key}/test` | Prueba la conectividad SSH / Telnet |
| POST | `/connections/test` | Prueba una conexión sin guardar (el cuerpo es una Connection; los secretos guardados se incorporan apuntando el campo `key` a la conexión original) |
| DELETE | `/connections/{key}/hostkey` | Restablece la huella del host |
| PUT | `/connections/{key}/quickaccess` | Cuerpo `{"dirs":["/a","/b"]}` |
| GET/POST | `/credentials` | Lista / crea preajustes de credenciales |
| PUT/DELETE | `/credentials/{key}` | Actualiza / elimina (devuelve 409 con los nombres de las conexiones mientras siga en uso) |
| PUT | `/credentials-order` | Reordena los preajustes `{"keys":[...]}` |
| GET | `/shells` | Lista de shells disponibles en esta máquina `["/bin/bash", ...]` (lee /etc/shells y filtra los que existen) |

Notas adicionales (la implementación es la referencia):
- `POST /connections/test` y `POST /connections/{key}/test` devuelven 200 con `{ok, error?, serverVersion?, newHostKey?, latencyMs}`; el tipo `local` devuelve 400. Para Telnet solo se verifica la accesibilidad TCP (también a través del proxy).
- `PUT /connections/{key}/quickaccess` devuelve `{ok, dirs}` (sin espacios sobrantes ni duplicados). Al editar una conexión sin `quickAccessDirs`, se conserva el valor anterior.
- `ssh.timeout` / `telnet.timeout` aceptan de 1 a 120 segundos, por defecto 15. `ssh.workDir` se aplica con `cd` tras iniciar sesión.
- En las conexiones con `authType=credential`, la contraseña, la clave privada, el archivo de clave y la frase de contraseña en línea se borran al guardar; las credenciales se obtienen del preajuste al conectar. Actualizar un preajuste descarta las conexiones SSH del pool que lo usan.
- El autorrellenado del inicio de sesión de Telnet se intenta una vez para cada campo, y solo cuando se detectan indicaciones como `login:` / `username:` / `password:`.

## 3. Sesiones de terminal (sesiones persistentes que se pueden reconectar tras una desconexión)

- `GET /connections/{key}/ws?cols=&rows=`: crea una sesión nueva y pasa a WebSocket.
- `GET /sessions/{id}/ws?cols=&rows=`: se conecta a una sesión existente, reproduce primero el búfer y después reenvía en tiempo real. Varios clientes pueden conectarse a la misma sesión a la vez (la entrada se comparte y la salida se difunde).

Protocolo WebSocket:
- Servidor → navegador: los **frames binarios** llevan la salida del terminal; los **frames de texto** llevan JSON:
  `{"t":"s","d":"<sessionId>"}` sesión establecida (se envía una vez tanto al crear como al conectarse);
  `{"t":"n","d":"notice"}` aviso; `{"t":"x","d":"reason"}` sesión terminada; `{"t":"p"}` respuesta al latido.
- Navegador → servidor: JSON de texto: `{"t":"i","d":"input"}`, `{"t":"r","c":cols,"r":rows}`, `{"t":"p"}` latido.
- Cuando se desconecta el WebSocket del navegador: si `sessionKeepAlive > 0`, la sesión se conserva (desconectada) y se cierra tras el tiempo de espera si nadie se conecta; si es 0, se cierra de inmediato.
- Cuando el proceso de la sesión termina por sí solo, se envía un mensaje `x`, se cierra el WebSocket y la sesión se elimina de la tabla de sesiones.

- Crear una sesión por encima de `maxSessions` devuelve HTTP 429; un fallo de conexión se notifica mediante un frame de aviso `n`, tras lo cual se cierra el WebSocket.
- `bytesIn` es el número de bytes de entrada del navegador, `bytesOut` es el número de bytes de salida del terminal y `attached` es el número de clientes conectados en ese momento (un número); los campos de tiempo están en RFC3339.

REST:

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/sessions` | `[{id, connKey, connName, type, remark, createdAt, lastActive, attached, cols, rows, bytesIn, bytesOut}]` |
| GET | `/sessions/{id}` | Una sola sesión |
| PUT | `/sessions/{id}/remark` | `{"remark":""}` |
| GET | `/sessions/{id}/stats` | `{bytesIn, bytesOut, durationSec, idleSec, attached, cols, rows}` |
| DELETE | `/sessions/{id}` | Termina la sesión |

## 4. Ajustes, accesos directos y disposición dividida

| Método | Ruta | Descripción |
|---|---|---|
| GET/PUT | `/settings` | `{settings, activeSessions}` / el cuerpo es Settings |
| GET/PUT | `/globalshortcuts` | Array de comandos de acceso directo |
| GET/PUT/DELETE | `/splitlayout` | Disposición JSON arbitraria |

## 5. Seguridad (acuerdo de usuario + verificación en dos pasos)

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/security` | `{agreementAccepted, twoFAEnable, verified}`; `verified` indica que el token de sesión actual ha superado la verificación en dos pasos |
| POST | `/security/agreement` | Acepta el acuerdo |
| POST | `/security/2fa/setup` | Genera un secreto pendiente `{secret, url}` (no se guarda) |
| POST | `/security/2fa/enable` | `{secret, code}` activa 2FA cuando la verificación es correcta |
| POST | `/security/2fa/disable` | `{code}` desactiva 2FA cuando la verificación es correcta |
| POST | `/security/verify2fa` | `{code}`; si es correcto, el token de sesión actual cuenta como verificado durante 30 minutos |
| GET | `/security/check2fa` | `{need: bool}`: si todavía se requiere verificación |

Tras 5 códigos 2FA fallidos seguidos, el token de sesión se bloquea durante 1 minuto (HTTP 429). `2fa/setup` devuelve 400 cuando 2FA ya está activado (desactívalo primero). Tras un `2fa/enable` correcto, el token actual cuenta como verificado.
Control de acceso: activar el shell local, crear una sesión, conectarse a una sesión y todos los endpoints SFTP devuelven 403 `{"error":"...","needAgreement":true}` cuando no se ha aceptado el acuerdo,
y devuelven 403 `{"error":"...","need2fa":true}` cuando 2FA está activado y el token actual no se ha verificado. «Activar el shell local» significa aquí un `PUT /settings` que cambia `allowLocalShell` de false a true. En el lado de Go, todos ellos llaman a `m.requireAccess(w, r) bool`.

## 6. Gestión de archivos por SFTP

El parámetro de ruta `{key}` es la **clave de la conexión**. Las conexiones SSH usan SFTP (cada petición abre un canal del subsistema SFTP sobre una conexión SSH reutilizada); las conexiones `local` operan directamente sobre el sistema de archivos del propio servidor (con los mismos endpoints).
Telnet y otros tipos devuelven 400; una conexión inexistente devuelve 404. El backend reutiliza una conexión SSH por conexión y la cierra tras 5 minutos de inactividad (`m.acquireSSH`).
Una conexión `local` requiere que «Permitir shell local» esté activado en los ajustes; en caso contrario devuelve 403 (los permisos sobre archivos locales equivalen a un shell local). Un fallo de conexión SSH devuelve 502.

Convenciones generales:
- Todo `path` / `from` / `to` / `dest` debe ser una ruta absoluta que empiece por `/` (cuando el `path` de `list` está vacío, se usa el directorio predeterminado: el directorio de inicio de sesión en SSH, y el workDir de la conexión o el directorio personal del usuario en local). Las rutas relativas devuelven 400.
- Los errores siempre son `{"error":"message"}`: no encontrado 404, permiso denegado 403, demasiado grande 413, tipo no admitido 415, el destino ya existe 409 (añadido en esta sección), parámetros incorrectos 400, todo lo demás 500.
- Los endpoints de escritura correctos devuelven `{"ok":true, ...}`.

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/sftp/{key}/list?path=` | `{path, parent, entries:[{name, path, isDir, isLink, size, mode, modeStr, modTime, owner?}]}`; primero los directorios y después ordenados por nombre sin distinguir mayúsculas. `mode` es una cadena de permisos octal como `"755"`, `modeStr` tiene la forma `drwxr-xr-x`, `modTime` está en RFC3339 y `owner` es el nombre de usuario en local y el uid numérico en SSH. Los enlaces simbólicos tienen `isLink=true`, e `isDir=true` cuando apuntan a un directorio |
| GET | `/sftp/{key}/read?path=&download=1` | Sin download: vista previa de texto, devuelve `{path, name, size, modTime, content}`; por encima de 1MB devuelve 413, y el contenido binario (con NUL o que no es UTF-8) devuelve 415. Con download=1: descarga en streaming (`application/octet-stream` + `Content-Disposition: attachment`), con soporte de `Range` e `If-Modified-Since` |
| POST | `/sftp/{key}/write` | `{path, content}` guarda texto (sobrescribe; un destino que es directorio devuelve 400) → `{ok, path, size}` |
| POST | `/sftp/{key}/upload` | Escritura multipart en streaming: `path` (el directorio de destino, que debe existir ya y **debe ir antes de los archivos**), `files` (se puede repetir) y `relativePaths` opcional (uno por archivo; se emparejan por orden de aparición con los archivos que siguen y deben ir antes de los archivos correspondientes; también se acepta una cadena con un array JSON). Sin `relativePaths`, se usa el nombre de archivo tal cual (puede llevar una subruta cuando un navegador sube un directorio). Los archivos con el mismo nombre se sobrescriben y los directorios intermedios que faltan se crean automáticamente; las rutas que contienen `..` devuelven 400, lo que garantiza que no se escribe nada fuera del directorio de destino → `{ok, count, files:[rutas absolutas]}` |
| POST | `/sftp/{key}/mkdir` | `{path}` crea un directorio (incluidos los padres que falten); 409 si ya existe |
| POST | `/sftp/{key}/touch` | `{path}` crea un archivo vacío (incluidos los padres que falten); 409 si ya existe |
| POST | `/sftp/{key}/rename` | `{from, to}`; 409 si `to` ya existe, 400 al mover un directorio dentro de sí mismo, 403 al renombrar el directorio raíz; en local, entre dispositivos distintos recurre automáticamente a copiar y después eliminar |
| POST | `/sftp/{key}/copy` | `{from:[...], to}`; `to` es el **directorio** de destino, y cada elemento se copia en `to/nombre-original` (recursivo; los enlaces simbólicos se copian como enlaces; se ignoran los archivos especiales como dispositivos y tuberías). Si el nombre coincide, la copia se **renombra automáticamente** a `name copy.ext`, `name copy 2.ext`, etc., sin error; copiar un directorio dentro de sí mismo devuelve 400 → `{ok, paths:[rutas nuevas]}` |
| POST | `/sftp/{key}/chmod` | `{path, mode:"755", recursive}`; `mode` es una cadena octal de 0 a 7777; la recursión no sigue los enlaces simbólicos, y archivos y directorios reciben todos el mismo permiso |
| DELETE | `/sftp/{key}/remove` | `{paths:[...]}` (cuerpo JSON; eliminación recursiva; en los enlaces simbólicos solo se elimina el enlace); si alguna ruta es `/`, devuelve 403 y no se elimina nada → `{ok, count}` |
| POST | `/sftp/{key}/compress` | `{paths:[...], dest, format:"zip|tar.gz"}` (format es zip por defecto; `tgz` se trata como tar.gz). `dest` es la ruta completa del archivo comprimido de salida; si `dest` es un directorio existente o termina en `/`, el archivo se nombra automáticamente dentro de él (el nombre del propio elemento si es uno solo, o el nombre del directorio padre si son varios, más `.zip` / `.tar.gz`). Cada elemento pasa a ser una entrada de primer nivel del archivo con su propio nombre; los elementos con nombres duplicados devuelven 400, un archivo de salida existente devuelve 409, y el archivo de salida nunca se empaqueta dentro de sí mismo → `{ok, path}` |
| POST | `/sftp/{key}/decompress` | `{path, dest}` admite zip / tar / tar.gz / tgz (se detecta por la extensión; cualquier otra cosa devuelve 415). Cuando `dest` está vacío, se extrae en el directorio que contiene el archivo comprimido, creándolo si hace falta; los archivos con el mismo nombre se sobrescriben. Un archivo comprimido que contenga cualquier entrada `..` se rechaza por completo (400, protección contra zip-slip); las entradas que no son archivos normales, como enlaces simbólicos y enlaces duros, se omiten; los permisos de archivo conservan solo los bits rwx → `{ok, dest, count}` |
| GET | `/sftp/{key}/preview-archive?path=` | Lista de entradas del archivo comprimido `[{name,size,isDir}]` (como máximo 10000 entradas; `name` no lleva `/` final) |

La compresión y la descompresión se hacen íntegramente en Go mediante la interfaz del sistema de archivos (os en local, lectura y escritura por SFTP en SSH), sin depender de comandos remotos.

## 7. Otros

- Registros: los registros del módulo ya se muestran en la página global de registros, así que no hace falta un endpoint aparte.
- Importación de Lucky: rellena telnet, proxy, keyFile, workDir, timeout, env, quickAccessDirs, sessionConfig, globalShortcuts
  y securityConfig (el acuerdo de usuario y el secreto 2FA se migran juntos).
