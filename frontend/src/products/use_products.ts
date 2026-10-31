import { useEffect, useState } from "react";
import { listProducts, type Product } from "./product_api";

const sortProducts = (products: Product[]) => products.sort((a, b) => a.name.localeCompare(b.name, "pt-BR") || a.id.localeCompare(b.id));

export function useProducts() {
  const [state, setState] = useState<{ items: Product[]; loading: boolean; error: string | null }>({ items: [], loading: true, error: null });
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    void listProducts(controller.signal).then(items => {
      if (!controller.signal.aborted) setState({ items: sortProducts(items), loading: false, error: null });
    }).catch(error => {
      if (!controller.signal.aborted) setState(previous => ({ ...previous, loading: false, error: error instanceof Error ? error.message : "Não foi possível carregar o catálogo." }));
    });
    return () => controller.abort();
  }, [revision]);

  const refresh = () => { setState(previous => ({ ...previous, loading: true, error: null })); setRevision(previous => previous + 1); };
  const upsert = (product: Product) => setState(previous => ({ ...previous, items: sortProducts([...previous.items.filter(item => item.id !== product.id), product]) }));
  const remove = (id: string) => setState(previous => ({ ...previous, items: previous.items.filter(item => item.id !== id) }));
  return { ...state, refresh, upsert, remove };
}
