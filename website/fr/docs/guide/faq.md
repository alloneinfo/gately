# FAQ

## Quel est le lien entre Gately et Lucky ?

Gately est une implémentation open source indépendante, sans affiliation avec le projet Lucky. L'idée du produit vient de Lucky, et sans Lucky, Gately n'existerait pas ; merci à son auteur et à sa communauté. La compatibilité repose sur des comportements, interfaces et formats de configuration observables publiquement, et le dépôt refuse tout code copié des binaires ou du frontend de Lucky.

## Faut-il se connecter aux serveurs du projet ?

Non. Par défaut, Gately ne se connecte à aucun serveur du projet et ne remonte aucune information sur l'appareil. Les adresses externes, comme la recherche de l'IP publique, utilisent des services publics tiers et se modifient dans les tâches correspondantes.

## Où trouver le mot de passe initial ?

Il est affiché dans le journal au premier démarrage, une seule fois. Pour un déploiement Docker, exécutez `docker logs gately 2>&1 | grep -i "initial password"`.

## Que faire si j'ai oublié le mot de passe administrateur ?

Arrêtez Gately, supprimez `account.json` du dossier de configuration, puis redémarrez-le. Le programme recrée le compte `admin` et affiche un nouveau mot de passe initial dans le journal. Après la connexion, vérifiez les paramètres de sécurité comme l'authentification à deux facteurs.

## Pourquoi recommander `--network host` ?

La redirection de ports, STUN et les services Web écoutent directement sur les ports de l'hôte. Avec le réseau bridge, il faut publier soi-même chaque port utilisé.

## Peut-il fonctionner en parallèle de Lucky ?

Oui. Le port d'administration est par défaut 16680, sans conflit avec le 16601 de Lucky. Les règles importées sont désactivées par défaut pour ne pas entrer en concurrence avec un Lucky encore actif sur les ports et les mappages UPnP du routeur.

## Pourquoi dois-je fournir moi-même les binaires de Tailscale, frp et cloudflared ?

Ces trois modules supervisent les programmes externes officiels. Gately ne les embarque pas, ce qui permet d'utiliser à tout moment la dernière version amont. Consultez [Configuration et options d'exécution](/fr/docs/guide/configuration) pour savoir où les placer.

## Ma règle STUN est activée mais je ne peux toujours pas me connecter de l'extérieur

Si l'interface WAN de votre routeur se trouve derrière un NAT de niveau opérateur (100.64.0.0/10), les hôtes extérieurs ne peuvent pas atteindre le port traversé. C'est une limite du réseau lui-même. Utilisez plutôt [Tailscale / frp / tunnel Cloudflare](/fr/docs/modules/tunnels).

## Comment signaler un problème ou contribuer ?

Ouvrez un ticket sur [GitHub Issues](https://github.com/wangwanjie/gately/issues). Les conventions de développement figurent dans [Contribuer](/fr/docs/guide/contributing).
