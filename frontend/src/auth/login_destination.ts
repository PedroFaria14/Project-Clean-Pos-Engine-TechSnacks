// Only application-owned routes can be used as a post-login destination.
export function loginDestination(state: unknown): "/caixa" | "/produtos" {
  return typeof state === "object" && state !== null && "from" in state && state.from === "/produtos" ? "/produtos" : "/caixa";
}
