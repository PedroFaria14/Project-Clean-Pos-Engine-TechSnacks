import type { ReactNode } from "react";

export function EmptyState({ title, children, action }: { title: string; children: ReactNode; action?: ReactNode }) {
  return <div className="empty-state"><span className="empty-state__symbol" aria-hidden="true">◇</span><h3>{title}</h3><p>{children}</p>{action}</div>;
}

export function CatalogLoading() {
  return <div className="catalog-loading" role="status" aria-label="Carregando produtos"><span className="sr-only">Carregando produtos...</span>
    {[1, 2, 3, 4, 5, 6].map(item => <div key={item} className="skeleton" aria-hidden="true" />)}
  </div>;
}
