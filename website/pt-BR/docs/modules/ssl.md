# Certificados SSL

Emita e renove certificados automaticamente para uso pelos [Serviços Web](/pt-BR/docs/modules/web) ou exporte-os para outros softwares.

## Recursos

- Emissão e renovação automáticas via **ACME DNS-01**, reaproveitando os provedores de DNS do [DNS dinâmico](/pt-BR/docs/modules/ddns), sem precisar abrir as portas 80 / 443;
- Certificados wildcard e EAB (por exemplo, ZeroSSL);
- Upload de um certificado existente ou leitura a partir de arquivos;
- Mapeamento para um diretório: quando um certificado muda, os arquivos são gravados no diretório indicado e um script é executado, para que outros serviços possam recarregar.

## Mapeamento de certificados

Com um certificado ativado, o Gately grava `.crt`, `.key`, `.pem` e `_issuerCertificate.crt` no diretório de mapeamento e executa o script de alteração.

## Nota de segurança

As chaves privadas ficam no `ssl.json` do diretório de configuração. Guarde esse arquivo com cuidado e, se for versioná-lo, confirme que o repositório é privado.

## Migrar do Lucky

Certificados já emitidos são importados como estão e não são solicitados de novo antes de expirar. No ZeroSSL, uma nova conta ACME é registrada com as credenciais EAB do Lucky. Veja os detalhes no [Guia de migração](/pt-BR/docs/guide/migration).
