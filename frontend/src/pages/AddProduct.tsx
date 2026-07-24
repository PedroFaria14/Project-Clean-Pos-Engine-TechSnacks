import { useState, type FormEvent } from "react";

export function AddProduct() {
  const [name, setName] = useState<string>("");
  const [price, setPrice] = useState<string>("");
  const [message, setMessage] = useState<string>("");

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    const priceInCents = Math.round(parseFloat(price) * 100);

    const id = crypto.randomUUID();

    const payload = {
      id: id,
      name: name,
      price: priceInCents,
    };

    try {
      const response = await fetch("http://localhost:8080/products", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (response.ok) {
        setMessage("Produto adicionado ao cardápio!");
        setName("");
        setPrice("");
      } else {
        setMessage("Erro ao cadastrar produto.");
      }
    } catch (error) {
      console.error(error);
      setMessage("Erro de conexão com o servidor.");
    }
  };

  return (
    <div style={{ padding: "20px", maxWidth: "400px", backgroundColor: "#f9f9f9", borderRadius: "8px" }}>
      <h2 style={{ textAlign: "center", color: "#ff6b6b" }}>Novo Produto</h2>

      <form
        onSubmit={handleSubmit}
        style={{ display: "flex", flexDirection: "column", gap: "15px", marginTop: "20px" }}
      >
        <div>
          <label style={{ display: "block", marginBottom: "5px", fontWeight: "bold" }}>Nome do Item</label>
          <input
            type="text"
            placeholder="Ex: Coxinha de Frango"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            style={{ width: "100%", padding: "10px", borderRadius: "4px", border: "1px solid #ccc" }}
          />
        </div>

        <div>
          <label style={{ display: "block", marginBottom: "5px", fontWeight: "bold" }}>Preço (R$)</label>
          <input
            type="number"
            step="0.01"
            placeholder="Ex: 6.50"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            required
            style={{ width: "100%", padding: "10px", borderRadius: "4px", border: "1px solid #ccc" }}
          />
        </div>

        <button
          type="submit"
          style={{
            padding: "12px",
            backgroundColor: "#4CAF50",
            color: "white",
            border: "none",
            borderRadius: "4px",
            fontWeight: "bold",
            cursor: "pointer",
          }}
        >
          Enviar
        </button>
      </form>

      {message && <p style={{ marginTop: "15px", textAlign: "center", fontWeight: "bold" }}>{message}</p>}
    </div>
  );
}
