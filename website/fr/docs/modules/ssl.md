# Certificats SSL

Émettez et renouvelez automatiquement des certificats pour les [Services Web](/fr/docs/modules/web), ou exportez-les vers d'autres logiciels.

## Fonctions

- Émission et renouvellement automatiques **ACME DNS-01**, en réutilisant les fournisseurs DNS du module [DNS dynamique](/fr/docs/modules/ddns), sans avoir à ouvrir les ports 80 / 443 ;
- **ACME HTTP-01** : aucun fournisseur DNS requis, mais le port 80 de chaque domaine doit être joignable depuis Internet ; les certificats génériques ne sont pas pris en charge ;
- Certificats génériques (joker) et EAB (par exemple ZeroSSL) ;
- Import d'un certificat existant, ou lecture depuis des fichiers ;
- Mappage vers un répertoire : après un changement de certificat, les fichiers sont écrits dans le répertoire indiqué et un script est exécuté, afin que les autres services puissent se recharger.

## Validation HTTP-01

Choisissez HTTP-01 dans la « Méthode » du certificat. Lors de l'émission, l'AC accède à `http://<domaine>/.well-known/acme-challenge/<token>` :

- Si un [service Web](/fr/docs/modules/web) écoute déjà sur le port 80, c'est lui qui répond, sans incidence sur les sites existants ;
- Si le port 80 est libre, Gately l'écoute temporairement pendant l'émission et le ferme aussitôt après ;
- Si le routeur redirige le port public 80 vers un autre port de cette machine, indiquez ce port dans « Port temporaire ».

De nombreux accès Internet résidentiels (en Chine continentale notamment) bloquent le port 80 entrant : utilisez alors DNS-01.

## Mappage de certificat

Une fois un certificat activé, Gately écrit `.crt`, `.key`, `.pem` et `_issuerCertificate.crt` dans le répertoire de mappage et exécute le script de changement.

## Note de sécurité

Les clés privées sont stockées dans `ssl.json`, dans le répertoire de configuration. Conservez ce fichier en lieu sûr, et assurez-vous que le dépôt est privé si vous le placez sous gestion de versions.

## Migration depuis Lucky

Les certificats déjà émis sont importés tels quels et ne sont pas redemandés avant leur expiration. ZeroSSL enregistre un nouveau compte ACME à partir des identifiants EAB de Lucky. Consultez le [Guide de migration](/fr/docs/guide/migration) pour plus de détails.
