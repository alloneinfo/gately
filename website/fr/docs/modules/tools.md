# Outils de fichiers et d'exploitation

## WebDAV

Configurez-le dans « WebDAV » du groupe « Stockage réseau » de la barre latérale.

| Champ | Par défaut | Description |
| --- | --- | --- |
| `enable` | Désactivé | Active le service ; au moins un utilisateur est requis lorsqu'il est activé |
| `listenIp` | Vide | Adresse d'écoute ; laissez vide pour écouter sur toutes les adresses |
| `port` | 18800 | Port d'écoute |
| `tls` | Désactivé | Active HTTPS ; le certificat est fourni par SNI par le module [Certificats SSL](/fr/docs/modules/ssl) |
| `users` | Aucun | Plusieurs utilisateurs. Chacun possède `name`, `password`, `root` (un chemin de répertoire absolu) et `readOnly`, ou utilise `mounts` pour monter plusieurs répertoires |

Chaque utilisateur peut être associé à un seul répertoire, ou monter plusieurs répertoires :

- **Répertoire unique** : définissez `root` et `readOnly`. Après la connexion, la racine est ce répertoire.
- **Plusieurs répertoires** : cliquez sur « Monter plusieurs répertoires » dans l'interface, ou renseignez plusieurs entrées dans `mounts`. Chaque entrée comporte `name` (nom affiché, ne peut pas contenir `/`, unique pour un même utilisateur), `path` (un chemin absolu) et `readOnly`. Après la connexion, la racine liste les répertoires par nom. La racine elle-même n'est pas modifiable, et les fichiers ne peuvent pas être déplacés d'un répertoire à un autre.

Lors d'une migration depuis Lucky, la liste de montages d'un utilisateur est convertie en plusieurs répertoires, et les utilisateurs qui apparaissent plusieurs fois dans Lucky avec le même nom et le même mot de passe sont fusionnés en un seul.

## FTP

Configurez-le dans « Service FTP » du groupe « Stockage réseau ». FTP et FTPS sont pris en charge.

| Champ | Par défaut | Description |
| --- | --- | --- |
| `enable` | Désactivé | Active le service ; au moins un utilisateur est requis lorsqu'il est activé |
| `listenIp` | Vide | Adresse d'écoute ; laissez vide pour écouter sur toutes les adresses |
| `port` | 2121 | Port d'écoute |
| `passiveStart` / `passiveEnd` | 0 | Plage de ports de données du mode passif. Si les deux valent 0, le système attribue les ports aléatoirement ; lorsqu'ils sont définis, 1 ≤ start ≤ end ≤ 65535 doit être respecté, et la plage doit être autorisée dans le pare-feu et les redirections de ports |
| `publicHost` | Vide | IP annoncée à l'extérieur en mode passif. Derrière un NAT, saisissez l'IP publique ou redirigée. Seules les adresses IP sont prises en charge ; laissez vide pour utiliser automatiquement l'adresse de connexion |
| `tlsMode` | 0 | 0 autorise à la fois le texte clair et FTPS, 1 exige FTPS explicite (`AUTH TLS`), 2 est FTPS implicite (chiffré dès la connexion). S'il est différent de 0, le certificat est fourni par SNI par le module [Certificats SSL](/fr/docs/modules/ssl) |
| `idleTimeout` | 900 | Secondes d'inactivité avant la fermeture d'une connexion |
| `connectionTimeout` | 30 | Délai en secondes pour établir une connexion de données |
| `disableActiveMode` | Désactivé | Utilise uniquement le mode passif, mieux adapté aux environnements NAT |
| `users` | Aucun | Plusieurs utilisateurs, avec les mêmes champs que WebDAV : `name`, `password`, `root`, `readOnly` ou `mounts` ; les noms d'utilisateur doivent être uniques |

Les paramètres du service et les utilisateurs peuvent être importés depuis la configuration FTP de Lucky (voir [Migrer depuis Lucky](/fr/docs/guide/migration)).

## Stockage

Ajoutez des montages dans « Stockage » du groupe « Stockage réseau », et gérez les fichiers en ligne depuis la page.

| Champ | Description |
| --- | --- |
| `name`, `remark`, `enable` | Nom, remarque et interrupteur d'activation |
| `type` | `local` (répertoire local) ou `webdav` (WebDAV distant) |
| `path` | Répertoire local, doit être un chemin absolu existant (utilisé par `local`) |
| `url`, `username`, `password` | Adresse WebDAV distante (commençant par `http://` ou `https://`) et compte ; le mot de passe est masqué dans l'interface (utilisés par `webdav`) |
| `skipVerify` | Ignore la vérification du certificat HTTPS (utilisé par `webdav`) |
| `readOnly` | Monte en lecture seule ; le gestionnaire de fichiers rejette toute opération d'écriture |

Les répertoires locaux et les stockages WebDAV peuvent être importés depuis la gestion du stockage de Lucky. Alibaba Cloud Drive et les montages système ne sont pas migrés.

Le gestionnaire de fichiers en ligne permet de parcourir les répertoires, télécharger des fichiers, téléverser en flux (le fichier entier n'est pas chargé en mémoire), créer des répertoires, renommer et supprimer. Un montage peut servir d'emplacement cible pour une tâche de sauvegarde : les fichiers de sauvegarde sont écrits dans le répertoire indiqué du montage, et les anciens fichiers sont supprimés selon la politique de rétention.

## Wake-on-LAN

Gérez les appareils dans « Wake-on-LAN » et envoyez-leur des paquets magiques (6 octets `0xFF` suivis de l'adresse MAC répétée 16 fois).

| Champ | Par défaut | Description |
| --- | --- | --- |
| `name` | Aucun | Nom de l'appareil, obligatoire |
| `macs` | Aucun | Adresses MAC, une par ligne, au moins une ; vous pouvez en saisir plusieurs pour un même appareil |
| `broadcastIps` | Vide | Adresses de diffusion IPv4. Si vide, les paquets sont envoyés à `255.255.255.255` et à l'adresse de diffusion de chaque interface réseau activée |
| `port` | 9 | Port UDP cible |
| `repeat` | 3 | Nombre d'envois pour chaque MAC, jusqu'à 20 |
| `checkAddr` | Vide | Adresse de vérification de présence au format `host:port`, par exemple `192.168.1.10:3389` ; laissez vide pour ignorer la vérification |
| `remark` | Vide | Remarque |

Cliquez sur Réveiller dans la liste des appareils pour envoyer le paquet magique ; la page indique si l'appareil est en ligne d'après `checkAddr`.

## Tâches planifiées

Exécutez des commandes ou envoyez des requêtes HTTP selon un calendrier cron.

- `spec` accepte les expressions standard à 5 champs, un champ de secondes facultatif (6 champs, par exemple `*/30 * * * * *`) et des descripteurs comme `@every 10m` et `@daily` ;
- Lorsque `type` vaut `command`, `command` s'exécute via `sh -c` (un `.bat` temporaire sous Windows), et l'environnement reçoit `taskName` ainsi qu'une variable en majuscules de même nom avec le préfixe `GATELY_` ;
- Lorsque `type` vaut `http`, une requête est envoyée selon la configuration `http`, avec les champs `url`, `method` (GET, POST, PUT, PATCH, DELETE), `headers` (chacun sous la forme `Key: Value`), `body`, `successContent` (le corps de la réponse doit contenir l'une de ses entrées pour compter comme un succès ; si vide, seul le statut 2xx est vérifié), `timeout` et `insecureSkipVerify` ;
- Les tâches peuvent être activées ou désactivées, la page affiche l'heure de la prochaine exécution, et une tâche peut être lancée immédiatement à la main ;
- Si l'exécution précédente de la même tâche n'est pas terminée, le nouveau déclenchement est ignoré ;
- Chaque tâche conserve ses 20 derniers enregistrements d'exécution, incluant l'heure de début, la durée, le succès ou l'échec, si elle a été déclenchée manuellement, et la sortie (un résumé de 4000 caractères maximum) ;
- Le formulaire de création valide et analyse l'expression.

## Vue d'ensemble et journaux système

- La page « Vue d'ensemble » affiche les informations de l'hôte : modèle de CPU, utilisation et charge, mémoire, disque, adresses des interfaces réseau et débits en direct, IPv4 et IPv6 publiques (cliquez pour copier), l'état du service de chaque module et, pour le processus Gately, sa version, son commit, sa date de build, son heure de démarrage, son PID, son CPU, sa mémoire résidente, son tas Go et son nombre de goroutines ;
- La page « Journaux système » rassemble les journaux de tous les modules, y compris les enregistrements d'exécution de modules tels que Docker, FTP et Tâches planifiées.

## Thèmes

L'interface propose plusieurs thèmes : Émeraude, Océan, Fraîcheur, Sakura, Coucher de soleil, Sombre, Minuit, Punk, et suivre le système (Sombre lorsque le système est sombre, sinon Émeraude). Le choix du thème est stocké localement dans le navigateur.
