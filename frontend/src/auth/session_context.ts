import { createContext, useContext } from "react";

export interface Session {
  user: { id: string; username: string; role: "operator" | "admin" };
  csrf_token: string;
  expires_at: string;
}

export interface SessionState {
  session: Session | null;
  loading: boolean;
  unavailable: boolean;
  login(username: string, password: string): Promise<void>;
  logout(): Promise<void>;
  refresh(): Promise<void>;
}

export const SessionContext = createContext<SessionState | null>(null);

export function useSession(): SessionState {
  const value = useContext(SessionContext);
  if (!value) throw new Error("SessionProvider is required");
  return value;
}

export function isSession(value: unknown): value is Session {
  if (typeof value !== "object" || value === null) return false;
  const result = value as Record<string, unknown>;
  if (typeof result.user !== "object" || result.user === null) return false;
  const user = result.user as Record<string, unknown>;
  return typeof user.id === "string" && /^[A-Za-z0-9_-]{1,128}$/.test(user.id) &&
    typeof user.username === "string" && user.username.length > 0 &&
    (user.role === "admin" || user.role === "operator") &&
    typeof result.csrf_token === "string" && /^[A-Za-z0-9_-]{43}$/.test(result.csrf_token) &&
    typeof result.expires_at === "string" && Number.isFinite(Date.parse(result.expires_at));
}
