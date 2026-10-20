# Instalación

## Docker

Imagen: `ghcr.io/wangwanjie/gately`, disponible para amd64, arm64 y arm/v7.

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
```

### Docker Compose

```yaml
services:
  gately:
    image: ghcr.io/wangwanjie/gately:latest
    container_name: gately
    restart: unless-stopped
    # El reenvío de puertos, el traversal STUN y los servicios web escuchan directamente en puertos del host
    network_mode: host
    volumes:
      # Directorio de configuración (JSON plano, se puede guardar en git; ssl.json contiene claves privadas de certificados)
      - ./data:/etc/gately
      # Gestión de Docker (quítalo si no lo necesitas)
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Asia/Shanghai
```

La imagen incluye `sh`, `bash` y `curl`, así que los scripts de notificación habituales funcionan sin cambios.

## Binario

1. Descarga el archivo para tu plataforma desde la [página de descargas](/es/download): Linux (amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows o FreeBSD.
2. Descomprímelo y ejecuta:

```bash
./gately -c /etc/gately
```

### systemd

El repositorio incluye un archivo de servicio completo, [deploy/systemd/gately.service](https://github.com/wangwanjie/gately/blob/main/deploy/systemd/gately.service), que ya concede las capacidades necesarias para escuchar en puertos bajos:

```bash
sudo install -m 0755 gately /usr/local/bin/gately
sudo install -m 0644 gately.service /etc/systemd/system/gately.service
sudo systemctl daemon-reload && sudo systemctl enable --now gately
journalctl -u gately | grep -i "initial password"
```

### OpenWrt

Usa el script de arranque de procd [deploy/openwrt/gately.init](https://github.com/wangwanjie/gately/blob/main/deploy/openwrt/gately.init):

```bash
cp gately /usr/bin/gately && chmod +x /usr/bin/gately
cp gately.init /etc/init.d/gately && chmod +x /etc/init.d/gately
/etc/init.d/gately enable && /etc/init.d/gately start
```

## Aplicación nativa para fnOS {#fnos}

Cada versión incluye paquetes sin conexión para fnOS: `gately_<version>_fnos_x86.fpk` (Intel / AMD) y `gately_<version>_fnos_arm.fpk` (ARM). No figuran en el Centro de aplicaciones oficial, así que debes instalarlos manualmente:

1. Descarga el `.fpk` correspondiente a la arquitectura de tu dispositivo desde la [página de descargas](/es/download#fnos);
2. En fnOS, abre «Centro de aplicaciones → Instalación manual» (el botón de la esquina inferior izquierda), elige el archivo y completa el asistente;
3. Haz clic en el icono de Gately del escritorio de fnOS para abrir la interfaz de administración.

### Funciona con la pasarela unificada de fnOS

El icono del escritorio abre la interfaz de administración a través de la pasarela unificada de fnOS en `/app/gately/` de la dirección que usas para fnOS, compartiendo el mismo punto de entrada. Funcionan tanto la dirección de la red local como tu dominio remoto de fnOS, sin necesidad de abrir ni redirigir el puerto 16680.

El acceso a través de la pasarela lo controla fnOS, y solo los administradores de fnOS pueden abrirla. La ruta de acceso segura y «Acceso solo desde la red local» se aplican únicamente al acceso directo por el puerto 16680. En la red local, `http://<dirección del NAS>:16680` sigue funcionando.

### Iniciar sesión con tu sesión de administrador de fnOS

Activa «Iniciar sesión con fnOS» en «Ajustes → Control de acceso» y los administradores que tengan la sesión iniciada en fnOS entrarán en Gately directamente desde el icono del escritorio, sin contraseña ni código de verificación en dos pasos:

- Solo los administradores de fnOS inician sesión de esta forma, como administrador de Gately; los usuarios normales de fnOS siguen introduciendo usuario y contraseña;
- Solo se aplica al acceso a través de la pasarela unificada; el acceso directo por el puerto 16680 sigue requiriendo un inicio de sesión normal;
- Tras cerrar la sesión, Gately espera a que vuelvas a entrar tú mismo: la página de inicio de sesión muestra el botón «Iniciar sesión como usuario de fnOS &lt;nombre&gt;»;
- El ajuste está desactivado por defecto y solo aparece en el paquete de fnOS.

### Notas

- La aplicación se ejecuta como `root`, de modo que puede escuchar en 80/443, gestionar Docker mediante `docker.sock` y abrir un terminal web real. Instálala solo en un NAS que controles.
- En el primer arranque, la contraseña inicial de `admin` se escribe en `/var/apps/gately/var/gately.log`: `grep -i "initial password" /var/apps/gately/var/gately.log`.
- La configuración se guarda en el directorio `var` de la aplicación y se conserva al desinstalar, salvo que elijas «Eliminar todo» en el asistente de desinstalación.
- El paquete también añade el comando `/usr/local/bin/gately` para `gately ctl` y `gately mcp`.
- El puerto 16680 se comparte con la imagen de Docker. Detén el contenedor `gately` antes de iniciar la aplicación nativa.

## Primer inicio de sesión

La interfaz de administración escucha por defecto en el puerto `16680`. El usuario es `admin` y la contraseña inicial se escribe en el registro en el primer arranque. En despliegues automatizados puedes fijar la contraseña inicial con la variable de entorno `GATELY_INITIAL_PASSWORD`; en ese caso no se genera ninguna contraseña aleatoria.

El primer inicio de sesión muestra una guía de bienvenida que te propone, paso a paso, cambiar la contraseña, activar la autenticación de dos factores o una passkey, ejecutar el autodiagnóstico de red, importar desde Lucky y configurar las notificaciones.

## Actualización

Gately no tiene autoactualización en línea integrada; las actualizaciones dependen de tu método de despliegue:

- Docker: ejecuta `docker pull` con la nueva imagen y recrea el contenedor (mantén montado el directorio de configuración);
- Binario: sustituye el ejecutable y reinicia el servicio;
- fnOS: instala manualmente el `.fpk` más reciente desde el Centro de aplicaciones; tu configuración se conserva.

Antes de actualizar, exporta una copia de seguridad de la configuración o guarda el directorio de configuración en git. Puedes comprobar la firma de los archivos e imágenes descargados según [Verificar versiones](/en/docs/guide/verify).
