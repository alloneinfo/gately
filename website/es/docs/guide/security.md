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
- Inicio de sesión con passkey (WebAuthn): solo se guarda la clave pública, se exige verificación del usuario (huella, rostro o PIN) y el contador de firmas impide la clonación; requiere acceder mediante un dominio con HTTPS (o localhost);
- Autenticación de dos factores (TOTP, verificada con los vectores de prueba de la RFC 6238);
- Cuentas de solo lectura: pueden ver el estado y la configuración, pero no modificarla, descargar copias de seguridad, leer secretos ni abrir terminales; al restablecer su contraseña o eliminarlas, sus sesiones dejan de ser válidas de inmediato;
- Límite de intentos de inicio de sesión;
- Proxy inverso local: cuando la interfaz de administración está detrás de nginx, haproxy o un servicio web de Gately en el mismo equipo, solo se confía en las cabeceras `X-Real-IP` / `X-Forwarded-For` que llegan desde direcciones locales. El acceso solo desde la red local, el límite de intentos de inicio de sesión y el registro de auditoría obtienen así el origen real. Puedes desactivarlo o añadir proxies de confianza adicionales en «Ajustes → Control de acceso»;
- Ruta de acceso segura, acceso solo desde la red local y HTTPS para la interfaz de administración;
- Pasarela unificada de fnOS: cuando se accede al paquete de fnOS a través de la pasarela unificada de fnOS, fnOS controla el acceso y solo los administradores de fnOS pueden abrirlo. «Iniciar sesión con fnOS» está desactivado por defecto; una vez activado, los administradores de fnOS con la sesión iniciada entran como administrador de Gately (omitiendo la verificación en dos pasos), mientras que los usuarios normales de fnOS siguen iniciando sesión. Consulta [Aplicación nativa para fnOS](/es/docs/guide/install#fnos);
- Interruptores de módulos: los módulos desactivados en «Ajustes → Módulos» no inician servicios ni ofrecen API de administración, lo que reduce la superficie de ataque;
- API con varios tokens: cada token tiene acceso de lectura y escritura únicamente a los módulos elegidos al crearlo, puede tener fecha de caducidad y registra la IP de origen de su último uso. Las operaciones de cuenta, autenticación de dos factores, gestión de tokens, copia de seguridad, restauración, reinicio y ajustes del sistema solo aceptan sesiones de inicio de sesión. Consulta [Tokens de API y OpenAPI](/es/docs/reference/api);
- Control de acceso a sitios: bloqueo automático tras fallos de autenticación, filtrado por país con GeoIP y reglas por ruta. Consulta [Servicios web](/es/docs/modules/web);
- Las copias de seguridad de la configuración admiten cifrado (scrypt + AES-GCM). Consulta [Copia de seguridad de la configuración](/es/docs/modules/backup).

## Auditoría y trazabilidad

El registro de auditoría guarda los inicios de sesión, los cambios de configuración y los accesos sensibles. El historial de configuración conserva la versión de cada archivo de configuración antes de sobrescribirlo y permite revertirla con un clic. Consulta [Auditoría e historial de configuración](/es/docs/guide/audit). Los artefactos publicados incluyen firma de cosign, atestación de procedencia de la compilación y SBOM; consulta cómo comprobarlos en [Verificar versiones](/en/docs/guide/verify).

## Valores por defecto de las funciones de alto riesgo

El terminal con shell local está desactivado por defecto. El terminal SSH usa una política de confianza en el primer uso para registrar las huellas de las claves del host.

## Informar de problemas de seguridad

Informa de las vulnerabilidades de forma privada a través de la página Security del repositorio y evita dar detalles en incidencias públicas: <https://github.com/wangwanjie/gately/security>

Para más recomendaciones prácticas, consulta la [Guía de refuerzo](/es/docs/guide/hardening).
