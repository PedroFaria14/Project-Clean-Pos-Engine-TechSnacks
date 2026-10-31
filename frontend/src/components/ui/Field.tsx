import { useId, type ComponentProps } from "react";

type Props = ComponentProps<"input"> & { label: string; error?: string; hint?: string };
export function Field({ label, error, hint, id, className = "", ...props }: Props) {
  const generated = useId();
  const inputId = id ?? generated;
  const describedBy = [hint ? `${inputId}-hint` : "", error ? `${inputId}-error` : ""].filter(Boolean).join(" ") || undefined;
  return <div className="field">
    <div className="field__label"><label htmlFor={inputId}>{label}</label>{props.required && <span className="field__required" aria-hidden="true">(obrigatório)</span>}</div>
    <input {...props} id={inputId} className={`input ${className}`} aria-invalid={!!error} aria-describedby={describedBy} />
    {hint && <small id={`${inputId}-hint`} className="field__hint">{hint}</small>}
    {error && <small id={`${inputId}-error`} className="field__error">{error}</small>}
  </div>;
}
