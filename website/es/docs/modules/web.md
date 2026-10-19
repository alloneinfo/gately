# Servicios web

Sirve proxy inverso, redirecciones, archivos estáticos y texto fijo por dominio. Es el punto de entrada web unificado de Gately. Ubicación en el menú: barra lateral «Web → Servicios web».

## Estructura: listeners, sitios y sitio por defecto

La configuración tiene tres niveles:

1. Listener: un puerto de escucha. Sus campos incluyen la red (`tcp` doble pila, `tcp4`, `tcp6`), la dirección de escucha (vacía para todas las direcciones), el puerto, el interruptor de HTTPS, la versión mínima de TLS (1.0 / 1.1 / 1.2 / 1.3) y si acepta el protocolo PROXY.
2. Sitio: un sitio dentro de un listener, que se selecciona por dominio. Los dominios pueden ser un dominio exacto, `*.example.com` (coincide con subdominios de cualquier profundidad) o `*` (coincide con todo). La prioridad de coincidencia es dominio exacto, comodín y después `*`.
3. Sitio por defecto: se usa cuando ningún sitio coincide. Si no está activado, se devuelve un 404.

## Tipos de sitio

| Tipo | Descripción |
| --- | --- |
| Proxy inverso | Reenvía a uno o varios backends. Las direcciones de backend tienen la forma `http(s)://host:port` y se admite WebSocket |
| Redirección | Redirige a una URL completa con código de estado 301 / 302 / 307 / 308 (por defecto 302), conservando opcionalmente la ruta y la cadena de consulta originales |
| Archivos estáticos | El directorio debe ser una ruta absoluta. Con «Listado de directorios» desactivado, una petición a un directorio solo devuelve respuesta si existe `index.html` |
| Texto fijo | Contenido personalizado, `Content-Type` (por defecto `text/plain; charset=utf-8`) y código de estado (100-599, por defecto 200) |

Otras opciones de los sitios de proxy inverso:

- Usar el Host del backend: la cabecera Host se cambia a la dirección del backend; si está desactivado, se conserva el Host de la petición del cliente;
- Omitir la verificación del certificado del backend, desactivar Keep-Alive;
- Enviar el protocolo PROXY al backend (cabecera v2, transmite la IP real del cliente, requiere compatibilidad del backend);
- Añadir `X-Forwarded-For` / `X-Forwarded-Proto` / `X-Forwarded-Host`.

Los sitios de cualquier tipo pueden definir un registro de acceso, orígenes cruzados (CORS), cabeceras de petición y respuesta personalizadas (un `Clave: Valor` por línea; un Valor vacío elimina esa cabecera) y pueden hacer referencia a una instancia de [Cortafuegos WAF](/es/docs/modules/waf) (si la instancia no existe o está desactivada, las peticiones pasan sin filtrar).

## Transporte y protocolos

- HTTPS: usa un certificado del módulo [Certificados SSL](/es/docs/modules/ssl) que cubra el dominio;
- HTTP/3 (QUIC): actívalo en un listener con HTTPS para servir también HTTP/3 por UDP en el mismo número de puerto; se anuncia al navegador con la cabecera de respuesta `Alt-Svc`. Hay que permitir ese puerto UDP en el cortafuegos y en el reenvío de puertos;
- WebSocket: el proxy inverso lo admite automáticamente;
- Protocolo PROXY: un listener puede aceptar cabeceras PROXY enviadas por componentes situados antes (como haproxy y frp). Solo confía en las IP o subredes indicadas en «Ajustes globales → Orígenes PROXY de confianza»; si se deja vacío, no confía en ningún origen. El proxy inverso puede enviar cabeceras PROXY al backend.

## Autenticación

Se configura en la página «Seguridad» del editor del sitio:

- Autenticación web: el navegador muestra una página de inicio de sesión;
- Autenticación básica: HTTP Basic;
- Usuarios: un `usuario:contraseña` por línea, y la contraseña puede ser un hash bcrypt. La autenticación básica necesita al menos un usuario cuando está activada; la autenticación web necesita al menos un usuario o una identidad de inicio de sesión de terceros cuando está activada;
- Inicio de sesión de terceros: se admiten GitHub, Google, QQ y OIDC, con tus propias aplicaciones OAuth. Las identidades permitidas se escriben una por línea con la forma `proveedor:cuenta`, por ejemplo `github:octocat` y `google:me@gmail.com`. Un `#` al final de una línea añade un comentario, y `github:*` significa cualquier cuenta de ese proveedor.

Los proveedores de inicio de sesión de terceros se configuran en «Ajustes globales»: introduce tu propio Client ID / Secret, activa el proveedor y define también una dirección de callback (debe ser atendida por uno de los listeners de Gately, por ejemplo `https://auth.example.com`). La URL de callback que debes registrar en GitHub / Google / QQ es:

```text
https://auth.example.com/__gately_auth/oauth/callback
```

El inicio de sesión de terceros de la propia interfaz de administración se configura por separado en «Ajustes → Cuenta y seguridad».

## Control de acceso

### Listas de IP permitidas y bloqueadas

El «Control de acceso por IP» de la página «Seguridad» del sitio puede estar desactivado, en modo lista de permitidas o en modo lista de bloqueadas. La lista admite una IP o subred por línea (como `192.168.1.0/24`).

### Reglas por ruta

En «Reglas por ruta», dentro de la página «Seguridad», sobrescribe la autenticación y la configuración de IP del sitio según el prefijo de ruta:

| Campo | Valores |
| --- | --- |
| `prefix` | Un prefijo de ruta que empieza por `/`, como `/admin` |
| `auth` | Heredar del sitio (vacío), `off` (sin autenticación), `on` (autenticación obligatoria) |
| `ipMode` / `ipList` | Heredar del sitio (vacío), `whitelist` o `blacklist`, con una lista de IP o subredes |

Reglas de coincidencia:

- Gana el prefijo más largo;
- La coincidencia respeta los límites de segmento de ruta: `/admin` coincide con `/admin` y `/admin/x`, pero no con `/administrator`;
- La ruta de la petición se normaliza primero (se eliminan `.`, `..` y las barras duplicadas) y después se compara, así que `/public/../admin` se trata como `/admin`;
- Solo se puede configurar una regla para el mismo prefijo.

La opción «Ámbito de autenticación» (`authOptional`): cuando está marcada, la autenticación solo se aplica a las rutas marcadas como «Obligatoria» en las reglas, y el resto de rutas son públicas. Esta opción requiere activar antes la autenticación básica o web.

### Enlaces temporales para compartir

Cuando la autenticación básica o web está activada, puedes crear enlaces para compartir de un sitio, de modo que los visitantes accedan sin cuenta:

| Campo | Descripción |
| --- | --- |
| Nombre | Una etiqueta, como «Para Alice» |
| Caducidad | Vacío para que no caduque |
| Usos `maxUses` | Número máximo de veces que se puede canjear el enlace; 0 significa sin límite. El contador se reinicia tras un reinicio |
| Límite de ruta `pathPrefix` | Si no está vacío, el enlace solo puede acceder a las rutas bajo este prefijo; debe empezar por `/` |

Un visitante abre `https://dominio/cualquier-ruta?_gs=token` para canjear una sesión. Tras un canje correcto, el servidor establece una cookie de acceso sin inicio de sesión y redirige a la dirección sin el parámetro `_gs`. La sesión dura como máximo 24 horas y nunca más allá de la caducidad del enlace. Un token no válido se trata como una visita normal y no revela el motivo del fallo. Las sesiones ya emitidas dejan de ser válidas de inmediato al eliminar el enlace. El token se genera al guardar (de al menos 16 caracteres) y el enlace completo se puede copiar desde la interfaz.

### Límite de intentos de autenticación y bloqueo automático

- Tras 10 autenticaciones fallidas desde la misma IP de origen en 15 minutos, los intentos posteriores se rechazan durante un tiempo;
- En «Ajustes globales → Bloqueo automático → Bloquear por fallos de autenticación», introduce un número de minutos (de 0 a 43200); la IP de origen se bloquea al llegar a 10 fallos. 0 significa sin bloqueo, solo limitación de intentos;
- Cuando se activa un bloqueo automático se publica un evento `AuthBanned`, que se puede enviar por el [Centro de notificaciones](/es/docs/modules/notify);
- Las direcciones IPv6 se cuentan y se bloquean por subred /64.

### Lista de bloqueos

La lista de bloqueos de «Ajustes globales» muestra las IP bloqueadas actualmente, el motivo (fallos de autenticación repetidos o bloqueo manual) y la hora de expiración, y cada entrada se puede levantar con «Desbloquear». También puedes bloquear manualmente: introduce una IP y un número de horas (de 1 a 720, es decir, 30 días como máximo); la API usa 24 horas por defecto.

### Filtro de países con GeoIP

El «Filtro de países» de un sitio tiene tres modos: desactivado, permitir solo y denegar. Introduce códigos de país ISO de dos letras separados por comas, como `CN, HK, US`. Cuando está activado se requiere al menos un país.

- El filtrado depende de una base de datos GeoIP, que debes importar antes en «Ajustes globales → Base de datos GeoIP»: sube un archivo `.mmdb` o introduce una URL de descarga. Se admiten MaxMind GeoLite2, DB-IP y otras bases de datos de países o ciudades; el límite de subida es de 129 MB;
- Si no hay ninguna base de datos cargada, el filtrado no tiene efecto (todos los visitantes pasan) y la interfaz muestra un aviso;
- Las direcciones de redes privadas siempre se permiten;
- En el modo «Denegar», se permiten las direcciones cuyo país no se encuentra en la base de datos.

## Mejoras del proxy inverso

Estas opciones están en la página «Tráfico» del editor del sitio.

### Backends, peso y reserva

La lista de backends admite una dirección por línea, seguida opcionalmente de parámetros:

```text
http://10.0.0.2:8080 weight=3
http://10.0.0.3:8080 backup
```

- `weight=N`: peso, un entero de 1 a 100, por defecto 1;
- `backup`: un backend de reserva que solo se usa cuando todos los backends que no son de reserva no están disponibles. Cuando todos los backends no están disponibles, se vuelven a probar los que no son de reserva.

### Política de balanceo de carga

| Opción de la interfaz | `lbPolicy` | Descripción |
| --- | --- | --- |
| Round robin (ponderado) | Vacío | Por defecto |
| Aleatorio (ponderado) | `random` | |
| Hash de IP (sesiones persistentes) | `ip_hash` | La misma IP de cliente siempre va al mismo backend |
| Menos conexiones | `least_conn` | Elige el menor valor de conexiones / peso |

### Comprobaciones de estado

El interruptor de comprobación activa está desactivado por defecto. Campos y valores por defecto una vez activado:

| Campo | Por defecto | Rango |
| --- | --- | --- |
| Modo `mode` | `http` | `http` (solicita la ruta de comprobación; un código de estado inferior a 400 cuenta como saludable) o `tcp` (solo prueba la conexión) |
| Ruta `path` | `/` | Empieza por `/`, solo en modo http |
| Intervalo `interval` | 10 segundos | De 1 a 3600 |
| Tiempo de espera `timeout` | 3 segundos | De 1 a 60 |
| Fallos consecutivos `fails` | 3 fallos lo marcan como no disponible | De 1 a 20 |
| Éxitos consecutivos `passes` | 2 éxitos lo recuperan | De 1 a 20 |

Si está activado el envío del protocolo PROXY al backend, las comprobaciones HTTP cambian automáticamente a TCP.

Las comprobaciones pasivas siempre están activas: entre varios backends, uno con 3 fallos de conexión consecutivos se pausa durante 30 segundos antes de volver a sondearlo (un backend único sin comprobación activa nunca se retira). Una vez activadas las comprobaciones activas, la recuperación se decide por completo según sus resultados. Cuando cambia el estado de un backend, se publican los eventos de backend caído y backend recuperado, que se pueden enviar por el [Centro de notificaciones](/es/docs/modules/notify).

### Compresión

La compresión se puede activar en los sitios de proxy inverso, archivos estáticos y texto fijo (no en los de redirección). Según `Accept-Encoding`, se prefiere `br` y después `gzip`.

- Solo se comprime el contenido de tipo texto: `text/*`, JSON, JavaScript, XML, XHTML, RSS / Atom, WebAssembly y SVG; se excluye `text/event-stream`;
- Las respuestas cuyo `Content-Length` declarado es inferior a 256 bytes se dejan tal cual;
- Las respuestas que ya tienen un `Content-Encoding` se dejan tal cual; las peticiones de actualización a WebSocket pasan directamente;
- Tras la compresión, los ETag fuertes se convierten en ETag débiles.

### Límite de peticiones por IP

Limitación de peticiones con token bucket por IP de cliente. Campos:

| Campo | Descripción |
| --- | --- |
| `rate` | Peticiones permitidas por segundo |
| `burst` | Capacidad de ráfaga. Cuando se define `rate` y `burst` es 0, por defecto vale `rate` redondeado hacia abajo más 1 |
| `maxConns` | Máximo de peticiones simultáneas por IP; 0 significa sin límite |

Cuando el límite de peticiones está activado, se debe definir al menos uno de `rate` y `maxConns`.

### Caché de respuestas en memoria

Se aplica solo a los sitios de proxy inverso. La caché vive en memoria y usa expulsión LRU.

| Campo | Por defecto | Rango |
| --- | --- | --- |
| Duración por defecto `ttl` | 0 | De 0 a 30 días (en segundos). Cuando el origen no declara ninguna directiva de caché, las respuestas 200 se almacenan durante este tiempo; 0 significa que solo se almacenan las respuestas que el origen permite explícitamente |
| Capacidad de la caché `maxMb` | 64 | De 1 a 4096 MB |

Reglas de caché:

- Solo se almacenan las peticiones GET, y la petición no debe contener las cabeceras `Authorization`, `Cookie` ni `Range`, ni `Cache-Control: no-cache`;
- Solo se almacenan las respuestas con estado 200; se omite una respuesta si contiene `Set-Cookie`, `Vary: *` o un `Cache-Control` con `no-store` / `private` / `no-cache`;
- Se respeta primero `s-maxage` y después `max-age`;
- Una entrada individual está limitada a 4 MB y también a un máximo de una cuarta parte de la capacidad total;
- Un acierto devuelve las cabeceras `X-Cache: HIT` y `Age`, y un fallo devuelve `X-Cache: MISS`;
- La clave de caché incluye el Host, la URI completa de la petición y `Accept-Encoding`.

### Reescritura y redirección de rutas

Las reglas se evalúan en orden y la evaluación se detiene en la primera coincidencia.

| Campo | Descripción |
| --- | --- |
| `pattern` | Una expresión regular que se compara con la ruta de la petición, como `^/old/(.*)$` |
| `to` | El resultado de la sustitución; usa `$1` para referirte a los grupos |
| `redirect` | Desactivado para una reescritura interna (la ruta reescrita se envía al backend y el destino debe empezar por `/`); activado para una redirección |
| `code` | Código de estado de la redirección 301 / 302 / 307 / 308, por defecto 302 |

Las redirecciones conservan los parámetros de consulta originales.

## Estadísticas de acceso y monitorización

### Estadísticas del sitio

En un listener en ejecución, pulsa «Estadísticas» en la fila de un sitio para abrir la ventana de estadísticas, donde puedes elegir la última hora, las últimas 6 horas o las últimas 24 horas:

- Estadísticas de acceso por minuto: un punto por minuto, conservadas durante las últimas 24 horas, con una curva de número de peticiones y curvas de errores 4xx / 5xx;
- Distribución de códigos de estado (2xx / 3xx / 4xx / 5xx) y tráfico de respuesta acumulado;
- Las 10 IP de origen y los 10 países principales (las estadísticas por país requieren una base de datos GeoIP importada);
- Los sitios de proxy inverso muestran el estado de salud, el peso, la marca de reserva y el número actual de conexiones de cada backend.

Las estadísticas se guardan en memoria y vuelven a empezar a contarse tras un reinicio o tras modificar el listener. Las clasificaciones siguen como máximo 5000 orígenes distintos.

### Métricas de Prometheus

El endpoint global `GET /api/metrics` reúne las métricas de todos los módulos; consulta [Métricas de monitorización](/en/docs/reference/monitoring). El endpoint del módulo que se describe a continuación solo incluye las métricas de los servicios web.

El endpoint de métricas es `GET /api/webservice/metrics`, en formato de texto de Prometheus, y requiere credenciales de inicio de sesión o un [token de API](/en/docs/reference/api-reference).

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/webservice/metrics
```

| Métrica | Tipo | Descripción |
| --- | --- | --- |
| `gately_site_requests_total` | counter | Peticiones acumuladas de un sitio |
| `gately_site_errors_total` | counter | Errores de backend de un sitio |
| `gately_site_active_requests` | gauge | Peticiones en curso |
| `gately_site_response_bytes_total` | counter | Bytes de respuesta acumulados |
| `gately_site_responses_total` | counter | Respuestas contadas por clase de estado (etiqueta `class`: de `2xx` a `5xx`) |
| `gately_upstream_up` | gauge | Si un backend está saludable: 1 saludable, 0 no disponible (con la etiqueta `upstream`) |

Todas las métricas llevan las etiquetas `listener` (nombre del listener#clave del listener) y `site` (nombre del sitio o, si no tiene, el primer dominio).

## Consejo de despliegue

Los servicios web necesitan escuchar en puertos del host como el 80 / 443. En despliegues con Docker se recomienda `--network host`.
