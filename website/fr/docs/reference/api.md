# Jetons d'API et OpenAPI

Toutes les API utilisées par l'interface de gestion de Gately peuvent être appelées en HTTP. Les scripts et l'automatisation s'authentifient avec des jetons d'API, et la référence de l'API est générée automatiquement à partir des routes enregistrées.

## Jetons d'API

Chemin dans l'interface : groupe « Système » de la barre latérale → « Jetons d'API ».

### Créer un jeton

Cliquez sur « Nouveau jeton » et renseignez :

| Élément | Description |
| --- | --- |
| Nom | Obligatoire, 64 caractères au maximum, indique l'usage, p. ex. « script de supervision » |
| Expiration | « N'expire jamais », 7 jours, 30 jours, 90 jours, 1 an ou une date personnalisée ; une date personnalisée doit être postérieure à l'instant présent |
| Permissions | Obligatoire, sélectionnez-en au moins une. Un nouveau jeton n'a aucune permission par défaut |

Une instance peut stocker jusqu'à 100 jetons.

Après la création, la boîte de dialogue n'affiche le jeton en clair qu'une seule fois, au format `gtl_` suivi de 48 caractères hexadécimaux. Seul le hachage SHA-256 est enregistré sur le disque. En cas de perte, vous pouvez seulement le « Régénérer », et l'ancienne valeur cesse aussitôt de fonctionner. La liste identifie les jetons par un indice de la forme `gtl_xxxx…yyyy` et affiche l'heure de dernière utilisation et l'IP source (écrites sur le disque avec un délai d'une minute au plus).

La modification d'un jeton permet de changer son nom, ses permissions, son expiration et son interrupteur d'activation. Les changements prennent effet immédiatement et le jeton en clair reste identique. Un jeton expiré peut conserver son expiration d'origine tandis que vous modifiez les autres champs.

### Authentification

Envoyez `Authorization: Bearer <jeton>` dans l'en-tête de la requête :

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/status
```

Le corps des requêtes utilise `Content-Type: application/json`, et les erreurs sont renvoyées sous la forme `{"error": "description"}`. Lorsqu'un jeton est désactivé, expiré ou inexistant, la requête est vérifiée comme une session de connexion et aboutit à une erreur de non-connexion.

### Portée des permissions

Le format d'une permission est `<domaine>:read` ou `<domaine>:write`, la permission d'écriture incluant la lecture. Le domaine est l'ID d'un module fonctionnel (p. ex. `ddns`, `portforward`, `notify`, `backup`) plus trois domaines intégrés :

| Domaine | Nom | Permissions disponibles | API correspondantes |
| --- | --- | --- | --- |
| `status` | État du système | `status:read` | `/api/status`, `/api/system/info`, `/api/system/update` |
| `logs` | Journaux système | `logs:read` | `/api/logs` |
| `import` | Import de configuration Lucky | `import:write` | `/api/import/lucky` |

Notations génériques :

| Notation | Signification |
| --- | --- |
| `*:read` | Lecture seule sur tous les modules |
| `*:write` ou `*` | Lecture et écriture sur tous les modules |
| `ddns:read` | Lecture seule sur DDNS uniquement |

La permission requise par une API dépend du chemin et de la méthode :

- Le chemin `/api/<module>/...` exige une permission du domaine `<module>` : GET et HEAD demandent `read`, les autres méthodes demandent `write`.
- Les journaux d'un module `GET /api/<module>/logs` sont en lecture seule ; leur effacement (DELETE) est une opération d'écriture.
- Les API GET suivantes renvoient du contenu sensible ou ouvrent un canal interactif, et sont traitées comme `write` : celles dont le chemin contient un segment `ws` (terminal WebSocket), `exec`, `download`, `export`, `read`, `config` ou `preview-archive`, par exemple le téléchargement de certificats, l'exécution de commandes dans un conteneur et la lecture de fichiers.
- `GET /api/session` et `GET /api/openapi.json` acceptent n'importe quel jeton valide.

### Opérations interdites aux jetons

Les API suivantes n'acceptent que les sessions de connexion ; tout accès par jeton renvoie 403 :

- Compte et authentification à deux facteurs : `/api/account`
- Gestion des jetons : `/api/tokens`
- Paramètres système, jeton système, redémarrage, mise à niveau en ligne : `/api/system/settings`, `/api/system/token`, `/api/system/restart`, `/api/system/upgrade`
- Téléchargement et restauration de la sauvegarde de configuration : `/api/system/backup`, `/api/system/restore`
- Téléchargement des fichiers de sauvegarde locaux : `/api/backup/local`
- Paramètres de sécurité du terminal Web : `/api/webterminal/security`
- Connexion et déconnexion : `/api/login`, `/api/logout`

Une requête sans permission suffisante reçoit `le jeton n'a pas la permission <permission>` et un avertissement est consigné dans les journaux système, avec le nom du jeton, l'IP source et l'API refusée.

### Jeton hérité

Le jeton global unique des anciennes versions est migré automatiquement lors de la mise à niveau vers un jeton nommé « Jeton hérité (migré depuis une ancienne version) » avec la permission `*`, marqué « Hérité » dans la liste. Créez de nouveaux jetons avec uniquement les permissions nécessaires, puis supprimez-le. « Régénérer » sur le jeton hérité retire aussi la marque d'héritage.

::: tip Conseil
Créez un jeton distinct pour chaque script, ne cochez que les permissions nécessaires et définissez une expiration. En cas de fuite d'un jeton, supprimez ou régénérez uniquement celui-ci.
:::

## OpenAPI et documentation de l'API

### La page de documentation dans l'interface

Groupe « Système » de la barre latérale → « Documentation de l'API ». La page affiche l'URL de base et les explications d'authentification, liste toutes les API par groupe fonctionnel et permet de rechercher par chemin ou par résumé. Le développement d'une API affiche la permission requise, les paramètres, les champs du corps de la requête et un exemple curl copiable. Les API qui nécessitent une session de connexion sont marquées « Session de connexion uniquement » et n'ont pas d'exemple curl.

### Obtenir le document OpenAPI

Lorsque l'instance est en cours d'exécution, `GET /api/openapi.json` renvoie un document OpenAPI 3.0, lisible avec n'importe quel jeton valide ou session de connexion :

```bash
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/openapi.json -o openapi.json
```

Le document utilise le schéma de sécurité `bearerAuth` et ajoute des champs d'extension à chaque API :

| Champ d'extension | Signification |
| --- | --- |
| `x-required-scope` | Permission de jeton requise pour accéder à l'API |
| `x-token-allowed` | Indique si un jeton est autorisé ; `false` signifie session de connexion uniquement |
| `x-scope-domains` (racine du document) | Liste de tous les domaines de permissions |

La liste des API de [Ligne de commande et MCP](/fr/docs/guide/cli-mcp) est lue depuis ce document.

### Export en ligne de commande

L'option `-api-docs` construit l'application dans un dossier de configuration temporaire (sans écouter de port ni toucher à la configuration existante), écrit les documents dans le dossier indiqué, puis quitte :

```bash
gately -api-docs ./api-docs
```

Deux fichiers sont produits :

- `openapi.json` : description OpenAPI 3.0
- `API.md` : documentation Markdown de l'API, avec démarrage rapide, tableau des domaines de permissions et tableaux d'API par groupe

## Exemples d'appels

```bash
# Consulter l'état du système (nécessite status:read)
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/status

# Lister toutes les tâches DDNS (nécessite ddns:read)
curl -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks

# Synchroniser immédiatement une tâche DDNS (nécessite ddns:write, key provient de la liste renvoyée à l'étape précédente)
curl -X POST -H "Authorization: Bearer gtl_xxxx" http://localhost:16680/api/ddns/tasks/KEY/sync

# Récupérer uniquement les journaux du module dont le numéro est supérieur à 10 (nécessite ddns:read)
curl -H "Authorization: Bearer gtl_xxxx" "http://localhost:16680/api/ddns/logs?since=10"

# Récupérer seulement les 200 entrées les plus récentes, puis passer le plus petit numéro en before pour remonter
curl -H "Authorization: Bearer gtl_xxxx" "http://localhost:16680/api/ddns/logs?limit=200"
curl -H "Authorization: Bearer gtl_xxxx" "http://localhost:16680/api/ddns/logs?before=1200&limit=200"
```

## Liste complète des API

Toutes les API, leurs paramètres et les permissions requises figurent dans la [référence de l'API](/en/docs/reference/api-reference).
