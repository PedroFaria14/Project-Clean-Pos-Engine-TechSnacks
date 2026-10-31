import { useEffect, useId, useRef, type ReactNode } from "react";
import { Button } from "./Button";

export function Dialog({ title, children, onClose, busy = false }: { title: string; children: ReactNode; onClose(): void; busy?: boolean }) {
  const ref = useRef<HTMLDialogElement>(null);
  const titleId = useId();
  useEffect(() => {
    const dialog = ref.current;
    const previous = document.activeElement;
    dialog?.showModal();
    dialog?.querySelector<HTMLElement>("[data-dialog-initial-focus], [autofocus]")?.focus({ preventScroll: true });
    return () => {
      dialog?.close();
      const target = previous instanceof HTMLElement && previous.isConnected ? previous : document.getElementById("main-content");
      target?.focus();
    };
  }, []);
  useEffect(() => { if (busy) ref.current?.focus({ preventScroll: true }); }, [busy]);
  return <dialog ref={ref} className="dialog" tabIndex={-1} aria-modal="true" aria-labelledby={titleId} aria-busy={busy}
    onKeyDown={event => {
      if (event.key !== "Tab") return;
      const elements = Array.from(event.currentTarget.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), a[href], [tabindex]:not([tabindex="-1"])')).filter(element => element.getClientRects().length > 0);
      const first = elements[0];
      const last = elements.at(-1);
      if (!first) { event.preventDefault(); event.currentTarget.focus(); return; }
      if (event.shiftKey && (document.activeElement === first || document.activeElement === event.currentTarget)) { event.preventDefault(); last?.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    }}
    onCancel={event => { event.preventDefault(); if (!busy) onClose(); }}>
    <div className="dialog__header"><h2 id={titleId}>{title}</h2><Button variant="ghost" className="button--icon" aria-label="Fechar diálogo" disabled={busy} onClick={onClose}>×</Button></div>
    {children}
  </dialog>;
}
