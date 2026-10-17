# Comparaison avec Lucky

Gately s'inspire de Lucky, couvre les fonctions courantes de façon open source et sait importer une configuration existante. Lucky garde l'avantage pour l'étendue des fonctions et le nombre d'utilisateurs, et sans Lucky, Gately n'existerait pas. Merci.

## Philosophie

| Point | Gately | Lucky |
| --- | --- | --- |
| Code source | Open source (Apache-2.0) | Propriétaire |
| Fichier de configuration | JSON en clair, compatible git | Format propriétaire (`.lkcf`) |
| Services officiels | Aucun par défaut ; toutes les adresses externes sont modifiables et désactivables | Mises à jour en ligne, relais de connexion tiers et autres services officiels |
| Connexion via des tiers | Vos propres applications OAuth | Via le service de relais officiel |
| Mises à niveau | Docker / gestionnaire de paquets | Mise à jour automatique en ligne intégrée |
| Étendue des fonctions | Couvre les fonctions essentielles courantes | Plus large, p. ex. DLNA et montage de disques cloud |

## Couverture fonctionnelle

| Fonction | Gately |
| --- | --- |
| DNS dynamique | ✅ |
| Traversée STUN | ✅ |
| Redirection de ports | ✅ |
| Certificats SSL | ✅ |
| Services Web (proxy inverse) | ✅ |
| WAF | ✅ |
| Tailscale / frp / cloudflared | ✅ |
| Gestion de Docker | ✅ |
| Terminal Web | ✅ |
| WebDAV / FTP / gestion du stockage | ✅ |
| Wake-on-LAN / tâches planifiées | ✅ |
| Centre de notifications / sauvegarde de la configuration / diagnostic réseau | ✅ |
| Jetons d'API / OpenAPI / CLI / MCP | ✅ |
| DLNA, montage de disques cloud, rclone, filtrage IP global, HTTP/3 | Prévu, planifié selon la demande de la communauté |

## Différences assumées

- Pas de mise à jour automatique en ligne qui écrase le binaire ; les mises à niveau sont confiées à votre gestionnaire de paquets ou à Docker.
- Pas de poussée de configuration à distance ni de remontée d'informations sur les appareils.
- La configuration est stockée en clair, et les champs sensibles sont masqués dans les réponses de l'API.

## Migration

Les deux peuvent fonctionner côte à côte, vous pouvez basculer module par module et revenir en arrière à tout moment. Consultez le [guide de migration](/fr/docs/guide/migration).
