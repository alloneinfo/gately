# Tailscale / frp / cloudflared

Estos tres módulos integran los programas oficiales como procesos gestionados: reinicio automático, registros recogidos en la interfaz y configuración desde la propia interfaz. Gately no incluye estos programas, de modo que siempre puedes usar la última versión oficial.

## Ubicación de los programas

Los ejecutables se buscan en este orden:

1. La ruta definida en los ajustes del módulo;
2. `PATH`;
3. El subdirectorio `bin/` del directorio de configuración.

En despliegues con Docker, coloca el binario en el directorio de configuración montado, por ejemplo `./gately-data/bin/frpc`. Tailscale necesita tanto `tailscale` como `tailscaled`.

Los tokens y las claves se guardan con permisos 0600. El token de cloudflared se pasa mediante una variable de entorno, por lo que no aparece en la lista de procesos.

## Tailscale

- Gestiona `tailscaled` (red en espacio de usuario, apta para contenedores) o se conecta a un demonio del sistema existente;
- Clave de autenticación o enlace de inicio de sesión, con compatibilidad con Headscale;
- Rutas de subred y nodos de salida;
- Lista de nodos y proxy SOCKS5;
- Puerto UDP de WireGuard fijo, que facilita permitir conexiones directas a través de un cortafuegos;
- Argumentos adicionales y variables de entorno de tailscaled. Si el iptables del host usa el backend nf_tables, añade `TS_DEBUG_FIREWALL_MODE=nftables` a las variables de entorno; de lo contrario, el reenvío de subredes no funcionará.

## frp

- **Cliente frpc**: proxies tcp / udp / http / https / stcp / xtcp; al añadir o quitar uno, el cambio se aplica de inmediato;
- **Servidor frps**: lista de puertos permitidos, subdominios, panel de control;
- **Direcciones de escucha de frps**: «Dirección de escucha» indica dónde escucha el puerto de conexión de frpc y «Dirección de escucha de proxies», dónde escuchan los puertos HTTP / HTTPS y los puertos remotos de los proxies tcp / udp; vacías escuchan en todas las direcciones. Para poner frps detrás de un proxy inverso como Gately, usa `127.0.0.1` como dirección de escucha, crea en Servicio web un sitio que haga proxy inverso a `127.0.0.1:<puerto de enlace>` y elige websocket o wss como protocolo de transporte de frpc apuntando a ese sitio; deja vacía la dirección de escucha de proxies para que los puertos remotos de los proxies tcp sigan accesibles desde fuera. Al importar una configuración de frps sin `proxyBindAddr`, se usa `bindAddr`, igual que hace frps;
- **Importar configuración existente**: «Importar configuración», en la cabecera de la página, admite los antiguos `frpc.ini` / `frps.ini` (incluida la sección `[common]`) y el `toml` de frp 0.52+. Puedes elegir un archivo o pegar el contenido. Revisa primero el resultado interpretado y los campos ignorados, y después confirma la importación. En el cliente puedes elegir entre añadir (los proxies con el mismo nombre no se modifican) o reemplazar todos los proxies. La importación no cambia el estado de activación y también se puede invocar por la API en `POST /api/frp/import`.

## Cloudflare Tunnel

- Túneles con token y túneles temporales de trycloudflare;
- Varios túneles con reinicio automático;
- Muestra la dirección pública.

## Cómo elegir

| Escenario | Recomendación |
| --- | --- |
| Montar una red privada entre varios dispositivos | Tailscale |
| Tienes un servidor público y quieres publicar a través de él los servicios internos | frp |
| Sin IP pública ni servidor, y quieres exponer un servicio web rápidamente | Cloudflare Tunnel |
| Tu operador ofrece un NAT que permite la perforación | [Traversal STUN](/es/docs/modules/stun) |
