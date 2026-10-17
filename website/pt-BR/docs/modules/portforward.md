# Redirecionamento de portas

Encaminhe uma porta em que o Gately escuta para outro dispositivo ou serviço.

## Recursos

- TCP e UDP;
- IPv4 e IPv6, com encaminhamento entre v4 e v6;
- Intervalos de portas e várias portas;
- Vários destinos com failover;
- Limite de conexões;
- Estatísticas de tráfego e logs de execução.

## Cenários típicos

- Expor as portas de serviço de um NAS por um único ponto de entrada;
- Fazer a ponte entre tráfego IPv6 e IPv4;
- Configurar um backend principal e outro em espera para um serviço.

## Dica de implantação

O redirecionamento de portas escuta diretamente nas portas do host. Em implantações com Docker, recomendamos `--network host`.

## Migrar do Lucky

As regras podem ser importadas de `lucky_portforward.lkcf`. Opções como criptografia TLS / de fluxo e limitação de velocidade ainda não são compatíveis, e um aviso é exibido durante a importação.
