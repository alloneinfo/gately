# Pare-feu WAF

Un pare-feu applicatif web basé sur [Coraza](https://coraza.io/) et OWASP CRS v4, rattachable aux sites des [Services Web](/fr/docs/modules/web).

## Fonctions

- Jeu de règles OWASP CRS v4 intégré, dont les fichiers de règles s'activent individuellement ;
- Règles SecLang personnalisées ;
- Les groupes de règles peuvent être réordonnés ; les journaux de détection enregistrent l'IP du client, le domaine, le chemin et l'ID de règle, filtrables par domaine, IP ou mots-clés ;
- Mode blocage et mode détection seule ;
- Limite d'inspection du corps de requête réglable : au-delà, la requête est rejetée ou seul le début est inspecté ; l'inspection du corps peut être ignorée par chemin ;
- Mode de compatibilité WebDAV : autorise les méthodes, les en-têtes et tout type de contenu d'envoi utilisés par les clouds personnels et WebDAV ;
- Plusieurs instances, référencées par les sites selon les besoins.

## Recommandations

1. Pour un nouveau site, commencez par « Détection seule » et surveillez les journaux pendant un certain temps pour vous assurer qu'il n'y a pas de faux positifs ;
2. Ajustez les fichiers de règles ou ajoutez des exceptions personnalisées en cas de faux positifs ;
3. Passez ensuite en « Blocage ».

## Ordre des groupes de règles

Les groupes de règles d'une instance se chargent de haut en bas ; les boutons monter et descendre permettent de les réordonner. Dans une même phase, les règles placées avant s'exécutent d'abord :

- Les règles de blocage (deny, etc.) fonctionnent à n'importe quelle position ;
- Les règles d'autorisation ou d'exclusion à l'exécution (`allow`, `ctl:ruleRemoveById`, `ctl:ruleRemoveTargetById`, etc.) doivent être au-dessus du groupe OWASP CRS, sinon CRS peut bloquer la requête avant ;
- Les directives de configuration comme `SecRuleRemoveById` et `SecRuleUpdateTargetById` n'agissent que sur les règles déjà chargées : placez-les sous le groupe CRS, au-dessus elles sont sans effet.

## Sites recevant de gros fichiers

Par défaut, le WAF n'inspecte que les corps de requête jusqu'à 12,5 Mo et rejette les requêtes plus volumineuses (413) : les clouds personnels, WebDAV ou gestionnaires de fichiers ne peuvent alors pas recevoir de gros fichiers. Réglez-le dans « Inspection du corps de requête » de l'instance :

- **Limite d'inspection** : 0 correspond à la valeur par défaut de 12,5 Mo, jusqu'à 1024 Mo ;
- **Au-delà de la limite** : « Rejeter la requête » conserve le comportement précédent ; « Inspecter uniquement le début » n'inspecte que la partie sous la limite et transmet le reste tel quel au backend, ce qui convient aux sites recevant de gros envois ;
- **Compatible WebDAV et envois** : par défaut, OWASP CRS n'autorise que GET, HEAD, POST et OPTIONS, et bloque les en-têtes `Lock-Token`, `If` et `Content-Range` ainsi que les types de contenu hors de sa liste autorisée ; les clients WebDAV ne peuvent donc ni lister les dossiers, ni envoyer, ni verrouiller des fichiers. Une fois l'option activée, PUT, DELETE, PROPFIND, PROPPATCH, MKCOL, COPY, MOVE, LOCK, UNLOCK et d'autres méthodes, ainsi que ces en-têtes, sont autorisés, et le type de contenu des envois n'est plus restreint ; le corps des requêtes PUT et `application/octet-stream` est traité comme contenu de fichier et n'est plus inspecté comme paramètres de formulaire. Les contrôles d'injection, de traversée de chemin et des autres attaques restent actifs ;
- **Chemins sans inspection du corps** : un préfixe de chemin par ligne (par exemple `/api/upload`, `/remote.php/dav/`). Le corps des requêtes correspondantes n'est pas inspecté ; les en-têtes, l'URL et les paramètres restent inspectés normalement.

Pour les clouds personnels et les sites WebDAV, créez de préférence une instance dédiée : activez « Compatible WebDAV et envois », choisissez « Inspecter uniquement le début » et, si nécessaire, ajoutez les points d'envoi aux chemins sans inspection ; les autres sites gardent les réglages par défaut.

curl, rclone et d'autres clients envoient l'en-tête `Expect: 100-continue` pour les envois volumineux, en-tête que CRS 4 bloque par défaut. Toutes les instances de Gately autorisent cet en-tête (Gately gère lui-même le 100-continue) : aucun réglage supplémentaire n'est nécessaire.

## Tests

Le WAF est testé avec de vraies charges d'attaque, qui couvrent le blocage de la traversée de chemin et d'attaques similaires.
