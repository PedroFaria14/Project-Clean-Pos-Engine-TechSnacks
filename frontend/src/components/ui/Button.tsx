import type { ComponentProps } from "react";

type Props = ComponentProps<"button"> & { variant?: "primary" | "secondary" | "danger" | "ghost" };
export function Button({ variant = "primary", className = "", type = "button", ...props }: Props) {
  return <button {...props} type={type} className={`button button--${variant} ${className}`} />;
}
