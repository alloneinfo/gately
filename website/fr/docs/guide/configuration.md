# Configuration et options d'exécution

## Options de ligne de commande

| Option | Description |
| --- | --- |
| `-c` | Dossier de configuration ; peut aussi être défini avec la variable d'environnement `GATELY_CONFIG_DIR` |
| `-listen` | Remplace l'adresse d'écoute de l'interface d'administration, p. ex. `:16680` |
| `-version` | Affiche les informations de version |

## Dossier de configuration

Toute la configuration réside dans le dossier de configuration, avec un fichier texte `<nom>.json` par module, écrit de façon atomique avec les droits 0600. Vous pouvez donc :

- l'inspecter avec n'importe quel éditeur de texte ;
- placer le dossier dans git, auditer chaque changement avec des diff et revenir en arrière à tout moment ;
- le sauvegarder en copiant simplement le dossier.

::: warning Attention
`ssl.json` contient les clés privées des certificats. Avant de le valider dans un gestionnaire de versions, assurez-vous que le dépôt est privé, ou ajoutez le fichier à votre liste d'exclusion.
:::

## Dossier des programmes externes

Les modules Tailscale, frp et cloudflared supervisent les programmes externes officiels. Gately cherche chaque exécutable dans cet ordre :

1. Le chemin défini dans les paramètres du module ;
2. `PATH` ;
3. Le sous-dossier `bin/` du dossier de configuration.

Pour un déploiement Docker, placez les binaires dans le dossier de configuration monté, par exemple `./gately-data/bin/frpc`. Tailscale nécessite à la fois `tailscale` et `tailscaled`. Consultez [Tailscale / frp / cloudflared](/fr/docs/modules/tunnels) pour les détails.

## Sauvegarde et restauration

« Paramètres » propose la sauvegarde et la restauration de la configuration, et sert aussi de point d'entrée pour importer des configurations Lucky.
