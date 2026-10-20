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

## Application native fnOS {#fnos}

Chaque version fournit des paquets hors ligne pour fnOS : `gately_<version>_fnos_x86.fpk` (Intel / AMD) et `gately_<version>_fnos_arm.fpk` (ARM). Ils ne figurent pas dans le Centre d'applications officiel ; installez-les donc manuellement :

1. Récupérez le `.fpk` correspondant à l'architecture de votre appareil sur la [page de téléchargement](/fr/download#fnos) ;
2. Dans fnOS, ouvrez « Centre d'applications → Installation manuelle » (le bouton en bas à gauche), sélectionnez le fichier et terminez l'assistant ;
3. Cliquez sur l'icône Gately du bureau fnOS pour ouvrir l'interface d'administration.

### Intégration à la passerelle unifiée fnOS

L'icône du bureau ouvre l'interface d'administration via la passerelle unifiée fnOS, sous `/app/gately/` sur l'adresse que vous utilisez pour fnOS, avec le même point d'entrée. L'adresse du réseau local comme votre domaine fnOS distant fonctionnent, sans avoir à ouvrir ni rediriger le port 16680.

L'accès via la passerelle est contrôlé par fnOS, et seuls les administrateurs fnOS peuvent l'ouvrir. Le chemin d'accès secret et « Accès réseau local uniquement » s'appliquent uniquement à l'accès direct sur le port 16680. Sur le réseau local, `http://<adresse du NAS>:16680` reste accessible.

### Connexion avec votre session d'administrateur fnOS

Activez « Connexion via fnOS » dans « Paramètres → Contrôle d'accès » et les administrateurs déjà connectés à fnOS entrent dans Gately directement depuis l'icône du bureau, sans mot de passe ni code à deux facteurs :

- Seuls les administrateurs fnOS sont connectés automatiquement, en tant qu'administrateur Gately ; les utilisateurs fnOS ordinaires saisissent toujours un nom d'utilisateur et un mot de passe ;
- Cela s'applique uniquement à l'accès via la passerelle unifiée ; l'accès direct sur le port 16680 exige toujours une connexion normale ;
- Après une déconnexion, Gately ne vous reconnecte pas silencieusement : la page de connexion affiche à la place un bouton « Se connecter avec le compte fnOS &lt;nom&gt; » ;
- Ce réglage est désactivé par défaut et n'apparaît que dans le paquet fnOS.

### Remarques

- L'application s'exécute en `root` afin de pouvoir écouter sur 80/443, gérer Docker via `docker.sock` et ouvrir un vrai terminal Web. Installez-la uniquement sur un NAS que vous contrôlez.
- Au premier démarrage, le mot de passe initial de `admin` est écrit dans `/var/apps/gately/var/gately.log` : `grep -i "initial password" /var/apps/gately/var/gately.log`.
- La configuration se trouve dans le répertoire `var` de l'application et est conservée lors de la désinstallation, sauf si vous choisissez « tout supprimer » dans l'assistant de désinstallation.
- Le paquet ajoute aussi une commande `/usr/local/bin/gately` pour `gately ctl` et `gately mcp`.
- Le port 16680 est partagé avec l'image Docker. Arrêtez le conteneur `gately` avant de démarrer l'application native.

## Première connexion

L'interface d'administration écoute par défaut sur le port `16680`. Le nom d'utilisateur est `admin` et le mot de passe initial est écrit dans le journal au premier démarrage. Pour un déploiement automatisé, la variable d'environnement `GATELY_INITIAL_PASSWORD` permet de fixer le mot de passe initial ; aucun mot de passe aléatoire n'est alors généré.

La première connexion ouvre un guide de démarrage qui propose successivement de changer le mot de passe, d'activer l'authentification à deux facteurs ou une passkey, de lancer l'auto-diagnostic réseau, d'importer depuis Lucky et de configurer les notifications.

## Mise à niveau

Gately n'a pas de mise à jour automatique en ligne intégrée ; les mises à niveau dépendent de votre mode de déploiement :

- Docker : `docker pull` de la nouvelle image, puis recréation du conteneur (en conservant le dossier de configuration monté) ;
- Binaire : remplacez l'exécutable et redémarrez le service ;
- fnOS : installez manuellement le `.fpk` plus récent depuis le Centre d'applications ; votre configuration est conservée.

Avant de mettre à niveau, exportez une sauvegarde de la configuration ou conservez le dossier de configuration dans git. Les archives et images téléchargées peuvent être vérifiées selon [Vérifier les versions](/en/docs/guide/verify).
