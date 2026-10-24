# Cortafuegos WAF

Un cortafuegos de aplicaciones web basado en [Coraza](https://coraza.io/) y OWASP CRS v4, que se puede asociar a los sitios de [Servicios web](/es/docs/modules/web).

## Capacidades

- Conjunto de reglas OWASP CRS v4 integrado, con archivos de reglas que se pueden activar individualmente;
- Reglas SecLang personalizadas;
- Los grupos de reglas se pueden reordenar; los registros de coincidencias guardan la IP del cliente, el dominio, la ruta y el ID de regla, y se pueden filtrar por dominio, IP o palabras clave;
- Modo de bloqueo y modo de solo detección;
- Límite de inspección del cuerpo de la solicitud ajustable: al superarlo se rechaza la solicitud o se inspecciona solo la primera parte; también se puede omitir la inspección del cuerpo por ruta;
- Modo de compatibilidad con WebDAV: permite los métodos, las cabeceras y cualquier tipo de contenido de subida que usan las nubes personales y WebDAV;
- Varias instancias, a las que los sitios hacen referencia según necesiten.

## Recomendaciones

1. En un sitio nuevo, empieza con «Solo detectar» y observa los registros durante un tiempo para confirmar que no hay falsos positivos;
2. Ajusta los archivos de reglas o añade excepciones personalizadas para los falsos positivos;
3. Después cambia a «Bloquear».

## Orden de los grupos de reglas

Los grupos de reglas de una instancia se cargan de arriba abajo; usa los botones subir y bajar para reordenarlos. Dentro de la misma fase, las reglas anteriores se ejecutan primero:

- Las reglas de bloqueo (deny y similares) funcionan en cualquier posición;
- Las reglas de permiso o exclusión en tiempo de ejecución (`allow`, `ctl:ruleRemoveById`, `ctl:ruleRemoveTargetById`, etc.) deben ir encima del grupo OWASP CRS, o CRS puede bloquear la solicitud antes;
- Las directivas de configuración como `SecRuleRemoveById` y `SecRuleUpdateTargetById` solo afectan a reglas ya cargadas, así que deben ir debajo del grupo CRS; encima no tienen efecto.

## Sitios con subidas de archivos grandes

Por defecto, el WAF solo inspecciona cuerpos de solicitud de hasta 12,5 MB y rechaza las solicitudes mayores (413), por lo que las nubes personales, WebDAV y los gestores de archivos no pueden recibir archivos grandes. Puedes ajustarlo en la sección «Inspección del cuerpo de la solicitud» de la instancia:

- **Límite de inspección**: 0 significa el valor predeterminado de 12,5 MB; el máximo es 1024 MB;
- **Al superar el límite**: «Rechazar la solicitud» mantiene el comportamiento anterior; «Inspeccionar solo la primera parte» inspecciona solo la parte dentro del límite y reenvía el resto al backend sin cambios, lo adecuado para sitios con subidas grandes;
- **Compatible con WebDAV y subidas**: por defecto, OWASP CRS solo permite GET, HEAD, POST y OPTIONS, y bloquea las cabeceras `Lock-Token`, `If` y `Content-Range` y los tipos de contenido fuera de su lista permitida, por lo que los clientes WebDAV no pueden listar carpetas, subir archivos ni bloquearlos. Al activarlo se permiten PUT, DELETE, PROPFIND, PROPPATCH, MKCOL, COPY, MOVE, LOCK, UNLOCK y otros métodos, además de esas cabeceras, y se deja de restringir el tipo de contenido de las subidas; el cuerpo de PUT y de `application/octet-stream` se trata como contenido de archivo y ya no se inspecciona como parámetros de formulario. Las comprobaciones de inyección, path traversal y otros ataques siguen activas;
- **Rutas sin inspección del cuerpo**: un prefijo de ruta por línea (como `/api/upload` o `/remote.php/dav/`); en las solicitudes coincidentes se omite la inspección del cuerpo, y las cabeceras, la URL y los parámetros se siguen inspeccionando con normalidad.

Se recomienda crear una instancia aparte para las nubes personales y los sitios WebDAV: activa «Compatible con WebDAV y subidas» y elige «Inspeccionar solo la primera parte»; si hace falta, añade las rutas de subida a las rutas sin inspección. Los demás sitios siguen con los ajustes predeterminados.

curl, rclone y otros clientes envían la cabecera `Expect: 100-continue` al subir contenido de cierto tamaño, y CRS 4 la bloquea por defecto. Todas las instancias de Gately permiten esa cabecera (Gately gestiona el 100-continue por su cuenta), así que no hace falta configurar nada.

## Pruebas

El WAF se prueba con cargas de ataque reales, que cubren el bloqueo de path traversal y ataques similares.
