import type { ReactNode } from "react";

export function Alert({ tone = "error", children, id }: { tone?: "error" | "success" | "warning" | "info"; children: ReactNode; id?: string }) {
  return <div id={id} className={`alert alert--${tone}`} role={tone === "error" ? "alert" : tone === "success" ? "status" : undefined}>
    <span className="alert__mark" aria-hidden="true">{tone === "success" ? "✓" : tone === "error" ? "!" : "i"}</span>
    <div>{children}</div>
  </div>;
}
