# Conventions de l'API du terminal Web

Tous les points de terminaison sont montés sous `/api/webterminal/` et exigent une connexion. Les erreurs sont toujours renvoyées sous la forme `{"error": "..."}`.

## 1. Structure de la configuration (le fichier webterminal dans `conf`)

```jsonc
{
  "settings": {
    "allowLocalShell": false,
    "maxSessions": 20,
    "idleTimeout": 0,          // minutes ; une session sans entrée ni sortie pendant cette durée est fermée automatiquement ; 0 = illimité
    "bufferSize": 256,         // Ko ; tampon de relecture de la sortie conservé par session et rejoué à la reconnexion
    "heartbeatInterval": 30,   // secondes ; battement de cœur WebSocket (ping du serveur / ping du frontend)
    "sessionKeepAlive": 0      // minutes ; durée de conservation d'une session après la déconnexion du navigateur ; 0 la ferme dès la déconnexion
  },
  "connections": [Connection],
  "shortcuts": [{ "name": "Show disk usage", "command": "df -h", "sendEnter": true }],
  "splitLayout": null,         // disposition des vues scindées personnalisée par le frontend ; le backend la stocke et la renvoie telle quelle (json.RawMessage)
  "security": {
    "agreementAccepted": false,
    "twoFAEnable": false,
    "twoFAKey": ""             // secret TOTP ; masqué dans les sorties de l'API
  },
  "credentials": [Credential]  // préréglages d'identifiants SSH
}
```

Connection (`type` vaut `ssh` / `telnet` / `local`) :

```jsonc
{
  "key": "", "name": "", "type": "ssh", "remark": "", "enable": true,
  "quickAccessDirs": ["/etc", "/var/log"],      // répertoires d'accès rapide SFTP
  "ssh": { "host", "port", "username", "authType": "password|key|credential",
           "credentialKey": "",  // clé du préréglage d'identifiants quand authType=credential ; username peut alors être vide et reprend la valeur par défaut du préréglage
           "password", "privateKey",
           "keyFile": "",        // chemin d'un fichier de clé privée sur le serveur lui-même ; utilisé quand authType=key et que privateKey est vide
           "passphrase", "hostKey", "keepAlive", "workDir": "", "timeout": 15,
           "proxy": Proxy },
  "telnet": { "host", "port": 23, "username", "password", "timeout": 15, "proxy": Proxy },
  "local": { "shell", "workDir", "env": {"K":"V"} }
}
Proxy = { "enabled": false, "type": "socks5|http", "host", "port", "username", "password" }
```

Credential (préréglage d'identifiants, référençable par plusieurs connexions SSH) :

```jsonc
{ "key": "", "name": "", "authType": "password|key", "username": "", // facultatif, utilisé quand la connexion n'a pas de nom d'utilisateur
  "password", "privateKey", "keyFile", "passphrase", "remark": "", "enable": true }
```

Les mots de passe, clés privées, phrases de passe des clés privées et mots de passe de proxy sont masqués avec `crud.Mask` en sortie et préservés avec `crud.KeepSecret` lors des mises à jour.

## 2. Gestion des connexions

| Méthode | Chemin | Description |
|---|---|---|
| GET/POST | `/connections` | Lister / créer |
| PUT/DELETE | `/connections/{key}` | Mettre à jour / supprimer |
| PUT | `/connections-order` | Réordonner `{"keys":[...]}` |
| POST | `/connections/{key}/test` | Tester la connectivité SSH / Telnet |
| POST | `/connections/test` | Tester une connexion non enregistrée (le corps est une Connection ; les secrets enregistrés sont fusionnés en faisant pointer le champ `key` vers la connexion d'origine) |
| DELETE | `/connections/{key}/hostkey` | Réinitialiser l'empreinte de l'hôte |
| PUT | `/connections/{key}/quickaccess` | Corps `{"dirs":["/a","/b"]}` |
| GET/POST | `/credentials` | Lister / créer des préréglages d'identifiants |
| PUT/DELETE | `/credentials/{key}` | Mettre à jour / supprimer (renvoie 409 avec la liste des noms de connexions tant qu'il est encore référencé) |
| PUT | `/credentials-order` | Réordonner les préréglages `{"keys":[...]}` |
| GET | `/shells` | Liste des shells disponibles sur cette machine `["/bin/bash", ...]` (lit /etc/shells et ne garde que ceux qui existent) |

Remarques complémentaires (l'implémentation fait foi) :
- `POST /connections/test` et `POST /connections/{key}/test` renvoient 200 avec `{ok, error?, serverVersion?, newHostKey?, latencyMs}` ; le type `local` renvoie 400. Pour Telnet, seule l'accessibilité TCP (y compris via le proxy) est vérifiée.
- `PUT /connections/{key}/quickaccess` renvoie `{ok, dirs}` (espaces et doublons retirés). Lors de la modification d'une connexion sans `quickAccessDirs`, l'ancienne valeur est conservée.
- `ssh.timeout` / `telnet.timeout` acceptent de 1 à 120 secondes, 15 par défaut. `ssh.workDir` est appliqué avec `cd` après la connexion.
- Pour les connexions avec `authType=credential`, le mot de passe, la clé privée, le fichier de clé et la phrase de passe saisis directement sont effacés à l'enregistrement ; les identifiants sont résolus depuis le préréglage au moment de la connexion. Mettre à jour un préréglage ferme les connexions SSH en réserve qui le référencent.
- Le remplissage automatique de la connexion Telnet n'essaie qu'une fois chacun, et seulement quand des invites comme `login:` / `username:` / `password:` sont détectées.

## 3. Sessions de terminal (sessions persistantes, reconnectables après une déconnexion)

- `GET /connections/{key}/ws?cols=&rows=` : crée une nouvelle session et passe en WebSocket.
- `GET /sessions/{id}/ws?cols=&rows=` : s'attache à une session existante, rejoue d'abord le tampon, puis transmet en temps réel. Plusieurs clients peuvent s'attacher à la même session en même temps (l'entrée est partagée, la sortie est diffusée à tous).

Protocole WebSocket :
- Serveur → navigateur : les **trames binaires** portent la sortie du terminal ; les **trames texte** portent du JSON :
  `{"t":"s","d":"<sessionId>"}` session établie (envoyé une fois, après une création comme après un attachement) ;
  `{"t":"n","d":"notice"}` avis ; `{"t":"x","d":"reason"}` session terminée ; `{"t":"p"}` réponse au battement de cœur.
- Navigateur → serveur : JSON texte : `{"t":"i","d":"input"}`, `{"t":"r","c":cols,"r":rows}`, `{"t":"p"}` battement de cœur.
- Quand le WebSocket du navigateur se déconnecte : si `sessionKeepAlive > 0`, la session est conservée (détachée) et fermée après le délai si personne ne s'y attache ; si 0, elle est fermée immédiatement.
- Quand le processus de la session se termine de lui-même, un message `x` est envoyé, le WebSocket est fermé et la session est retirée de la table des sessions.

- Créer une session au-delà de `maxSessions` renvoie HTTP 429 ; un échec de connexion est signalé par une trame d'avis `n`, après quoi le WebSocket est fermé.
- `bytesIn` est le nombre d'octets saisis dans le navigateur, `bytesOut` le nombre d'octets de sortie du terminal, et `attached` le nombre de clients actuellement attachés (un nombre) ; les champs de date sont en RFC3339.

REST :

| Méthode | Chemin | Description |
|---|---|---|
| GET | `/sessions` | `[{id, connKey, connName, type, remark, createdAt, lastActive, attached, cols, rows, bytesIn, bytesOut}]` |
| GET | `/sessions/{id}` | Une session |
| PUT | `/sessions/{id}/remark` | `{"remark":""}` |
| GET | `/sessions/{id}/stats` | `{bytesIn, bytesOut, durationSec, idleSec, attached, cols, rows}` |
| DELETE | `/sessions/{id}` | Fermer la session |

## 4. Réglages, raccourcis et disposition des vues scindées

| Méthode | Chemin | Description |
|---|---|---|
| GET/PUT | `/settings` | `{settings, activeSessions}` / le corps est Settings |
| GET/PUT | `/globalshortcuts` | Tableau de commandes raccourcies |
| GET/PUT/DELETE | `/splitlayout` | Disposition JSON libre |

## 5. Sécurité (accord d'utilisation + vérification à deux facteurs)

| Méthode | Chemin | Description |
|---|---|---|
| GET | `/security` | `{agreementAccepted, twoFAEnable, verified}` ; `verified` indique que le jeton de connexion actuel a passé la vérification à deux facteurs |
| POST | `/security/agreement` | Accepter l'accord |
| POST | `/security/2fa/setup` | Générer un secret en attente `{secret, url}` (non persisté) |
| POST | `/security/2fa/enable` | `{secret, code}` active la 2FA une fois la vérification réussie |
| POST | `/security/2fa/disable` | `{code}` désactive la 2FA une fois la vérification réussie |
| POST | `/security/verify2fa` | `{code}` ; une fois réussie, le jeton de connexion actuel est considéré comme vérifié pendant 30 minutes |
| GET | `/security/check2fa` | `{need: bool}` : indique si une vérification est encore requise |

Après 5 codes 2FA erronés consécutifs, le jeton de connexion est bloqué pendant 1 minute (HTTP 429). `2fa/setup` renvoie 400 quand la 2FA est déjà activée (désactivez-la d'abord). Après un `2fa/enable` réussi, le jeton actuel est considéré comme vérifié.
Contrôle d'accès : activer le shell local, créer une session, s'attacher à une session et tous les points de terminaison SFTP renvoient 403 `{"error":"...","needAgreement":true}` quand l'accord n'a pas été accepté,
et 403 `{"error":"...","need2fa":true}` quand la 2FA est activée et que le jeton actuel n'a pas été vérifié. « Activer le shell local » désigne ici un `PUT /settings` qui fait passer `allowLocalShell` de false à true. Côté Go, tous ces cas appellent `m.requireAccess(w, r) bool`.

## 6. Gestion de fichiers SFTP

Le paramètre de chemin `{key}` est la **clé de la connexion**. Les connexions SSH utilisent SFTP (chaque requête ouvre un canal de sous-système SFTP sur une connexion SSH réutilisée) ; les connexions `local` agissent directement sur le système de fichiers du serveur lui-même (via les mêmes points de terminaison).
Telnet et les autres types renvoient 400 ; une connexion introuvable renvoie 404. Le backend réutilise une connexion SSH par connexion et la ferme après 5 minutes d'inactivité (`m.acquireSSH`).
Une connexion `local` exige que « Autoriser le shell local » soit activé dans les réglages, sinon elle renvoie 403 (les permissions sur les fichiers locaux équivalent à un shell local). Un échec de connexion SSH renvoie 502.

Conventions générales :
- Chaque `path` / `from` / `to` / `dest` doit être un chemin absolu commençant par `/` (quand le `path` de `list` est vide, le répertoire par défaut est utilisé : le répertoire de connexion pour SSH, et le workDir de la connexion ou le répertoire personnel de l'utilisateur pour local). Les chemins relatifs renvoient 400.
- Les erreurs sont toujours `{"error":"message"}` : introuvable 404, permission refusée 403, trop volumineux 413, type non pris en charge 415, cible déjà existante 409 (ajouté dans cette section), paramètres invalides 400, tout le reste 500.
- Les points de terminaison d'écriture renvoient `{"ok":true, ...}` en cas de succès.

| Méthode | Chemin | Description |
|---|---|---|
| GET | `/sftp/{key}/list?path=` | `{path, parent, entries:[{name, path, isDir, isLink, size, mode, modeStr, modTime, owner?}]}` ; répertoires d'abord, puis tri par nom sans tenir compte de la casse. `mode` est une chaîne de permissions octale comme `"755"`, `modeStr` est au format `drwxr-xr-x`, `modTime` est en RFC3339, et `owner` est le nom d'utilisateur pour local et l'uid numérique pour SSH. Les liens symboliques ont `isLink=true`, et `isDir=true` quand ils pointent vers un répertoire |
| GET | `/sftp/{key}/read?path=&download=1` | Sans download : aperçu texte, renvoie `{path, name, size, modTime, content}` ; au-delà de 1 Mo renvoie 413, et un contenu binaire (contenant NUL ou non UTF-8) renvoie 415. Avec download=1 : téléchargement en flux (`application/octet-stream` + `Content-Disposition: attachment`), avec prise en charge de `Range` et `If-Modified-Since` |
| POST | `/sftp/{key}/write` | `{path, content}` enregistre du texte (écrasement ; une cible qui est un répertoire renvoie 400) → `{ok, path, size}` |
| POST | `/sftp/{key}/upload` | Écriture multipart en flux : `path` (le répertoire cible, qui doit déjà exister et **doit précéder les fichiers**), `files` (peut être répété) et `relativePaths` facultatif (un par fichier ; associé dans l'ordre d'apparition aux fichiers qui suivent, et doit précéder les fichiers correspondants ; une chaîne de tableau JSON est aussi acceptée). Sans `relativePaths`, le nom de fichier est utilisé tel quel (il peut contenir un sous-chemin quand un navigateur téléverse un répertoire). Les fichiers de même nom sont écrasés et les répertoires intermédiaires manquants sont créés automatiquement ; les chemins contenant `..` renvoient 400, ce qui garantit que rien n'est écrit hors du répertoire cible → `{ok, count, files:[chemins absolus]}` |
| POST | `/sftp/{key}/mkdir` | `{path}` crée un répertoire (parents manquants compris) ; 409 s'il existe déjà |
| POST | `/sftp/{key}/touch` | `{path}` crée un fichier vide (parents manquants compris) ; 409 s'il existe déjà |
| POST | `/sftp/{key}/rename` | `{from, to}` ; 409 si `to` existe déjà, 400 quand on déplace un répertoire dans lui-même, 403 quand on renomme le répertoire racine ; en local, entre deux périphériques, bascule automatiquement sur copie puis suppression |
| POST | `/sftp/{key}/copy` | `{from:[...], to}` ; `to` est le **répertoire** cible, et chaque élément est copié vers `to/nom-d-origine` (récursif ; les liens symboliques sont copiés comme liens ; les fichiers spéciaux comme les périphériques et les tubes sont ignorés). En cas de conflit de nom, la copie est **renommée automatiquement** en `name copy.ext`, `name copy 2.ext`, etc., sans erreur ; copier un répertoire dans lui-même renvoie 400 → `{ok, paths:[nouveaux chemins]}` |
| POST | `/sftp/{key}/chmod` | `{path, mode:"755", recursive}` ; `mode` est une chaîne octale de 0 à 7777 ; la récursion ne suit pas les liens symboliques, et fichiers comme répertoires reçoivent tous la même permission |
| DELETE | `/sftp/{key}/remove` | `{paths:[...]}` (corps JSON ; suppression récursive ; pour les liens symboliques, seul le lien est supprimé) ; si un chemin vaut `/`, renvoie 403 et rien n'est supprimé → `{ok, count}` |
| POST | `/sftp/{key}/compress` | `{paths:[...], dest, format:"zip|tar.gz"}` (format zip par défaut ; `tgz` est traité comme tar.gz). `dest` est le chemin complet de l'archive produite ; si `dest` est un répertoire existant ou se termine par `/`, le fichier est nommé automatiquement à l'intérieur (nom de l'élément pour un seul élément, ou nom du répertoire parent pour plusieurs, plus `.zip` / `.tar.gz`). Chaque élément devient une entrée de premier niveau de l'archive sous son propre nom ; des éléments de même nom renvoient 400, un fichier de sortie existant renvoie 409, et le fichier de sortie n'est jamais inclus dans lui-même → `{ok, path}` |
| POST | `/sftp/{key}/decompress` | `{path, dest}` prend en charge zip / tar / tar.gz / tgz (détecté par l'extension ; tout le reste renvoie 415). Quand `dest` est vide, l'extraction se fait dans le répertoire qui contient l'archive, créé si besoin ; les fichiers de même nom sont écrasés. Une archive contenant une entrée `..` est entièrement refusée (400, protection contre le zip-slip) ; les entrées qui ne sont pas des fichiers ordinaires, comme les liens symboliques et les liens physiques, sont ignorées ; les permissions des fichiers ne conservent que les bits rwx → `{ok, dest, count}` |
| GET | `/sftp/{key}/preview-archive?path=` | Liste des entrées de l'archive `[{name,size,isDir}]` (10000 entrées au maximum ; `name` n'a pas de `/` final) |

La compression et la décompression se font entièrement en Go via l'interface du système de fichiers (os pour local, lectures et écritures SFTP pour SSH), sans dépendre de commandes distantes.

## 7. Divers

- Journaux : les journaux du module s'affichent déjà dans la page globale des journaux ; aucun point de terminaison séparé n'est donc nécessaire.
- Import depuis Lucky : renseigne telnet, proxy, keyFile, workDir, timeout, env, quickAccessDirs, sessionConfig, globalShortcuts
  et securityConfig (l'accord d'utilisation et le secret 2FA sont migrés ensemble).
