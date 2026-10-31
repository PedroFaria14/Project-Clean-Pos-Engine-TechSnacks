import assert from "node:assert/strict";
import { test } from "node:test";
import {
  clearPendingCheckout, createPendingCheckout, loadPendingCheckout,
  pendingCheckoutStorageKey, pendingCheckoutKey, clearLegacyPendingCheckout, savePendingCheckout, type CheckoutStorage,
} from "../src/checkout/pendingCheckout.ts";

class MemoryStorage implements CheckoutStorage {
  values = new Map<string, string>();
  getItem(key: string) { return this.values.get(key) ?? null; }
  setItem(key: string, value: string) { this.values.set(key, value); }
  removeItem(key: string) { this.values.delete(key); }
}

const items = () => [{ id: "product-1", name: "Coxinha", price: 650, quantity: 2 }];
const user = "operator-one";

test("pending checkout survives recovery with exactly the original key and items", () => {
  const storage = new MemoryStorage();
  assert.deepEqual(loadPendingCheckout(user, storage), { pending: null, error: null, legacy: false });
  const cart = items();
  const attempt = createPendingCheckout(cart, user);
  cart[0].quantity = 99;
  savePendingCheckout(attempt, storage);
  const recovered = loadPendingCheckout(user, storage);
  assert.deepEqual(recovered, { pending: attempt, error: null, legacy: false });
  assert.equal(recovered.pending?.items[0].quantity, 2);
  assert.notEqual(createPendingCheckout(items(), user).key, attempt.key);
  clearPendingCheckout(user, attempt.key, storage);
  assert.equal(loadPendingCheckout(user, storage).pending, null);
});

test("corrupt or incompatible storage is preserved and blocks silent recovery", () => {
  const storage = new MemoryStorage();
  const valid = createPendingCheckout(items(), user);
  for (const raw of [
    "{", "null", "[]",
    JSON.stringify({ ...valid, version: 3 }),
    JSON.stringify({ ...valid, user_id: "another-user" }),
    JSON.stringify({ ...valid, key: "bad key" }),
    JSON.stringify({ ...valid, items: [] }),
    JSON.stringify({ ...valid, items: [{ ...items()[0], quantity: 0 }] }),
    JSON.stringify({ ...valid, items: [{ ...items()[0], price: Number.MAX_SAFE_INTEGER }] }),
  ]) {
    storage.setItem(pendingCheckoutKey(user), raw);
    const recovered = loadPendingCheckout(user, storage);
    assert.equal(recovered.pending, null);
    assert.ok(recovered.error);
    assert.equal(storage.getItem(pendingCheckoutKey(user)), raw);
  }
});

test("cleanup cannot discard a different pending sale", () => {
  const storage = new MemoryStorage();
  const attempt = createPendingCheckout(items(), user);
  savePendingCheckout(attempt, storage);
  assert.throws(() => clearPendingCheckout(user, "another-sale", storage));
  assert.equal(loadPendingCheckout(user, storage).pending?.key, attempt.key);
  clearPendingCheckout(user, undefined, storage); // Explicit operator discard.
  assert.equal(loadPendingCheckout(user, storage).pending, null);
});

test("blocked storage never pretends to save or discard a sale", () => {
  const blocked: CheckoutStorage = {
    getItem() { throw new Error("blocked"); },
    setItem() { throw new Error("blocked"); },
    removeItem() { throw new Error("blocked"); },
  };
  assert.ok(loadPendingCheckout(user, blocked).error);
  assert.throws(() => savePendingCheckout(createPendingCheckout(items(), user), blocked));
  assert.throws(() => clearPendingCheckout(user, undefined, blocked));
});

test("invalid amounts and quantities are rejected before a request is created", () => {
  for (const item of [
    { ...items()[0], price: -1 },
    { ...items()[0], quantity: 0 },
    { ...items()[0], quantity: 1.5 },
    { ...items()[0], price: Number.MAX_SAFE_INTEGER },
  ]) assert.throws(() => createPendingCheckout([item], user));
});

test("accounts never recover or overwrite another account's pending attempt", () => {
  const storage = new MemoryStorage();
  const first = createPendingCheckout(items(), user);
  savePendingCheckout(first, storage);
  assert.equal(loadPendingCheckout("operator-two", storage).pending, null);
  const second = createPendingCheckout(items(), "operator-two");
  savePendingCheckout(second, storage);
  clearPendingCheckout("operator-two", second.key, storage);
  assert.equal(loadPendingCheckout(user, storage).pending?.key, first.key);
});

test("legacy anonymous attempts are not silently attached to a logged-in account", () => {
  const storage = new MemoryStorage();
  storage.setItem(pendingCheckoutStorageKey, JSON.stringify({ version: 1, key: "legacy", items: items() }));
  const recovered = loadPendingCheckout(user, storage);
  assert.equal(recovered.pending, null);
  assert.equal(recovered.legacy, true);
  assert.ok(recovered.error);
  clearLegacyPendingCheckout(storage);
  assert.equal(loadPendingCheckout(user, storage).error, null);
});
