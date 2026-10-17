# Установка и развёртывание

## Docker

Образ `ghcr.io/wangwanjie/gately`, поддерживаются amd64, arm64, arm/v7.

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
    # Проброс портов, STUN и веб-службы должны напрямую слушать порты хоста
    network_mode: host
    volumes:
      # Каталог конфигурации (открытый JSON, можно хранить в git; ssl.json содержит закрытые ключи сертификатов)
      - ./data:/etc/gately
      # Управление Docker (если не нужно, удалите)
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Asia/Shanghai
```

В образ встроены `sh`, `bash` и `curl`, поэтому типовые скрипты уведомлений работают без доработок.

## Бинарный файл

1. Скачайте архив для своей платформы на [странице загрузки](/ru/download): Linux (amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows, FreeBSD.
2. Распакуйте и запустите:

```bash
./gately -c /etc/gately
```

### Пример для systemd

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

## Первый вход

Интерфейс управления по умолчанию слушает порт `16680`, имя пользователя `admin`, начальный пароль выводится в журнал при первом запуске. После первого входа смените пароль и включите двухфакторную аутентификацию.

## Обновление

В Gately нет встроенного автообновления, обновлением занимается способ развёртывания:

- Docker: выполните `docker pull` для нового образа и пересоздайте контейнер (каталог конфигурации должен оставаться смонтированным);
- бинарный файл: замените исполняемый файл и перезапустите службу.

Перед обновлением рекомендуется экспортировать резервную копию конфигурации или просто вести каталог конфигурации в git.
