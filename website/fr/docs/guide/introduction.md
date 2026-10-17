# Introduction

Gately est une boîte à outils de passerelle réseau open source et auditable, destinée aux réseaux domestiques, aux NAS, aux serveurs cloud et à tous types de serveurs. Inspiré de [Lucky](https://github.com/gdy666/lucky), il offre des fonctions comparables sous une forme entièrement libre. DDNS, traversée NAT STUN, redirection de ports, proxy inverse, certificats, WAF et bien plus sont réunis dans un seul binaire d'environ 22 Mo, et les configurations Lucky s'importent en un clic.

## Pourquoi Gately

Lucky est un outil très apprécié des utilisateurs de NAS et de réseaux domestiques, surtout en Chine. Il rassemble DDNS, traversée STUN, proxy inverse et bien d'autres choses, et il a donné son orientation à Gately. Sans Lucky, Gately n'existerait pas. Lucky est distribué gratuitement mais son code est fermé ; Gately est entièrement open source, si bien que les utilisateurs et la communauté peuvent examiner directement le code et la configuration. Pour un logiciel placé à la frontière du réseau et qui détient des clés DNS et des clés privées de certificats, l'open source permet de voir exactement ce qu'il fait. Les engagements de Gately :

- Code source ouvert sous Apache-2.0, ouvert à la relecture par la communauté
- Aucune connexion aux serveurs du projet et aucune remontée d'informations par défaut ; la connexion via des tiers utilise vos propres applications OAuth, sans relais
- Chaque adresse externe (recherche de l'IP publique, serveurs STUN, etc.) est visible, modifiable et désactivable
- La configuration est du JSON en clair, prête pour l'audit et le retour arrière avec git

## Principes de conception

1. **Transparence** : aucune connexion aux serveurs du projet par défaut ; chaque adresse externe est visible, modifiable et désactivable.
2. **Migration facile** : les champs de configuration restent aussi proches que possible de ceux de Lucky, avec import de `.lkcf`.
3. **Configuration auditable** : JSON en clair (droits 0600) qui se versionne directement avec git pour les diff et les retours arrière.
4. **Déploiement en un seul fichier** : un binaire statique avec le frontend intégré, compatible Docker, systemd et OpenWrt.

## Pile technique

| Couche | Choix |
| --- | --- |
| Backend | Go, compilé statiquement, compilation croisée pour routeurs et NAS arm / arm64 / mips / riscv |
| HTTP | Bibliothèque standard `net/http` |
| Stockage | Un fichier JSON en clair par module, écrit de façon atomique |
| Authentification | Mots de passe bcrypt + jetons de session signés HMAC, limitation des tentatives de connexion, authentification à deux facteurs TOTP |
| Frontend | Vue 3 + TypeScript + Vite + Element Plus, intégré au binaire via `go:embed` |

## Aperçu des fonctions

DNS dynamique, traversée STUN, redirection de ports, certificats SSL, services Web (proxy inverse, contrôle d'accès, vérifications d'état, cache et limitation de débit, statistiques de trafic), pare-feu WAF, Tailscale, frp, tunnel Cloudflare, gestion de Docker, terminal Web, diagnostic réseau, Wake-on-LAN, WebDAV, FTP, gestion du stockage, tâches planifiées, centre de notifications et sauvegarde de la configuration, ainsi que l'authentification à deux facteurs, le chemin d'accès secret, les jetons d'API, la CLI et MCP. Consultez la [vue d'ensemble des modules](/fr/docs/modules/) pour les détails.

## Étapes suivantes

- Pour essayer tout de suite : [Démarrage rapide](/fr/docs/guide/quick-start)
- Vous utilisez déjà Lucky : [Guide de migration](/fr/docs/guide/migration)
- Vous tenez à la confidentialité et à la sécurité : [Modèle de sécurité et engagement de confidentialité](/fr/docs/guide/security)
