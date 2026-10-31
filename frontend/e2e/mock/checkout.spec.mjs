import { expect, test } from '@playwright/test';

const headers = {
  'access-control-allow-origin': '*',
  'access-control-allow-methods': 'GET, POST, OPTIONS',
  'access-control-allow-headers': 'Content-Type, Idempotency-Key',
};
const successfulResult = {
  order_id: 'ord-BROWSER-TEST',
  total_cents: 1400,
  status: 'AWAITING_PAYMENT',
};

test.beforeEach(async ({ page }) => {
  await page.route('**/api/sessions/current', route => route.fulfill({ json: {
    user: { id: 'operator-one', username: 'operator.one', role: 'operator' },
    csrf_token: 'c'.repeat(43), expires_at: '2027-01-01T00:00:00Z',
  } }));
  await page.route('**/api/products', async route => {
    await route.fulfill({
      headers,
      json: [{ id: 'product-1', name: 'Coxinha', price: 650 }],
    });
  });
});

async function openCart(page, quantity = 1) {
  await page.goto('/caixa');
  const product = page.getByRole('button', { name: /^Coxinha/ });
  await expect(product).toBeVisible();
  for (let i = 0; i < quantity; i++) await product.click();
}

async function routeCheckout(page, handler) {
  await page.route('**/api/checkout', async route => {
    if (route.request().method() === 'OPTIONS') {
      await route.fulfill({ status: 204, headers });
      return;
    }
    await handler(route);
  });
}

test('empty cart cannot be submitted', async ({ page }) => {
  let requests = 0;
  await routeCheckout(page, async route => {
    requests++;
    await route.fulfill({ headers, json: successfulResult });
  });
  await page.goto('/caixa');
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Finalizar Pedido' })).toBeDisabled();
  await expect(page.getByRole('button', { name: 'Cancelar Venda' })).toBeDisabled();
  expect(requests).toBe(0);
});

test('pending checkout blocks edits and duplicate submissions; confirmation uses server data', async ({ page }, testInfo) => {
  let requests = 0;
  let payload;
  let release;
  const pending = new Promise(resolve => { release = resolve; });
  await routeCheckout(page, async route => {
    requests++;
    payload = route.request().postDataJSON();
    await pending;
    await route.fulfill({ headers, json: successfulResult });
  });
  await openCart(page, 2);
  try {
    await page.evaluate(() => {
      const buttons = [...document.querySelectorAll('button')];
      const finish = buttons.find(button => button.textContent.trim() === 'Finalizar Pedido');
      const cancel = buttons.find(button => button.textContent.trim() === 'Cancelar Venda');
      finish.click();
      finish.click();
      buttons.find(button => button.classList.contains('product-card')).click();
      buttons.find(button => button.getAttribute('aria-label')?.startsWith('Diminuir')).click();
      cancel.click();
    });
    await expect(page.getByRole('button', { name: 'Processando...' })).toBeDisabled();
    await expect.poll(() => requests).toBe(1);
    await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeDisabled();
    await expect(page.getByRole('button', { name: 'Diminuir quantidade de Coxinha' })).toBeDisabled();
    await expect(page.getByRole('button', { name: 'Iniciar nova venda' })).toBeDisabled();
    await expect(page.locator('li')).toContainText('x 2');
    expect(payload).toEqual({ items: [{ product_id: 'product-1', quantity: 2 }] });
  } finally {
    release();
  }
  await expect(page.getByRole('status')).toContainText('ord-BROWSER-TEST');
  await expect(page.getByRole('status')).toContainText('aguardando pagamento');
  await expect(page.getByRole('status')).toContainText('14,00');
  await expect(page.getByText('O carrinho está vazio.')).toBeVisible();
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeEnabled();
  expect(requests).toBe(1);
  await page.screenshot({ path: testInfo.outputPath('checkout-confirmed.png'), fullPage: true });
  await page.getByRole('button', { name: /^Coxinha/ }).click();
  await expect(page.getByRole('status')).toHaveCount(0);
});

const failures = [
  { name: 'validation rejection', respond: route => route.fulfill({ status: 422, headers, json: { error: 'invalid quantity' } }) },
  { name: 'server failure', respond: route => route.fulfill({ status: 500, headers, json: { error: 'internal failure' } }) },
  { name: 'connection lost', respond: route => route.abort('failed') },
  { name: 'malformed JSON', respond: route => route.fulfill({ status: 200, headers, contentType: 'application/json', body: '{' }) },
  { name: 'missing order ID', respond: route => route.fulfill({ headers, json: { ...successfulResult, order_id: '' } }) },
  { name: 'wrong status', respond: route => route.fulfill({ headers, json: { ...successfulResult, status: 'COMPLETED' } }) },
  { name: 'unsafe total', respond: route => route.fulfill({ headers, json: { ...successfulResult, total_cents: Number.MAX_SAFE_INTEGER + 1 } }) },
  { name: 'negative total', respond: route => route.fulfill({ headers, json: { ...successfulResult, total_cents: -1 } }) },
  { name: 'fractional total', respond: route => route.fulfill({ headers, json: { ...successfulResult, total_cents: 1.5 } }) },
  { name: 'null response', respond: route => route.fulfill({ headers, contentType: 'application/json', body: 'null' }) },
];

for (const failure of failures) {
  test(`${failure.name} preserves cart and never reports payment success`, async ({ page }) => {
    let requests = 0;
    await routeCheckout(page, async route => {
      requests++;
      await failure.respond(route);
    });
    await openCart(page);
    await page.getByRole('button', { name: 'Finalizar Pedido' }).click();
    await expect(page.getByRole('alert')).toContainText('carrinho foi mantido');
    await expect(page.getByRole('status')).toHaveCount(0);
    await expect(page.locator('li')).toContainText('Coxinha');
    await expect(page.locator('li')).toContainText('x 1');
    await expect(page.getByRole('button', { name: 'Retomar pedido pendente' })).toBeEnabled();
    await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeDisabled();
    expect(requests).toBe(1);
  });
}

test('successful zero-total order remains valid', async ({ page }) => {
  await routeCheckout(page, route => route.fulfill({ headers, json: { ...successfulResult, total_cents: 0 } }));
  await openCart(page);
  await page.getByRole('button', { name: 'Finalizar Pedido' }).click();
  await expect(page.getByRole('status')).toContainText('0,00');
  await expect(page.getByText('O carrinho está vazio.')).toBeVisible();
});

const storageKey = 'techsnacks.checkout.pending.v2:operator-one';

test('lost response and reload reuse the same key and payload; a new sale gets a new key', async ({ page }) => {
  const requests = [];
  const recorded = new Map();
  await routeCheckout(page, async route => {
    const key = route.request().headers()['idempotency-key'];
    const body = route.request().postDataJSON();
    expect(key).toMatch(/^[A-Za-z0-9_-]{1,128}$/);
    requests.push({ key, body });
    if (!recorded.has(key)) recorded.set(key, { ...successfulResult, order_id: `ord-${recorded.size + 1}` });
    if (requests.length === 1) await route.abort('failed');
    else await route.fulfill({ headers, json: recorded.get(key) });
  });
  await openCart(page, 2);
  await page.getByRole('button', { name: 'Finalizar Pedido' }).click();
  await expect(page.getByRole('alert')).toContainText('Retomar pedido pendente');
  const stored = await page.evaluate(key => JSON.parse(sessionStorage.getItem(key)), storageKey);
  expect(stored.key).toBe(requests[0].key);
  await page.reload();
  await expect(page.locator('li')).toContainText('x 2');
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeDisabled();
  await page.getByRole('button', { name: 'Retomar pedido pendente' }).click();
  await expect(page.getByRole('status')).toContainText('ord-1');
  expect(requests[1]).toEqual(requests[0]);
  expect(recorded.size).toBe(1);
  expect(await page.evaluate(key => sessionStorage.getItem(key), storageKey)).toBeNull();
  await page.getByRole('button', { name: /^Coxinha/ }).click();
  await page.getByRole('button', { name: 'Finalizar Pedido' }).click();
  await expect(page.getByRole('status')).toContainText('ord-2');
  expect(requests[2].key).not.toBe(requests[0].key);
  expect(recorded.size).toBe(2);
});

test('conflict preserves pending key; discard requires explicit confirmation', async ({ page }) => {
  const keys = [];
  await routeCheckout(page, async route => {
    keys.push(route.request().headers()['idempotency-key']);
    await route.fulfill({ status: 409, headers, json: { error: 'idempotency conflict' } });
  });
  await openCart(page);
  await page.getByRole('button', { name: 'Finalizar Pedido' }).click();
  await expect(page.getByRole('alert')).toContainText('conflita');
  await page.getByRole('button', { name: 'Iniciar nova venda' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Manter pedido' }).click();
  await expect(page.getByRole('button', { name: 'Retomar pedido pendente' })).toBeVisible();
  expect(JSON.parse(await page.evaluate(key => sessionStorage.getItem(key), storageKey)).key).toBe(keys[0]);
  await page.getByRole('button', { name: 'Iniciar nova venda' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Descartar tentativa local' }).click();
  await expect(page.getByText('O carrinho está vazio.')).toBeVisible();
  expect(await page.evaluate(key => sessionStorage.getItem(key), storageKey)).toBeNull();
  await page.getByRole('button', { name: /^Coxinha/ }).click();
  await page.getByRole('button', { name: 'Finalizar Pedido' }).click();
  await expect(page.getByRole('alert')).toContainText('conflita');
  expect(keys).toHaveLength(2);
  expect(keys[1]).not.toBe(keys[0]);
});

test('storage write failure prevents the first request', async ({ page }) => {
  await page.addInitScript(() => {
    Storage.prototype.setItem = function () { throw new Error('storage disabled'); };
  });
  let requests = 0;
  await routeCheckout(page, async route => {
    requests++;
    await route.fulfill({ headers, json: successfulResult });
  });
  await openCart(page);
  await page.getByRole('button', { name: 'Finalizar Pedido' }).click();
  await expect(page.getByRole('alert')).toContainText('não foi enviado');
  expect(requests).toBe(0);
  await expect(page.locator('li')).toContainText('x 1');
});

test('corrupt recovery data blocks checkout until an explicit discard', async ({ page }) => {
  await page.addInitScript(key => sessionStorage.setItem(key, '{broken'), storageKey);
  await page.goto('/caixa');
  await expect(page.getByRole('alert')).toContainText('recuperar o registro local');
  await expect(page.getByRole('button', { name: 'Finalizar Pedido' })).toBeDisabled();
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeDisabled();
  expect(await page.evaluate(key => sessionStorage.getItem(key), storageKey)).toBe('{broken');
  await page.getByRole('button', { name: 'Iniciar nova venda' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Descartar tentativa local' }).click();
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeEnabled();
  expect(await page.evaluate(key => sessionStorage.getItem(key), storageKey)).toBeNull();
});

test('cleanup failure after success retains the same attempt for safe recovery', async ({ page }) => {
  await page.addInitScript(key => {
    const original = Storage.prototype.removeItem;
    let failed = false;
    Storage.prototype.removeItem = function (name) {
      if (name === key && !failed) { failed = true; throw new Error('temporary storage failure'); }
      return original.call(this, name);
    };
  }, storageKey);
  const keys = [];
  await routeCheckout(page, async route => {
    keys.push(route.request().headers()['idempotency-key']);
    await route.fulfill({ headers, json: successfulResult });
  });
  await openCart(page);
  await page.getByRole('button', { name: 'Finalizar Pedido' }).click();
  await expect(page.getByRole('status')).toContainText('ord-BROWSER-TEST');
  await expect(page.getByRole('alert')).toContainText('limpar a tentativa local');
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeDisabled();
  await page.getByRole('button', { name: 'Retomar pedido pendente' }).click();
  await expect(page.getByText('O carrinho está vazio.')).toBeVisible();
  expect(keys).toHaveLength(2);
  expect(keys[0]).toBe(keys[1]);
  expect(await page.evaluate(key => sessionStorage.getItem(key), storageKey)).toBeNull();
});
