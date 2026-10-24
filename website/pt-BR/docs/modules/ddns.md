# DNS dinâmico (DDNS)

O IP público de uma banda larga residencial muda com o tempo. O DDNS atualiza os registros DNS automaticamente depois de uma mudança de IP, para que o seu domínio sempre aponte para o endereço atual.

## Provedores compatíveis

Alibaba Cloud, Cloudflare, Tencent Cloud, DNSPod, Huawei Cloud, GoDaddy, Porkbun, dynv6, DuckDNS, Amazon Route 53, OVHcloud, Gandi, NS1, Bunny DNS, Namecheap, DynDNS2 e um callback personalizado (para provedores sem suporte embutido ou endpoints próprios).

| Provedor | Credenciais | Pode ser usado na validação DNS de certificados |
| --- | --- | --- |
| Amazon Route 53 | Access Key ID / Secret Access Key, ID da zona hospedada opcional | Sim |
| OVHcloud | Application Key / Secret e Consumer Key, endpoint ovh-eu / ovh-ca / ovh-us | Sim |
| Gandi | Personal Access Token | Sim |
| NS1 | API Key | Sim |
| Bunny DNS | Account API Key | Sim |
| Namecheap | Senha de Dynamic DNS do domínio (somente registros A) | Não |
| DynDNS2 | Usuário, senha e URL de atualização; para No-IP, Dynu e outros serviços compatíveis com o protocolo dyndns2 | Não |

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
