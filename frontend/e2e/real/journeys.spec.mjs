import { test, expect } from '@playwright/test';
import { resetFixtures } from './database.mjs';

test.beforeEach(async () => resetFixtures());

test('public project home and dedicated login work before authentication and after entering the application', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1, name: 'Do catálogo ao pedido, um caixa com clareza.' })).toBeVisible();
  await page.getByRole('link', { name: 'Entrar no sistema', exact: true }).first().click();
  await expect(page).toHaveURL(/\/login$/);
  await page.reload();
  await expect(page.getByRole('heading', { name: 'Entrar no TechSnacks' })).toBeVisible();
  await login(page);
  await expect(page).toHaveURL(/\/caixa$/);
  await page.getByRole('link', { name: 'TechSnacks, sobre o projeto', exact: true }).click();
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByRole('heading', { level: 1, name: 'Do catálogo ao pedido, um caixa com clareza.' })).toBeVisible();
});

async function login(page, username = 'e2e.admin') {
  await page.goto('/login');
  await page.getByLabel('Usuário', { exact: true }).fill(username);
  await page.getByLabel('Senha', { exact: true }).fill(process.env.E2E_PASSWORD);
  const response = page.waitForResponse(r => r.url().endsWith('/api/sessions') && r.request().method() === 'POST');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  const result = await (await response).json();
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeVisible();
  return result;
}

test('real login sets an HttpOnly cookie; CRUD and SPA navigation work through the same-origin proxy', async ({ page, context }) => {
  await login(page);
  const cookies = await context.cookies();
  expect(cookies.find(cookie => cookie.name === 'techsnacks_session')?.httpOnly).toBe(true);
  expect(await page.evaluate(() => document.cookie)).not.toContain('techsnacks_session');
  await page.getByRole('link', { name: 'Gerenciar Produtos' }).click();
  await page.reload();
  await expect(page.getByRole('table')).toBeVisible();
  await page.getByRole('button', { name: 'Novo produto', exact: true }).click();
  let dialog = page.getByRole('dialog');
  await dialog.getByLabel('Nome do produto', { exact: true }).fill('Produto E2E');
  await dialog.getByLabel('Preço (R$)', { exact: true }).fill('6,50');
  await dialog.getByRole('button', { name: 'Cadastrar produto' }).click();
  await expect(dialog).toHaveCount(0);
  await page.getByRole('button', { name: 'Editar Produto E2E', exact: true }).click();
  dialog = page.getByRole('dialog');
  await dialog.getByLabel('Preço (R$)', { exact: true }).fill('7,25');
  await dialog.getByRole('button', { name: 'Salvar alterações' }).click();
  await expect(dialog).toHaveCount(0);
  await expect(page.getByRole('row').filter({ has: page.getByRole('button', { name: 'Editar Produto E2E', exact: true }) })).toContainText('7,25');
  await page.getByRole('button', { name: 'Excluir Produto E2E', exact: true }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Excluir produto', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Editar Produto E2E', exact: true })).toHaveCount(0);
});

test('real checkout is replayed after reload, price edits preserve history, and operators cannot read another account sale', async ({ page }) => {
  const admin = await login(page);
  const payload = { items: [{ product_id: 'p1', quantity: 2 }] };
  const key = crypto.randomUUID();
  const sale = await page.evaluate(async ({ csrf, payload, key }) => {
    const response = await fetch('/api/checkout', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': key }, body: JSON.stringify(payload) });
    return { status: response.status, body: await response.json() };
  }, { csrf: admin.csrf_token, payload, key });
  expect(sale.status).toBe(200);
  expect(sale.body.total_cents).toBe(1300);
  await page.evaluate(({ id, key }) => sessionStorage.setItem(`techsnacks.checkout.pending.v2:${id}`, JSON.stringify({ version: 2, user_id: id, key, items: [{ id: 'p1', name: 'Coxinha de frango', price: 650, quantity: 2 }] })), { id: admin.user.id, key });
  await page.reload();
  await page.getByRole('button', { name: 'Retomar pedido pendente' }).click();
  await expect(page.getByRole('status')).toContainText(sale.body.order_id);
  await page.getByRole('link', { name: 'Gerenciar Produtos' }).click();
  await page.getByRole('button', { name: 'Editar Coxinha de frango', exact: true }).click();
  await page.getByRole('dialog').getByLabel('Preço (R$)', { exact: true }).fill('9,00');
  await page.getByRole('dialog').getByRole('button', { name: 'Salvar alterações' }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  const receipt = await page.request.get(`/api/orders/${sale.body.order_id}`);
  expect((await receipt.json()).total_cents).toBe(1300);
  await page.getByRole('button', { name: 'Sair', exact: true }).click();
  await login(page, 'e2e.operator');
  expect((await page.request.get(`/api/orders/${sale.body.order_id}`)).status()).toBe(404);
  await page.goto('/produtos');
  await expect(page.getByRole('alert')).toContainText('não tem permissão');
});

test('real logout revokes the session and anonymous mutations are denied', async ({ page, context }) => {
  const session = await login(page, 'e2e.operator');
  const cookie = (await context.cookies()).find(value => value.name === 'techsnacks_session');
  await page.getByRole('button', { name: 'Sair', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Entrar no TechSnacks' })).toBeVisible();
  const replay = await page.request.get('/api/sessions/current', { headers: { Cookie: `${cookie.name}=${cookie.value}` } });
  expect(replay.status()).toBe(401);
  expect((await page.request.post('/api/checkout', { headers: { Origin: process.env.E2E_BASE_URL, 'X-CSRF-Token': session.csrf_token }, data: { items: [] } })).status()).toBe(401);
});

test('real CSRF and role boundaries reject writes even when the client sends valid JSON', async ({ page }) => {
  const operator = await login(page, 'e2e.operator');
  const origin = process.env.E2E_BASE_URL;
  const denied = await page.request.patch('/api/products/p1', { headers: { Origin: origin, 'X-CSRF-Token': operator.csrf_token }, data: { price: 1 } });
  expect(denied.status()).toBe(403);
  const missingCsrf = await page.request.post('/api/checkout', { headers: { Origin: origin }, data: { items: [{ product_id: 'p1', quantity: 1 }] } });
  expect(missingCsrf.status()).toBe(403);
  expect((await page.request.get('/api/health/live')).status()).toBe(200);
  expect((await page.request.get('/api/health/ready')).status()).toBe(200);
});
