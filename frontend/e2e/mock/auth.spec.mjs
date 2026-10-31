import { test, expect } from '@playwright/test';

const accounts = {
  'operator.one': { id: 'operator-one', username: 'operator.one', role: 'operator' },
  'operator.two': { id: 'operator-two', username: 'operator.two', role: 'operator' },
  'admin.one': { id: 'admin-one', username: 'admin.one', role: 'admin' },
};
const password = 'browser-test-only password';
const keyFor = id => `techsnacks.checkout.pending.v2:${id}`;

async function fixture(page) {
  const state = { session: null, generation: 0, logoutFails: false, bootstrapFails: false, loginLimited: false, sales: [], checkout: null };
  const newSession = username => {
    state.generation++;
    return { user: accounts[username], csrf_token: (`csrf-${state.generation}`).padEnd(43, 'x'), expires_at: '2027-01-01T00:00:00Z' };
  };
  await page.route('http://127.0.0.1:4173/api/**', async route => {
    const request = route.request();
    const path = new URL(request.url()).pathname.replace('/api', '');
    const method = request.method();
    const json = (body, status = 200) => route.fulfill({ status, json: body });
    if (path === '/sessions' && method === 'POST') {
      if (state.loginLimited) return json({ error: 'limited' }, 429);
      const input = request.postDataJSON();
      if (!accounts[input.username] || input.password !== password) return json({ error: 'invalid credentials' }, 401);
      state.session = newSession(input.username);
      return route.fulfill({ status: 201, headers: { 'set-cookie': 'techsnacks_session=opaque-test-only; HttpOnly; SameSite=Lax; Path=/' }, json: state.session });
    }
    if (path === '/sessions/current') {
      if (method === 'GET') {
        if (state.bootstrapFails) return json({ error: 'unavailable' }, 503);
        return state.session ? json(state.session) : json({ error: 'expired' }, 401);
      }
      if (state.logoutFails) return json({ error: 'unavailable' }, 503);
      expect(request.headers()['x-csrf-token']).toBe(state.session.csrf_token);
      state.session = null;
      return route.fulfill({ status: 204 });
    }
    if (!state.session) return json({ error: 'unauthenticated' }, 401);
    if (path === '/products' && method === 'GET') return json([{ id: 'p', name: 'Coxinha', price: 650 }]);
    if (path === '/checkout') {
      expect(request.headers()['x-csrf-token']).toBe(state.session.csrf_token);
      const sale = { key: request.headers()['idempotency-key'], body: request.postDataJSON(), user: state.session.user.id };
      state.sales.push(sale);
      if (state.checkout) return state.checkout(route, sale);
      return json({ order_id: `ord-${sale.user}`, total_cents: 650, status: 'AWAITING_PAYMENT' });
    }
    return json({ error: 'not found' }, 404);
  });
  return state;
}

async function login(page, username = 'operator.one') {
  await expect(page.getByRole('heading', { name: 'Entrar no TechSnacks' })).toBeVisible();
  await page.getByLabel('Usuário').fill(username);
  await page.getByLabel('Senha', { exact: true }).fill(password);
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Sair', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeVisible();
}

test('login rejects invalid credentials; operator cannot navigate to product management', async ({ page }) => {
  await fixture(page);
  await page.goto('/produtos');
  await page.getByLabel('Usuário').fill('operator.one');
  await page.getByLabel('Senha', { exact: true }).fill('wrong');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('Usuário ou senha inválidos');
  await expect(page.getByLabel('Senha', { exact: true })).toHaveValue('');
  await page.getByLabel('Senha', { exact: true }).fill(password);
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('não tem permissão');
  await expect(page.getByRole('link', { name: 'Gerenciar Produtos' })).toHaveCount(0);
  await page.getByRole('link', { name: 'Caixa (PDV)' }).click();
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeVisible();
  expect(await page.evaluate(() => localStorage.length)).toBe(0);
  expect(await page.evaluate(() => document.cookie)).not.toContain('techsnacks_session');
});

test('expired session preserves the pending sale for the same account after reauthentication', async ({ page }) => {
  const state = await fixture(page);
  state.checkout = async route => { state.session = null; await route.fulfill({ status: 401, json: { error: 'expired' } }); };
  await page.goto('/login');
  await login(page);
  await page.getByRole('button', { name: /^Coxinha/ }).click();
  await page.getByRole('button', { name: 'Finalizar Pedido' }).click();
  await expect(page.getByRole('heading', { name: 'Entrar no TechSnacks' })).toBeVisible();
  const first = state.sales[0];
  expect(JSON.parse(await page.evaluate(key => sessionStorage.getItem(key), keyFor('operator-one'))).key).toBe(first.key);
  state.checkout = null;
  await login(page);
  await expect(page.getByRole('button', { name: 'Retomar pedido pendente' })).toBeVisible();
  await page.getByRole('button', { name: 'Retomar pedido pendente' }).click();
  await expect(page.getByRole('status')).toContainText('aguardando pagamento');
  expect(state.sales[1]).toEqual(first);
  expect(await page.evaluate(key => sessionStorage.getItem(key), keyFor('operator-one'))).toBeNull();
});

test('switching accounts isolates pending sales and permits recovery only in the original account', async ({ page }) => {
  const state = await fixture(page);
  state.checkout = route => route.abort('failed');
  await page.goto('/login');
  await login(page);
  await page.getByRole('button', { name: /^Coxinha/ }).click();
  await page.getByRole('button', { name: 'Finalizar Pedido' }).click();
  await expect(page.getByRole('button', { name: 'Retomar pedido pendente' })).toBeVisible();
  const first = state.sales[0];
  await page.getByRole('button', { name: 'Sair', exact: true }).click();
  await login(page, 'operator.two');
  await expect(page.getByText('O carrinho está vazio.')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Retomar pedido pendente' })).toHaveCount(0);
  await page.getByRole('button', { name: /^Coxinha/ }).click();
  await page.getByRole('button', { name: 'Finalizar Pedido' }).click();
  await expect(page.getByRole('button', { name: 'Retomar pedido pendente' })).toBeVisible();
  const second = state.sales[1];
  expect(second.key).not.toBe(first.key);
  await page.getByRole('button', { name: 'Sair', exact: true }).click();
  await login(page);
  state.checkout = null;
  await page.getByRole('button', { name: 'Retomar pedido pendente' }).click();
  await expect(page.getByRole('status')).toContainText('ord-operator-one');
  expect(state.sales[2]).toEqual(first);
  expect(JSON.parse(await page.evaluate(key => sessionStorage.getItem(key), keyFor('operator-two'))).key).toBe(second.key);
});

test('logout failure keeps the session visible; successful logout returns to login', async ({ page }) => {
  const state = await fixture(page);
  await page.goto('/login');
  await login(page, 'admin.one');
  await expect(page.getByRole('link', { name: 'Gerenciar Produtos' })).toBeVisible();
  state.logoutFails = true;
  await page.getByRole('button', { name: 'Sair', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('Não foi possível encerrar');
  await expect(page.getByRole('button', { name: 'Sair', exact: true })).toBeVisible();
  state.logoutFails = false;
  await page.getByRole('button', { name: 'Sair', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Entrar no TechSnacks' })).toBeVisible();
});

test('bootstrap outage blocks business screens and login rate limit has actionable feedback', async ({ page }) => {
  const state = await fixture(page);
  state.bootstrapFails = true;
  await page.goto('/login');
  await expect(page.getByRole('alert')).toContainText('verificar sua sessão');
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toHaveCount(0);
  state.bootstrapFails = false;
  await page.getByRole('button', { name: 'Tentar novamente' }).click();
  state.loginLimited = true;
  await page.getByLabel('Usuário').fill('operator.one');
  await page.getByLabel('Senha', { exact: true }).fill(password);
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('Limite de tentativas');
});

test('legacy anonymous attempt is not attached to an operator; only admin can explicitly discard it', async ({ page }) => {
  await fixture(page);
  await page.addInitScript(() => sessionStorage.setItem('techsnacks.checkout.pending.v1', '{legacy-checkout}'));
  await page.goto('/login');
  await login(page);
  await expect(page.getByRole('alert')).toContainText('anterior à autenticação');
  await expect(page.getByRole('button', { name: 'Finalizar Pedido' })).toBeDisabled();
  await expect(page.getByRole('button', { name: 'Iniciar nova venda' })).toBeDisabled();
  await page.getByRole('button', { name: 'Sair', exact: true }).click();
  await login(page, 'admin.one');
  await page.getByRole('button', { name: 'Iniciar nova venda' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Descartar tentativa local' }).click();
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeEnabled();
  expect(await page.evaluate(() => sessionStorage.getItem('techsnacks.checkout.pending.v1'))).toBeNull();
});
