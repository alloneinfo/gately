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

## Primer inicio de sesión

La interfaz de administración escucha por defecto en el puerto `16680`. El usuario es `admin` y la contraseña inicial se escribe en el registro en el primer arranque. En despliegues automatizados puedes fijar la contraseña inicial con la variable de entorno `GATELY_INITIAL_PASSWORD`; en ese caso no se genera ninguna contraseña aleatoria.

El primer inicio de sesión muestra una guía de bienvenida que te propone, paso a paso, cambiar la contraseña, activar la autenticación de dos factores o una passkey, ejecutar el autodiagnóstico de red, importar desde Lucky y configurar las notificaciones.

## Actualización

Gately no tiene autoactualización en línea integrada; las actualizaciones dependen de tu método de despliegue:

- Docker: ejecuta `docker pull` con la nueva imagen y recrea el contenedor (mantén montado el directorio de configuración);
- Binario: sustituye el ejecutable y reinicia el servicio.

Antes de actualizar, exporta una copia de seguridad de la configuración o guarda el directorio de configuración en git. Puedes comprobar la firma de los archivos e imágenes descargados según [Verificar versiones](/en/docs/guide/verify).
