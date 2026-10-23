# Certificados SSL

Emita e renove certificados automaticamente para uso pelos [Serviços Web](/pt-BR/docs/modules/web) ou exporte-os para outros softwares.

## Recursos

- Emissão e renovação automáticas via **ACME DNS-01**, reaproveitando os provedores de DNS do [DNS dinâmico](/pt-BR/docs/modules/ddns), sem precisar abrir as portas 80 / 443;
- **ACME HTTP-01**: dispensa provedor de DNS, mas exige que a porta 80 de cada domínio seja acessível pela internet; não suporta wildcard;
- Certificados wildcard e EAB (por exemplo, ZeroSSL);
- Upload de um certificado existente ou leitura a partir de arquivos;
- Mapeamento para um diretório: quando um certificado muda, os arquivos são gravados no diretório indicado e um script é executado, para que outros serviços possam recarregar;
- Implantação automática: quando um certificado muda, ele é copiado para vários diretórios locais, enviado por SSH a outros hosts com a execução de um comando, ou os contêineres Docker que o usam são reiniciados.

## Validação HTTP-01

Escolha HTTP-01 em "Método de validação" do certificado. Na emissão, a CA acessa `http://<domínio>/.well-known/acme-challenge/<token>`:

- Se um [Serviço Web](/pt-BR/docs/modules/web) já escuta na porta 80, ele responde diretamente, sem afetar os sites existentes;
- Se a porta 80 estiver livre, o Gately escuta nela temporariamente durante a emissão e a fecha logo em seguida;
- Se o roteador redireciona a porta 80 pública para outra porta deste host, informe essa porta em "Porta temporária".

Muitos provedores de banda larga residencial bloqueiam a entrada na porta 80; nesse caso, use DNS-01.

## Mapeamento de certificados

Com um certificado ativado, o Gately grava `.crt`, `.key`, `.pem` e `_issuerCertificate.crt` no diretório de mapeamento e executa o script de alteração.

## Implantação automática

Em "Implantação automática" do certificado, você pode adicionar vários destinos, executados automaticamente depois que o certificado é emitido, renovado ou atualizado:

- **Copiar para diretório local**: grava o certificado em um diretório desta máquina e executa localmente o comando "Executar após copiar". É possível adicionar vários diretórios, cada um com um comando diferente. Se os arquivos de um diretório forem apagados ou alterados, a próxima verificação os copia de novo. Quando o Gately roda no Docker, o diretório e o comando ficam dentro do contêiner;
- **Enviar para host SSH**: escolha uma conexão SSH já existente no [Terminal Web](/pt-BR/docs/modules/terminal), reaproveitando seu endereço, método de autenticação e proxy. Os arquivos do certificado (`<observação>.crt`, `.key`, `.pem` e `_issuerCertificate.crt`, iguais aos do mapeamento de certificados, por isso a observação é obrigatória) são gravados no diretório remoto e, em seguida, o comando de "Executar após enviar" roda no host remoto, por exemplo `systemctl reload nginx`. O comando pode usar `$certName`, `$certDir`, `$certFile` e `$keyFile`. Cada arquivo é gravado primeiro em um arquivo temporário com as permissões definidas e depois renomeado para substituir o original; a chave privada fica com permissão 0600;
- **Reiniciar contêineres locais**: quando o certificado muda, os contêineres Docker selecionados são reiniciados em sequência. Útil para serviços que montam o diretório de mapeamento e precisam reiniciar para carregar o novo certificado.

"Copiar para diretório local" e "Enviar para host SSH" aceitam nomes de arquivo para o certificado (cadeia completa) e a chave privada, como `fullchain.pem` e `privkey.pem`. Com ambos preenchidos, apenas esses dois arquivos são gravados e a observação pode ficar vazia; vazios, são usados os nomes do mapeamento de certificados. `$certFile` e `$keyFile` no comando apontam para os arquivos realmente gravados.

A implantação automática nunca confia em um host desconhecido: conecte-se uma vez pela conexão SSH escolhida no Terminal Web para registrar a chave do host antes do primeiro envio. Implantações com falha são repetidas a cada hora e geram o evento "Certificado: falha na implantação", que pode ser assinado na [Central de notificações](/pt-BR/docs/modules/notify). O botão "Implantar agora" no cartão do certificado ignora o intervalo de nova tentativa, implanta imediatamente em todos os destinos ativados e mostra o resultado de cada um.

Ao editar o certificado, destinos com configuração inalterada não são implantados de novo; alterar o diretório remoto, o comando ou os contêineres faz uma nova implantação.

## Nota de segurança

As chaves privadas ficam no `ssl.json` do diretório de configuração. Guarde esse arquivo com cuidado e, se for versioná-lo, confirme que o repositório é privado.

## Migrar do Lucky

Certificados já emitidos são importados como estão e não são solicitados de novo antes de expirar. No ZeroSSL, uma nova conta ACME é registrada com as credenciais EAB do Lucky. Veja os detalhes no [Guia de migração](/pt-BR/docs/guide/migration).
