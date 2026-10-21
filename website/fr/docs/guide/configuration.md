# Configuration et options d'exécution

## Options de ligne de commande

| Option | Description |
| --- | --- |
| `-c` | Dossier de configuration ; peut aussi être défini avec la variable d'environnement `GATELY_CONFIG_DIR` |
| `-listen` | Remplace l'adresse d'écoute de l'interface d'administration, p. ex. `:16680` |
| `-apply` | Dossier de configuration déclarative ; peut aussi être défini avec la variable d'environnement `GATELY_APPLY_DIR`, voir ci-dessous |
| `-version` | Affiche les informations de version |

| Variable d'environnement | Description |
| --- | --- |
| `GATELY_INITIAL_PASSWORD` | Mot de passe administrateur initial au premier démarrage (8 caractères minimum) ; à défaut, le mot de passe par défaut `gately.cn` est utilisé. Sans effet si le compte existe déjà |

## Dossier de configuration

Toute la configuration réside dans le dossier de configuration, avec un fichier texte `<nom>.json` par module, écrit de façon atomique avec les droits 0600. Vous pouvez donc :

- l'inspecter avec n'importe quel éditeur de texte ;
- placer le dossier dans git, auditer chaque changement avec des diff et revenir en arrière à tout moment ;
- le sauvegarder en copiant simplement le dossier.

::: warning Attention
`ssl.json` contient les clés privées des certificats. Avant de le valider dans un gestionnaire de versions, assurez-vous que le dépôt est privé, ou ajoutez le fichier à votre liste d'exclusion.
:::

## Configuration déclarative

Avec `-apply <dossier>`, chaque démarrage copie les fichiers `<nom>.json` de ce dossier par-dessus le dossier de configuration ; les fichiers au contenu identique ne sont pas réécrits. C'est pratique pour distribuer la configuration depuis un dépôt git, un ConfigMap Kubernetes ou un script de déploiement.

- Chaque fichier est vérifié comme JSON valide avant l'écriture ; si un seul fichier est invalide, Gately refuse de démarrer ;
- `account.json` et `tokens.json` ne peuvent pas être écrasés : le compte et les jetons se gèrent uniquement dans l'interface d'administration ;
- L'ancien contenu écrasé est conservé dans l'[historique de configuration](/fr/docs/guide/audit) et peut être restauré à tout moment.

## Dossier des programmes externes

Les modules Tailscale, frp et cloudflared supervisent les programmes externes officiels. Gately cherche chaque exécutable dans cet ordre :

1. Le chemin défini dans les paramètres du module ;
2. `PATH` ;
3. Le sous-dossier `bin/` du dossier de configuration.

Pour un déploiement Docker, placez les binaires dans le dossier de configuration monté, par exemple `./gately-data/bin/frpc`. Tailscale nécessite à la fois `tailscale` et `tailscaled`. Consultez [Tailscale / frp / cloudflared](/fr/docs/modules/tunnels) pour les détails.

## Journaux d'exécution

Les journaux d'exécution du système et de chaque module sont écrits par défaut dans `logs/` du répertoire de configuration ; ils restent donc consultables dans la page des journaux système et dans chaque module après un redémarrage ou une mise à jour :

- Un fichier JSON Lines par source : `logs/<ID du module>.log`, et `logs/gately.log` pour le système. Au démarrage, les entrées les plus récentes, dans la limite de « Lignes de journal conservées », sont restaurées depuis le fichier ;
- Les nouvelles entrées sont écrites par lots, au plus 1 seconde plus tard. Un fichier qui dépasse la limite (2 Mo par défaut) est renouvelé en `.1` et seul ce fichier précédent est conservé : chaque source occupe au plus le double de la limite ;
- Dans Paramètres → Réseau et exécution → Paramètres d'exécution, vous pouvez désactiver « Conserver les journaux sur disque » ou modifier « Limite du fichier journal », avec effet immédiat. Désactivés, les journaux restent uniquement en mémoire et sont effacés au redémarrage ;
- Effacer les journaux d'un module dans l'interface supprime aussi ses fichiers ;
- Les sauvegardes de configuration n'incluent ni `logs/` ni le journal d'audit `audit.log`.

## Sauvegarde et restauration

« Paramètres » propose la sauvegarde et la restauration de la configuration, et sert aussi de point d'entrée pour importer des configurations Lucky.
