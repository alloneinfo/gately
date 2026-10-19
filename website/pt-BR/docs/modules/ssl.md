# Certificados SSL

Emita e renove certificados automaticamente para uso pelos [Serviços Web](/pt-BR/docs/modules/web) ou exporte-os para outros softwares.

## Recursos

- Emissão e renovação automáticas via **ACME DNS-01**, reaproveitando os provedores de DNS do [DNS dinâmico](/pt-BR/docs/modules/ddns), sem precisar abrir as portas 80 / 443;
- **ACME HTTP-01**: dispensa provedor de DNS, mas exige que a porta 80 de cada domínio seja acessível pela internet; não suporta wildcard;
- Certificados wildcard e EAB (por exemplo, ZeroSSL);
- Upload de um certificado existente ou leitura a partir de arquivos;
- Mapeamento para um diretório: quando um certificado muda, os arquivos são gravados no diretório indicado e um script é executado, para que outros serviços possam recarregar.

## Validação HTTP-01

Escolha HTTP-01 em "Método de validação" do certificado. Na emissão, a CA acessa `http://<domínio>/.well-known/acme-challenge/<token>`:

- Se um [Serviço Web](/pt-BR/docs/modules/web) já escuta na porta 80, ele responde diretamente, sem afetar os sites existentes;
- Se a porta 80 estiver livre, o Gately escuta nela temporariamente durante a emissão e a fecha logo em seguida;
- Se o roteador redireciona a porta 80 pública para outra porta deste host, informe essa porta em "Porta temporária".

Muitos provedores de banda larga residencial bloqueiam a entrada na porta 80; nesse caso, use DNS-01.

## Mapeamento de certificados

Com um certificado ativado, o Gately grava `.crt`, `.key`, `.pem` e `_issuerCertificate.crt` no diretório de mapeamento e executa o script de alteração.

## Nota de segurança

As chaves privadas ficam no `ssl.json` do diretório de configuração. Guarde esse arquivo com cuidado e, se for versioná-lo, confirme que o repositório é privado.

## Migrar do Lucky

Certificados já emitidos são importados como estão e não são solicitados de novo antes de expirar. No ZeroSSL, uma nova conta ACME é registrada com as credenciais EAB do Lucky. Veja os detalhes no [Guia de migração](/pt-BR/docs/guide/migration).
