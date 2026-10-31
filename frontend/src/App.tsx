import { useEffect, useState } from "react";
import { BrowserRouter, Routes, Route, NavLink, Navigate, Link, Outlet, useLocation } from "react-router-dom";
import { SessionProvider } from "./auth/SessionProvider";
import { useSession } from "./auth/session_context";
import { Login } from "./pages/Login";
import { GerenciarProdutos } from "./pages/GerenciarProdutos";
import { CaixaPDV } from "./components/CaixaPDV";
import { Button } from "./components/ui/Button";
import { Brand } from "./components/ui/Brand";
import { Alert } from "./components/ui/Alert";
import { Home } from "./pages/Home";
import { loginDestination } from "./auth/login_destination";

export default function App() {
  return <SessionProvider><BrowserRouter><RouteMetadata /><Routes>
    <Route path="/" element={<Home />} />
    <Route path="/login" element={<LoginEntry />} />
    <Route element={<AuthenticatedApp />}>
      <Route path="/caixa" element={<CashRegisterEntry />} />
      <Route path="/produtos" element={<ProductsEntry />} />
    </Route>
    <Route path="*" element={<Navigate to="/" replace />} />
  </Routes></BrowserRouter></SessionProvider>;
}

function RouteMetadata() {
  const { pathname } = useLocation();
  useEffect(() => {
    const titles: Record<string, string> = { "/": "TechSnacks | Sobre o projeto", "/login": "Entrar | TechSnacks", "/caixa": "Caixa | TechSnacks", "/produtos": "Produtos | TechSnacks" };
    document.title = titles[pathname] ?? "TechSnacks";
    window.scrollTo(0, 0);
  }, [pathname]);
  return null;
}

function SessionStatus() {
  const { loading, refresh } = useSession();
  return <main className="session-screen"><div className="panel session-screen__card"><Brand />
    {loading ? <p role="status">Verificando sessão...</p> : <><h1>Serviço indisponível</h1><Alert>Não foi possível verificar sua sessão. Nenhuma operação foi iniciada.</Alert><Button onClick={() => void refresh()}>Tentar novamente</Button></>}
    <Link to="/" className="button button--secondary">Sobre o projeto</Link>
  </div></main>;
}

function LoginEntry() {
  const { session, loading, unavailable } = useSession();
  const location = useLocation();
  if (loading || unavailable) return <SessionStatus />;
  if (session) return <Navigate to={loginDestination(location.state)} replace />;
  return <Login />;
}

function CashRegisterEntry() {
  const { session } = useSession();
  return <CaixaPDV key={session?.user.id} />;
}

function ProductsEntry() {
  const { session } = useSession();
  return session?.user.role === "admin" ? <GerenciarProdutos /> : <><h1>Acesso restrito</h1><Alert>Você não tem permissão para gerenciar produtos.</Alert></>;
}

function AuthenticatedApp() {
  const { session, loading, unavailable, logout } = useSession();
  const location = useLocation();
  const [logoutError, setLogoutError] = useState("");
  const [leaving, setLeaving] = useState(false);
  if (loading || unavailable) return <SessionStatus />;
  if (!session) return <Navigate to="/login" replace state={{ from: location.pathname }} />;

  const leave = async () => {
    setLeaving(true);
    setLogoutError("");
    try { await logout(); } catch (error) { setLogoutError(error instanceof Error ? error.message : "Não foi possível sair."); }
    finally { setLeaving(false); }
  };
  return (
      <>
        <a className="skip-link" href="#main-content">Pular para o conteúdo</a>
        <header className="app-header"><div className="app-header__inner">
          <Link to="/" className="brand-link" aria-label="TechSnacks, sobre o projeto"><Brand /></Link>
          <nav className="app-nav" aria-label="Navegação principal"><NavLink end to="/caixa">Caixa (PDV)</NavLink>
            {session.user.role === "admin" && <NavLink to="/produtos">Gerenciar Produtos</NavLink>}</nav>
          <div className="account"><span className="account__avatar" aria-hidden="true">{session.user.username.slice(0, 2).toUpperCase()}</span>
            <div className="account__info"><strong title={session.user.username}>{session.user.username}</strong><small>{session.user.role === "admin" ? "Administrador" : "Operador"}</small></div>
            <Button variant="ghost" disabled={leaving} onClick={() => void leave()}>{leaving ? "Saindo..." : "Sair"}</Button>
          </div>
        </div></header>
        <main id="main-content" className="app-main" tabIndex={-1}>
          {logoutError && <div className="notice"><Alert>{logoutError}</Alert></div>}
          <Outlet />
        </main>
      </>
  );
}
