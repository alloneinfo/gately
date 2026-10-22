---
title: Journal des modifications
description: Annonces et notes de version de chaque version de Gately.
---

# Journal des modifications

Les annonces des versions majeures sont listées ci-dessous. Consultez [GitHub Releases](https://github.com/wangwanjie/gately/releases) pour la liste complète des changements et les téléchargements de chaque version.

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
