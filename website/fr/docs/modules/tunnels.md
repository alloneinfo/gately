# Tailscale / frp / cloudflared

Ces trois modules intègrent les programmes officiels en tant que processus gérés : redémarrage automatique, journaux collectés dans l'interface et configuration directement depuis l'interface. Gately n'embarque pas ces programmes, vous pouvez donc toujours utiliser la dernière version publiée en amont.

## Emplacement des programmes

Les exécutables sont recherchés dans cet ordre :

1. Le chemin défini dans les paramètres du module ;
2. `PATH` ;
3. Le sous-répertoire `bin/` du répertoire de configuration.

Pour un déploiement Docker, placez le binaire dans le répertoire de configuration monté, par exemple `./gately-data/bin/frpc`. Tailscale nécessite à la fois `tailscale` et `tailscaled`.

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
