# Redirection de ports

Redirigez un port sur lequel Gately écoute vers un autre appareil ou service.

## Fonctions

- TCP et UDP ;
- IPv4 et IPv6, avec redirection entre v4 et v6 ;
- Plages de ports et ports multiples ;
- Cibles multiples avec basculement ;
- Limites de connexions ;
- Statistiques de trafic et journaux d'exécution ;
- Envoi du protocole PROXY (v1 / v2) à la cible, pour transmettre l'adresse réelle du client en aval.

## Protocole PROXY

La redirection de ports ne fait que relayer le TCP : par défaut, le service en aval ne voit que l'adresse de Gately. Si vous réglez « Protocole PROXY » sur v1 ou v2 dans une règle, Gately envoie d'abord un en-tête PROXY après s'être connecté à la cible ; le service en aval obtient alors l'adresse réelle du client pour ses journaux, sa limitation de débit et son contrôle d'accès.

La cible doit activer l'option de réception correspondante, sinon la connexion sera refusée. Par exemple :

- haproxy : `bind 192.168.1.8:4434 accept-proxy` ;
- nginx : `listen 4435 ssl proxy_protocol;` ;
- Service Web Gately : activez « Accepter le protocole PROXY » sur l'écouteur et ajoutez la source aux sources PROXY de confiance.

Cette option ne concerne que le TCP ; la redirection UDP n'est pas affectée.

## Scénarios typiques

- Exposer les ports des services d'un NAS via un point d'entrée unique ;
- Faire le pont entre le trafic IPv6 et IPv4 ;
- Configurer un backend principal et un backend de secours pour un même service.

## Conseil de déploiement

La redirection de ports écoute directement sur les ports de l'hôte. Pour un déploiement Docker, `--network host` est recommandé.

## Migration depuis Lucky

Les règles peuvent être importées depuis `lucky_portforward.lkcf`. Les options telles que le chiffrement TLS / de flux et la limitation de débit ne sont pas encore prises en charge, et un avertissement s'affiche lors de l'import.
