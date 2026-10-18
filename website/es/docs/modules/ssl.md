# Certificados SSL

Emite y renueva certificados automáticamente para que los usen los [Servicios web](/es/docs/modules/web), o expórtalos a otro software.

## Capacidades

- Emisión y renovación automáticas con **ACME DNS-01**, reutilizando los proveedores DNS de [DNS dinámico](/es/docs/modules/ddns), sin necesidad de abrir los puertos 80 / 443;
- Certificados comodín y EAB (por ejemplo, ZeroSSL);
- Subida de un certificado existente o lectura desde archivos;
- Exportación a un directorio: cuando cambia un certificado, los archivos se escriben en el directorio indicado y se ejecuta un script, para que otros servicios puedan recargarlo.

## Mapeo de certificados

Cuando un certificado está activado, Gately escribe `.crt`, `.key`, `.pem` y `_issuerCertificate.crt` en el directorio de mapeo y ejecuta el script de cambio.

## Nota de seguridad

Las claves privadas se guardan en `ssl.json` dentro del directorio de configuración. Guarda ese archivo con cuidado y, si lo pones bajo control de versiones, asegúrate de que el repositorio sea privado.

## Migrar desde Lucky

Los certificados ya emitidos se importan tal cual y no se vuelven a solicitar antes de que caduquen. ZeroSSL registra una nueva cuenta ACME con las credenciales EAB de Lucky. Consulta la [Guía de migración](/es/docs/guide/migration) para más detalles.
