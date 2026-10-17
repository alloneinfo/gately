# Guide de durcissement

L'interface d'administration contrôle le DNS, les certificats et les ports. Parcourez la liste de contrôle suivante.

## Interface d'administration

- [ ] Changer le mot de passe de l'administrateur après la première connexion
- [ ] Activer l'authentification à deux facteurs (TOTP)
- [ ] Définir un chemin d'accès secret pour que l'interface ne soit joignable que par un chemin personnalisé
- [ ] Activer « l'accès limité au réseau local » et utiliser un VPN (comme [Tailscale](/fr/docs/modules/tunnels)) pour l'administration à distance
- [ ] Activer le HTTPS de l'interface d'administration pour un accès via des réseaux non fiables
- [ ] Donner aux scripts d'automatisation et à la [CLI / MCP](/fr/docs/guide/cli-mcp) des [jetons d'API](/fr/docs/reference/api) aux privilèges minimaux, avec une date d'expiration, et éviter de partager le mot de passe administrateur
- [ ] Définir une phrase secrète de chiffrement pour les [sauvegardes de configuration](/fr/docs/modules/backup) et activer les sauvegardes planifiées
- [ ] Configurer des notifications dans le [centre de notifications](/fr/docs/modules/notify) pour les événements tels que les connexions, les certificats et les échecs de sauvegarde

## Services exposés sur Internet

- Lorsque vous exposez des services du réseau local via les [Services Web](/fr/docs/modules/web), activez le HTTPS et combinez-le avec l'authentification de base, l'authentification Web ou la connexion via des tiers ;
- Activez le [WAF](/fr/docs/modules/waf) sur les sites publics : commencez en mode « détection seule » pour repérer les faux positifs, puis passez en « blocage » ;
- Restreignez les sources avec des listes d'autorisation/blocage d'IP, le filtrage GeoIP par pays et le bannissement automatique après échecs d'authentification.

## Docker

Monter `docker.sock` revient à accorder un accès de niveau root à l'hôte. Retirez ce montage si vous n'avez pas besoin de la [gestion de Docker](/fr/docs/modules/docker).

## Terminal Web

Le shell local est désactivé par défaut. Ne l'activez qu'en cas de réel besoin, et vérifiez que l'authentification à deux facteurs est activée pour l'interface d'administration.

## Sauvegarde de la configuration

Conservez le dossier de configuration dans un dépôt git privé ou sauvegardez-le régulièrement. `ssl.json` contient les clés privées des certificats : stockez-le avec soin.
