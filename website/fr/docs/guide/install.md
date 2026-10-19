# Installation

## Docker

Image : `ghcr.io/wangwanjie/gately`, disponible pour amd64, arm64 et arm/v7.

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
    # La redirection de ports, la traversée STUN et les services Web écoutent directement sur les ports de l'hôte
    network_mode: host
    volumes:
      # Dossier de configuration (JSON en clair, peut être versionné avec git ; ssl.json contient les clés privées des certificats)
      - ./data:/etc/gately
      # Gestion de Docker (à retirer si inutile)
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Europe/Paris
```

L'image embarque `sh`, `bash` et `curl` : les scripts de notification courants fonctionnent donc sans configuration supplémentaire.

## Binaire

1. Récupérez l'archive de votre plateforme sur la [page de téléchargement](/fr/download) : Linux (amd64 / arm64 / arm / mips / mipsle / riscv64), macOS, Windows ou FreeBSD.
2. Extrayez-la puis lancez :

```bash
./gately -c /etc/gately
```

### systemd

Le dépôt fournit un fichier de service complet, [deploy/systemd/gately.service](https://github.com/wangwanjie/gately/blob/main/deploy/systemd/gately.service), qui accorde déjà les capacités nécessaires pour écouter sur les ports bas :

```bash
sudo install -m 0755 gately /usr/local/bin/gately
sudo install -m 0644 gately.service /etc/systemd/system/gately.service
sudo systemctl daemon-reload && sudo systemctl enable --now gately
journalctl -u gately | grep -i "initial password"
```

### OpenWrt

Utilisez le script de démarrage procd [deploy/openwrt/gately.init](https://github.com/wangwanjie/gately/blob/main/deploy/openwrt/gately.init) :

```bash
cp gately /usr/bin/gately && chmod +x /usr/bin/gately
cp gately.init /etc/init.d/gately && chmod +x /etc/init.d/gately
/etc/init.d/gately enable && /etc/init.d/gately start
```

## Première connexion

L'interface d'administration écoute par défaut sur le port `16680`. Le nom d'utilisateur est `admin` et le mot de passe initial est écrit dans le journal au premier démarrage. Pour un déploiement automatisé, la variable d'environnement `GATELY_INITIAL_PASSWORD` permet de fixer le mot de passe initial ; aucun mot de passe aléatoire n'est alors généré.

La première connexion ouvre un guide de démarrage qui propose successivement de changer le mot de passe, d'activer l'authentification à deux facteurs ou une passkey, de lancer l'auto-diagnostic réseau, d'importer depuis Lucky et de configurer les notifications.

## Mise à niveau

Gately n'a pas de mise à jour automatique en ligne intégrée ; les mises à niveau dépendent de votre mode de déploiement :

- Docker : `docker pull` de la nouvelle image, puis recréation du conteneur (en conservant le dossier de configuration monté) ;
- Binaire : remplacez l'exécutable et redémarrez le service.

Avant de mettre à niveau, exportez une sauvegarde de la configuration ou conservez le dossier de configuration dans git. Les archives et images téléchargées peuvent être vérifiées selon [Vérifier les versions](/en/docs/guide/verify).
