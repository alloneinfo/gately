# Certificados SSL

Emite y renueva certificados automáticamente para que los usen los [Servicios web](/es/docs/modules/web), o expórtalos a otro software.

## Capacidades

- Emisión y renovación automáticas con **ACME DNS-01**, reutilizando los proveedores DNS de [DNS dinámico](/es/docs/modules/ddns), sin necesidad de abrir los puertos 80 / 443;
- **ACME HTTP-01**: no necesita proveedor DNS, pero el puerto 80 de cada dominio debe ser accesible desde Internet; no admite comodines;
- Certificados comodín y EAB (por ejemplo, ZeroSSL);
- Subida de un certificado existente o lectura desde archivos;
- Exportación a un directorio: cuando cambia un certificado, los archivos se escriben en el directorio indicado y se ejecuta un script, para que otros servicios puedan recargarlo;
- Despliegue automático: cuando cambia un certificado, se envía por SSH a otros hosts y se ejecuta un comando, o se reinician los contenedores Docker que lo usan.

## Validación HTTP-01

Elige HTTP-01 en el «Método» del certificado. Durante la emisión, la CA accede a `http://<dominio>/.well-known/acme-challenge/<token>`:

- Si un [servicio web](/es/docs/modules/web) ya escucha en el puerto 80, responde él directamente sin afectar a los sitios existentes;
- Si el puerto 80 está libre, Gately escucha temporalmente en él durante la emisión y lo cierra en cuanto termina;
- Si el router reenvía el puerto público 80 a otro puerto de este equipo, indica ese puerto en «Puerto temporal».

Muchas conexiones domésticas (por ejemplo, en China continental) bloquean el puerto 80 entrante; en ese caso, usa DNS-01.

## Mapeo de certificados

Cuando un certificado está activado, Gately escribe `.crt`, `.key`, `.pem` y `_issuerCertificate.crt` en el directorio de mapeo y ejecuta el script de cambio.

## Despliegue automático

En el «Despliegue automático» del certificado puedes añadir varios destinos de despliegue, que se ejecutan automáticamente cada vez que el certificado se emite, se renueva o se actualiza:

- **Subir a host SSH**: elige una conexión SSH ya existente en el [Terminal web](/es/docs/modules/terminal) y se reutilizan su dirección, su método de autenticación y su proxy. Los archivos del certificado (`<nota>.crt`, `.key`, `.pem` y `_issuerCertificate.crt`, igual que en el mapeo de certificados, por lo que la nota es obligatoria) se escriben en el «Directorio remoto» y después se ejecuta en el host remoto el comando de «Ejecutar tras subir», por ejemplo `systemctl reload nginx`; el comando puede usar `$certName`, `$certDir`, `$certFile` y `$keyFile`. Cada archivo se escribe primero en un archivo temporal con los permisos ya ajustados y luego se renombra para sustituir al anterior; la clave privada queda con permisos 0600;
- **Reiniciar contenedores locales**: cuando cambia el certificado, los contenedores Docker elegidos se reinician uno tras otro; útil para servicios en contenedores que montan el directorio de mapeo y necesitan reiniciarse para cargar el certificado nuevo.

El despliegue automático nunca confía en un host desconocido: la conexión SSH elegida debe haberse abierto una vez desde el Terminal web para registrar la clave del host antes de que se envíe nada. Si un despliegue falla, se reintenta cada hora y se emite el evento «Certificado: despliegue fallido», al que puedes suscribirte en el [Centro de notificaciones](/es/docs/modules/notify). El botón «Desplegar ahora» de la tarjeta del certificado ignora el intervalo de reintento, despliega de inmediato en todos los destinos activados y muestra el resultado de cada uno.

Al editar un certificado, los despliegues cuyos ajustes de destino no han cambiado no se repiten; si cambias el directorio remoto, el comando o los contenedores, se vuelve a desplegar una vez.

## Nota de seguridad

Las claves privadas se guardan en `ssl.json` dentro del directorio de configuración. Guarda ese archivo con cuidado y, si lo pones bajo control de versiones, asegúrate de que el repositorio sea privado.

## Migrar desde Lucky

Los certificados ya emitidos se importan tal cual y no se vuelven a solicitar antes de que caduquen. ZeroSSL registra una nueva cuenta ACME con las credenciales EAB de Lucky. Consulta la [Guía de migración](/es/docs/guide/migration) para más detalles.
