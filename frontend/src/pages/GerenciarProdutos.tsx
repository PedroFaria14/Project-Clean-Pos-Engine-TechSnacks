import { useState, useEffect } from "react";
import { AddProduct } from "./AddProduct";

interface Product {
  id: string;
  name: string;
  price: number;
}

export function GerenciarProdutos() {
  const [products, setProducts] = useState<Product[]>([]);
  const [showAddForm, setShowAddForm] = useState(false);
  const [loading, setLoading] = useState(true);

  const fetchProducts = async () => {
    try {
      const response = await fetch("http://localhost:8080/products");
      if (response.ok) {
        const data = await response.json();
        setProducts(data || []);
      }
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProducts();
  }, []);

  const formatPrice = (cents: number) => {
    return (cents / 100).toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
  };

  const handleDelete = async (id: string) => {
    const confirmDelete = window.confirm("Tem certeza que deseja excluir este produto?");
    if (!confirmDelete) return;

    try {
      const response = await fetch(`http://localhost:8080/products/${id}`, {
        method: "DELETE",
      });

      if (response.ok) {
        setProducts(products.filter((product) => product.id !== id));
        alert("Produto excluído com sucesso!");
      } else {
        alert("Erro ao excluir o produto no servidor.");
      }
    } catch (error) {
      console.error(error);
      alert("Erro de conexão com o servidor.");
    }
  };

  return (
    <div
      style={{
        maxWidth: "800px",
        margin: "0 auto",
        backgroundColor: "white",
        padding: "20px",
        borderRadius: "8px",
        boxShadow: "0 2px 10px rgba(0,0,0,0.05)",
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          borderBottom: "2px solid #f0f2f5",
          paddingBottom: "15px",
          marginBottom: "20px",
        }}
      >
        <h2 style={{ color: "#333", margin: 0 }}>Estoque e Cardápio</h2>
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          style={{
            padding: "10px 15px",
            backgroundColor: showAddForm ? "#888" : "#ff6b6b",
            color: "white",
            border: "none",
            borderRadius: "6px",
            fontWeight: "bold",
            cursor: "pointer",
          }}
        >
          {showAddForm ? "Voltar para Lista" : "+ Novo Produto"}
        </button>
      </div>

      {showAddForm ? (
        <div style={{ display: "flex", justifyContent: "center", padding: "20px 0" }}>
          <AddProduct />
        </div>
      ) : (
        <>
          {loading ? (
            <p style={{ textAlign: "center" }}>Carregando cardápio...</p>
          ) : products.length === 0 ? (
            <p style={{ textAlign: "center", color: "#666" }}>Nenhum produto cadastrado ainda.</p>
          ) : (
            <table style={{ width: "100%", borderCollapse: "collapse", textAlign: "left" }}>
              <thead>
                <tr style={{ backgroundColor: "#f9f9f9", borderBottom: "2px solid #eee" }}>
                  <th style={{ padding: "12px" }}>Nome do Produto</th>
                  <th style={{ padding: "12px" }}>Preço</th>
                  <th style={{ padding: "12px", textAlign: "center" }}>Ações</th>
                </tr>
              </thead>
              <tbody>
                {products.map((product) => (
                  <tr key={product.id} style={{ borderBottom: "1px solid #eee" }}>
                    <td style={{ padding: "12px", fontWeight: "500" }}>{product.name}</td>
                    <td style={{ padding: "12px", color: "#2ecc71", fontWeight: "bold" }}>
                      {formatPrice(product.price)}
                    </td>
                    <td style={{ padding: "12px", textAlign: "center" }}>
                      <button
                        style={{
                          marginRight: "10px",
                          padding: "6px 12px",
                          backgroundColor: "#3498db",
                          color: "white",
                          border: "none",
                          borderRadius: "4px",
                          cursor: "pointer",
                        }}
                      >
                        Editar
                      </button>
                      <button
                        onClick={() => handleDelete(product.id)}
                        style={{
                          padding: "6px 12px",
                          backgroundColor: "#e74c3c",
                          color: "white",
                          border: "none",
                          borderRadius: "4px",
                          cursor: "pointer",
                        }}
                      >
                        Excluir
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </>
      )}
    </div>
  );
}
