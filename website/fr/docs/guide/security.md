# Modèle de sécurité et engagement de confidentialité

Gately fonctionne à la frontière du réseau et manipule des clés de fournisseurs DNS, des clés privées de certificats et d'autres identifiants : la sécurité et la transparence sont donc ses principaux objectifs de conception.

## Engagement de confidentialité

- **Zéro connexion sortante par défaut** : aucune connexion aux serveurs du projet, aucune remontée d'informations sur l'appareil, aucune poussée de configuration à distance.
- **Adresses externes visibles et modifiables** : la recherche de l'IP publique, les serveurs STUN et les adresses similaires sont affichés dans l'interface et peuvent être modifiés ou désactivés.
- **Connexion via des tiers sous votre contrôle** : les connexions GitHub, Google, QQ et OIDC utilisent des applications OAuth que vous enregistrez vous-même, sans relais intermédiaire.
- **Mises à niveau sous votre contrôle** : le programme ne remplace jamais son propre binaire en arrière-plan ; vous décidez quand mettre à niveau.

## Protection des données

- La configuration est du JSON en clair avec les droits 0600, ce qui permet de l'auditer et de revenir en arrière.
- Les champs sensibles sont masqués dans les réponses de l'API.
- Le jeton cloudflared est transmis par variable d'environnement et n'apparaît pas dans la liste des processus ; les jetons et clés frp et Tailscale sont stockés avec les droits 0600.

## Authentification et contrôle d'accès

- Hachage des mots de passe avec bcrypt et jetons de session signés HMAC ;
- Authentification à deux facteurs (TOTP, vérifiée avec les vecteurs de test de la RFC 6238) ;
- Limitation des tentatives de connexion ;
- Chemin d'accès secret, accès limité au réseau local et HTTPS pour l'interface d'administration ;
- API multi-jetons : chaque jeton n'a un accès en lecture/écriture qu'aux modules choisis à sa création, peut avoir une date d'expiration et enregistre l'IP source de sa dernière utilisation. Les points d'accès du compte, de l'authentification à deux facteurs, de la gestion des jetons, ainsi que de la sauvegarde, de la restauration, du redémarrage et des paramètres système n'acceptent que les sessions de connexion. Consultez [Jetons d'API et OpenAPI](/fr/docs/reference/api) ;
- Contrôle d'accès aux sites : bannissement automatique après échecs d'authentification, filtrage GeoIP par pays et règles de chemins. Consultez [Services Web](/fr/docs/modules/web) ;
- Les sauvegardes de configuration peuvent être chiffrées (scrypt + AES-GCM). Consultez [Sauvegarde de la configuration](/fr/docs/modules/backup).

## Valeurs par défaut des fonctions à risque

Le terminal de shell local est désactivé par défaut. Le terminal SSH applique une politique de confiance à la première utilisation (TOFU) pour enregistrer les empreintes des clés d'hôte.

## Signaler un problème de sécurité

Signalez les vulnérabilités en privé via la page Security du dépôt, et évitez d'en divulguer les détails dans les tickets publics : <https://github.com/wangwanjie/gately/security>

Pour d'autres recommandations pratiques, consultez le [guide de durcissement](/fr/docs/guide/hardening).
