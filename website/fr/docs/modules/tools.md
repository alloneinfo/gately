# Outils de fichiers et d'exploitation

## WebDAV

Configurez-le dans « Service WebDAV » du groupe « Services de fichiers » de la barre latérale.

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

## FTP

Configurez-le dans « Service FTP » du groupe « Services de fichiers ». FTP et FTPS sont pris en charge.

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

## SMB

Configurez-le dans « Service SMB » du groupe « Services de fichiers » pour partager des répertoires locaux avec l’Explorateur Windows, le Finder de macOS, Linux et les clients SMB des téléphones et téléviseurs. L’implémentation du protocole est intégrée à Gately (basée sur [gosamba](https://github.com/ahmetozer/gosamba)) et prend en charge SMB 2.0.2 à 3.1.1, l’authentification NTLMv2 et la signature, avec un chiffrement SMB 3 obligatoire en option ; SMB 1 n’est pas pris en charge. Disponible uniquement sous Linux et macOS.

| Champ | Par défaut | Description |
| --- | --- | --- |
| `enable` | Désactivé | Active le service ; au moins un utilisateur est requis lorsqu'il est activé |
| `listenIp` | Vide | Adresse d'écoute ; laissez vide pour écouter sur toutes les adresses |
| `port` | 445 | Port d'écoute. Windows se connecte uniquement au 445 ; si le Samba fourni avec un NAS comme fnOS ou Synology utilise déjà le 445, choisissez un autre port, que macOS, Linux et la plupart des lecteurs peuvent indiquer dans l’adresse |
| `requireEncryption` | Désactivé | Exige le chiffrement SMB 3 pour chaque connexion ; les anciens appareils et certains téléviseurs limités à SMB 2 ne pourront plus se connecter |
| `mdns` | Désactivé | Annonce le service via Bonjour pour qu’il apparaisse dans Réseau du Finder de macOS |
| `users` | Aucun | Plusieurs utilisateurs avec les mêmes champs que WebDAV : `name`, `password`, `root`, `readOnly` ou `mounts` ; les noms d’utilisateur sont insensibles à la casse et uniques |

Les partages appartiennent aux utilisateurs, et un utilisateur connecté ne voit et n’ouvre que les siens :

- **Répertoire unique** : le partage porte le nom de l’utilisateur, par exemple `\\192.168.1.2\alice` pour l’utilisateur `alice`.
- **Plusieurs répertoires** : chaque répertoire est un partage dont `name` est le nom (insensible à la casse et unique par utilisateur), et chacun peut être en lecture seule. Des utilisateurs différents peuvent avoir des partages du même nom, chacun pointant vers son propre répertoire.

Les noms d’utilisateur et de partage ne peuvent pas contenir `" / \ [ ] : ; | = , + * ? < >` ni commencer ou se terminer par `.`. L’authentification NTLM a besoin du mot de passe en clair : les utilisateurs dont le mot de passe est stocké sous forme de hachage bcrypt doivent le redéfinir. Les fichiers sont lus et écrits sous l’identité du processus Gately.

« Adresses de connexion », en bas de la page, liste l’adresse de chaque partage d’après les paramètres enregistrés :

- Windows : saisissez `\\hôte\partage` dans la barre d’adresse de l’Explorateur, ou exécutez `net use Z: \\hôte\partage /user:utilisateur`
- macOS : dans le Finder, choisissez Aller → Se connecter au serveur et saisissez `smb://hôte:port/partage` (le port peut être omis pour 445)
- Linux : `mount -t cifs //hôte/partage /mnt -o username=utilisateur,port=port`

## Montages de stockage

Ajoutez des montages dans « Montages de stockage » du groupe « Services de fichiers », et gérez les fichiers en ligne depuis la page.

| Champ | Description |
| --- | --- |
| `name`, `remark`, `enable` | Nom, remarque et interrupteur d'activation |
| `type` | `local` (répertoire local), `webdav` (WebDAV distant) ou `smb` (partage SMB) |
| `path` | Répertoire local, doit être un chemin absolu existant (utilisé par `local`) |
| `url`, `username`, `password` | Adresse distante et compte ; le mot de passe est masqué dans l'interface (utilisés par `webdav` et `smb`). Une adresse WebDAV commence par `http://` ou `https://` ; une adresse SMB a la forme `smb://hôte[:port]/partage[/sous-dossier]`, SMB2/3 est pris en charge et un utilisateur vide se connecte en invité |
| `skipVerify` | Ignore la vérification du certificat HTTPS (utilisé par `webdav`) |
| `domain` | Domaine ou groupe de travail pour l'authentification SMB, généralement vide (utilisé par `smb`) |
| `readOnly` | Monte en lecture seule ; le gestionnaire de fichiers rejette toute opération d'écriture |

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

L'interface propose 15 palettes en deux groupes, les originales Gately (Émeraude, Océan, Fraîcheur, Sakura, Coucher de soleil, Minuit, Punk) et les classiques (GitHub, Nord, Dracula, Catppuccin, Tokyo Night, Gruvbox, Rosé Pine, Solarized), chacune avec une variante claire et une variante sombre. Le mode peut être clair, sombre ou suivre le système, qui utilise la variante claire ou sombre de la palette en cours. Toute palette accepte l'une des 10 couleurs d'accent prédéfinies ou une couleur choisie avec le sélecteur du système ; le logo et le dégradé de la page de connexion la suivent, et choisir à nouveau une palette efface l'accent personnalisé. Les coins peuvent être droits, ceux de la palette ou arrondis. Les réglages du thème sont stockés localement dans le navigateur.
