# Certificats SSL

Émettez et renouvelez automatiquement des certificats pour les [Services Web](/fr/docs/modules/web), ou exportez-les vers d'autres logiciels.

## Fonctions

- Émission et renouvellement automatiques **ACME DNS-01**, en réutilisant les fournisseurs DNS du module [DNS dynamique](/fr/docs/modules/ddns), sans avoir à ouvrir les ports 80 / 443 ;
- Certificats génériques (joker) et EAB (par exemple ZeroSSL) ;
- Import d'un certificat existant, ou lecture depuis des fichiers ;
- Mappage vers un répertoire : après un changement de certificat, les fichiers sont écrits dans le répertoire indiqué et un script est exécuté, afin que les autres services puissent se recharger.

## Mappage de certificat

Une fois un certificat activé, Gately écrit `.crt`, `.key`, `.pem` et `_issuerCertificate.crt` dans le répertoire de mappage et exécute le script de changement. Les noms de fichiers sont identiques à ceux de Lucky, et le contenu écrit est strictement identique octet pour octet à celui de Lucky.

## Note de sécurité

Les clés privées sont stockées dans `ssl.json`, dans le répertoire de configuration. Conservez ce fichier en lieu sûr, et assurez-vous que le dépôt est privé si vous le placez sous gestion de versions.

## Migration depuis Lucky

Les certificats déjà émis sont importés tels quels et ne sont pas redemandés avant leur expiration. ZeroSSL enregistre un nouveau compte ACME à partir des identifiants EAB de Lucky. Consultez le [Guide de migration](/fr/docs/guide/migration) pour plus de détails.
