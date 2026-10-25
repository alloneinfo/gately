---
title: Journal des modifications
description: Annonces et notes de version de chaque version de Gately.
---

# Journal des modifications

Les annonces des versions majeures sont listées ci-dessous. Consultez [GitHub Releases](https://github.com/wangwanjie/gately/releases) pour la liste complète des changements et les téléchargements de chaque version.

## [v0.5.8.1](/fr/news/v0.5.8.1)

26 octobre 2026 · WebDAV, FTP et SMB partagent une liste d'utilisateurs partagés ; le Service web gagne le type de site Fermer la connexion et journalise le Referer et le User-Agent ; frps peut écouter sur des adresses choisies ; les boutons de copie fonctionnent en HTTP.

## [v0.5.8](/fr/news/v0.5.8)

25 octobre 2026 · Un fournisseur d'identité OIDC intégré permet aux utilisateurs partagés de se connecter à Gitea, Grafana et d'autres applications ; nouveau Service SMB ; tous les journaux sont filtrables ; les groupes de règles du WAF se réordonnent ; les en-têtes des sites se choisissent parmi les entrées courantes.

## [v0.5.7](/fr/news/v0.5.7)

24 octobre 2026 · Les Montages de stockage se connectent aux partages SMB ; les thèmes deviennent 15 palettes en clair et en sombre, avec accent et coins personnalisables ; l'interface d'administration est compressée et le binaire est environ 11 % plus léger.

## [v0.5.6.4](/fr/news/v0.5.6.4)

23 octobre 2026 · Le déploiement automatique des certificats copie vers plusieurs répertoires locaux, chacun avec sa commande, et accepte des noms de fichiers personnalisés pour le certificat et la clé.

## [v0.5.6.3](/fr/news/v0.5.6.3)

23 octobre 2026 · Corrige la vue d'ensemble qui masquait l'interface br-lan sous OpenWrt, ainsi que le fuseau horaire impossible à régler, l'heure restant en UTC.

## [v0.5.6.2](/fr/news/v0.5.6.2)

23 octobre 2026 · Nouvelles pages de connexion et d'erreur du service web, avec la vérification en deux étapes dans une fenêtre. Corrige l'échec de connexion du module Docker sous Docker 29, l'affichage furtif de la vue d'ensemble avant la page de connexion, ainsi que la synchronisation des identifiants Cloudflare et des enregistrements DDNS renommés importés de Lucky.

## [v0.5.6.1](/fr/news/v0.5.6.1)

22 octobre 2026 · Corrige le plantage d'un nft ancien sur l'hôte causé par les règles de pare-feu écrites par l'image Docker.

## [v0.5.6](/fr/news/v0.5.6)

22 octobre 2026 · Ajoute la prévention d'intrusion : règles de journaux à la fail2ban, ports pièges et détection de scan, avec des bannissements synchronisés dans nftables.

## [v0.5.5.2](/fr/news/v0.5.5.2)

22 octobre 2026 · Corrige la lecture possible par d'autres utilisateurs locaux des fichiers sensibles du répertoire de configuration sur fnOS et les systèmes de fichiers similaires.

## [v0.5.5.1](/fr/news/v0.5.5.1)

21 octobre 2026 · Les journaux d'exécution sont écrits sur disque et survivent aux redémarrages et mises à jour.

## [v0.5.5](/fr/news/v0.5.5)

21 octobre 2026 · Import de configurations nginx, déploiement automatique des certificats vers d'autres hôtes et conteneurs, gros envois et WebDAV derrière le WAF, listes d'IP abonnées ; l'auto-diagnostic vérifie désormais la couverture des certificats et les conflits frp.

## [v0.5.4](/fr/news/v0.5.4)

21 octobre 2026 · Mise à niveau en ligne pour les installations binaires Linux : plus besoin de recréer les conteneurs LXC de PVE ; nouveaux paquets OpenWrt (.ipk / .apk).

## [v0.5.3](/fr/news/v0.5.3)

21 octobre 2026 · Application native fnOS qui s'intègre à la passerelle unifiée fnOS et connecte les administrateurs fnOS avec leur session fnOS ; les modules fonctionnels peuvent être activés ou désactivés.

## [v0.5.2](/fr/news/v0.5.2)

20 octobre 2026 · Le routage SNI permet à plusieurs services de partager le port 443, les Services Web peuvent remplacer nginx, les utilisateurs partagés gagnent TOTP et l'authentification unique, et une comparaison des performances avec haproxy + nginx.

## [v0.5.0](/fr/news/v0.5.0)

19 octobre 2026 · Journal d'audit, historique de configuration et restauration, passkeys, comptes en lecture seule, ACME HTTP-01, HTTP/3, protocole PROXY, métriques Prometheus globales, auto-diagnostic réseau et versions signées.
