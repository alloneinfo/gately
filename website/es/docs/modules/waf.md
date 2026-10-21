# Cortafuegos WAF

Un cortafuegos de aplicaciones web basado en [Coraza](https://coraza.io/) y OWASP CRS v4, que se puede asociar a los sitios de [Servicios web](/es/docs/modules/web).

## Capacidades

- Conjunto de reglas OWASP CRS v4 integrado, con archivos de reglas que se pueden activar individualmente;
- Reglas SecLang personalizadas;
- Modo de bloqueo y modo de solo detección;
- Varias instancias, a las que los sitios hacen referencia según necesiten.

## Recomendaciones

1. En un sitio nuevo, empieza con «Solo detectar» y observa los registros durante un tiempo para confirmar que no hay falsos positivos;
2. Ajusta los archivos de reglas o añade excepciones personalizadas para los falsos positivos;
3. Después cambia a «Bloquear».

## Pruebas

El WAF se prueba con cargas de ataque reales, que cubren el bloqueo de path traversal y ataques similares.
