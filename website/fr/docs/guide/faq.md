# FAQ

## Faut-il se connecter aux serveurs du projet ?

Non. Par défaut, Gately ne se connecte à aucun serveur du projet et ne remonte aucune information sur l'appareil. Les adresses externes, comme la recherche de l'IP publique, utilisent des services publics tiers et se modifient dans les tâches correspondantes.

## Quels sont le compte et le mot de passe par défaut ?

Le compte par défaut est `admin` et le mot de passe par défaut `gately.cn`. Ce mot de passe est public : changez-le dans « Paramètres → Compte et sécurité » dès la première connexion ; d'ici là, un avertissement reste affiché en haut de l'interface d'administration. Si la variable d'environnement `GATELY_INITIAL_PASSWORD` a été définie lors du déploiement, c'est ce mot de passe qui s'applique.

## Que faire si j'ai oublié le mot de passe administrateur ?

Exécutez `gately reset-password` sur le serveur (avec Docker : `docker exec -it gately gately reset-password`) et tapez `yes` pour confirmer. Le mot de passe redevient le mot de passe par défaut `gately.cn` ; une instance en cours d'exécution applique le changement en quelques secondes. Consultez [Réinitialiser le mot de passe et l'authentification à deux facteurs](/fr/docs/guide/install#reset).

## Que faire si j'ai perdu mon application d'authentification ?

Exécutez `gately reset-totp` sur le serveur (avec Docker : `docker exec -it gately gately reset-totp`) et confirmez. L'authentification à deux facteurs est désactivée : connectez-vous avec le mot de passe, puis réactivez-la.

## fnOS est-il pris en charge ?

Oui. Chaque version fournit des paquets fnOS hors ligne (`.fpk`) à installer manuellement depuis le Centre d'applications. Gately s'intègre à la passerelle unifiée fnOS : il s'ouvre depuis l'icône du bureau et fonctionne aussi via votre domaine fnOS distant. Activez « Connexion via fnOS » dans « Paramètres → Contrôle d'accès » et les administrateurs fnOS déjà connectés entrent directement, sans mot de passe ni code à deux facteurs. Consultez [Application native fnOS](/fr/docs/guide/install#fnos).

## OpenWrt / iStoreOS est-il pris en charge ?

Oui. Chaque version fournit des paquets OpenWrt : le `.ipk` pour OpenWrt 24.10 et antérieur (y compris iStoreOS), installé avec `opkg install`, et le `.apk` pour OpenWrt 25.12 et ultérieur, installé avec `apk add --allow-untrusted`. Le service démarre dès l'installation et conserve sa configuration dans `/etc/gately` ; pour mettre à jour, installez le nouveau paquet. Le binaire pèse environ 35 Mo, ce qui convient aux routeurs logiciels disposant d'un stockage confortable. Voir [OpenWrt](/fr/docs/guide/install#openwrt).

## Pourquoi recommander `--network host` ?

La redirection de ports, STUN et les services Web écoutent directement sur les ports de l'hôte. Avec le réseau bridge, il faut publier soi-même chaque port utilisé.

## Peut-il fonctionner en parallèle de Lucky ?

Oui. Le port d'administration est par défaut 16680, sans conflit avec le 16601 de Lucky. Les règles importées sont désactivées par défaut pour ne pas entrer en concurrence avec un Lucky encore actif sur les ports et les mappages UPnP du routeur.

## Pourquoi dois-je fournir moi-même les binaires de Tailscale, frp et cloudflared ?

Ces trois modules supervisent les programmes externes officiels. Gately ne les embarque pas, ce qui permet d'utiliser à tout moment la dernière version amont. Consultez [Configuration et options d'exécution](/fr/docs/guide/configuration) pour savoir où les placer.

## Ma règle STUN est activée mais je ne peux toujours pas me connecter de l'extérieur

Si l'interface WAN de votre routeur se trouve derrière un NAT de niveau opérateur (100.64.0.0/10), les hôtes extérieurs ne peuvent pas atteindre le port traversé. C'est une limite du réseau lui-même. Utilisez plutôt [Tailscale / frp / tunnel Cloudflare](/fr/docs/modules/tunnels).

## Comment signaler un problème ou contribuer ?

Ouvrez un ticket sur [GitHub Issues](https://github.com/wangwanjie/gately/issues). Les conventions de développement figurent dans [Contribuer](/fr/docs/guide/contributing).
