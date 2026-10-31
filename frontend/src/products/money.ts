import type { ProductValues } from "./product_api.ts";

const brl = new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL" });
export function formatPrice(cents: number): string {
  if (!Number.isSafeInteger(cents) || cents < 0) return "—";
  const value = BigInt(cents);
  const fraction = String(value % 100n).padStart(2, "0");
  return brl.formatToParts(value / 100n).map(part => part.type === "fraction" ? fraction : part.value).join("");
}
export function priceInput(cents: number): string { const value = BigInt(cents); return `${value / 100n},${String(value % 100n).padStart(2, "0")}`; }

// Decimal strings are converted without multiplying floating point currency values.
export function parsePrice(value: string): number | null {
  const normalized = value.trim();
  if (!/^\d{1,14}([,.]\d{1,2})?$/.test(normalized)) return null;
  const [units, fraction = ""] = normalized.split(/[,.]/);
  const cents = BigInt(units) * 100n + BigInt(fraction.padEnd(2, "0"));
  return cents <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(cents) : null;
}

export function validateProduct(name: string, price: string): { values: ProductValues | null; errors: Record<string, string> } {
  const errors: Record<string, string> = {};
  const cleanName = name.trim();
  const cents = parsePrice(price);
  if (!cleanName || Array.from(cleanName).length > 120) errors.name = "Informe um nome com 1 a 120 caracteres.";
  if (cents === null) errors.price = "Informe um preço válido, como 6,50, com até duas casas decimais.";
  return { values: Object.keys(errors).length || cents === null ? null : { name: cleanName, price: cents }, errors };
}

export function cartTotal(items: ReadonlyArray<{ price: number; quantity: number }>): number | null {
  let total = 0;
  for (const item of items) {
    if (!Number.isSafeInteger(item.price) || item.price < 0 || !Number.isSafeInteger(item.quantity) || item.quantity < 1) return null;
    const subtotal = item.price * item.quantity;
    total += subtotal;
    if (!Number.isSafeInteger(subtotal) || !Number.isSafeInteger(total)) return null;
  }
  return total;
}

export function searchText(value: string): string { return value.normalize("NFD").replace(/\p{Diacritic}/gu, "").toLocaleLowerCase("pt-BR").trim(); }
