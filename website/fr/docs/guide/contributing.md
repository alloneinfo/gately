# Contribuer

Gately est un projet open source porté par la communauté. Les tickets et les pull requests sont les bienvenus. Dépôt : <https://github.com/wangwanjie/gately>

## Développement local

Prérequis : Go 1.26+, Node.js 24+, pnpm.

```bash
make web      # compile le frontend (sortie dans web/dist, intégré via go:embed)
make build    # compile bin/gately
make test     # go test -race ./...
make lint     # gofmt, go vet, vérification des types du frontend
```

Pour le développement du frontend, lancez `cd web && pnpm dev` ; `/api` est relayé vers `127.0.0.1:16680` en local.

## Conventions

- Le code de compatibilité doit reposer uniquement sur des comportements, interfaces et formats de configuration observables publiquement. Ne soumettez pas de code copié des binaires ou du frontend d'un logiciel tiers.
- Un nouveau module requiert : une structure de configuration avec validation, un démarrage/arrêt à l'exécution, des points d'accès REST et des tests unitaires. Les modules qui prennent en charge l'import de configuration implémentent `luckyimport.Importer`, et ceux qui doivent afficher leur état sur la page de vue d'ensemble implémentent `module.Summarizer`.
- Les commentaires du code sont rédigés en chinois ; les instructions de contrôle doivent utiliser des accolades.

## Tests supplémentaires

Certains tests sont ignorés par défaut et demandent une préparation supplémentaire :

| Test | Comment l'activer |
| --- | --- |
| ACME de bout en bout (Pebble) | Installez pebble et pebble-challtestsrv, puis définissez `GATELY_PEBBLE_BIN` et `GATELY_PEBBLE_SRC` |
| Vrais serveurs STUN publics | `GATELY_NET_TEST=1` |
| Vérification de l'import avec une vraie configuration Lucky | `GATELY_LUCKY_SAMPLE` et autres ; voir `TestImportRealSample` de chaque module |

## Licence

[Apache License 2.0](https://github.com/wangwanjie/gately/blob/main/LICENSE)
