# Prevención de intrusiones

Prevención de intrusiones al estilo de fail2ban: Gately detecta ataques de fuerza bruta en los registros, coloca trampas en puertos sin usar, cuenta los escaneos de puertos, añade las IP maliciosas a una lista de bloqueos y la sincroniza con el cortafuegos del sistema (nftables), que descarta en el kernel todo su tráfico.

## Funciones

- **Reglas de registros**: leen archivos de registro, el diario de systemd, la salida de un comando (como `logread -f` en OpenWrt) o los registros de un contenedor Docker, buscan fallos con expresiones regulares y bloquean la IP que alcanza el número de fallos en la ventana de tiempo. Incluye plantillas para SSH, Dropbear, Nginx, Vaultwarden, Home Assistant, Proxmox VE y otros, y en el editor puedes pegar líneas de ejemplo para probar los filtros;
- **Eventos de Gately**: los inicios de sesión fallidos en el panel y los bloqueos del WAF también pueden alimentar una regla, sin expresiones regulares;
- **Puertos trampa**: trampas en puertos que este equipo no usa (por defecto 23, 135, 1433, 3389, 5900, 6379 y otros). Cualquier origen que se conecte queda bloqueado;
- **Detección de escaneos**: se bloquea el origen que se conecta a puertos cerrados de este equipo más veces por minuto que el umbral;
- **Una sola lista de bloqueos**: aquí se gestionan los bloqueos manuales (incluidas redes CIDR y bloqueos permanentes) y los bloqueos automáticos por fallos de autenticación del servicio web. Los bloqueos se guardan y se restauran tras reiniciar;
- **Bloqueos progresivos**: si una dirección vuelve a bloquearse en 30 días, la duración se duplica hasta un máximo;
- **Lista blanca**: por defecto se ignoran las direcciones de la LAN, de loopback, de 100.64.0.0/10 (Tailscale) y las redes IPv6 /64 de las interfaces de este equipo (los bloqueos son por /64, así que los dispositivos con IPv6 pública de la misma LAN no caen junto con el infractor), y puedes añadir IP o redes. Una sola dirección IPv6 de la lista blanca protege todo su /64.

## Dónde se aplican los bloqueos

| Entorno | Efecto de un bloqueo | Puertos trampa | Detección de escaneos |
| --- | --- | --- | --- |
| Linux (root o `CAP_NET_ADMIN`, nftables instalado) | Se descarta en el kernel todo el tráfico, incluido SSH, otros programas y el tráfico reenviado a contenedores Docker y dispositivos de la LAN | Los gestiona el kernel y detectan escaneos que solo envían SYN | Disponible |
| Docker (red host + `cap_add: NET_ADMIN`) | Igual que arriba; las reglas se escriben en el anfitrión | Igual | Disponible |
| Paquete de fnOS, OpenWrt 22.03 y posteriores | Igual que arriba (se ejecuta como root y nftables viene incluido) | Igual | Disponible |
| Resto de casos (Windows, macOS, FreeBSD, Linux sin nftables o sin permisos) | Solo los servicios de Gately: reenvío de puertos, servicio web, STUN, WebDAV, FTP, SMB y el panel | Escucha Gately; solo detecta conexiones que completan el saludo TCP | No disponible |

La parte superior de la página muestra el estado de la sincronización con el cortafuegos y, si no está disponible, el motivo y cómo solucionarlo.

## Reglas del cortafuegos

Con «Sincronizar con el cortafuegos» activado, Gately crea su propia tabla `inet gately` en nftables y no toca las reglas del sistema (fw4, Docker, ufw, etc.):

- La cadena `prerouting` (prioridad -190, después del seguimiento de conexiones y antes del NAT de destino) descarta los paquetes que llegan de direcciones bloqueadas, tanto los dirigidos a este equipo como los reenviados a contenedores o dispositivos de la LAN. Las respuestas a las conexiones que abre este equipo (DNS, repositorios, etc.) siguen pasando aunque el otro extremo esté bloqueado. En IPv6 se bloquea por /64;
- La cadena `input` solo mira las conexiones nuevas a este equipo, para los puertos trampa y la detección de escaneos. Las direcciones de la lista blanca no participan;
- La tabla completa se elimina al desactivar la sincronización o al detener Gately. Si otro programa la borra, Gately la vuelve a escribir en menos de 10 segundos.

«Ver reglas» en la página muestra todas las reglas vigentes.

::: tip Docker
Usa `network_mode: host` y concede `NET_ADMIN` para que las reglas se escriban en el anfitrión:

```yaml
    network_mode: host
    cap_add:
      - NET_ADMIN
```

Sin ello, los bloqueos siguen aplicándose a los servicios de Gately. Para leer registros del anfitrión, monta el directorio de registros en el contenedor, por ejemplo `- /var/log:/var/log/host:ro`, e indica `/var/log/host/auth.log` en la regla. El origen «diario de systemd» no está disponible dentro de un contenedor.
:::

## Reglas de registros

| Origen | Qué indicar | Notas |
| --- | --- | --- |
| Archivo de registro | Rutas de archivo, una por línea | Lee las líneas nuevas desde el final del archivo y sigue la rotación (renombrado o truncado) |
| Diario de systemd | Unidades de servicio como `ssh.service` | Se lee con `journalctl -f` |
| Salida de un comando | Un comando como `logread -f` | Se reinicia si termina |
| Contenedor Docker | Nombre del contenedor | Se lee a través de la conexión a Docker del módulo Docker |
| Evento de Gately | Inicio de sesión fallido / bloqueo del WAF | Los eventos incluyen la IP de origen; no hace falta filtro |

Los filtros son expresiones regulares de Go (sintaxis RE2), una por línea, y `<HOST>` representa la dirección de origen, por ejemplo:

```
Failed password for (?:invalid user )?\S* ?from <HOST>
```

Las líneas que coinciden con «Excluir» no se cuentan. La ventana de tiempo, el número de fallos y la duración del bloqueo se configuran por regla; la duración puede ser la predeterminada, personalizada o permanente.

## Compromisos de la detección de escaneos

La detección cuenta las conexiones nuevas a puertos **en los que este equipo no escucha**. Los puertos TCP en escucha se actualizan cada 10 segundos y se consideran abiertos; los puertos que otras reglas del cortafuegos reenvían no pasan por la cadena `input` y no se cuentan. El umbral predeterminado es 12 por minuto. Un cliente normal reintenta un puerto cerrado unas 5 veces por minuto, así que no conviene bajar de 10.

Los puertos trampa que están en uso se excluyen automáticamente para no bloquear a visitantes legítimos.

## Relación con el servicio web

Los bloqueos de «Bloquear tras fallos de autenticación» del servicio web van a la misma lista, por lo que con la sincronización activada afectan a todo el equipo y no solo a los sitios web. Con «Ignorar direcciones de la LAN» activado, las direcciones de la LAN nunca se bloquean automáticamente.

El servicio web y el WAF solo conocen la dirección real del cliente mediante el protocolo PROXY. En sitios detrás de una CDN que pasa la IP real en una cabecera, los bloqueos por fallos de autenticación y la regla «Bloqueos del WAF» bloquearían los nodos de la CDN, así que no los actives en esos sitios.

## Notificaciones

Los bloqueos generan el evento `Prevención de intrusiones: IP bloqueada`, al que puedes suscribirte en el [Centro de notificaciones](/es/docs/modules/notify). Las direcciones que los puertos trampa y la detección de escaneos bloquean en una misma comprobación se agrupan en una sola notificación.

## Verificación

Verificado en espacios de nombres de red aislados en fnOS (Debian 12, nftables 1.0.6) e iStoreOS 24.10 (nftables 1.1.1): los puertos trampa y la detección de escaneos (IPv4 e IPv6) bloquean en el kernel y se importan a la lista de bloqueos; los archivos de registro (con rotación), el diario de systemd, los registros de contenedores Docker y los inicios de sesión fallidos en el panel activan bloqueos; los bloqueos manuales de redes y permanentes, el desbloqueo, las duraciones progresivas, la restauración tras reiniciar, la reescritura tras borrar la tabla desde fuera y la limpieza al detener funcionan como se espera; un contenedor Docker sin `NET_ADMIN` se detecta como sin permisos y pasa a usar puertos trampa en espacio de usuario.
