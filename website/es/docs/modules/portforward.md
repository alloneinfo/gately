# Reenvío de puertos

Reenvía un puerto en el que escucha Gately a otro dispositivo o servicio.

## Capacidades

- TCP y UDP;
- IPv4 e IPv6, con reenvío entre v4 y v6;
- Rangos de puertos y varios puertos;
- Varios destinos con conmutación por error;
- Límites de conexiones;
- Estadísticas de tráfico y registros de ejecución.

## Escenarios típicos

- Exponer los puertos de los servicios de un NAS a través de un único punto de entrada;
- Conectar el tráfico IPv6 con IPv4;
- Configurar un backend principal y otro de reserva para un mismo servicio.

## Consejo de despliegue

El reenvío de puertos escucha directamente en puertos del host. En despliegues con Docker se recomienda `--network host`.

## Migrar desde Lucky

Las reglas se pueden importar desde `lucky_portforward.lkcf`. Las opciones como el cifrado TLS / de flujo y el límite de velocidad aún no se admiten, y durante la importación se muestra un aviso.
