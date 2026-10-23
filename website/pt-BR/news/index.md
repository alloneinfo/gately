---
title: Changelog
description: Anúncios e notas de cada versão do Gately.
---

# Changelog

Abaixo estão os anúncios das versões principais. Veja as [GitHub Releases](https://github.com/wangwanjie/gately/releases) para as mudanças completas e os downloads de cada versão.

## [v0.5.6.4](/pt-BR/news/v0.5.6.4)

23 de outubro de 2026 · A implantação automática de certificados copia para vários diretórios locais, cada um com seu comando, e aceita nomes de arquivo personalizados para o certificado e a chave.

## [v0.5.6.3](/pt-BR/news/v0.5.6.3)

23 de outubro de 2026 · Corrige a visão geral que ocultava a interface br-lan no OpenWrt e o fuso horário que não podia ser definido, com o horário sempre em UTC.

## [v0.5.6.2](/pt-BR/news/v0.5.6.2)

23 de outubro de 2026 · Novo visual das páginas de login e de erro do serviço web, com a verificação em duas etapas em uma janela. Corrige o módulo Docker que não conectava no Docker 29, a visão geral que aparecia por um instante antes da página de login e credenciais Cloudflare e registros DDNS renomeados importados do Lucky que não sincronizavam.

## [v0.5.6.1](/pt-BR/news/v0.5.6.1)

22 de outubro de 2026 · Corrige as regras de firewall gravadas pela imagem Docker que faziam um nft antigo no host travar.

## [v0.5.6](/pt-BR/news/v0.5.6)

22 de outubro de 2026 · Nova prevenção de intrusão: regras de logs no estilo fail2ban, portas armadilha e detecção de varredura, com bloqueios sincronizados no nftables.

## [v0.5.5.2](/pt-BR/news/v0.5.5.2)

22 de outubro de 2026 · Corrige a leitura de arquivos sensíveis do diretório de configuração por outros usuários locais no fnOS e em sistemas de arquivos semelhantes.

## [v0.5.5.1](/pt-BR/news/v0.5.5.1)

21 de outubro de 2026 · Os logs de execução são gravados em disco e sobrevivem a reinícios e atualizações.

## [v0.5.5](/pt-BR/news/v0.5.5)

21 de outubro de 2026 · Importação de configurações do nginx, implantação automática de certificados em outros hosts e contêineres, WAF com suporte a uploads grandes e WebDAV, listas de IP assinadas e novas verificações de cobertura de certificados e conflitos do frp no autodiagnóstico.

## [v0.5.4](/pt-BR/news/v0.5.4)

21 de outubro de 2026 · Atualização online para instalações binárias no Linux, sem precisar recriar contêineres LXC do PVE; novos pacotes para OpenWrt (.ipk / .apk).

## [v0.5.3](/pt-BR/news/v0.5.3)

21 de outubro de 2026 · Aplicativo nativo para fnOS, integrado ao gateway unificado do fnOS e com login automático para administradores do fnOS pela sessão do fnOS; módulos de recursos podem ser ativados ou desativados.

## [v0.5.2](/pt-BR/news/v0.5.2)

20 de outubro de 2026 · O roteamento SNI permite que vários serviços compartilhem a porta 443, os Serviços Web podem substituir o nginx, os usuários compartilhados ganham TOTP e login único, e uma comparação de desempenho com haproxy + nginx.

## [v0.5.0](/pt-BR/news/v0.5.0)

19 de outubro de 2026 · Log de auditoria, histórico de configuração e reversão, passkey, contas somente leitura, ACME HTTP-01, HTTP/3, protocolo PROXY, métricas globais do Prometheus, autodiagnóstico de rede e versões assinadas.
