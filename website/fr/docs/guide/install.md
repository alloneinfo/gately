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

### OpenWrt {#openwrt}

Chaque version fournit des paquets OpenWrt qui contiennent le binaire, le script de démarrage procd et une règle qui conserve votre configuration lors des mises à jour du firmware :

- OpenWrt 24.10 et antérieur (y compris iStoreOS et les autres firmwares basés sur 24.10) utilisent le `.ipk`, à installer avec `opkg install` ;
- OpenWrt 25.12 et ultérieur utilisent le `.apk`, à installer avec `apk add --allow-untrusted` (les paquets ne sont pas signés, l'option `--allow-untrusted` est donc obligatoire).

Exécutez `grep DISTRIB_ARCH /etc/openwrt_release` pour connaître l'architecture de l'appareil, puis choisissez le paquet dans le groupe OpenWrt de la [page de téléchargement](/fr/download) :

| DISTRIB_ARCH | Paquet |
| --- | --- |
| `x86_64` | `gately_<version>_openwrt_amd64` |
| `aarch64_*` | `gately_<version>_openwrt_arm64` |
| `arm_cortex-a*_*` (ARMv7 avec VFP / NEON) | `gately_<version>_openwrt_arm` |
| `mipsel_*` | `gately_<version>_openwrt_mipsle` |
| `mips_*` | `gately_<version>_openwrt_mips` |
| `riscv64_*` | `gately_<version>_openwrt_riscv64` |

```bash
cd /tmp
# OpenWrt 24.10 / iStoreOS (exemple pour x86_64)
opkg install ./gately_<version>_openwrt_amd64.ipk
# OpenWrt 25.12
apk add --allow-untrusted ./gately_<version>_openwrt_amd64.apk

# Mot de passe initial (écrit dans le journal système au premier démarrage)
logread | grep -i "initial password"
```

- Le service est activé au démarrage et lancé dès l'installation ; l'interface d'administration est accessible à `http://<adresse du routeur>:16680`. La configuration se trouve dans `/etc/gately` et est conservée lors des mises à jour du firmware (sysupgrade) ;
- Mise à niveau : téléchargez le paquet plus récent et relancez `opkg install` ou `apk add --allow-untrusted` ; votre configuration est conservée. Si vous avez désactivé le lancement au démarrage, il reste désactivé après la mise à niveau ;
- Désinstallation : `opkg remove gately` ou `apk del gately`. Le répertoire de configuration est conservé ; exécutez `rm -rf /etc/gately` pour le supprimer aussi ;
- L'architecture de l'appareil est vérifiée avant l'installation. En cas de non-correspondance, opkg interrompt l'installation ; l'apk d'OpenWrt 25.12 installe tout de même le paquet mais laisse le service arrêté : exécutez alors `apk del gately` et installez le bon paquet ;
- Le binaire pèse environ 35 Mo et requiert autant d'espace libre (vérifiez avec `df -h /overlay`), ce qui convient aux routeurs logiciels x86, aux boîtiers ARM64 et aux autres appareils dotés d'un stockage confortable ; les routeurs avec 16 Mo / 32 Mo de mémoire flash sont trop petits ;
- Le paquet arm nécessite un ARMv7 avec unité de calcul en virgule flottante matérielle. Les appareils ARMv5 / ARMv6 et les cœurs sans FPU comme `arm_cortex-a9` ne sont pas encore pris en charge.

Vous pouvez aussi déployer manuellement : téléchargez l'archive Linux et utilisez le script de démarrage procd [deploy/openwrt/gately.init](https://github.com/wangwanjie/gately/blob/main/deploy/openwrt/gately.init) :

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

### Mise à niveau en ligne {#online-upgrade}

Les installations binaires Linux (systemd, conteneurs LXC de PVE, OpenWrt déployé manuellement, etc.) peuvent être mises à niveau en ligne depuis la page À propos de l'interface d'administration : cliquez sur « Rechercher des mises à jour », puis sur « Mettre à niveau » si une nouvelle version est disponible. Cela fonctionne aussi pour les conteneurs LXC créés dans PVE à partir de l'image OCI, sans les supprimer ni les recréer. Gately va :

1. Télécharger depuis GitHub le `gately_<version>_linux_<architecture>.tar.gz` de cette plateforme ;
2. Vérifier son SHA-256 avec le `checksums.txt` de la même version ;
3. Tester le nouveau binaire avec `-version` pour s'assurer qu'il s'exécute sur cette machine ;
4. Remplacer l'exécutable et redémarrer avec les mêmes arguments, en conservant la version précédente sous `gately.old` dans le même répertoire.

Tout se déroule dans une boîte de dialogue : elle affiche d’abord les notes de version, puis, une fois la mise à niveau lancée, l’étape en cours, la progression et la vitesse du téléchargement. Tant que le binaire n’est pas remplacé (récupération, téléchargement et vérification), vous pouvez cliquer sur « Annuler la mise à niveau » à tout moment ; le binaire actuel reste intact.

Si GitHub est difficile à joindre, définissez un **proxy de mise à jour** sur la page À propos ou dans Réglages → Réseau et exécution. Il prend en charge `http://`, `https://` et `socks5://`, et la recherche comme le téléchargement passent par lui. Vide, Gately se connecte directement et tient toujours compte de la variable d'environnement `HTTPS_PROXY`.

Sans accès à Internet, récupérez le tar.gz sur la [page de téléchargement](/fr/download) et cliquez sur « Téléverser un paquet de mise à niveau » dans la page À propos. Toute version est acceptée, ce qui permet aussi de revenir à une version antérieure.

La mise à niveau en ligne ne s'exécute que sur action d'un administrateur ; Gately ne télécharge ni n'installe jamais de version de lui-même. Si la nouvelle version ne démarre pas, revenez en arrière manuellement avec la copie conservée (exemple avec systemd) :

```bash
mv /usr/local/bin/gately.old /usr/local/bin/gately
systemctl restart gately
```

Si vous avez accordé au binaire l'accès aux ports bas avec `setcap`, relancez `setcap` après la mise à niveau.

### Autres déploiements

- Docker : `docker pull` de la nouvelle image, puis recréation du conteneur (en conservant le dossier de configuration monté) ;
- fnOS : installez manuellement le `.fpk` plus récent depuis le Centre d'applications ; votre configuration est conservée ;
- Paquets OpenWrt : téléchargez le `.ipk` / `.apk` plus récent et exécutez `opkg install` / `apk add --allow-untrusted` ; votre configuration est conservée (voir [OpenWrt](#openwrt)) ;
- Windows, macOS, FreeBSD : remplacez l'exécutable et redémarrez le service.

Avant de mettre à niveau, exportez une sauvegarde de la configuration ou conservez le dossier de configuration dans git. Les archives et images téléchargées peuvent être vérifiées selon [Vérifier les versions](/en/docs/guide/verify).
