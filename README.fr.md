# Gately

[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md) | [한국어](README.ko.md) | [Español](README.es.md) | **Français** | [Deutsch](README.de.md) | [Русский](README.ru.md) | [Português (Brasil)](README.pt-BR.md)

Boîte à outils de passerelle réseau open source et auditable, pour les réseaux domestiques, les NAS, les hôtes cloud et toutes sortes de serveurs. Inspirée de [Lucky](https://github.com/gdy666/lucky), elle offre des fonctionnalités équivalentes de manière entièrement open source. DDNS, traversée NAT STUN, redirection de ports, reverse proxy, certificats, WAF et autres fonctions courantes sont réunis dans un seul fichier d'environ 22 Mo, avec import de la configuration Lucky en un clic.

## Pourquoi Gately

Lucky est un excellent outil, bien connu des utilisateurs chinois de NAS et de réseaux domestiques : il réunit DDNS, traversée NAT STUN, reverse proxy et bien d'autres fonctions. Il a aussi fourni l'idée de départ de Gately : sans Lucky, Gately n'existerait pas. Lucky est distribué gratuitement mais en source fermée ; Gately choisit d'être entièrement open source afin que les utilisateurs et la communauté puissent examiner directement le code source et la configuration. Pour un logiciel qui s'exécute à la frontière du réseau et manipule des clés DNS et des clés privées de certificats, l'open source permet de voir ce qu'il fait. Les engagements de Gately :

- Le code source est publié sous licence Apache-2.0 et ouvert à l'examen de la communauté
- Aucune connexion aux serveurs du projet et aucun envoi d'informations sur l'appareil par défaut ; la connexion via un tiers utilise votre propre application OAuth, sans intermédiaire
- Toutes les adresses externes (interrogation de l'IP publique, serveurs STUN, etc.) sont visibles, modifiables et désactivables
- La configuration est du JSON en clair, que vous pouvez placer dans git pour l'audit et le retour en arrière

## Fonctionnalités

| Module | Capacités |
| --- | --- |
| DNS dynamique | Alibaba Cloud, Cloudflare, Tencent Cloud, DNSPod, Huawei Cloud, GoDaddy, Porkbun, dynv6, DuckDNS, rappel personnalisé ; récupération de l'IP par URL/interface réseau/commande ; IPv4/IPv6 ; Webhook |
| Traversée STUN | TCP/UDP, keep-alive, UPnP/NAT-PMP, redirection intégrée, déclenchement de Webhook et de scripts lors d'un changement d'adresse |
| Redirection de ports | TCP/UDP, IPv4/IPv6, plages de ports, basculement multi-cibles, limite de connexions, statistiques de trafic |
| Certificats SSL | Émission et renouvellement automatiques ACME DNS-01 (wildcard, EAB), téléversement, lecture depuis un fichier, copie vers un répertoire avec exécution de scripts |
| Tailscale | Gère tailscaled (réseau en espace utilisateur, adapté aux conteneurs) ou se connecte à un démon système existant ; clé d'authentification/lien de connexion, Headscale, routes de sous-réseau, nœud de sortie, liste des nœuds et proxy SOCKS5, port fixe, arguments supplémentaires et variables d'environnement de tailscaled |
| frp | Client frpc (proxys tcp/udp/http/https/stcp/xtcp, ajout et suppression pris en compte immédiatement) et serveur frps (liste blanche de ports, sous-domaines, tableau de bord) ; import en un clic des configurations ini anciennes et toml récentes |
| Tunnel Cloudflare | Tunnel à jeton cloudflared et tunnel temporaire trycloudflare, plusieurs tunnels, redémarrage automatique, affichage de l'adresse publique |
| Service web | Reverse proxy/redirection/fichiers statiques/texte fixe par domaine ; HTTPS ; WebSocket ; envoi et réception du protocole PROXY ; authentification de base, authentification web, connexion GitHub/Google/QQ/OIDC ; listes blanche et noire d'IP, règles par chemin, liens de partage temporaires, bannissement automatique après échecs d'authentification, filtrage par pays GeoIP ; poids des backends/secours/politiques d'équilibrage de charge, contrôles de santé actifs et passifs, compression, limitation de débit, cache des réponses, réécriture de chemins ; statistiques d'accès à la minute et métriques Prometheus |
| Pare-feu WAF | Coraza + OWASP CRS v4, activation des règles par fichier, SecLang personnalisé, blocage/détection seule |
| Gestion Docker | Création et édition de conteneurs, journaux et terminal, Compose, vérification des mises à jour d'images et mise à niveau en un clic (retour arrière automatique en cas d'échec), réseaux et volumes |
| Terminal web | SSH (confiance dans la clé d'hôte à la première connexion), Telnet, shell local (désactivé par défaut) ; proxy, sessions persistantes, gestion de fichiers SFTP, 2FA et confirmation du protocole, commandes rapides, écran partagé |
| Diagnostic réseau | Ping, traceroute, sondage de ports, requêtes DNS, test de débit, scan du réseau local |
| Centre de notifications | Webhook, Telegram, Bark, WeCom, DingTalk, Feishu, e-mail ; niveaux, jokers et règles de mise en sourdine et de déduplication |
| Sauvegarde de la configuration | Sauvegardes chiffrées (scrypt + AES-GCM), politique de rétention, sauvegardes planifiées vers un répertoire local ou un point de montage de stockage |
| Ligne de commande et MCP | `gately ctl` interroge une instance en cours d'exécution ; `gately mcp` fournit un service MCP via stdio (lecture seule par défaut) |
| Wake-on-LAN | Diffusion de paquets magiques, détection de présence en ligne |
| WebDAV | Multi-utilisateur, répertoire indépendant par utilisateur, lecture seule |
| FTP | Multi-utilisateur, répertoire indépendant par utilisateur, lecture seule, prise en charge de FTPS et d'une plage de ports passifs |
| Gestion du stockage | Répertoires locaux et montages WebDAV distants, gestion de fichiers en ligne |
| Tâches planifiées | Expressions cron, commandes ou requêtes HTTP, historique d'exécution |
| Sécurité et paramètres | Authentification à deux facteurs (TOTP), limitation des échecs de connexion, point d'entrée sécurisé, accès réservé au réseau interne, HTTPS pour l'interface d'administration, plusieurs jetons d'API (autorisations en lecture/écriture par module), DNS personnalisé (DoH pris en charge), fuseau horaire, sauvegarde et restauration de la configuration |
| Migration | Import du `.lkcf` de Lucky ou du zip de configuration complète : redirection de ports, DDNS, STUN, certificats, reverse proxy, WAF, Wake-on-LAN, terminal web, WebDAV, paramètres de base |

## Démarrage rapide

### Docker

L'image prend en charge amd64, arm64 et arm/v7 :

```bash
docker run -d --name gately --network host \
  -v ./gately-data:/etc/gately \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep -i "initial password"
```

Ouvrez `http://<IP de l'appareil>:16680`, connectez-vous avec `admin` et le mot de passe initial affiché dans les journaux, puis changez le mot de passe et activez l'authentification à deux facteurs dans « Paramètres ». Un exemple Compose complet est disponible dans [deploy/docker-compose.yml](deploy/docker-compose.yml).

La redirection de ports, STUN et le service web doivent écouter sur les ports de l'hôte ; `--network host` est recommandé. Si vous n'avez pas besoin de la gestion Docker, vous pouvez supprimer le montage de `docker.sock`.

### Binaire

Téléchargez l'archive correspondant à votre plateforme depuis [Releases](https://github.com/wangwanjie/gately/releases) (Linux amd64/arm64/arm/mips/mipsle/riscv64, macOS, Windows, FreeBSD), décompressez-la puis exécutez :

```bash
./gately -c /etc/gately
```

Paramètres : `-c` répertoire de configuration (la variable d'environnement `GATELY_CONFIG_DIR` fonctionne aussi), `-listen` remplace l'adresse d'écoute de l'interface d'administration, `-version` affiche la version.

### Appels d'API

Créez un jeton dans la page « Jetons d'API » de la console d'administration (plusieurs jetons possibles, avec lecture seule ou lecture-écriture par module, date d'expiration configurable ; la valeur en clair n'est affichée qu'une seule fois après la création), puis vos scripts peuvent appeler l'API d'administration en masse :

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks
```

Les API de compte, d'authentification à deux facteurs, de gestion des jetons, ainsi que celles de sauvegarde, de restauration, de redémarrage et de paramètres système ne fonctionnent qu'avec une session connectée. L'ensemble des API, les autorisations requises et les structures de données sont décrits dans la page « Documentation de l'API » de la console d'administration ; vous pouvez aussi les exporter avec `make api-docs` vers [docs/api](docs/api/fr) (`openapi.json` et `API.md`), que le site https://gately.cn référence directement lors de sa publication.

Lors d'une mise à niveau depuis une ancienne version, l'ancien jeton d'API global devient automatiquement un « jeton hérité » (valeur inchangée, avec toutes les autorisations de module). Il est recommandé de restreindre ses autorisations ou de le remplacer par un nouveau jeton dans la page des jetons d'API.

### Tailscale, frp, cloudflared

Ces trois modules gèrent des programmes externes officiels (redémarrage automatique, journaux regroupés dans l'interface). Gately ne les embarque pas, ce qui permet d'utiliser toujours la dernière version amont. Les programmes sont recherchés dans cet ordre : le chemin indiqué dans les paramètres du module, `PATH`, puis le sous-répertoire `bin/` du répertoire de configuration. Avec un déploiement Docker, placez simplement le binaire dans le répertoire de configuration monté, par exemple `./gately-data/bin/frpc` ; Tailscale nécessite de fournir à la fois `tailscale` et `tailscaled`. Les jetons et clés des fichiers de configuration sont enregistrés avec les permissions 0600, et le jeton de cloudflared est transmis par variable d'environnement, il n'apparaît donc pas dans la liste des processus.

### Migrer depuis Lucky

Consultez le [guide de migration](https://gately.cn/fr/docs/guide/migration). En bref : dans « Paramètres → Importer depuis Lucky », sélectionnez le fichier `lucky_*.lkcf` du répertoire de configuration de Lucky, vérifiez l'aperçu puis importez ; les règles importées sont désactivées par défaut, désactivez-les dans Lucky et activez-les dans Gately module par module.

## Développement

Prérequis : Go 1.26+, Node.js 24+, pnpm.

```bash
make web      # compile le frontend (sortie dans web/dist, intégré via go:embed)
make build    # compile bin/gately
make test     # go test -race ./...
make lint     # gofmt, go vet, vérification des types du frontend
```

Pour développer le frontend, lancez `cd web && pnpm dev` ; `/api` est relayé vers `127.0.0.1:16680` en local.

Certains tests sont ignorés par défaut et nécessitent des conditions supplémentaires :

| Test | Activation |
| --- | --- |
| ACME de bout en bout (Pebble) | Installer pebble et pebble-challtestsrv, définir `GATELY_PEBBLE_BIN` et `GATELY_PEBBLE_SRC` (exécuté automatiquement en CI) |
| Serveurs STUN publics réels | `GATELY_NET_TEST=1` |
| Vérification de l'import avec une vraie configuration Lucky | `GATELY_LUCKY_SAMPLE`, etc. ; voir `TestImportRealSample` dans chaque module |

La structure des répertoires et les choix techniques sont décrits dans [docs/ROADMAP.md](docs/ROADMAP.md).

Le code source du site web ([gately.cn](https://gately.cn)) se trouve dans `website/` et repose sur VitePress : `cd website && pnpm install && pnpm dev` ; `pnpm deploy:site` compile et synchronise vers le serveur.

## Règles de contribution

- La compatibilité avec Lucky repose sur des comportements, interfaces et formats de configuration publiquement observables. Ne soumettez pas de code copié depuis les binaires ou les artefacts frontend de Lucky.
- Un nouveau module nécessite : une structure de configuration avec validation, un démarrage/arrêt à l'exécution, une interface REST et des tests unitaires ; les modules migrables depuis Lucky implémentent `luckyimport.Importer`, et ceux qui souhaitent afficher leur état sur la page de synthèse implémentent `module.Summarizer`.
- Les commentaires de code sont rédigés en chinois ; les instructions de contrôle doivent toujours utiliser des accolades.

## Historique des étoiles

[![Star History Chart](https://api.star-history.com/svg?repos=wangwanjie/gately&type=Date)](https://star-history.com/#wangwanjie/gately&Date)

## Licence

[Apache License 2.0](LICENSE)
