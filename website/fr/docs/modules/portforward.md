# Redirection de ports

Redirigez un port sur lequel Gately écoute vers un autre appareil ou service.

## Fonctions

- TCP et UDP ;
- IPv4 et IPv6, avec redirection entre v4 et v6 ;
- Plages de ports et ports multiples ;
- Cibles multiples avec basculement ;
- Limites de connexions ;
- Statistiques de trafic et journaux d'exécution.

## Scénarios typiques

- Exposer les ports des services d'un NAS via un point d'entrée unique ;
- Faire le pont entre le trafic IPv6 et IPv4 ;
- Configurer un backend principal et un backend de secours pour un même service.

## Conseil de déploiement

La redirection de ports écoute directement sur les ports de l'hôte. Pour un déploiement Docker, `--network host` est recommandé.

## Migration depuis Lucky

Les règles peuvent être importées depuis `lucky_portforward.lkcf`. Les options telles que le chiffrement TLS / de flux et la limitation de débit ne sont pas encore prises en charge, et un avertissement s'affiche lors de l'import.
