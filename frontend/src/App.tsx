import { BrowserRouter, Routes, Route, Link } from "react-router-dom";
import { GerenciarProdutos } from "./pages/GerenciarProdutos";
import { CaixaPDV } from "./components/CaixaPDV";

export default function App() {
  return (
    <BrowserRouter>
      <div style={{ minHeight: "100vh", backgroundColor: "#f0f2f5", fontFamily: "sans-serif" }}>
        <nav
          style={{
            backgroundColor: "#ff6b6b",
            padding: "15px 30px",
            display: "flex",
            gap: "30px",
            boxShadow: "0 2px 4px rgba(0,0,0,0.1)",
          }}
        >
          <Link to="/" style={{ color: "white", textDecoration: "none", fontWeight: "bold", fontSize: "18px" }}>
            Caixa (PDV)
          </Link>
          <Link to="/produtos" style={{ color: "white", textDecoration: "none", fontWeight: "bold", fontSize: "18px" }}>
            Gerenciar Produtos
          </Link>
        </nav>

        <main style={{ padding: "30px" }}>
          <Routes>
            <Route path="/" element={<CaixaPDV />} />

            <Route path="/produtos" element={<GerenciarProdutos />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}
