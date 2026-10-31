export interface StoredCartItem {
  id: string;
  name: string;
  price: number;
  quantity: number;
}

export interface PendingCheckout {
  version: 2;
  user_id: string;
  key: string;
  items: StoredCartItem[];
}

export interface CheckoutStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

export const pendingCheckoutStorageKey = "techsnacks.checkout.pending.v1";

export function pendingCheckoutKey(userId: string): string {
  if (!/^[A-Za-z0-9_-]{1,128}$/.test(userId)) throw new Error("Invalid account ID");
  return `techsnacks.checkout.pending.v2:${userId}`;
}

function isPendingCheckout(value: unknown): value is PendingCheckout {
  if (typeof value !== "object" || value === null) return false;
  const attempt = value as Record<string, unknown>;
  if (
    attempt.version !== 2 || typeof attempt.user_id !== "string" || !/^[A-Za-z0-9_-]{1,128}$/.test(attempt.user_id) || typeof attempt.key !== "string" ||
    !/^[A-Za-z0-9_-]{1,128}$/.test(attempt.key) ||
    !Array.isArray(attempt.items) || attempt.items.length === 0
  ) return false;

  let total = 0;
  for (const value of attempt.items) {
    if (typeof value !== "object" || value === null) return false;
    const item = value as Record<string, unknown>;
    if (
      typeof item.id !== "string" || item.id.length === 0 ||
      typeof item.name !== "string" || item.name.length === 0 ||
      typeof item.price !== "number" || !Number.isSafeInteger(item.price) || item.price < 0 ||
      typeof item.quantity !== "number" || !Number.isSafeInteger(item.quantity) || item.quantity < 1
    ) return false;
    const subtotal = item.price * item.quantity;
    total += subtotal;
    if (!Number.isSafeInteger(subtotal) || !Number.isSafeInteger(total)) return false;
  }
  return true;
}

export function loadPendingCheckout(userId: string, storage?: CheckoutStorage): { pending: PendingCheckout | null; error: string | null; legacy: boolean } {
  try {
    const target = storage ?? window.sessionStorage;
    const raw = target.getItem(pendingCheckoutKey(userId));
    if (raw === null) {
      if (target.getItem(pendingCheckoutStorageKey) !== null) return { pending: null, error: "Existe uma tentativa anterior à autenticação. Um administrador precisa conferir os pedidos no servidor antes de descartar o registro antigo.", legacy: true };
      return { pending: null, error: null, legacy: false };
    }
    const value: unknown = JSON.parse(raw);
    if (!isPendingCheckout(value) || value.user_id !== userId) throw new Error("Invalid pending checkout");
    return { pending: value, error: null, legacy: false };
  } catch {
    return {
      pending: null,
      legacy: false,
      error: "Não foi possível recuperar o registro local. Confira os pedidos no servidor antes de descartar esse registro e iniciar outra venda.",
    };
  }
}

export function createPendingCheckout(items: StoredCartItem[], userId: string): PendingCheckout {
  const attempt: PendingCheckout = { version: 2, user_id: userId, key: crypto.randomUUID(), items: items.map(item => ({ ...item })) };
  if (!isPendingCheckout(attempt)) throw new Error("Invalid checkout items or monetary total");
  return attempt;
}

export function savePendingCheckout(attempt: PendingCheckout, storage?: CheckoutStorage): void {
  if (!isPendingCheckout(attempt)) throw new Error("Invalid pending checkout");
  (storage ?? window.sessionStorage).setItem(pendingCheckoutKey(attempt.user_id), JSON.stringify(attempt));
}

export function clearPendingCheckout(userId: string, expectedKey?: string, storage?: CheckoutStorage): void {
  const target = storage ?? window.sessionStorage;
  const storageKey = pendingCheckoutKey(userId);
  const raw = target.getItem(storageKey);
  if (expectedKey && raw !== null) {
    const attempt: unknown = JSON.parse(raw);
    if (!isPendingCheckout(attempt) || attempt.user_id !== userId || attempt.key !== expectedKey) {
      throw new Error("Pending checkout changed; refusing to discard a different attempt");
    }
  }
  target.removeItem(storageKey);
}

export function clearLegacyPendingCheckout(storage?: CheckoutStorage): void {
  (storage ?? window.sessionStorage).removeItem(pendingCheckoutStorageKey);
}
