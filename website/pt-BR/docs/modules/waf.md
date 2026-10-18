# Firewall WAF

Um firewall de aplicações web baseado no [Coraza](https://coraza.io/) e no OWASP CRS v4, que pode ser associado a sites dos [Serviços Web](/pt-BR/docs/modules/web).

## Recursos

- Conjunto de regras OWASP CRS v4 embutido, com arquivos de regras que podem ser ativados individualmente;
- Regras SecLang personalizadas;
- Modo de bloqueio e modo apenas de detecção;
- Várias instâncias, referenciadas pelos sites conforme necessário.

## Recomendações

1. Em um site novo, comece com "Apenas detectar" e acompanhe os logs por algum tempo para confirmar que não há falsos positivos;
2. Ajuste os arquivos de regras ou adicione exceções personalizadas para os falsos positivos;
3. Depois mude para "Bloquear".

## Verificação

O WAF é testado com payloads de ataque reais e bloqueia ataques como path traversal.
