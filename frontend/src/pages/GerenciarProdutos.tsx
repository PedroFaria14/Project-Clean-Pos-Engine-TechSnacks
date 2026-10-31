import { useRef, useState } from "react";
import { AddProduct } from "./AddProduct";
import { ProductForm } from "../components/ProductForm";
import { Alert } from "../components/ui/Alert";
import { Button } from "../components/ui/Button";
import { Dialog } from "../components/ui/Dialog";
import { Field } from "../components/ui/Field";
import { CatalogLoading, EmptyState } from "../components/ui/EmptyState";
import { deleteProduct, type Product } from "../products/product_api";
import { formatPrice, searchText } from "../products/money";
import { useProducts } from "../products/use_products";

export function GerenciarProdutos() {
  const catalog = useProducts();
  const [search, setSearch] = useState("");
  const [form, setForm] = useState<Product | null | undefined>(undefined);
  const [deleting, setDeleting] = useState<Product | null>(null);
  const [busy, setBusy] = useState(false);
  const busyRef = useRef(false);
  const [deleteError, setDeleteError] = useState("");
  const [notice, setNotice] = useState("");
  const filtered = catalog.items.filter(item => searchText(item.name).includes(searchText(search)));
  const changeBusy = (value: boolean) => { busyRef.current = value; setBusy(value); };
  const closeForm = () => { if (!busyRef.current) setForm(undefined); };
  const closeDelete = () => { if (!busyRef.current) setDeleting(null); };
  const saved = (product: Product) => {
    catalog.upsert(product);
    if (!searchText(product.name).includes(searchText(search))) setSearch("");
    setForm(undefined);
    setNotice(form ? "Produto atualizado. Os pedidos anteriores permanecem inalterados." : "Produto cadastrado no catálogo.");
  };
  const confirmDelete = async () => {
    if (!deleting || busyRef.current) return;
    changeBusy(true);
    setDeleteError("");
    try {
      await deleteProduct(deleting.id);
      catalog.remove(deleting.id);
      setDeleting(null);
      setNotice("Produto excluído do catálogo. O histórico de pedidos foi preservado.");
    } catch (error) { setDeleteError(error instanceof Error ? error.message : "Não foi possível excluir o produto."); }
    finally { changeBusy(false); }
  };
  const formProps = { onSaved: saved, onCancel: closeForm, onBusyChange: changeBusy };

  return <>
    <header className="page-header"><div><span className="eyebrow">GESTÃO DO CATÁLOGO</span><h1>Produtos</h1><p>Organize o cardápio e mantenha os preços atualizados.</p></div>
      <Button disabled={catalog.loading || !!catalog.error} onClick={() => setForm(null)}><span aria-hidden="true">＋</span> Novo produto</Button>
    </header>
    {notice && <div className="notice"><Alert tone="success">{notice}</Alert></div>}
    <section className="panel" aria-label="Catálogo de produtos">
      <div className="products-toolbar"><Field label="Buscar produtos" type="search" value={search} placeholder="Buscar por nome..." onChange={event => setSearch(event.target.value)} />
        <span className="products-toolbar__count">{filtered.length} de {catalog.items.length} produtos</span></div>
      {catalog.loading ? <CatalogLoading /> : catalog.error ? <div className="panel-error"><Alert>{catalog.error}</Alert><Button variant="secondary" onClick={catalog.refresh}>Tentar novamente</Button></div> : catalog.items.length === 0 ?
        <EmptyState title="Seu catálogo começa aqui" action={<Button onClick={() => setForm(null)}>Cadastrar primeiro produto</Button>}>Adicione os produtos que estarão disponíveis no caixa.</EmptyState> : filtered.length === 0 ?
          <EmptyState title="Nenhum produto encontrado" action={<Button variant="secondary" onClick={() => setSearch("")}>Limpar busca</Button>}>Tente outro nome ou limpe a busca para ver o catálogo.</EmptyState> :
          <table className="product-table"><caption className="sr-only">Produtos cadastrados, preços e ações administrativas</caption>
            <thead><tr><th scope="col">Produto</th><th scope="col">Preço</th><th scope="col" className="actions-heading">Ações</th></tr></thead>
            <tbody>{filtered.map(product => <tr key={product.id}>
              <td className="product-table__name">{product.name}<small className="product-table__id">ID {product.id}</small></td>
              <td className="product-table__price">{formatPrice(product.price)}</td>
              <td><div className="product-table__actions"><Button variant="secondary" aria-label={`Editar ${product.name}`} onClick={() => setForm(product)}>Editar</Button>
                <Button variant="ghost" aria-label={`Excluir ${product.name}`} onClick={() => { setDeleteError(""); setDeleting(product); }}>Excluir</Button></div></td>
            </tr>)}</tbody>
          </table>}
    </section>
    {form !== undefined && <Dialog title={form ? "Editar produto" : "Novo produto"} onClose={closeForm} busy={busy}>
      <div className="dialog__body"><p>{form ? "Altere os dados do catálogo. Vendas já registradas não serão modificadas." : "Informe o nome e o preço para disponibilizar este item no caixa."}</p>
        {form ? <ProductForm key={form.id} product={form} {...formProps} /> : <AddProduct {...formProps} />}</div>
    </Dialog>}
    {deleting && <Dialog title="Excluir produto?" onClose={closeDelete} busy={busy}><div className="dialog__body">
      {deleteError && <Alert>{deleteError}</Alert>}
      <p><strong>{deleting.name}</strong> sairá do catálogo e não poderá ser adicionado a novas vendas. Pedidos anteriores continuarão disponíveis.</p>
      <div className="dialog__actions"><Button variant="secondary" autoFocus data-dialog-initial-focus disabled={busy} onClick={closeDelete}>Manter produto</Button>
        <Button variant="danger" disabled={busy} onClick={() => void confirmDelete()}>{busy ? "Excluindo..." : "Excluir produto"}</Button></div>
    </div></Dialog>}
  </>;
}
