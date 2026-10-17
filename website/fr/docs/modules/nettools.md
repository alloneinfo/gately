# Diagnostic réseau

Exécutez Ping, traceroute, test de port, requête DNS, test de débit et analyse du réseau local dans votre navigateur. Emplacement dans le menu : « Outils → Diagnostic réseau » dans la barre latérale.

## Tâches en arrière-plan

Chaque clic sur « Démarrer » crée une tâche en arrière-plan. L'API renvoie immédiatement l'ID de la tâche, et l'interface interroge régulièrement la sortie.

- Jusqu'à 4 tâches s'exécutent simultanément ; au-delà, un message « réessayez plus tard » s'affiche ;
- Une tâche s'exécute au maximum 5 minutes et est annulée automatiquement en cas de dépassement ;
- Les 50 dernières tâches sont conservées, et les tâches en cours ne sont jamais nettoyées ; chaque tâche conserve au plus 500 lignes de sortie ;
- Une tâche en cours peut être annulée avec « Arrêter ». « Tâches récentes » en bas de la page liste les tâches passées, et un clic sur l'une d'elles affiche sa sortie ;
- États des tâches : en cours, terminé, échec, annulé.

## Ping

Envoie des requêtes ICMP echo à la cible et affiche chaque temps d'aller-retour, le taux de perte de paquets et la latence minimale / moyenne / maximale.

| Paramètre | Description |
| --- | --- |
| Cible | Nom d'hôte ou IP, IPv4 et IPv6 pris en charge |
| Nombre | 4 par défaut, jusqu'à 20 |

Chaque requête expire après 2 secondes, avec un intervalle d'environ 0,7 seconde. Un socket ICMP non privilégié de type UDP est essayé en premier, puis un socket brut. Dans un conteneur Docker, cela nécessite la capability `NET_RAW` ou le réglage `net.ipv4.ping_group_range` ; si les permissions sont insuffisantes, utilisez plutôt le test de port. Si la cible ne répond pas du tout, la tâche se termine en échec.

## Traceroute

Appelle une commande système pour tracer le chemin et affiche le résultat brut de la commande.

- Windows utilise `tracert` ; les autres systèmes préfèrent `traceroute`, puis `tracepath` ;
- 20 sauts au maximum, sans résolution inverse (les IP sont affichées directement) ;
- Si le système n'a aucune commande correspondante, la tâche signale une erreur. Dans les images Docker, vous pouvez installer le paquet `traceroute`.

## Test de port

Effectue des tentatives de connexion TCP vers la cible et ne liste que les ports ouverts.

| Paramètre | Description |
| --- | --- |
| Cible | Nom d'hôte ou IP |
| Ports | Par exemple `22,80,443,8000-8100` ; les listes séparées par des virgules et les plages sont prises en charge, avec des ports de 1 à 65535 |

- 1024 ports au maximum par test ;
- Concurrence de 100, avec un délai de connexion de 1,5 seconde par port ;
- Les ports courants affichent le nom du service (ssh, http, https, mysql, redis, postgres, rdp, etc.) et le temps de connexion.

## Requête DNS

| Paramètre | Description |
| --- | --- |
| Domaine / IP | Le nom à interroger ; saisissez une IP pour le type `PTR` |
| Type | `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, `PTR` ; `A` par défaut |
| Serveur DNS | Laissez vide pour utiliser le résolveur système ; vous pouvez saisir `223.5.5.5`, `223.5.5.5:5353` ou une adresse DoH comme `https://dns.alidns.com/dns-query` |

Les requêtes expirent après 8 secondes et affichent les enregistrements et le temps écoulé.

## Test de débit

Un test de débit en téléchargement avec 4 flux simultanés. Il affiche le débit courant (Mbps) chaque seconde et, à la fin, le total téléchargé et le débit moyen.

| Paramètre | Description |
| --- | --- |
| URL de téléchargement | Une URL `http(s)://` complète. Laissez vide pour utiliser le point de terminaison de test de débit de Cloudflare (`https://speed.cloudflare.com/__down?bytes=200000000`) |
| Durée | En secondes, de 2 à 30 dans l'interface, 8 par défaut |

Les requêtes du test de débit ne suivent pas les redirections, elles ne peuvent donc pas être détournées vers des adresses interdites au sondage.

## Analyse du réseau local

Analyse un sous-réseau pour trouver les appareils en ligne et affiche les résultats dans un tableau : IP, MAC, nom d'hôte, ports ouverts.

| Paramètre | Description |
| --- | --- |
| Sous-réseau | CIDR, par exemple `192.168.1.0/24`. Laissez vide pour utiliser le sous-réseau de la première interface IPv4 privée de cette machine (la liste déroulante affiche les sous-réseaux locaux disponibles, et les sous-réseaux trop grands sont réduits à /24) |

Limites et fonctionnement :

- Seuls les sous-réseaux privés IPv4 sont pris en charge (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`) ;
- 1024 adresses au maximum par analyse, soit l'équivalent d'un /22 ; les adresses de réseau et de diffusion sont exclues automatiquement ;
- Pour chaque adresse, les ports courants 80, 443, 22, 445, 139, 3389, 8080, 62078, 5000 et 53 sont sondés, et une connexion réussie ou refusée vaut présence en ligne ;
- Les adresses MAC proviennent de la table de voisinage du système, et un hôte qui figure dans la table de voisinage mais n'a aucun port courant ouvert est également listé comme en ligne ;
- Les noms d'hôte sont résolus par DNS inverse (délai de 1,5 seconde) ;
- Pour les appareils disposant d'une MAC, cliquez sur « Ajouter à Wake-on-LAN » pour les ajouter à la liste d'appareils de [Wake-on-LAN](/fr/docs/modules/tools).

## Liste de blocage des sondages

« Cibles bloquées » en haut à droite de la page définit les cibles auxquelles les outils de diagnostic n'ont pas le droit d'accéder.

- Les adresses de métadonnées des fournisseurs cloud sont toujours bloquées : `169.254.169.254/32`, `100.100.100.200/32`, `fd00:ec2::254/128`, ce qui empêche d'utiliser ces outils pour voler les identifiants d'une instance ;
- Vous pouvez ajouter des IP ou des sous-réseaux, un par ligne, par exemple `10.0.0.0/8` ;
- Si l'une des adresses auxquelles se résout le nom d'hôte de la cible correspond à la liste, la tâche échoue avec une erreur. L'URL de téléchargement du test de débit est également vérifiée, et les analyses ignorent les adresses correspondantes.

::: tip Astuce
Les outils de diagnostic s'exécutent sur l'hôte ou le conteneur où tourne Gately ; les résultats reflètent donc le réseau de cet environnement. Pour limiter ce qui peut être sondé, ajoutez les sous-réseaux internes sensibles à la liste de blocage des sondages.
:::
