import { apiRequest } from "../api/http.ts";

export interface Product { id: string; name: string; price: number }
export interface ProductValues { name: string; price: number }

export class ProductRequestError extends Error {
  status: number;
  fields: Record<string, string>;
  constructor(message: string, status = 0, fields: Record<string, string> = {}) {
    super(message);
    this.status = status;
    this.fields = fields;
  }
}

export function isProduct(value: unknown): value is Product {
  if (typeof value !== "object" || value === null) return false;
  const p = value as Record<string, unknown>;
  return typeof p.id === "string" && p.id.length > 0 && typeof p.name === "string" && p.name.trim().length > 0 &&
    typeof p.price === "number" && Number.isSafeInteger(p.price) && p.price >= 0;
}

export async function listProducts(signal?: AbortSignal): Promise<Product[]> {
  let response: Response;
  try { response = await apiRequest("/products", { signal }); }
  catch { throw new ProductRequestError("Não foi possível conectar ao servidor. Tente carregar o catálogo novamente."); }
  if (!response.ok) throw new ProductRequestError("Não foi possível carregar o catálogo. Tente novamente.", response.status);
  let value: unknown;
  try { value = await response.json(); } catch { throw new ProductRequestError("O catálogo retornou uma resposta inválida. Tente novamente."); }
  if (!Array.isArray(value) || !value.every(isProduct)) throw new ProductRequestError("O catálogo retornou dados inválidos. Nenhum preço foi usado sem validação.");
  return value;
}

export async function saveProduct(id: string, values: ProductValues, original?: Product): Promise<Product> {
  const payload: Record<string, unknown> = original ? {} : { id };
  if (!original || values.name !== original.name) payload.name = values.name;
  if (!original || values.price !== original.price) payload.price = values.price;
  let response: Response;
  try {
    response = await apiRequest(original ? `/products/${encodeURIComponent(id)}` : "/products", {
      method: original ? "PATCH" : "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload),
    });
  } catch {
    throw new ProductRequestError("Não foi possível confirmar o salvamento. Os campos foram mantidos; confira o catálogo antes de repetir.");
  }
  if (response.status !== (original ? 200 : 201)) {
    const fields: Record<string, string> = {};
    try {
      const problem: unknown = await response.json();
      if (typeof problem === "object" && problem !== null && "details" in problem && Array.isArray(problem.details)) {
        for (const detail of problem.details) {
          if (detail?.field === "name") fields.name = "Informe um nome válido, com até 120 caracteres.";
          if (detail?.field === "price") fields.price = "Informe um preço válido em reais, com até duas casas decimais.";
        }
      }
    } catch { /* Never expose raw dependency responses in the form. */ }
    const message = response.status === 404 ? "Este produto não está mais no catálogo. Atualize a lista." :
      response.status === 403 ? "Sua conta não tem permissão para salvar produtos, ou a sessão precisa ser atualizada." :
        "Não foi possível salvar o produto. Os campos foram mantidos; confira o catálogo antes de repetir.";
    throw new ProductRequestError(message, response.status, fields);
  }
  let value: unknown;
  try { value = await response.json(); } catch { throw new ProductRequestError("O servidor não confirmou o salvamento. Confira o catálogo antes de repetir."); }
  if (!isProduct(value) || value.id !== id || ("name" in payload && value.name !== payload.name) || ("price" in payload && value.price !== payload.price)) throw new ProductRequestError("O servidor não confirmou um produto válido. Confira o catálogo antes de repetir.");
  return value;
}

export async function deleteProduct(id: string): Promise<void> {
  let response: Response;
  try { response = await apiRequest(`/products/${encodeURIComponent(id)}`, { method: "DELETE" }); }
  catch { throw new ProductRequestError("Não foi possível confirmar a exclusão. Confira o catálogo antes de repetir."); }
  if (response.status !== 204) throw new ProductRequestError("Não foi possível confirmar a exclusão. Confira o catálogo e tente novamente.", response.status);
}
