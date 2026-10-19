# Reenvío de puertos

Reenvía un puerto en el que escucha Gately a otro dispositivo o servicio.

## Capacidades

- TCP y UDP;
- IPv4 e IPv6, con reenvío entre v4 y v6;
- Rangos de puertos y varios puertos;
- Varios destinos con conmutación por error;
- Límites de conexiones;
- Estadísticas de tráfico y registros de ejecución;
- Envío del protocolo PROXY (v1 / v2) al destino, para entregar la dirección real del cliente al siguiente salto.

## Protocolo PROXY

El reenvío de puertos solo retransmite TCP, así que por defecto el servicio de destino solo ve la dirección de Gately. Si en una regla defines «Protocolo PROXY» como v1 o v2, Gately envía primero una cabecera PROXY tras conectar con el destino, y este obtiene la dirección real del cliente para sus registros, límites de peticiones y control de acceso.

El destino debe tener activada la opción de recepción correspondiente; de lo contrario, rechazará la conexión. Por ejemplo:

- haproxy: `bind 192.168.1.8:4434 accept-proxy`;
- nginx: `listen 4435 ssl proxy_protocol;`;
- Servicio web de Gately: activa «Aceptar protocolo PROXY» en el listener y añade el origen a los orígenes PROXY de confianza.

Esta opción solo afecta a TCP; el reenvío UDP no cambia.

## Escenarios típicos

- Exponer los puertos de los servicios de un NAS a través de un único punto de entrada;
- Conectar el tráfico IPv6 con IPv4;
- Configurar un backend principal y otro de reserva para un mismo servicio.

## Consejo de despliegue

El reenvío de puertos escucha directamente en puertos del host. En despliegues con Docker se recomienda `--network host`.

## Migrar desde Lucky

Las reglas se pueden importar desde `lucky_portforward.lkcf`. Las opciones como el cifrado TLS / de flujo y el límite de velocidad aún no se admiten, y durante la importación se muestra un aviso.
