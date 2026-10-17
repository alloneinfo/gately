# DNS dinâmico (DDNS)

O IP público de uma banda larga residencial muda com o tempo. O DDNS atualiza os registros DNS automaticamente depois de uma mudança de IP, para que o seu domínio sempre aponte para o endereço atual.

## Provedores compatíveis

Alibaba Cloud, Cloudflare, Tencent Cloud, DNSPod, Huawei Cloud, GoDaddy, Porkbun, dynv6, DuckDNS e um callback personalizado (para provedores sem suporte embutido ou endpoints próprios).

## Como o IP é obtido

- **URL**: consulta um endereço de verificação de IP público (substitua por qualquer serviço ou hospede o seu);
- **Interface de rede**: lê diretamente o endereço de uma interface de rede específica, indicado para dispositivos que já têm IP público;
- **Comando**: executa um comando e usa a saída dele, indicado para ambientes de rede incomuns.

IPv4 e IPv6 são compatíveis.

## Notificações

Um Webhook pode ser chamado quando o IP muda ou a sincronização falha.

## Recomendações

- Crie, no provedor de DNS, uma chave de subconta limitada à edição de registros DNS e evite usar a chave da conta principal;
- Depois de criar uma tarefa, clique em "Forçar sincronização" para confirmar que o registro foi atualizado;
- O endereço de consulta do IP público não depende mais de nenhum serviço operado pelo projeto e pode ser trocado em uma tarefa a qualquer momento.

## Migrar do Lucky

As tarefas de DDNS podem ser importadas de `lucky_ddns.lkcf`, com 11 provedores compatíveis. Os endereços oficiais de consulta de IP do Lucky são removidos na importação. Veja os detalhes no [Guia de migração](/pt-BR/docs/guide/migration).
