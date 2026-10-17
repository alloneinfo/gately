# Comparación con Lucky

Gately está inspirado en Lucky, cubre las capacidades habituales de forma abierta y puede importar la configuración existente. Lucky lidera en amplitud de funciones y en base de usuarios, y sin Lucky no existiría Gately. Gracias.

## Filosofía

| Aspecto | Gately | Lucky |
| --- | --- | --- |
| Código fuente | Código abierto (Apache-2.0) | Código cerrado |
| Archivo de configuración | JSON plano, apto para git | Formato propietario (`.lkcf`) |
| Servicios oficiales | Ninguno por defecto; todas las direcciones salientes se pueden editar y desactivar | Actualizaciones en línea, relé de inicio de sesión de terceros y otros servicios oficiales |
| Inicio de sesión de terceros | Tus propias aplicaciones OAuth | A través del servicio de relé oficial |
| Actualizaciones | Docker / gestor de paquetes | Autoactualización en línea integrada |
| Amplitud de funciones | Cubre las funciones básicas habituales | Más amplia, p. ej. DLNA y montaje de unidades en la nube |

## Cobertura de funciones

| Capacidad | Gately |
| --- | --- |
| DNS dinámico | ✅ |
| Traversal STUN | ✅ |
| Reenvío de puertos | ✅ |
| Certificados SSL | ✅ |
| Servicios web (proxy inverso) | ✅ |
| WAF | ✅ |
| Tailscale / frp / cloudflared | ✅ |
| Gestión de Docker | ✅ |
| Terminal web | ✅ |
| WebDAV / FTP / gestión de almacenamiento | ✅ |
| Wake-on-LAN / tareas programadas | ✅ |
| Centro de notificaciones / copia de seguridad de la configuración / diagnóstico de red | ✅ |
| Tokens de API / OpenAPI / CLI / MCP | ✅ |
| DLNA, montaje de unidades en la nube, rclone, filtrado global de IP, HTTP/3 | Planificado, según la demanda de la comunidad |

## Diferencias deliberadas

- Sin autoactualización en línea integrada que sobrescriba el binario; las actualizaciones las gestiona tu gestor de paquetes o Docker.
- Sin envío remoto de configuración ni envío de datos del dispositivo.
- La configuración se guarda en texto plano y los campos sensibles se enmascaran en las respuestas de la API.

## Migración

Ambos pueden funcionar en paralelo, pasar módulo a módulo y volver atrás en cualquier momento. Consulta la [Guía de migración](/es/docs/guide/migration).
