import { useEffect, useState, type ReactNode } from "react";
import { apiRequest, setCsrfToken } from "../api/http";
import { isSession, SessionContext, type Session } from "./session_context";

async function readSession(signal?: AbortSignal): Promise<Session | null> {
  const response = await apiRequest("/sessions/current", { signal });
  if (response.status === 401) return null;
  const value: unknown = await response.json();
  if (!response.ok || !isSession(value)) throw new Error("Session unavailable");
  return value;
}

export function SessionProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);
  const [loading, setLoading] = useState(true);
  const [unavailable, setUnavailable] = useState(false);

  const refresh = async () => {
    setLoading(true);
    try {
      const value = await readSession();
      setCsrfToken(value?.csrf_token ?? "");
      setSession(value);
      setUnavailable(false);
    } catch {
      setCsrfToken("");
      setSession(null);
      setUnavailable(true);
    } finally { setLoading(false); }
  };

  useEffect(() => {
    const expired = () => { setCsrfToken(""); setSession(null); setUnavailable(false); };
    window.addEventListener("techsnacks:session-expired", expired);
    const controller = new AbortController();
    void readSession(controller.signal).then(value => {
      if (controller.signal.aborted) return;
      setCsrfToken(value?.csrf_token ?? "");
      setSession(value);
      setUnavailable(false);
    }).catch(() => {
      if (controller.signal.aborted) return;
      setCsrfToken("");
      setSession(null);
      setUnavailable(true);
    }).finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => { controller.abort(); window.removeEventListener("techsnacks:session-expired", expired); };
  }, []);

  const login = async (username: string, password: string) => {
    const response = await apiRequest("/sessions", {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ username, password }),
    });
    if (!response.ok) {
      if (response.status === 429) throw new Error("Limite de tentativas atingido. Aguarde 15 minutos antes de tentar novamente.");
      if (response.status === 401 || response.status === 400) throw new Error("Usuário ou senha inválidos.");
      throw new Error("Não foi possível entrar. Verifique a conexão e a disponibilidade do serviço.");
    }
    const value: unknown = await response.json();
    if (!isSession(value)) throw new Error("Não foi possível validar a sessão recebida.");
    setCsrfToken(value.csrf_token);
    setSession(value);
    setUnavailable(false);
  };

  const logout = async () => {
    const response = await apiRequest("/sessions/current", { method: "DELETE" });
    if (response.status !== 204 && response.status !== 401) throw new Error("Não foi possível encerrar a sessão. Tente novamente antes de deixar o caixa.");
    setCsrfToken("");
    setSession(null);
  };

  return <SessionContext.Provider value={{ session, loading, unavailable, login, logout, refresh }}>{children}</SessionContext.Provider>;
}
