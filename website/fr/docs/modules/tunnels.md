# Tailscale / frp / cloudflared

Ces trois modules intègrent les programmes officiels en tant que processus gérés : redémarrage automatique, journaux collectés dans l'interface et configuration directement depuis l'interface. Gately n'embarque pas ces programmes ; ils s'installent, se mettent à jour et se restaurent en ligne depuis la page.

## Installation, mise à jour et retour arrière en ligne {#install}

Les pages frp, tunnel Cloudflare, Tailscale et [File Browser](/fr/docs/modules/tools#filebrowser) comportent en bas une carte « Installation en ligne » :

- **Télécharger et installer / Mettre à jour** : télécharge la dernière version stable depuis l'amont, la vérifie avec le SHA-256 publié par l'amont, contrôle qu'elle s'exécute sur cette machine, l'installe dans `bin/` sous le répertoire de configuration, puis redémarre les processus qui utilisent le programme. Une version déjà à jour n'est pas retéléchargée ;
- **Installer une version** : saisissez un numéro de version (par exemple `v0.65.0` pour frp, `2025.8.0` pour cloudflared, `1.90.2` pour Tailscale), utile aussi pour revenir à une version antérieure ;
- **Proxy de téléchargement** : réglable par programme, prend en charge http, https et socks5 et est enregistré au lancement de l'installation ; vide, le proxy de mise à jour des « Paramètres » est utilisé ;
- **Versions précédentes et retour arrière** : les versions remplacées par une mise à jour ou un retour arrière sont conservées dans `bin/.versions/`, 3 au plus. Si la nouvelle version pose problème, cliquez sur « Revenir » dans la liste ; la version actuelle est conservée elle aussi, vous pouvez donc y revenir ensuite ;
- **Désinstaller** : supprime le programme installé dans `bin/` et toutes les versions précédentes ; les paramètres du module sont conservés. Les processus qui utilisent le programme redémarrent et se rabattent sur un programme du même nom dans `PATH` ; sans lui, ils ne peuvent pas s'exécuter.

| Programme | Source | Fichiers installés |
| --- | --- | --- |
| frp | GitHub fatedier/frp | `frpc`, `frps` |
| cloudflared | GitHub cloudflare/cloudflared | `cloudflared` |
| Tailscale | Paquets statiques pkgs.tailscale.com, Linux uniquement | `tailscale`, `tailscaled` |
| FileBrowser Quantum | GitHub gtsteffaniak/filebrowser | `filebrowser` |

Pour macOS et Windows, Tailscale ne fournit pas de paquet directement exécutable : utilisez l'installateur officiel, puis indiquez le chemin dans les paramètres. La carte signale quand la plateforme actuelle n'est pas prise en charge.

## Rétablir les valeurs par défaut {#reset}

Le bouton « Rétablir les valeurs par défaut » de l'en-tête est distinct de la désinstallation : il arrête tous les processus du module, remet les paramètres (chemin du programme et proxy de téléchargement compris) à leurs valeurs par défaut et supprime les fichiers que Gately a générés pour lui. Les programmes installés sont conservés.

| Module | Contenu supprimé |
| --- | --- |
| frp | Paramètres du client, du serveur et de tous les proxys, fichiers de configuration générés dans `frp/` |
| Tunnel Cloudflare | Tous les tunnels et les paramètres du programme |
| Tailscale | Tous les paramètres et l'état de connexion du répertoire d'état par défaut ; si le tailscaled géré par Gately tourne, il se déconnecte d'abord. Les répertoires d'état personnalisés et le tailscaled déjà présent sur le système restent intacts |
| File Browser | Toutes les instances et les paramètres du programme, avec les utilisateurs, liens de partage et caches de chaque instance ; les fichiers des répertoires partagés restent intacts |

L'ancienne configuration est conservée dans l'[historique de configuration](/fr/docs/guide/audit) et peut être restaurée en cas d'erreur. Pour tout retirer, désinstallez d'abord le programme, puis rétablissez les valeurs par défaut. Les API sont `DELETE /api/{module}/install` (désinstallation) et `POST /api/{module}/reset` (rétablissement des valeurs par défaut).

## Emplacement des programmes

Les exécutables sont recherchés dans cet ordre :

1. Le chemin défini dans les paramètres du module ;
2. Le sous-répertoire `bin/` du répertoire de configuration (emplacement de l'installation en ligne) ;
3. `PATH`.

Quand un chemin est indiqué, la version installée en ligne n'est pas utilisée et la carte le signale. Vous pouvez aussi placer le binaire vous-même dans le répertoire de configuration, avec Docker par exemple `./gately-data/bin/frpc`. Tailscale nécessite à la fois `tailscale` et `tailscaled` ; avec le tailscaled déjà présent sur le système, la commande `tailscale` installée devrait avoir la même version que le démon.

Les jetons et les clés sont stockés avec les permissions 0600. Le jeton cloudflared est transmis via une variable d'environnement, il n'apparaît donc pas dans la liste des processus.

## Tailscale

- Gère `tailscaled` (réseau en espace utilisateur, adapté aux conteneurs) ou se rattache à un démon système existant ;
- Clé d'authentification ou lien de connexion, Headscale pris en charge ;
- Routes de sous-réseau et nœuds de sortie ;
- Liste des nœuds et proxy SOCKS5 ;
- Port UDP WireGuard fixe, pratique pour autoriser les connexions directes à travers un pare-feu ;
- Arguments supplémentaires de tailscaled et variables d'environnement. Si iptables de l'hôte utilise le backend nf_tables, ajoutez `TS_DEBUG_FIREWALL_MODE=nftables` aux variables d'environnement, faute de quoi le transfert des sous-réseaux ne fonctionnera pas.

## frp

- **Client frpc** : proxys tcp / udp / http / https / stcp / xtcp ; l'ajout ou la suppression d'un proxy est appliqué immédiatement ;
- **Serveur frps** : liste d'autorisation de ports, sous-domaines, tableau de bord ;
- **Adresses d'écoute de frps** : « Adresse d'écoute » définit où écoute le port de connexion de frpc, et « Adresse d'écoute des proxys » où écoutent les ports HTTP / HTTPS et les ports distants des proxys tcp / udp ; vides, elles écoutent sur toutes les adresses. Pour placer frps derrière un proxy inverse comme Gately, mettez `127.0.0.1` comme adresse d'écoute, créez dans le Service web un site en proxy inverse vers `127.0.0.1:<port d'écoute>` et choisissez websocket ou wss comme protocole de transport de frpc vers ce site ; laissez l'adresse d'écoute des proxys vide pour que les ports distants des proxys tcp restent joignables de l'extérieur. À l'import d'une configuration frps sans `proxyBindAddr`, `bindAddr` est repris, comme le fait frps ;
- **Importer une configuration existante** : « Importer la configuration » dans l'en-tête de la page prend en charge les anciens `frpc.ini` / `frps.ini` (y compris la section `[common]`) et le `toml` de frp 0.52+. Vous pouvez choisir un fichier ou coller son contenu. Vérifiez d'abord l'aperçu du résultat et des champs ignorés, puis confirmez l'import. Pour le client, vous pouvez choisir d'ajouter les proxys (ceux du même nom restent inchangés) ou de remplacer tous les proxys. L'import ne modifie pas l'état d'activation et peut aussi être appelé via l'API avec `POST /api/frp/import`.

## Tunnel Cloudflare

- Tunnels à jeton et tunnels temporaires trycloudflare ;
- Plusieurs tunnels avec redémarrage automatique ;
- Affichage de l'adresse publique.

## Comment choisir

| Scénario | Recommandation |
| --- | --- |
| Construire un réseau privé entre plusieurs appareils | Tailscale |
| Vous disposez d'un serveur public et souhaitez exposer des services internes à travers lui | frp |
| Pas d'IP publique ni de serveur, et vous voulez exposer rapidement un service web | Tunnel Cloudflare |
| Votre FAI fournit un NAT qui autorise le hole punching | [Traversée STUN](/fr/docs/modules/stun) |
