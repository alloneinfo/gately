# DNS dynamique (DDNS)

L'IP publique d'une connexion internet domestique change avec le temps. Le DDNS met à jour automatiquement vos enregistrements DNS après un changement d'IP, afin que votre domaine pointe toujours vers votre adresse actuelle.

## Fournisseurs pris en charge

Alibaba Cloud, Cloudflare, Tencent Cloud, DNSPod, Huawei Cloud, GoDaddy, Porkbun, dynv6, DuckDNS, ainsi qu'un rappel personnalisé (pour les fournisseurs non intégrés ou les points de terminaison auto-hébergés).

## Obtention de l'IP

- **URL** : interroge une adresse de consultation d'IP publique (remplacez-la par le service de votre choix, ou hébergez le vôtre) ;
- **Interface réseau** : lit directement l'adresse d'une interface réseau précise, adapté aux appareils qui détiennent une IP publique ;
- **Commande** : exécute une commande et utilise sa sortie, adapté aux configurations réseau atypiques.

IPv4 et IPv6 sont tous deux pris en charge.

## Notifications

Un Webhook peut être appelé lorsque l'IP change ou qu'une synchronisation échoue.

## Recommandations

- Créez chez votre fournisseur DNS une clé de sous-compte limitée à la modification des enregistrements DNS. Évitez d'utiliser la clé du compte principal ;
- Après avoir créé une tâche, cliquez sur « Forcer la synchronisation » pour vérifier que l'enregistrement a bien été mis à jour ;
- L'adresse de consultation de l'IP publique ne dépend plus d'aucun service exploité par le projet, et vous pouvez la modifier à tout moment dans une tâche.

## Migration depuis Lucky

Les tâches DDNS peuvent être importées depuis `lucky_ddns.lkcf`, pour 11 fournisseurs. Les adresses de consultation d'IP officielles de Lucky sont supprimées lors de l'import. Consultez le [Guide de migration](/fr/docs/guide/migration) pour plus de détails.
