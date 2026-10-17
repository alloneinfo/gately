# Comparação com o Lucky

O Gately é inspirado no Lucky, cobre os recursos mais comuns de forma open source e importa configurações existentes. O Lucky está à frente em abrangência de recursos e base de usuários. Sem o Lucky não haveria o Gately, e fica aqui o nosso agradecimento.

## Diferenças de filosofia

| Item | Gately | Lucky |
| --- | --- | --- |
| Código-fonte | Código aberto (Apache-2.0) | Código fechado |
| Arquivo de configuração | JSON em texto simples, compatível com git | Formato proprietário (`.lkcf`) |
| Serviços oficiais | Nenhum por padrão; todos os endereços externos são editáveis e podem ser desativados | Atualização online, intermediário de login de terceiros e outros serviços oficiais |
| Login de terceiros | Seus próprios aplicativos OAuth | Pelo serviço intermediário oficial |
| Atualizações | Docker / gerenciador de pacotes | Autoatualização online embutida |
| Abrangência de recursos | Cobre os recursos centrais mais usados | Mais ampla, como DLNA e montagem de armazenamento em nuvem |

## Cobertura de recursos

| Recurso | Gately |
| --- | --- |
| DNS dinâmico | ✅ |
| Travessia STUN | ✅ |
| Redirecionamento de portas | ✅ |
| Certificados SSL | ✅ |
| Serviços Web (proxy reverso) | ✅ |
| WAF | ✅ |
| Tailscale / frp / cloudflared | ✅ |
| Gerenciamento do Docker | ✅ |
| Terminal Web | ✅ |
| WebDAV / FTP / gerenciamento de armazenamento | ✅ |
| Wake-on-LAN / tarefas agendadas | ✅ |
| Central de notificações / backup da configuração / diagnóstico de rede | ✅ |
| Tokens de API / OpenAPI / CLI / MCP | ✅ |
| DLNA, montagem de armazenamento em nuvem, rclone, filtro global de IP, HTTP/3 | Planejado, conforme a demanda da comunidade |

## Diferenças intencionais

- Sem autoatualização online que sobrescreva o binário; as atualizações ficam a cargo do gerenciador de pacotes ou do Docker.
- Sem envio de configuração remota e sem envio de informações do dispositivo.
- A configuração é armazenada em texto simples, e campos sensíveis são mascarados na saída da API.

## Migração

Os dois podem rodar lado a lado, com troca módulo por módulo e rollback a qualquer momento. Veja o [Guia de migração](/pt-BR/docs/guide/migration).
