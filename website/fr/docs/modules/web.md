# Services Web

Servez un proxy inverse, des redirections, des fichiers statiques et du texte fixe par domaine. C'est le point d'entrée web unifié de Gately. Emplacement dans le menu : barre latérale « Web → Services Web ».

## Structure : écouteurs, sites et site par défaut

La configuration comporte trois niveaux :

1. Écouteur : un port d'écoute. Les champs comprennent le réseau (`tcp` double pile, `tcp4`, `tcp6`), l'adresse d'écoute (vide pour toutes les adresses), le port, l'interrupteur HTTPS, la version TLS minimale (1.0 / 1.1 / 1.2 / 1.3) et l'acceptation ou non du protocole PROXY.
2. Site : un site rattaché à un écouteur, sélectionné par domaine. Un domaine peut être un domaine exact, `*.example.com` (correspond aux sous-domaines à n'importe quelle profondeur) ou `*` (correspond à tout). Priorité de correspondance : domaine exact, puis joker, puis `*`.
3. Site par défaut : utilisé lorsqu'aucun site ne correspond. S'il n'est pas activé, une erreur 404 est renvoyée.

## Types de sites

| Type | Description |
| --- | --- |
| Proxy inverse | Transmet à un ou plusieurs backends. Les adresses de backend ont la forme `http(s)://host:port`, et WebSocket est pris en charge |
| Redirection | Redirige vers une URL complète avec le code de statut 301 / 302 / 307 / 308 (302 par défaut), en conservant éventuellement le chemin et la chaîne de requête d'origine |
| Fichiers statiques | Le répertoire doit être un chemin absolu. Avec « Liste du répertoire » désactivée, une requête sur un répertoire n'obtient de réponse que si `index.html` existe |
| Texte fixe | Contenu personnalisé, `Content-Type` (`text/plain; charset=utf-8` par défaut) et code de statut (100-599, 200 par défaut) |

Autres options des sites en proxy inverse :

- Utiliser l'hôte du backend : l'en-tête Host est remplacé par l'adresse du backend ; lorsque l'option est désactivée, le Host de la requête du client est conservé ;
- Ignorer la vérification du certificat du backend, désactiver Keep-Alive ;
- Envoyer le protocole PROXY au backend (en-tête v2, transmet l'IP réelle du client, nécessite la prise en charge côté backend) ;
- Ajouter `X-Forwarded-For` / `X-Forwarded-Proto` / `X-Forwarded-Host`.

Les sites de tous types peuvent définir un journal d'accès, le cross-origin (CORS), des en-têtes de requête et de réponse personnalisés (un `Key: Value` par ligne ; une Value vide supprime cet en-tête), et peuvent référencer une instance de [Pare-feu WAF](/fr/docs/modules/waf) (si l'instance n'existe pas ou est désactivée, les requêtes passent sans filtrage).

## Transport et protocoles

- HTTPS : utilise un certificat du module [Certificats SSL](/fr/docs/modules/ssl) qui couvre le domaine ;
- HTTP/3 (QUIC) : à activer sur un écouteur HTTPS pour servir aussi HTTP/3 en UDP sur le même numéro de port, annoncé au navigateur par l'en-tête de réponse `Alt-Svc` ; ce port UDP doit être autorisé dans le pare-feu et la redirection de ports ;
- WebSocket : pris en charge automatiquement par le proxy inverse ;
- Protocole PROXY : un écouteur peut accepter les en-têtes PROXY envoyés par des composants en amont (comme haproxy et frp). Il ne fait confiance qu'aux IP ou sous-réseaux listés dans « Paramètres globaux → Sources PROXY de confiance » ; une liste vide ne fait confiance à aucune source. Le proxy inverse peut envoyer des en-têtes PROXY au backend.

## Authentification

À configurer dans la page « Sécurité » de l'éditeur de site :

- Authentification web : le navigateur affiche une page de connexion ;
- Authentification Basic : HTTP Basic ;
- Utilisateurs : un `username:password` par ligne, le mot de passe pouvant être un hash bcrypt. L'authentification Basic exige au moins un utilisateur lorsqu'elle est activée ; l'authentification web exige au moins un utilisateur ou une identité de connexion tierce lorsqu'elle est activée ;
- Connexion tierce : GitHub, Google, QQ et OIDC sont pris en charge, à partir de vos propres applications OAuth. Les identités autorisées s'écrivent une par ligne sous la forme `provider:account`, par exemple `github:octocat` et `google:me@gmail.com`. Un `#` en fin de ligne ajoute un commentaire, et `github:*` désigne n'importe quel compte de ce fournisseur.

Les fournisseurs de connexion tierce se configurent dans « Paramètres globaux » : saisissez votre propre Client ID / Secret et activez le fournisseur, puis définissez aussi une adresse de rappel (elle doit être servie par l'un des écouteurs de Gately, par exemple `https://auth.example.com`). L'URL de rappel à déclarer auprès de GitHub / Google / QQ est :

```text
https://auth.example.com/__gately_auth/oauth/callback
```

La connexion tierce pour l'interface d'administration elle-même se configure séparément dans « Paramètres → Compte et sécurité ».

## Contrôle d'accès

### Liste d'autorisation et liste de blocage d'IP

Le « Contrôle d'accès par IP » de la page « Sécurité » du site peut être désactivé, en liste d'autorisation ou en liste de blocage. La liste accepte une IP ou un sous-réseau par ligne (comme `192.168.1.0/24`).

### Règles de chemin

Dans « Règles de chemin » de la page « Sécurité », remplacez l'authentification et les paramètres d'IP du site par préfixe de chemin :

| Champ | Valeurs |
| --- | --- |
| `prefix` | Un préfixe de chemin commençant par `/`, comme `/admin` |
| `auth` | Hériter du site (vide), `off` (aucune authentification), `on` (authentification requise) |
| `ipMode` / `ipList` | Hériter du site (vide), `whitelist` ou `blacklist`, avec une liste d'IP ou de sous-réseaux |

Règles de correspondance :

- Le préfixe le plus long l'emporte ;
- La correspondance respecte les frontières de segments de chemin : `/admin` correspond à `/admin` et `/admin/x`, mais pas à `/administrator` ;
- Le chemin de la requête est d'abord normalisé (`.`, `..` et les barres obliques en double sont supprimés) puis comparé, de sorte que `/public/../admin` est traité comme `/admin` ;
- Une seule règle peut être configurée pour un même préfixe.

L'option « Portée de l'authentification » (`authOptional`) : une fois cochée, l'authentification ne s'applique qu'aux chemins marqués « Requis » dans les règles, et tous les autres chemins sont publics. Cette option nécessite d'activer d'abord l'authentification Basic ou web.

### Liens de partage temporaires

Une fois l'authentification Basic ou web activée, vous pouvez créer des liens de partage pour un site afin que des visiteurs y accèdent sans compte :

| Champ | Description |
| --- | --- |
| Nom | Une étiquette, par exemple « Pour Alice » |
| Expiration | Laissez vide pour aucune expiration |
| Utilisations `maxUses` | Nombre maximal de fois où le lien peut être utilisé ; 0 signifie illimité. Le compteur est réinitialisé après un redémarrage |
| Limite de chemin `pathPrefix` | Lorsqu'il n'est pas vide, le lien ne donne accès qu'aux chemins sous ce préfixe ; il doit commencer par `/` |

Un visiteur ouvre `https://domain/any-path?_gs=token` pour obtenir une session. Après une utilisation réussie, le serveur dépose un cookie d'accès sans connexion et redirige vers l'adresse sans le paramètre `_gs`. La session dure au plus 24 heures et jamais au-delà de l'expiration du lien. Un jeton invalide est traité comme une visite normale et ne révèle pas la cause de l'échec. Les sessions déjà émises deviennent immédiatement invalides une fois le lien supprimé. Le jeton est généré après l'enregistrement (16 caractères au minimum), et le lien complet peut être copié dans l'interface.

### Limitation de débit des échecs d'authentification et bannissement automatique

- Après 10 échecs d'authentification depuis la même IP source en 15 minutes, les tentatives suivantes sont rejetées pendant un certain temps ;
- Sous « Paramètres globaux → Bannissement automatique → Bannir après échec d'authentification », saisissez un nombre de minutes (0 à 43200) ; l'IP source est bannie à 10 échecs. 0 signifie aucun bannissement, limitation de débit uniquement ;
- Un événement `AuthBanned` est publié lorsqu'un bannissement automatique se déclenche, et il peut être envoyé via le [Centre de notifications](/fr/docs/modules/notify) ;
- Les adresses IPv6 sont comptées et bannies par sous-réseau /64.

### Liste de bannissement

La liste de bannissement dans « Paramètres globaux » affiche les IP actuellement bannies, la raison (échecs d'authentification répétés ou bannissement manuel) et l'heure d'expiration, et chaque entrée peut être levée avec « Débannir ». Vous pouvez aussi bannir manuellement : saisissez une IP et un nombre d'heures (1 à 720, soit 30 jours au maximum) ; l'API utilise 24 heures par défaut.

### Filtre de pays GeoIP

Le « Filtre de pays » d'un site propose trois modes : désactivé, autoriser uniquement et refuser. Saisissez des codes pays ISO à deux lettres séparés par des virgules, comme `CN, HK, US`. Au moins un pays est requis lorsque le filtre est activé.

- Le filtrage dépend d'une base GeoIP, que vous devez d'abord importer sous « Paramètres globaux → Base GeoIP » : téléversez un fichier `.mmdb`, ou saisissez une URL de téléchargement. MaxMind GeoLite2, DB-IP et d'autres bases de pays ou de villes sont pris en charge ; la limite de téléversement est de 129 Mo ;
- Lorsqu'aucune base n'est chargée, le filtrage n'a aucun effet (tous les visiteurs passent) et l'interface affiche un avertissement ;
- Les adresses de réseaux privés sont toujours autorisées ;
- En mode « Refuser », les adresses dont le pays est introuvable dans la base sont autorisées.

## Améliorations du proxy inverse

Ces options se trouvent dans la page « Trafic » de l'éditeur de site.

### Backends, poids et secours

La liste des backends accepte une adresse par ligne, éventuellement suivie de paramètres :

```text
http://10.0.0.2:8080 weight=3
http://10.0.0.3:8080 backup
```

- `weight=N` : poids, un entier de 1 à 100, 1 par défaut ;
- `backup` : un backend de secours, utilisé uniquement lorsque tous les backends hors secours sont indisponibles. Lorsque tous les backends sont indisponibles, tous ceux hors secours sont réessayés.

### Politique de répartition de charge

| Option de l'interface | `lbPolicy` | Description |
| --- | --- | --- |
| Tourniquet (pondéré) | Vide | Par défaut |
| Aléatoire (pondéré) | `random` | |
| Hash d'IP (sessions persistantes) | `ip_hash` | Une même IP cliente va toujours vers le même backend |
| Moins de connexions | `least_conn` | Choisit la plus petite valeur de connexions / poids |

### Vérifications d'état

L'interrupteur de vérification active est désactivé par défaut. Champs et valeurs par défaut une fois activée :

| Champ | Valeur par défaut | Plage |
| --- | --- | --- |
| Mode `mode` | `http` | `http` (requête sur le chemin de vérification ; un code de statut inférieur à 400 est considéré comme sain) ou `tcp` (teste seulement la connexion) |
| Chemin `path` | `/` | Commence par `/`, mode http uniquement |
| Intervalle `interval` | 10 secondes | 1 à 3600 |
| Délai d'attente `timeout` | 3 secondes | 1 à 60 |
| Échecs consécutifs `fails` | 3 échecs le marquent indisponible | 1 à 20 |
| Succès consécutifs `passes` | 2 succès le rétablissent | 1 à 20 |

Lorsque l'envoi du protocole PROXY au backend est activé, les vérifications HTTP passent automatiquement en TCP.

Les vérifications passives sont toujours actives : parmi plusieurs backends, un backend ayant subi 3 échecs de connexion consécutifs est mis en pause 30 secondes avant d'être sondé de nouveau (un backend unique sans vérification active n'est jamais retiré). Une fois les vérifications actives activées, le rétablissement dépend entièrement de leurs résultats. Lorsque l'état d'un backend change, des événements upstream-down et upstream-recovered sont publiés, et ils peuvent être envoyés via le [Centre de notifications](/fr/docs/modules/notify).

### Compression

La compression peut être activée pour les sites en proxy inverse, en fichiers statiques et en texte fixe (pas pour les sites de redirection). Selon `Accept-Encoding`, `br` est préféré, puis `gzip`.

- Seul le contenu de type texte est compressé : `text/*`, JSON, JavaScript, XML, XHTML, RSS / Atom, WebAssembly et SVG ; `text/event-stream` est exclu ;
- Les réponses dont le `Content-Length` déclaré est inférieur à 256 octets sont laissées telles quelles ;
- Les réponses ayant déjà un `Content-Encoding` sont laissées telles quelles ; les requêtes de mise à niveau WebSocket passent directement ;
- Après compression, les ETag forts sont convertis en ETag faibles.

### Limitation de débit par IP

Limitation de débit à jeton (token bucket) par IP cliente. Champs :

| Champ | Description |
| --- | --- |
| `rate` | Requêtes autorisées par seconde |
| `burst` | Capacité de rafale. Lorsque `rate` est défini et que `burst` vaut 0, il prend par défaut la valeur de `rate` arrondie à l'inférieur plus 1 |
| `maxConns` | Nombre maximal de requêtes simultanées par IP ; 0 signifie illimité |

Lorsque la limitation de débit est activée, au moins l'un de `rate` et `maxConns` doit être défini.

### Cache de réponses en mémoire

S'applique uniquement aux sites en proxy inverse. Le cache réside en mémoire et utilise l'éviction LRU.

| Champ | Valeur par défaut | Plage |
| --- | --- | --- |
| `ttl` durée par défaut | 0 | 0 à 30 jours (en secondes). Lorsque l'amont ne déclare aucune directive de cache, les réponses 200 sont mises en cache pendant cette durée ; 0 signifie que seules les réponses explicitement autorisées par l'amont sont mises en cache |
| `maxMb` capacité du cache | 64 | 1 à 4096 Mo |

Règles de mise en cache :

- Seules les requêtes GET sont mises en cache, et la requête ne doit contenir ni en-tête `Authorization`, `Cookie` ou `Range`, ni `Cache-Control: no-cache` ;
- Seules les réponses de statut 200 sont mises en cache ; une réponse est ignorée si elle contient `Set-Cookie`, `Vary: *`, ou un `Cache-Control` valant `no-store` / `private` / `no-cache` ;
- `s-maxage` est respecté en priorité, puis `max-age` ;
- Une entrée unique est limitée à 4 Mo et à au plus un quart de la capacité totale ;
- Un succès (hit) renvoie les en-têtes `X-Cache: HIT` et `Age`, et un échec (miss) renvoie `X-Cache: MISS` ;
- La clé de cache inclut Host, l'URI complète de la requête et `Accept-Encoding`.

### Réécriture et redirection de chemin

Les règles sont évaluées dans l'ordre, et l'évaluation s'arrête à la première correspondance.

| Champ | Description |
| --- | --- |
| `pattern` | Une expression régulière comparée au chemin de la requête, comme `^/old/(.*)$` |
| `to` | Le résultat de remplacement ; utilisez `$1` pour référencer les groupes |
| `redirect` | Désactivé pour une réécriture interne (le chemin réécrit est envoyé au backend, et la cible doit commencer par `/`) ; activé pour une redirection |
| `code` | Code de statut de redirection 301 / 302 / 307 / 308, 302 par défaut |

Les redirections conservent les paramètres de requête d'origine.

## Statistiques d'accès et supervision

### Statistiques de site

Pour un écouteur en cours d'exécution, cliquez sur « Statistiques » sur la ligne d'un site pour ouvrir la fenêtre de statistiques, où vous pouvez choisir la dernière heure, les 6 dernières heures ou les 24 dernières heures :

- Statistiques d'accès à la minute : un point par minute, conservées sur les dernières 24 heures, avec une courbe du nombre de requêtes et des courbes d'erreurs 4xx / 5xx ;
- Répartition des codes de statut (2xx / 3xx / 4xx / 5xx) et trafic cumulé des réponses ;
- Top 10 des IP sources et top 10 des pays (les statistiques par pays nécessitent une base GeoIP importée) ;
- Les sites en proxy inverse affichent l'état de santé, le poids, l'indicateur de secours et le nombre de connexions actuelles de chaque backend.

Les statistiques sont conservées en mémoire et repartent de zéro après un redémarrage ou après la modification de l'écouteur. Les classements suivent au plus 5000 sources distinctes.

### Métriques Prometheus

Le point de terminaison global `GET /api/metrics` regroupe les métriques de tous les modules ; voir [Métriques de supervision](/en/docs/reference/monitoring). Le point de terminaison du module décrit ci-dessous ne contient que les métriques des services Web.

Le point de terminaison des métriques est `GET /api/webservice/metrics`, au format texte Prometheus, et il exige des identifiants de connexion ou un [jeton d'API](/en/docs/reference/api-reference).

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/webservice/metrics
```

| Métrique | Type | Description |
| --- | --- | --- |
| `gately_site_requests_total` | counter | Requêtes cumulées d'un site |
| `gately_site_errors_total` | counter | Erreurs de backend d'un site |
| `gately_site_active_requests` | gauge | Requêtes en cours |
| `gately_site_response_bytes_total` | counter | Octets de réponse cumulés |
| `gately_site_responses_total` | counter | Réponses comptées par classe de statut (étiquette `class` : `2xx` à `5xx`) |
| `gately_upstream_up` | gauge | Indique si un backend est sain : 1 sain, 0 indisponible (avec l'étiquette `upstream`) |

Toutes les métriques portent les étiquettes `listener` (nom de l'écouteur#clé de l'écouteur) et `site` (nom du site, ou premier domaine s'il n'a pas de nom).

## Conseil de déploiement

Les Services Web doivent écouter sur des ports de l'hôte tels que 80 / 443. Pour un déploiement Docker, `--network host` est recommandé.
