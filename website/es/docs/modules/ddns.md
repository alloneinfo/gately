# DNS dinámico (DDNS)

La IP pública de una conexión de banda ancha doméstica cambia con el tiempo. El DDNS actualiza automáticamente tus registros DNS tras un cambio de IP, de modo que tu dominio siempre apunte a tu dirección actual.

## Proveedores compatibles

Alibaba Cloud, Cloudflare, Tencent Cloud, DNSPod, Huawei Cloud, GoDaddy, Porkbun, dynv6, DuckDNS y una devolución de llamada personalizada (para proveedores que no están integrados o para endpoints autoalojados).

## Cómo se obtiene la IP

- **URL**: solicita una dirección de consulta de IP pública (sustitúyela por el servicio que prefieras o aloja el tuyo);
- **Interfaz de red**: lee directamente la dirección de una interfaz de red concreta, adecuado para dispositivos que tienen IP pública;
- **Comando**: ejecuta un comando y usa su salida, adecuado para configuraciones de red poco habituales.

Se admiten tanto IPv4 como IPv6.

## Notificaciones

Se puede llamar a un webhook cuando cambia la IP o falla una sincronización.

## Recomendaciones

- Crea en tu proveedor DNS una clave de subcuenta limitada a la edición de registros DNS. Evita usar la clave de la cuenta principal;
- Tras crear una tarea, pulsa «Forzar sincronización» para confirmar que el registro se ha actualizado;
- La dirección de consulta de IP pública ya no depende de ningún servicio operado por el proyecto y puedes cambiarla en una tarea en cualquier momento.

## Migrar desde Lucky

Las tareas DDNS se pueden importar desde `lucky_ddns.lkcf`, con 11 proveedores. Las direcciones oficiales de consulta de IP de Lucky se eliminan durante la importación. Consulta la [Guía de migración](/es/docs/guide/migration) para más detalles.
