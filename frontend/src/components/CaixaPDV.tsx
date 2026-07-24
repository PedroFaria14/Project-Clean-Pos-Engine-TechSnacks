import { useEffect } from "react";
import { useState } from "react";

interface Product {
  id: string;
  name: string;
  price: number;
}

interface CartItem extends Product {
  quantity: number;
}

export function CaixaPDV() {
  const [products, setProducts] = useState<Product[]>([]);
  const [cart, setCart] = useState<CartItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCheckoutLoading, setIsCheckoutLoading] = useState(false);

  useEffect(() => {
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
    fetchProducts();
  }, []);

  const formatPrice = (cents: number) => {
    return (cents / 100).toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
  };

  const addToCart = (product: Product) => {
    setCart((prevCart) => {
      const existingItem = prevCart.find((item) => item.id === product.id);
      if (existingItem) {
        return prevCart.map((item) => (item.id === product.id ? { ...item, quantity: item.quantity + 1 } : item));
      }
      return [...prevCart, { ...product, quantity: 1 }];
    });
  };

  const removeFromCart = (productId: string) => {
    setCart((prevCart) => {
      const existingItem = prevCart.find((item) => item.id === productId);
      if (existingItem?.quantity === 1) {
        return prevCart.filter((item) => item.id !== productId);
      }
      return prevCart.map((item) => (item.id === productId ? { ...item, quantity: item.quantity - 1 } : item));
    });
  };

  const clearCart = () => setCart([]);

  const handleCheckout = async () => {
    setIsCheckoutLoading(true);

    const orderPayload = {
      items: cart.map((item) => ({
        product_id: item.id,
        quantity: item.quantity,
      })),
    };

    try {
      const response = await fetch("http://localhost:8080/checkout", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(orderPayload),
      });

      if (response.ok) {
        alert("Venda registrada com sucesso!");
        clearCart();
      } else {
        alert("Erro ao registrar a venda no servidor.");
      }
    } catch (error) {
      console.error(error);
      alert("Erro de conexão com o servidor.");
    } finally {
      setIsCheckoutLoading(false);
    }
  };

  const cartTotalCents = cart.reduce((total, item) => total + item.price * item.quantity, 0);

  return (
    <div style={{ maxWidth: "1200px", margin: "0 auto", display: "flex", gap: "20px" }}>
      <div
        style={{
          flex: "2",
          backgroundColor: "white",
          padding: "20px",
          borderRadius: "8px",
          boxShadow: "0 2px 10px rgba(0,0,0,0.05)",
        }}
      >
        <h2 style={{ color: "#ff6b6b", borderBottom: "2px solid #f0f2f5", paddingBottom: "10px" }}>
          Estoque Disponível
        </h2>

        {loading ? (
          <p style={{ textAlign: "center", color: "#888", marginTop: "20px" }}>Procurando produtos...</p>
        ) : products.length === 0 ? (
          <p style={{ textAlign: "center", color: "#888", marginTop: "20px" }}>Nenhum produto cadastrado.</p>
        ) : (
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))",
              gap: "15px",
              marginTop: "20px",
            }}
          >
            {products.map((product) => (
              <div
                key={product.id}
                onClick={() => addToCart(product)}
                style={{
                  border: "1px solid #eee",
                  borderRadius: "8px",
                  padding: "15px",
                  cursor: "pointer",
                  transition: "transform 0.1s, boxShadow 0.1s",
                  textAlign: "center",
                  backgroundColor: "#fafafa",
                }}
                onMouseOver={(e) => (e.currentTarget.style.borderColor = "#ff6b6b")}
                onMouseOut={(e) => (e.currentTarget.style.borderColor = "#eee")}
              >
                <h3 style={{ margin: "0 0 10px 0", fontSize: "16px", color: "#333" }}>{product.name}</h3>
                <span style={{ fontWeight: "bold", color: "#2ecc71", fontSize: "18px" }}>
                  {formatPrice(product.price)}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div
        style={{
          flex: "1",
          backgroundColor: "white",
          padding: "20px",
          borderRadius: "8px",
          boxShadow: "0 2px 10px rgba(0,0,0,0.05)",
          display: "flex",
          flexDirection: "column",
        }}
      >
        <h2 style={{ color: "#333", borderBottom: "2px solid #f0f2f5", paddingBottom: "10px" }}>Pedido Atual</h2>

        <div style={{ flexGrow: 1, overflowY: "auto", marginTop: "10px" }}>
          {cart.length === 0 ? (
            <p style={{ textAlign: "center", color: "#888", fontStyle: "italic", marginTop: "40px" }}>
              O carrinho está vazio.
            </p>
          ) : (
            <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
              {cart.map((item) => (
                <li
                  key={item.id}
                  style={{
                    display: "flex",
                    justifyContent: "space-between",
                    alignItems: "center",
                    padding: "10px 0",
                    borderBottom: "1px dashed #eee",
                  }}
                >
                  <div style={{ flex: 1 }}>
                    <div style={{ fontWeight: "bold", color: "#333" }}>{item.name}</div>
                    <div style={{ fontSize: "13px", color: "#888" }}>
                      {formatPrice(item.price)} x {item.quantity}
                    </div>
                  </div>

                  <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                    <span style={{ fontWeight: "bold" }}>{formatPrice(item.price * item.quantity)}</span>
                    <button
                      onClick={() => removeFromCart(item.id)}
                      style={{
                        background: "#e74c3c",
                        color: "white",
                        border: "none",
                        borderRadius: "4px",
                        width: "25px",
                        height: "25px",
                        cursor: "pointer",
                        fontWeight: "bold",
                      }}
                    >
                      -
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div style={{ marginTop: "20px", paddingTop: "20px", borderTop: "2px solid #f0f2f5" }}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              fontSize: "20px",
              fontWeight: "bold",
              marginBottom: "20px",
            }}
          >
            <span>Total:</span>
            <span style={{ color: "#2ecc71" }}>{formatPrice(cartTotalCents)}</span>
          </div>

          <button
            onClick={handleCheckout}
            disabled={cart.length === 0 || isCheckoutLoading}
            style={{
              width: "100%",
              padding: "15px",
              backgroundColor: cart.length === 0 ? "#ccc" : "#4CAF50",
              color: "white",
              border: "none",
              borderRadius: "6px",
              fontSize: "18px",
              fontWeight: "bold",
              cursor: cart.length === 0 || isCheckoutLoading ? "not-allowed" : "pointer",
              marginBottom: "10px",
            }}
          >
            {isCheckoutLoading ? "Processando..." : "Finalizar Pedido"}
          </button>

          <button
            onClick={clearCart}
            disabled={cart.length === 0}
            style={{
              width: "100%",
              padding: "10px",
              backgroundColor: "transparent",
              color: "#e74c3c",
              border: "1px solid #e74c3c",
              borderRadius: "6px",
              cursor: cart.length === 0 ? "not-allowed" : "pointer",
            }}
          >
            Cancelar Venda
          </button>
        </div>
      </div>
    </div>
  );
}
