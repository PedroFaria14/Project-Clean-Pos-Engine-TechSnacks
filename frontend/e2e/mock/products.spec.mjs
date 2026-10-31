import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const csrf = 'x'.repeat(43);
async function fixture(page, role = 'admin') {
  const state = {
    items: [{ id: 'p1', name: 'Coxinha de frango', price: 650 }, { id: 'p2', name: 'Pão de queijo', price: 350 }, { id: 'p3', name: 'Suco natural', price: 800 }],
    creates: [], patches: [], deletes: [], failList: false, failSave: false, failDelete: false, malformed: false, saveGate: null,
  };
  await page.route('http://127.0.0.1:4173/api/**', async route => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const method = request.method();
    const json = (body, status = 200) => route.fulfill({ status, json: body });
    if (path === '/api/sessions/current') return json({ user: { id: 'ui-user', username: 'admin.local', role }, csrf_token: csrf, expires_at: '2027-01-01T00:00:00Z' });
    if (method !== 'GET') expect(request.headers()['x-csrf-token']).toBe(csrf);
    if (path === '/api/products' && method === 'GET') {
      if (state.failList) return json({ error: 'unavailable' }, 503);
      if (state.malformed) return route.fulfill({ contentType: 'application/json', body: '{invalid' });
      return json(state.items);
    }
    if (path === '/api/products' && method === 'POST') {
      const body = request.postDataJSON();
      state.creates.push(body);
      if (state.saveGate) await state.saveGate;
      if (state.failSave) return json({ error: 'unavailable' }, 503);
      if (state.items.some(item => item.id === body.id)) return json({ error: 'duplicate' }, 422);
      state.items.push(body);
      return json(body, 201);
    }
    if (path.startsWith('/api/products/') && method === 'PATCH') {
      const id = decodeURIComponent(path.split('/').at(-1));
      const body = request.postDataJSON();
      state.patches.push({ id, body });
      if (state.failSave) return json({ error: 'unavailable' }, 503);
      const index = state.items.findIndex(item => item.id === id);
      if (index < 0) return json({ error: 'not found' }, 404);
      state.items[index] = { ...state.items[index], ...body };
      return json(state.items[index]);
    }
    if (path.startsWith('/api/products/') && method === 'DELETE') {
      const id = decodeURIComponent(path.split('/').at(-1));
      state.deletes.push(id);
      if (state.failDelete) return json({ error: 'unavailable' }, 500);
      state.items = state.items.filter(item => item.id !== id);
      return route.fulfill({ status: 204 });
    }
    if (path === '/api/checkout') return json({ order_id: 'ord-ui', status: 'AWAITING_PAYMENT', total_cents: 650 });
    return json({ error: 'not found' }, 404);
  });
  return state;
}

async function management(page) {
  await page.goto('/produtos');
  await expect(page.getByRole('table')).toBeVisible();
}

function productRow(page, name) { return page.getByRole('row').filter({ has: page.getByRole('button', { name: `Editar ${name}`, exact: true }) }); }

test('CRUD validates decimal fields, blocks duplicate saves, and updates the visible list', async ({ page }) => {
  const state = await fixture(page);
  await management(page);
  await page.getByRole('button', { name: 'Novo produto', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Novo produto' });
  await expect(dialog.getByLabel('Nome do produto', { exact: true })).toBeFocused();
  await dialog.getByRole('button', { name: 'Cadastrar produto' }).click();
  await expect(dialog.getByText('Informe um nome com 1 a 120 caracteres.')).toBeVisible();
  expect(state.creates).toHaveLength(0);
  await dialog.getByLabel('Nome do produto', { exact: true }).fill('Bolinho da casa');
  await dialog.getByLabel('Preço (R$)', { exact: true }).fill('12,345');
  await dialog.getByRole('button', { name: 'Cadastrar produto' }).click();
  await expect(dialog.getByLabel('Preço (R$)', { exact: true })).toHaveAttribute('aria-invalid', 'true');
  expect(state.creates).toHaveLength(0);
  await dialog.getByLabel('Preço (R$)', { exact: true }).fill('12,35');
  let release;
  state.saveGate = new Promise(resolve => { release = resolve; });
  await dialog.getByRole('button', { name: 'Cadastrar produto' }).click();
  try {
    await expect.poll(() => state.creates.length).toBe(1);
    await expect(dialog.getByRole('button', { name: 'Salvando...' })).toBeDisabled();
    await expect(dialog.getByRole('button', { name: 'Fechar diálogo' })).toBeDisabled();
    await expect(dialog).toBeFocused();
    await page.keyboard.press('Escape');
    await expect(dialog).toBeVisible();
    expect(state.creates[0].price).toBe(1235);
  } finally { release(); }
  await expect(dialog).toHaveCount(0);
  await expect(productRow(page, 'Bolinho da casa')).toContainText('12,35');
  await expect(page.getByRole('status')).toContainText('cadastrado');
  await page.getByRole('button', { name: 'Editar Bolinho da casa', exact: true }).click();
  const edit = page.getByRole('dialog', { name: 'Editar produto' });
  await expect(edit.getByLabel('Preço (R$)', { exact: true })).toHaveValue('12,35');
  await expect(edit.getByRole('button', { name: 'Salvar alterações' })).toBeDisabled();
  await edit.getByLabel('Preço (R$)', { exact: true }).fill('0');
  await edit.getByRole('button', { name: 'Salvar alterações' }).click();
  await expect(edit).toHaveCount(0);
  expect(state.patches).toHaveLength(1);
  expect(state.patches[0].body).toEqual({ price: 0 });
  await expect(productRow(page, 'Bolinho da casa')).toContainText('0,00');
  await page.getByRole('button', { name: 'Excluir Bolinho da casa', exact: true }).click();
  const remove = page.getByRole('dialog', { name: 'Excluir produto?' });
  await expect(remove.getByRole('button', { name: 'Manter produto' })).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('button', { name: 'Excluir Bolinho da casa', exact: true })).toBeFocused();
  await page.getByRole('button', { name: 'Excluir Bolinho da casa', exact: true }).click();
  state.failDelete = true;
  await remove.getByRole('button', { name: 'Excluir produto', exact: true }).click();
  await expect(remove.getByRole('alert')).toContainText('Não foi possível');
  await expect(productRow(page, 'Bolinho da casa')).toHaveCount(1);
  state.failDelete = false;
  await remove.getByRole('button', { name: 'Excluir produto', exact: true }).click();
  await expect(remove).toHaveCount(0);
  await expect(productRow(page, 'Bolinho da casa')).toHaveCount(0);
  await expect(page.getByRole('status')).toContainText('histórico de pedidos foi preservado');
});

test('save failures preserve form values and the same create ID for an explicit retry', async ({ page }) => {
  const state = await fixture(page);
  await management(page);
  await page.getByRole('button', { name: 'Novo produto', exact: true }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByLabel('Nome do produto', { exact: true }).fill('Produto em teste');
  await dialog.getByLabel('Preço (R$)', { exact: true }).fill('0,10');
  state.failSave = true;
  await dialog.getByRole('button', { name: 'Cadastrar produto' }).click();
  await expect(dialog.getByRole('alert')).toContainText('campos foram mantidos');
  await expect(dialog.getByLabel('Nome do produto', { exact: true })).toHaveValue('Produto em teste');
  await expect(dialog.getByLabel('Preço (R$)', { exact: true })).toHaveValue('0,10');
  const id = state.creates[0].id;
  state.failSave = false;
  await dialog.getByRole('button', { name: 'Cadastrar produto' }).click();
  await expect(dialog).toHaveCount(0);
  expect(state.creates).toHaveLength(2);
  expect(state.creates[1].id).toBe(id);
  await expect(productRow(page, 'Produto em teste')).toContainText('0,10');
});

test('search is accent-insensitive and network errors do not masquerade as an empty catalog', async ({ page }) => {
  const state = await fixture(page);
  state.failList = true;
  await page.goto('/produtos');
  await expect(page.getByRole('alert')).toContainText('carregar o catálogo');
  await expect(page.getByText('Seu catálogo começa aqui')).toHaveCount(0);
  state.failList = false;
  await page.getByRole('button', { name: 'Tentar novamente' }).click();
  await expect(page.getByRole('table')).toBeVisible();
  await page.getByLabel('Buscar produtos').fill('pao');
  await expect(page.getByRole('row')).toHaveCount(2);
  await expect(productRow(page, 'Pão de queijo')).toHaveCount(1);
  await page.getByLabel('Buscar produtos').fill('missing');
  await expect(page.getByText('Nenhum produto encontrado')).toBeVisible();
  await page.getByRole('button', { name: 'Limpar busca' }).click();
  await expect(page.getByRole('row')).toHaveCount(4);
});

test('cash register has working quantity controls, safe totals and an accessible cancel dialog', async ({ page }) => {
  const state = await fixture(page, 'operator');
  state.items.push({ id: 'max', name: 'Limite monetário', price: Number.MAX_SAFE_INTEGER });
  await page.goto('/caixa');
  await page.getByLabel('Buscar produtos').fill('cox');
  await expect(page.locator('.product-card')).toHaveCount(1);
  await page.getByRole('button', { name: /^Coxinha/ }).click();
  await page.getByRole('button', { name: 'Aumentar quantidade de Coxinha de frango' }).click();
  await expect(page.locator('.cart-summary__total')).toContainText('13,00');
  await page.getByRole('button', { name: 'Diminuir quantidade de Coxinha de frango' }).click();
  await expect(page.locator('.cart-summary__total')).toContainText('6,50');
  await page.getByRole('button', { name: 'Cancelar Venda', exact: true }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Manter pedido' }).click();
  await expect(page.locator('.cart-item')).toHaveCount(1);
  await page.getByRole('button', { name: 'Cancelar Venda', exact: true }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Cancelar venda', exact: true }).click();
  await expect(page.getByText('O carrinho está vazio.')).toBeVisible();
  await page.getByLabel('Buscar produtos').fill('Limite');
  await page.getByRole('button', { name: /^Limite monetário/ }).click();
  await page.getByRole('button', { name: 'Aumentar quantidade de Limite monetário' }).click();
  await expect(page.getByRole('alert')).toContainText('limite monetário seguro');
  await expect(page.locator('.cart-item__unit')).toContainText('x 1');
  await expect(page.locator('.cart-summary__total')).toContainText('409,91');
});

test('dialog traps keyboard focus and returns focus when dismissed', async ({ page }) => {
  await fixture(page);
  await management(page);
  const opener = page.getByRole('button', { name: 'Novo produto', exact: true });
  await opener.click();
  expect(await page.getByRole('dialog').evaluate(dialog => dialog.matches(':modal'))).toBe(true);
  for (let index = 0; index < 9; index++) {
    await page.keyboard.press('Tab');
    expect(await page.evaluate(() => document.querySelector('dialog').contains(document.activeElement))).toBe(true);
  }
  await page.keyboard.press('Escape');
  await expect(opener).toBeFocused();
});

for (const size of [{ width: 320, height: 800 }, { width: 768, height: 1024 }, { width: 1440, height: 900 }]) {
  test(`responsive register and management at ${size.width}px have no horizontal overflow`, async ({ page }, info) => {
    await fixture(page);
    await page.setViewportSize(size);
    await page.goto('/caixa');
    await page.getByRole('button', { name: /^Coxinha/ }).click();
    await page.getByRole('button', { name: /^Pão de queijo/ }).click();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await page.screenshot({ path: info.outputPath(`register-${size.width}.png`), fullPage: true });
    await page.getByRole('link', { name: 'Gerenciar Produtos' }).click();
    await expect(page.getByRole('table')).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await page.screenshot({ path: info.outputPath(`products-${size.width}.png`), fullPage: true });
    if (size.width === 320) {
      for (const button of await page.getByRole('button').all()) {
        const box = await button.boundingBox();
        if (box) expect(box.height).toBeGreaterThanOrEqual(44);
      }
    }
  });
}

test('WCAG A/AA automated checks cover register, management and the product dialog', async ({ page }) => {
  await fixture(page);
  await page.goto('/caixa');
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeVisible();
  const check = async () => {
    const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze();
    expect(results.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([]);
  };
  await check();
  await page.getByRole('link', { name: 'Gerenciar Produtos' }).click();
  await expect(page.getByRole('table')).toBeVisible();
  await check();
  await page.getByRole('button', { name: 'Novo produto', exact: true }).click();
  await check();
});

test('login is responsive and passes automated WCAG checks', async ({ page }, info) => {
  await page.route('**/api/sessions/current', route => route.fulfill({ status: 401, json: { error: 'not authenticated' } }));
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: 'Entrar no TechSnacks' })).toBeVisible();
  const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze();
  expect(results.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => n.target) }))).toEqual([]);
  await page.screenshot({ path: info.outputPath('login-desktop.png'), fullPage: true });
  await page.setViewportSize({ width: 320, height: 740 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.screenshot({ path: info.outputPath('login-mobile.png'), fullPage: true });
});
