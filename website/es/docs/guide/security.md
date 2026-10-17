# Modelo de seguridad y compromiso de privacidad

Gately funciona en el borde de la red y maneja claves de proveedores DNS, claves privadas de certificados y otras credenciales, por lo que la seguridad y la transparencia son sus objetivos de diseño principales.

## Compromiso de privacidad

- **Cero conexiones salientes por defecto**: sin conexiones a servidores del proyecto, sin envío de datos del dispositivo y sin envío remoto de configuración.
- **Direcciones salientes visibles y editables**: la consulta de IP pública, los servidores STUN y direcciones similares se muestran en la interfaz y se pueden cambiar o desactivar.
- **Inicio de sesión de terceros bajo tu control**: el acceso con GitHub, Google, QQ y OIDC usa aplicaciones OAuth que registras tú mismo, sin ningún relé intermedio.
- **Actualizaciones bajo tu control**: el programa nunca reemplaza su propio binario en segundo plano; tú decides cuándo actualizar.

## Protección de los datos

- La configuración es JSON plano con permisos 0600, por lo que se puede auditar y revertir.
- Los campos sensibles se enmascaran en las respuestas de la API.
- El token de cloudflared se pasa mediante una variable de entorno y no aparece en la lista de procesos; los tokens y claves de frp y Tailscale se guardan con permisos 0600.

## Autenticación y control de acceso

- Contraseñas con hash bcrypt y tokens de sesión firmados con HMAC;
- Autenticación de dos factores (TOTP, verificada con los vectores de prueba de la RFC 6238);
- Límite de intentos de inicio de sesión;
- Ruta de acceso segura, acceso solo desde la red local y HTTPS para la interfaz de administración;
- API con varios tokens: cada token tiene acceso de lectura y escritura únicamente a los módulos elegidos al crearlo, puede tener fecha de caducidad y registra la IP de origen de su último uso. Las operaciones de cuenta, autenticación de dos factores, gestión de tokens, copia de seguridad, restauración, reinicio y ajustes del sistema solo aceptan sesiones de inicio de sesión. Consulta [Tokens de API y OpenAPI](/es/docs/reference/api);
- Control de acceso a sitios: bloqueo automático tras fallos de autenticación, filtrado por país con GeoIP y reglas por ruta. Consulta [Servicios web](/es/docs/modules/web);
- Las copias de seguridad de la configuración admiten cifrado (scrypt + AES-GCM). Consulta [Copia de seguridad de la configuración](/es/docs/modules/backup).

## Valores por defecto de las funciones de alto riesgo

El terminal con shell local está desactivado por defecto. El terminal SSH usa una política de confianza en el primer uso para registrar las huellas de las claves del host.

## Informar de problemas de seguridad

Informa de las vulnerabilidades de forma privada a través de la página Security del repositorio y evita dar detalles en incidencias públicas: <https://github.com/wangwanjie/gately/security>

Para más recomendaciones prácticas, consulta la [Guía de refuerzo](/es/docs/guide/hardening).
