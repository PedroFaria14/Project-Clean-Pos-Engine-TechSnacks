import { useRef, useState, type FormEvent } from "react";
import { useSession } from "../auth/session_context";
import { Brand } from "../components/ui/Brand";
import { Button } from "../components/ui/Button";
import { Field } from "../components/ui/Field";
import { Alert } from "../components/ui/Alert";
import { Link } from "react-router-dom";

export function Login() {
  const { login } = useSession();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState("");
  const inFlight = useRef(false);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (inFlight.current) return;
    inFlight.current = true;
    setSending(true);
    setError("");
    try { await login(username, password); }
    catch (failure) { setError(failure instanceof Error ? failure.message : "Não foi possível entrar."); }
    finally { inFlight.current = false; setPassword(""); setSending(false); }
  };

  return (
    <div className="login-page">
      <aside className="login-story" aria-label="TechSnacks"><Link to="/" className="brand-link" aria-label="TechSnacks, sobre o projeto"><Brand /></Link>
        <div className="login-story__content"><span className="login-story__tag">SIMPLES PARA QUEM ATENDE</span><p className="login-story__title">Seu atendimento,<br />no ritmo certo.</p><p>Catálogo organizado, pedidos claros e uma operação mais tranquila. Tudo no mesmo lugar.</p></div>
        <span className="login-story__footer">TechSnacks · Gestão e frente de caixa</span><div className="login-story__ring" aria-hidden="true" />
      </aside>
      <main className="login-main"><div className="login-card"><div className="login-mobile-brand"><Link to="/" className="brand-link" aria-label="TechSnacks, sobre o projeto"><Brand /></Link></div>
        <span className="eyebrow">BEM-VINDO DE VOLTA</span><h1>Entrar no TechSnacks</h1><p className="login-card__intro">Use a conta disponibilizada pelo administrador para acessar sua operação.</p>
        <form onSubmit={submit} className="login-form" aria-busy={sending}>
          <Field id="login-username" label="Usuário" autoComplete="username" value={username} onChange={event => setUsername(event.target.value)} required maxLength={64} disabled={sending} autoFocus placeholder="Seu usuário" />
          <Field id="login-password" label="Senha" type="password" autoComplete="current-password" value={password} onChange={event => setPassword(event.target.value)} required maxLength={128} disabled={sending} placeholder="Sua senha" />
          {error && <Alert>{error}</Alert>}
          <Button className="button--full" disabled={sending} type="submit">{sending ? "Entrando..." : "Entrar"}</Button>
        </form>
        <p className="login-card__footnote">Precisa de acesso? Fale com o administrador.</p>
        <p className="login-card__footnote"><Link to="/">Conhecer o projeto</Link></p>
      </div></main>
    </div>
  );
}
