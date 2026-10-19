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

### systemd

В репозитории есть готовый unit-файл [deploy/systemd/gately.service](https://github.com/wangwanjie/gately/blob/main/deploy/systemd/gately.service), в котором уже выданы возможности (capabilities), нужные для прослушивания низких портов:

```bash
sudo install -m 0755 gately /usr/local/bin/gately
sudo install -m 0644 gately.service /etc/systemd/system/gately.service
sudo systemctl daemon-reload && sudo systemctl enable --now gately
journalctl -u gately | grep -i "initial password"
```

### OpenWrt

Используйте скрипт запуска procd [deploy/openwrt/gately.init](https://github.com/wangwanjie/gately/blob/main/deploy/openwrt/gately.init):

```bash
cp gately /usr/bin/gately && chmod +x /usr/bin/gately
cp gately.init /etc/init.d/gately && chmod +x /etc/init.d/gately
/etc/init.d/gately enable && /etc/init.d/gately start
```

## Первый вход

Интерфейс управления по умолчанию слушает порт `16680`, имя пользователя `admin`, начальный пароль выводится в журнал при первом запуске. При автоматизированном развёртывании начальный пароль можно задать переменной окружения `GATELY_INITIAL_PASSWORD` — тогда случайный пароль не создаётся.

При первом входе откроется мастер начальной настройки, который по очереди предложит сменить пароль, включить двухфакторную аутентификацию или passkey, запустить самопроверку сети, импортировать данные из Lucky и настроить уведомления.

## Обновление

В Gately нет встроенного автообновления, обновлением занимается способ развёртывания:

- Docker: выполните `docker pull` для нового образа и пересоздайте контейнер (каталог конфигурации должен оставаться смонтированным);
- бинарный файл: замените исполняемый файл и перезапустите службу.

Перед обновлением рекомендуется экспортировать резервную копию конфигурации или просто вести каталог конфигурации в git. Подписи загруженных архивов и образов можно проверить по инструкции [Проверка релизов](/en/docs/guide/verify).
