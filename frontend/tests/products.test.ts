import assert from "node:assert/strict";
import { test } from "node:test";
import { cartTotal, formatPrice, parsePrice, priceInput, searchText, validateProduct } from "../src/products/money.ts";
import { isProduct, listProducts, ProductRequestError, saveProduct } from "../src/products/product_api.ts";

test("decimal prices convert to cents without floating point currency multiplication", () => {
  for (const [input, expected] of [["0", 0], ["0,01", 1], ["6,5", 650], ["6.50", 650], [" 19,99 ", 1999], ["90071992547409,91", Number.MAX_SAFE_INTEGER]] as const) assert.equal(parsePrice(input), expected);
  for (const input of ["", "-1", "1e3", "1,234", "1.234,56", "Infinity", "NaN", "90071992547409,92"]) assert.equal(parsePrice(input), null);
  assert.equal(priceInput(Number.MAX_SAFE_INTEGER), "90071992547409,91");
  assert.ok(formatPrice(Number.MAX_SAFE_INTEGER).endsWith("409,91"));
  assert.equal(parsePrice(priceInput(1999)), 1999);
});

test("product validation and search handle empty values, accents and trimmed names", () => {
  assert.deepEqual(validateProduct("  Pão de queijo  ", "3,50").values, { name: "Pão de queijo", price: 350 });
  assert.ok(validateProduct("  ", "3,50").errors.name);
  assert.ok(validateProduct("Produto", "x").errors.price);
  assert.equal(searchText(" PÃO de QUEIJO "), "pao de queijo");
  assert.equal(isProduct({ id: "p", name: "Produto", price: 0 }), true);
  for (const value of [null, { id: "p", name: " ", price: 1 }, { id: "p", name: "Produto", price: "650" }, { id: "p", name: "Produto", price: Number.MAX_SAFE_INTEGER + 1 }]) assert.equal(isProduct(value), false);
});

test("cart totals reject unsafe arithmetic and invalid quantities", () => {
  assert.equal(cartTotal([]), 0);
  assert.equal(cartTotal([{ price: 650, quantity: 2 }, { price: 350, quantity: 1 }]), 1650);
  assert.equal(cartTotal([{ price: Number.MAX_SAFE_INTEGER, quantity: 2 }]), null);
  assert.equal(cartTotal([{ price: 0, quantity: Number.MAX_SAFE_INTEGER + 1 }]), null);
  assert.equal(cartTotal([{ price: 650, quantity: 0 }]), null);
});

test("catalog rejects invalid responses rather than silently using bad data", async context => {
  context.mock.method(globalThis, "fetch", async () => new Response("{", { status: 200 }));
  await assert.rejects(listProducts(), ProductRequestError);
});

test("editing sends only changed fields and preserves a zero price", async context => {
  const requests: RequestInit[] = [];
  context.mock.method(globalThis, "fetch", async (_url, options: RequestInit) => {
    requests.push(options);
    const body = JSON.parse(String(options.body));
    return Response.json({ id: "p", name: "New name", price: body.price ?? 650 });
  });
  await saveProduct("p", { name: "New name", price: 650 }, { id: "p", name: "Old name", price: 650 });
  assert.equal(requests[0].method, "PATCH");
  assert.deepEqual(JSON.parse(String(requests[0].body)), { name: "New name" });
  await saveProduct("p", { name: "New name", price: 0 }, { id: "p", name: "New name", price: 650 });
  assert.deepEqual(JSON.parse(String(requests[1].body)), { price: 0 });
});
