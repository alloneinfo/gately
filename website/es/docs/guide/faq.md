# Preguntas frecuentes

## ¿Necesita conectarse a los servidores del proyecto?

No. Gately no se conecta a ningún servidor del proyecto por defecto y no envía información del dispositivo. Las direcciones salientes, como la consulta de IP pública, usan servicios públicos de terceros y se pueden cambiar en las tareas correspondientes.

## ¿Dónde está la contraseña inicial?

Se imprime en el registro en el primer arranque, una sola vez. En despliegues con Docker, ejecuta `docker logs gately 2>&1 | grep -i "initial password"`.

## ¿Qué hago si olvido la contraseña del administrador?

Detén Gately, elimina `account.json` del directorio de configuración y vuelve a iniciarlo. El programa recrea la cuenta `admin` e imprime una nueva contraseña inicial en el registro. Tras iniciar sesión, revisa los ajustes de seguridad, como la autenticación de dos factores.

## ¿Por qué se recomienda `--network host`?

El reenvío de puertos, STUN y los servicios web escuchan directamente en puertos del host. Con una red bridge tienes que publicar tú mismo cada puerto en uso.

## ¿Puede funcionar junto con Lucky?

Sí. El puerto de administración es por defecto el 16680, que no entra en conflicto con el 16601 de Lucky. Las reglas importadas quedan desactivadas por defecto para que no compitan por los puertos ni por los mapeos UPnP del router con un Lucky que siga en marcha.

## ¿Por qué tengo que aportar yo mismo los binarios de Tailscale, frp y cloudflared?

Estos tres módulos supervisan los programas oficiales externos. Gately no los incluye, de modo que siempre puedes usar la última versión oficial. Consulta [Configuración y parámetros de ejecución](/es/docs/guide/configuration) para saber dónde colocarlos.

## Mi regla STUN está activada, pero sigo sin poder conectar desde fuera

Si la interfaz WAN de tu router está detrás de un NAT de operador (100.64.0.0/10), los equipos externos no pueden alcanzar el puerto traspasado. Es una limitación de la propia red. Usa en su lugar [Tailscale / frp / Cloudflare Tunnel](/es/docs/modules/tunnels).

## ¿Cómo informo de problemas o contribuyo?

Abre una incidencia en [GitHub Issues](https://github.com/wangwanjie/gately/issues). Las convenciones de desarrollo están en [Cómo contribuir](/es/docs/guide/contributing).
