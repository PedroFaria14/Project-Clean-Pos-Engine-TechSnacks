import { useRef, useState } from "react";
import {
  clearPendingCheckout, clearLegacyPendingCheckout, createPendingCheckout, loadPendingCheckout, savePendingCheckout,
  type PendingCheckout,
} from "../checkout/pendingCheckout";
import { apiRequest } from "../api/http";
import { useSession } from "../auth/session_context";
import type { Product } from "../products/product_api";
import { cartTotal, formatPrice, searchText } from "../products/money";
import { useProducts } from "../products/use_products";
import { Alert } from "./ui/Alert";
import { Button } from "./ui/Button";
import { Dialog } from "./ui/Dialog";
import { Field } from "./ui/Field";
import { CatalogLoading, EmptyState } from "./ui/EmptyState";

interface CartItem extends Product {
  quantity: number;
}

interface CheckoutResult {
  order_id: string;
  total_cents: number;
  status: "AWAITING_PAYMENT";
}

function isCheckoutResult(value: unknown): value is CheckoutResult {
  if (typeof value !== "object" || value === null) return false;

  const result = value as Record<string, unknown>;
  return (
    typeof result.order_id === "string" &&
    result.order_id.trim().length > 0 &&
    typeof result.total_cents === "number" &&
    Number.isSafeInteger(result.total_cents) &&
    result.total_cents >= 0 &&
    result.status === "AWAITING_PAYMENT"
  );
}

export function CaixaPDV() {
  const { session } = useSession();
  const userId = session?.user.id ?? "";
  const isAdmin = session?.user.role === "admin";
  const [recovery] = useState(() => loadPendingCheckout(userId));
  const [legacyRecovery, setLegacyRecovery] = useState(recovery.legacy);
  const [pendingCheckout, setPendingCheckout] = useState<PendingCheckout | null>(recovery.pending);
  const pendingCheckoutRef = useRef<PendingCheckout | null>(recovery.pending);
  const [storageError, setStorageError] = useState<string | null>(recovery.error);
  const catalog = useProducts();
  const [search, setSearch] = useState("");
  const [confirmClear, setConfirmClear] = useState(false);
  const [cart, setCart] = useState<CartItem[]>(recovery.pending?.items ?? []);
  const cartRef = useRef<CartItem[]>(recovery.pending?.items ?? []);
  const [isCheckoutLoading, setIsCheckoutLoading] = useState(false);
  const [checkoutResult, setCheckoutResult] = useState<CheckoutResult | null>(null);
  const [checkoutError, setCheckoutError] = useState<string | null>(null);
  const checkoutInFlight = useRef(false);
  const cartLocked = isCheckoutLoading || pendingCheckout !== null || storageError !== null;

  const commitCart = (items: CartItem[]) => {
    if (cartTotal(items) === null) { setCheckoutError("Essa quantidade ultrapassa o limite monetário seguro. O pedido foi mantido sem alterações."); return; }
    cartRef.current = items;
    setCart(items);
  };

  const addToCart = (product: Product) => {
    if (checkoutInFlight.current || pendingCheckoutRef.current || storageError) return;
    setCheckoutResult(null);
    setCheckoutError(null);
    const previous = cartRef.current;
    const exists = previous.some(item => item.id === product.id);
    commitCart(exists ? previous.map(item => item.id === product.id ? { ...item, quantity: item.quantity + 1 } : item) : [...previous, { ...product, quantity: 1 }]);
  };

  const removeFromCart = (productId: string) => {
    if (checkoutInFlight.current || pendingCheckoutRef.current || storageError) return;
    setCheckoutError(null);
    commitCart(cartRef.current.flatMap(item => item.id !== productId ? [item] : item.quantity === 1 ? [] : [{ ...item, quantity: item.quantity - 1 }]));
  };

  const discardCart = () => {
    if (checkoutInFlight.current) return;
    if (pendingCheckoutRef.current || storageError) {
      try {
        if (legacyRecovery) {
          if (!isAdmin) return;
          clearLegacyPendingCheckout();
        } else clearPendingCheckout(userId);
      } catch {
        setCheckoutError("Não foi possível limpar o registro local. A tentativa foi mantida.");
        setConfirmClear(false);
        return;
      }
      pendingCheckoutRef.current = null;
      setPendingCheckout(null);
      setStorageError(null);
      setLegacyRecovery(false);
    }
    cartRef.current = [];
    setCart([]);
    setCheckoutResult(null);
    setCheckoutError(null);
    setConfirmClear(false);
  };

  const handleCheckout = async () => {
    if (checkoutInFlight.current || storageError || cartRef.current.length === 0) return;

    let attempt = pendingCheckoutRef.current;
    if (!attempt) {
      try {
        attempt = createPendingCheckout(cartRef.current, userId);
        // Persist before sending: a reload or lost response must reuse this exact sale.
        savePendingCheckout(attempt);
      } catch {
        setCheckoutError("O pedido não foi enviado. Não foi possível salvar a tentativa local; confira o armazenamento do navegador e os valores dos itens.");
        return;
      }
      pendingCheckoutRef.current = attempt;
      setPendingCheckout(attempt);
    }

    checkoutInFlight.current = true;
    setIsCheckoutLoading(true);
    setCheckoutResult(null);
    setCheckoutError(null);

    const orderPayload = {
      items: attempt.items.map((item) => ({
        product_id: item.id,
        quantity: item.quantity,
      })),
    };

    try {
      const response = await apiRequest("/checkout", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": attempt.key,
        },
        body: JSON.stringify(orderPayload),
      });

      if (!response.ok) {
        setCheckoutError(
          response.status === 409
            ? "A tentativa conflita com um registro existente. Confira o pedido no servidor antes de iniciar outra venda. Seu carrinho foi mantido."
            : response.status >= 500
              ? "O servidor não confirmou o pedido. Use Retomar pedido pendente para reenviar a mesma tentativa. Seu carrinho foi mantido."
              : "O pedido não foi aceito. Confira os itens e o registro no servidor antes de iniciar outra venda. Seu carrinho foi mantido.",
        );
        return;
      }

      const result: unknown = await response.json();
      if (!isCheckoutResult(result)) {
        setCheckoutError(
          "A resposta do servidor não pôde ser confirmada. Use Retomar pedido pendente para recuperar o resultado da mesma tentativa. Seu carrinho foi mantido.",
        );
        return;
      }

      setCheckoutResult(result);
      try {
        clearPendingCheckout(userId, attempt.key);
      } catch {
        setCheckoutError("Pedido confirmado, mas não foi possível limpar a tentativa local. Retome o pedido pendente para concluir a recuperação antes de iniciar outra venda.");
        return;
      }
      pendingCheckoutRef.current = null;
      setPendingCheckout(null);
      cartRef.current = [];
      setCart([]);
    } catch (error) {
      console.error(error);
      setCheckoutError(
        "Não foi possível confirmar o pedido. Use Retomar pedido pendente para reenviar com a mesma chave, sem iniciar outra venda. Seu carrinho foi mantido.",
      );
    } finally {
      checkoutInFlight.current = false;
      setIsCheckoutLoading(false);
    }
  };

  const total = cartTotal(cart);
  const filtered = catalog.items.filter(product => searchText(product.name).includes(searchText(search)));
  const hasPending = pendingCheckout !== null || storageError !== null;

  return <>
    <header className="page-header"><div><span className="eyebrow">OPERAÇÃO DO CAIXA</span><h1>Frente de caixa</h1><p>Escolha os produtos e confira o pedido antes de finalizar.</p></div>
      <a href="#pedido-atual" className="button button--secondary mobile-cart-link">Ver pedido · {cart.length}</a>
    </header>
    <div className="pdv-layout">
      <section className="panel" aria-labelledby="catalog-title"><div className="panel__heading"><h2 id="catalog-title">Catálogo</h2><span className="badge">{catalog.items.length} produtos</span></div>
        <div className="catalog-toolbar"><Field label="Buscar produtos" type="search" placeholder="Buscar por nome..." value={search} onChange={event => setSearch(event.target.value)} /></div>
        {catalog.loading ? <CatalogLoading /> : catalog.error ? <div className="panel-error"><Alert>{catalog.error}</Alert><Button variant="secondary" onClick={catalog.refresh}>Tentar novamente</Button></div> :
          catalog.items.length === 0 ? <EmptyState title="Nenhum produto cadastrado.">O administrador pode adicionar produtos na gestão do catálogo.</EmptyState> :
            filtered.length === 0 ? <EmptyState title="Nenhum produto encontrado" action={<Button variant="secondary" onClick={() => setSearch("")}>Limpar busca</Button>}>Tente outro nome para encontrar o produto.</EmptyState> :
              <div className="catalog-grid">{filtered.map(product => <button type="button" key={product.id} className="product-card" disabled={cartLocked}
                aria-label={`${product.name}, ${formatPrice(product.price)}. Adicionar ao pedido.`} onClick={() => addToCart(product)}>
                <span className="product-card__mark" aria-hidden="true">{product.name.trim().split(/\s+/).slice(0, 2).map(word => Array.from(word)[0]).join("").toUpperCase()}</span>
                <span className="product-card__name">{product.name}</span><span className="product-card__bottom"><span className="product-card__price">{formatPrice(product.price)}</span><span className="product-card__add" aria-hidden="true">＋</span></span>
              </button>)}</div>}
      </section>
      <section id="pedido-atual" className="panel cart-panel" tabIndex={-1} aria-labelledby="cart-title" aria-busy={isCheckoutLoading}>
        <div className="panel__heading"><h2 id="cart-title">Pedido atual</h2><span className="badge">{hasPending ? "Tentativa pendente" : `${cart.length} produtos`}</span></div>
        <div className="cart-panel__body">
          {pendingCheckout && <Alert tone="warning">Existe uma tentativa pendente. Retome este pedido para recuperar o resultado; os itens estão bloqueados para preservar a mesma venda.</Alert>}
          {storageError && <Alert>{storageError}</Alert>}
          {checkoutResult && <Alert tone="success">Pedido {checkoutResult.order_id} registrado — aguardando pagamento. Total confirmado: {formatPrice(checkoutResult.total_cents)}.</Alert>}
          {checkoutError && <Alert>{checkoutError}</Alert>}
          {cart.length === 0 ? <EmptyState title="O carrinho está vazio.">Selecione um produto do catálogo para começar o atendimento.</EmptyState> :
            <ul className="cart-list">{cart.map(item => <li key={item.id} className="cart-item"><div>
              <strong className="cart-item__name">{item.name}</strong><span className="cart-item__unit">{formatPrice(item.price)} x {item.quantity}</span>
              <div className="quantity" role="group" aria-label={`Quantidade de ${item.name}`}>
                <Button variant="ghost" disabled={cartLocked} aria-label={`Diminuir quantidade de ${item.name}`} onClick={() => removeFromCart(item.id)}>−</Button>
                <span aria-live="polite">{item.quantity}</span>
                <Button variant="ghost" disabled={cartLocked} aria-label={`Aumentar quantidade de ${item.name}`} onClick={() => addToCart(item)}>＋</Button>
              </div>
            </div><span className="cart-item__price">{formatPrice(item.price * item.quantity)}</span></li>)}</ul>}
        </div>
        <div className="cart-summary"><div className="cart-summary__total"><span>Total</span><strong>{total === null ? "—" : formatPrice(total)}</strong></div>
          <p className="cart-summary__note">O valor final será confirmado pelo servidor. Registrar o pedido não confirma seu pagamento.</p>
          <Button className="button--full" disabled={cart.length === 0 || isCheckoutLoading || storageError !== null || total === null} onClick={() => void handleCheckout()}>
            {isCheckoutLoading ? "Processando..." : pendingCheckout ? "Retomar pedido pendente" : "Finalizar Pedido"}
          </Button>
          <Button className="button--full" variant="ghost" disabled={isCheckoutLoading || (legacyRecovery && !isAdmin) || (cart.length === 0 && !hasPending)}
            onClick={() => { if (!checkoutInFlight.current) setConfirmClear(true); }}>{hasPending ? "Iniciar nova venda" : "Cancelar Venda"}</Button>
        </div>
      </section>
    </div>
    {confirmClear && <Dialog title={hasPending ? "Descartar tentativa pendente?" : "Cancelar venda atual?"} onClose={() => setConfirmClear(false)}>
      <div className="dialog__body"><p>{hasPending ? "O pedido pode já estar registrado. Descartar a tentativa local não cancela o pedido no servidor. Confira o registro antes de iniciar outra venda." : "Os itens deste carrinho serão removidos. Nenhum pedido foi enviado ao servidor."}</p>
        <div className="dialog__actions"><Button variant="secondary" autoFocus data-dialog-initial-focus onClick={() => setConfirmClear(false)}>Manter pedido</Button>
          <Button variant="danger" onClick={discardCart}>{hasPending ? "Descartar tentativa local" : "Cancelar venda"}</Button></div>
      </div>
    </Dialog>}
  </>;
}
