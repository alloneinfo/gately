# Pare-feu WAF

Un pare-feu applicatif web basé sur [Coraza](https://coraza.io/) et OWASP CRS v4, rattachable aux sites des [Services Web](/fr/docs/modules/web).

## Fonctions

- Jeu de règles OWASP CRS v4 intégré, dont les fichiers de règles s'activent individuellement ;
- Règles SecLang personnalisées ;
- Mode blocage et mode détection seule ;
- Plusieurs instances, référencées par les sites selon les besoins.

## Recommandations

1. Pour un nouveau site, commencez par « Détection seule » et surveillez les journaux pendant un certain temps pour vous assurer qu'il n'y a pas de faux positifs ;
2. Ajustez les fichiers de règles ou ajoutez des exceptions personnalisées en cas de faux positifs ;
3. Passez ensuite en « Blocage ».

## Tests

Le WAF est testé avec de vraies charges d'attaque, qui couvrent le blocage de la traversée de chemin et d'attaques similaires.
