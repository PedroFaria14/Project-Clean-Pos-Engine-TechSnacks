import assert from "node:assert/strict";
import { test } from "node:test";
import { loginDestination } from "../src/auth/login_destination.ts";

test("post-login redirects allow only application routes", () => {
  assert.equal(loginDestination({ from: "/produtos" }), "/produtos");
  assert.equal(loginDestination({ from: "/caixa" }), "/caixa");
  for (const state of [null, undefined, {}, [], "/produtos", { from: "https://evil.example" }, { from: "//evil.example" }, { from: "/login" }, { from: "/produtos?redirect=external" }]) {
    assert.equal(loginDestination(state), "/caixa");
  }
});
