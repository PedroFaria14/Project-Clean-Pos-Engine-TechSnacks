import { useId, useRef, useState, type FormEvent } from "react";
import { ProductRequestError, saveProduct, type Product } from "../products/product_api";
import { parsePrice, priceInput, validateProduct } from "../products/money";
import { Alert } from "./ui/Alert";
import { Button } from "./ui/Button";
import { Field } from "./ui/Field";

export interface ProductFormProps {
  product?: Product;
  onSaved(product: Product): void;
  onCancel(): void;
  onBusyChange(busy: boolean): void;
}

export function ProductForm({ product, onSaved, onCancel, onBusyChange }: ProductFormProps) {
  const formId = useId();
  const [id] = useState(() => product?.id ?? crypto.randomUUID());
  const [name, setName] = useState(product?.name ?? "");
  const [price, setPrice] = useState(product ? priceInput(product.price) : "");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const inFlight = useRef(false);
  const unchanged = !!product && name.trim() === product.name && parsePrice(price) === product.price;

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (inFlight.current || unchanged) return;
    const result = validateProduct(name, price);
    setErrors(result.errors);
    setError("");
    if (!result.values) {
      document.getElementById(`${formId}-${result.errors.name ? "name" : "price"}`)?.focus();
      return;
    }
    inFlight.current = true;
    setSaving(true);
    onBusyChange(true);
    try { onSaved(await saveProduct(id, result.values, product)); }
    catch (failure) {
      setError(failure instanceof Error ? failure.message : "Não foi possível salvar o produto.");
      if (failure instanceof ProductRequestError) setErrors(failure.fields);
    } finally { inFlight.current = false; setSaving(false); onBusyChange(false); }
  };

  return <form className="product-form" onSubmit={submit} noValidate aria-busy={saving}>
    {error && <Alert>{error}</Alert>}
    <Field id={`${formId}-name`} label="Nome do produto" value={name} onChange={event => setName(event.target.value)}
      error={errors.name} required maxLength={120} autoFocus data-dialog-initial-focus disabled={saving} placeholder="Ex.: Coxinha de frango" autoComplete="off" />
    <Field id={`${formId}-price`} label="Preço (R$)" value={price} onChange={event => setPrice(event.target.value)}
      error={errors.price} required inputMode="decimal" maxLength={18} disabled={saving} placeholder="Ex.: 6,50" autoComplete="off" hint="Use vírgula ou ponto, sem separador de milhares. Zero é permitido." />
    <div className="dialog__actions">
      <Button variant="secondary" disabled={saving} onClick={onCancel}>Cancelar</Button>
      <Button type="submit" disabled={saving || unchanged}>{saving ? "Salvando..." : product ? "Salvar alterações" : "Cadastrar produto"}</Button>
    </div>
  </form>;
}
