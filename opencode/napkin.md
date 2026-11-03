# Guardanapo (Napkin)

## Regras de Curadoria

- Manter apenas orientações recorrentes, com data e ação explícita.
- Ordenar por prioridade e limitar cada categoria a 10 itens.

## Execução e Validação (Prioridade Máxima)

1. **[2026-11-03] Proxy da API no deploy Vercel**
   Faça isto em vez disso: mantenha `frontend/vercel.json` com o rewrite de `/api/:path*` para o backend sem o prefixo `/api`, antes do fallback SPA. O navegador usa chamadas same-origin com cookies; o proxy de desenvolvimento do Vite e o Nginx do Docker não configuram a Vercel.
