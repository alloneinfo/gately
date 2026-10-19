# Redirecionamento de portas

Encaminhe uma porta em que o Gately escuta para outro dispositivo ou serviço.

## Recursos

- TCP e UDP;
- IPv4 e IPv6, com encaminhamento entre v4 e v6;
- Intervalos de portas e várias portas;
- Vários destinos com failover;
- Limite de conexões;
- Estatísticas de tráfego e logs de execução;
- Envio do protocolo PROXY (v1 / v2) ao destino, entregando ao serviço seguinte o endereço real do cliente.

## Protocolo PROXY

O redirecionamento de portas apenas repassa o TCP, então por padrão o serviço seguinte só vê o endereço do próprio Gately. Ao definir "Protocolo PROXY" como v1 ou v2 na regra, o Gately envia primeiro um cabeçalho PROXY depois de conectar ao destino, e o serviço seguinte obtém o endereço real do cliente para logs, limitação de taxa e controle de acesso.

O destino precisa ter a opção de recebimento correspondente ativada, caso contrário a conexão é recusada. Por exemplo:

- haproxy: `bind 192.168.1.8:4434 accept-proxy`;
- nginx: `listen 4435 ssl proxy_protocol;`;
- Serviço Web do Gately: ative "Aceitar protocolo PROXY" no listener e adicione a origem às origens PROXY confiáveis.

A opção vale apenas para TCP; os redirecionamentos UDP não são afetados.

## Cenários típicos

- Expor as portas de serviço de um NAS por um único ponto de entrada;
- Fazer a ponte entre tráfego IPv6 e IPv4;
- Configurar um backend principal e outro em espera para um serviço.

## Dica de implantação

O redirecionamento de portas escuta diretamente nas portas do host. Em implantações com Docker, recomendamos `--network host`.

## Migrar do Lucky

As regras podem ser importadas de `lucky_portforward.lkcf`. Opções como criptografia TLS / de fluxo e limitação de velocidade ainda não são compatíveis, e um aviso é exibido durante a importação.
