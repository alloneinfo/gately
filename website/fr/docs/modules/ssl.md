# Certificats SSL

Émettez et renouvelez automatiquement des certificats pour les [Services Web](/fr/docs/modules/web), ou exportez-les vers d'autres logiciels.

## Fonctions

- Émission et renouvellement automatiques **ACME DNS-01**, en réutilisant les fournisseurs DNS du module [DNS dynamique](/fr/docs/modules/ddns), sans avoir à ouvrir les ports 80 / 443 ;
- **ACME HTTP-01** : aucun fournisseur DNS requis, mais le port 80 de chaque domaine doit être joignable depuis Internet ; les certificats génériques ne sont pas pris en charge ;
- Certificats génériques (joker) et EAB (par exemple ZeroSSL) ;
- Import d'un certificat existant, ou lecture depuis des fichiers ;
- Mappage vers un répertoire : après un changement de certificat, les fichiers sont écrits dans le répertoire indiqué et un script est exécuté, afin que les autres services puissent se recharger ;
- Déploiement automatique : après un changement de certificat, il est copié vers plusieurs répertoires locaux, envoyé par SSH vers d'autres hôtes avec exécution d'une commande, ou les conteneurs Docker qui l'utilisent sont redémarrés.

## Validation HTTP-01

Choisissez HTTP-01 dans la « Méthode » du certificat. Lors de l'émission, l'AC accède à `http://<domaine>/.well-known/acme-challenge/<token>` :

- Si un [service Web](/fr/docs/modules/web) écoute déjà sur le port 80, c'est lui qui répond, sans incidence sur les sites existants ;
- Si le port 80 est libre, Gately l'écoute temporairement pendant l'émission et le ferme aussitôt après ;
- Si le routeur redirige le port public 80 vers un autre port de cette machine, indiquez ce port dans « Port temporaire ».

De nombreux accès Internet résidentiels (en Chine continentale notamment) bloquent le port 80 entrant : utilisez alors DNS-01.

## Mappage de certificat

Une fois un certificat activé, Gately écrit `.crt`, `.key`, `.pem` et `_issuerCertificate.crt` dans le répertoire de mappage et exécute le script de changement.

## Déploiement automatique

Dans « Déploiement automatique » d'un certificat, vous pouvez ajouter plusieurs cibles de déploiement, exécutées automatiquement après l'émission, le renouvellement ou la mise à jour du certificat :

- **Copier vers un répertoire local** : écrit le certificat dans un répertoire de cette machine, puis exécute localement la commande « Exécuter après la copie ». Vous pouvez ajouter plusieurs répertoires, chacun avec une commande différente. Si les fichiers d'un répertoire sont supprimés ou modifiés, la vérification suivante les recopie. Si Gately tourne dans Docker, le répertoire et la commande sont dans le conteneur ;
- **Envoyer vers un hôte SSH** : choisissez une connexion SSH existante du [terminal web](/fr/docs/modules/terminal), dont l'adresse, la méthode d'authentification et le proxy sont réutilisés. Les fichiers du certificat (`<note>.crt`, `.key`, `.pem`, `_issuerCertificate.crt`, comme pour le mappage de certificat : la note est donc obligatoire) sont écrits dans le « Répertoire distant », puis la commande « Exécuter après l'envoi » s'exécute sur l'hôte distant, par exemple `systemctl reload nginx` ; elle peut utiliser `$certName`, `$certDir`, `$certFile` et `$keyFile`. Chaque fichier est d'abord écrit dans un fichier temporaire avec ses permissions, puis renommé pour remplacer l'ancien ; la clé privée a les permissions 0600 ;
- **Redémarrer des conteneurs locaux** : après un changement de certificat, les conteneurs Docker choisis sont redémarrés l'un après l'autre. Utile pour les services dont le conteneur monte le répertoire de mappage et doit redémarrer pour charger le nouveau certificat.

« Copier vers un répertoire local » et « Envoyer vers un hôte SSH » acceptent des noms de fichiers pour le certificat (chaîne complète) et la clé privée, comme `fullchain.pem` et `privkey.pem`. Si les deux sont renseignés, seuls ces deux fichiers sont écrits et la note peut rester vide ; laissés vides, les noms du mappage des certificats sont utilisés. `$certFile` et `$keyFile` désignent les fichiers réellement écrits.

Le déploiement automatique ne fait jamais confiance à un hôte inconnu : connectez-vous une fois à la connexion SSH choisie depuis le terminal web pour enregistrer sa clé d'hôte avant tout envoi. En cas d'échec, le déploiement est retenté toutes les heures et déclenche l'événement « Certificat : échec du déploiement », auquel vous pouvez vous abonner dans le [Centre de notifications](/fr/docs/modules/notify). Le bouton « Déployer maintenant » de la carte du certificat ignore l'intervalle de nouvel essai, déploie immédiatement vers toutes les cibles activées et affiche le résultat de chacune.

Lors de la modification d'un certificat, les déploiements dont les réglages de cible n'ont pas changé ne sont pas réexécutés ; modifier le répertoire distant, la commande ou les conteneurs déclenche un nouveau déploiement.

## Note de sécurité

Les clés privées sont stockées dans `ssl.json`, dans le répertoire de configuration. Conservez ce fichier en lieu sûr, et assurez-vous que le dépôt est privé si vous le placez sous gestion de versions.

## Migration depuis Lucky

Les certificats déjà émis sont importés tels quels et ne sont pas redemandés avant leur expiration. ZeroSSL enregistre un nouveau compte ACME à partir des identifiants EAB de Lucky. Consultez le [Guide de migration](/fr/docs/guide/migration) pour plus de détails.
