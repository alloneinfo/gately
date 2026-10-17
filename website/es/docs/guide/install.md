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

### Ejemplo con systemd

```ini
[Unit]
Description=Gately
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/gately -c /etc/gately
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Primer inicio de sesión

La interfaz de administración escucha por defecto en el puerto `16680`. El usuario es `admin` y la contraseña inicial se escribe en el registro en el primer arranque. Tras iniciar sesión por primera vez, cambia la contraseña y activa la autenticación de dos factores.

## Actualización

Gately no tiene autoactualización en línea integrada; las actualizaciones dependen de tu método de despliegue:

- Docker: ejecuta `docker pull` con la nueva imagen y recrea el contenedor (mantén montado el directorio de configuración);
- Binario: sustituye el ejecutable y reinicia el servicio.

Antes de actualizar, exporta una copia de seguridad de la configuración o guarda el directorio de configuración en git.
