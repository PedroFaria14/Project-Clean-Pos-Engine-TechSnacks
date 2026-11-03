import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const homeTitle = 'Do catálogo ao pedido, um caixa com clareza.';
const session = { user: { id: 'home-user', username: 'operator.home', role: 'operator' }, csrf_token: 'x'.repeat(43), expires_at: '2027-01-01T00:00:00Z' };

test('public presentation renders before session lookup and remains available during an API outage', async ({ page }) => {
  let release;
  const pending = new Promise(resolve => { release = resolve; });
  let catalogCalls = 0;
  await page.route('**/api/sessions/current', async route => { await pending; await route.fulfill({ status: 503, json: { error: 'unavailable' } }); });
  await page.route('**/api/products', async route => { catalogCalls++; await route.fulfill({ json: [] }); });
  await page.goto('/');
  try {
    await expect(page.getByRole('heading', { level: 1, name: homeTitle })).toBeVisible();
    await expect(page.getByRole('textbox', { name: 'Senha' })).toHaveCount(0);
    expect(catalogCalls).toBe(0);
    await page.getByRole('navigation', { name: 'Navegação da apresentação' }).getByRole('link', { name: 'Tecnologia', exact: true }).click();
    await expect(page).toHaveURL(/\/#tecnologia$/);
    await expect(page.getByRole('heading', { name: 'Tecnologia com propósito. Responsabilidades separadas.' })).toBeVisible();
  } finally { release(); }
  await page.getByRole('link', { name: 'Entrar no sistema', exact: true }).first().click();
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('alert')).toContainText('verificar sua sessão');
  await page.getByRole('link', { name: 'Sobre o projeto', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1, name: homeTitle })).toBeVisible();
});

test('anonymous users reach public home and login, while business URLs stay protected', async ({ page }) => {
  let catalogCalls = 0;
  await page.route('**/api/sessions/current', route => route.fulfill({ status: 401, json: { error: 'unauthenticated' } }));
  await page.route('**/api/products', route => { catalogCalls++; return route.fulfill({ json: [] }); });
  await page.goto('/caixa');
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('heading', { name: 'Entrar no TechSnacks' })).toBeVisible();
  await page.getByRole('link', { name: 'Conhecer o projeto', exact: true }).click();
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByRole('heading', { level: 1, name: homeTitle })).toBeVisible();
  await page.goto('/produtos');
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('button', { name: 'Cadastrar produto' })).toHaveCount(0);
  expect(catalogCalls).toBe(0);
  await page.goto('/unknown-route');
  await expect(page).toHaveURL(/\/$/);
});

test('authenticated users can read the presentation and access the moved cash register without logging in again', async ({ page }) => {
  let loginCalls = 0;
  await page.route('**/api/sessions/current', route => route.fulfill({ json: session }));
  await page.route('**/api/sessions', route => { loginCalls++; return route.fulfill({ status: 201, json: session }); });
  await page.route('**/api/products', route => route.fulfill({ json: [{ id: 'p', name: 'Coxinha', price: 650 }] }));
  await page.goto('/');
  await page.getByRole('link', { name: 'Entrar no sistema', exact: true }).first().click();
  await expect(page).toHaveURL(/\/caixa$/);
  await expect(page.getByRole('button', { name: /^Coxinha/ })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Caixa (PDV)', exact: true })).toHaveAttribute('aria-current', 'page');
  await page.getByRole('link', { name: 'TechSnacks, sobre o projeto', exact: true }).click();
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByRole('heading', { level: 1, name: homeTitle })).toBeVisible();
  expect(loginCalls).toBe(0);
});

for (const width of [320, 768, 1440]) {
  test(`public home is responsive at ${width}px without horizontal overflow`, async ({ page }, info) => {
    await page.route('**/api/sessions/current', route => route.fulfill({ status: 401, json: { error: 'unauthenticated' } }));
    await page.setViewportSize({ width, height: 900 });
    await page.goto('/');
    await expect(page.getByRole('heading', { level: 1, name: homeTitle })).toBeVisible();
    const layout = await page.evaluate(() => ({
      width: document.documentElement.scrollWidth,
      viewport: window.innerWidth,
      overflow: Array.from(document.querySelectorAll('body *')).filter(element => element.getBoundingClientRect().right > window.innerWidth + 1).map(element => ({ tag: element.tagName, class: element.className, right: element.getBoundingClientRect().right })),
      contentOverflow: Array.from(document.querySelectorAll('body *')).filter(element => element.scrollWidth > element.clientWidth + 1).map(element => ({ tag: element.tagName, class: element.className, width: element.clientWidth, scrollWidth: element.scrollWidth })),
    }));
    expect(layout.width, JSON.stringify({ elements: layout.overflow, content: layout.contentOverflow })).toBeLessThanOrEqual(layout.viewport);
    const header = page.getByRole('banner');
    await expect(header.getByRole('link', { name: 'Entrar no sistema', exact: true })).toBeVisible();
    if (width === 320) {
      for (const link of await page.getByRole('link').all()) {
        const box = await link.boundingBox();
        if (box && await link.isVisible() && !(await link.getAttribute('class'))?.includes('skip-link')) expect(box.height).toBeGreaterThanOrEqual(44);
      }
    }
    await page.screenshot({ path: info.outputPath(`home-${width}.png`), fullPage: true });
  });
}

test('home keyboard access, scope disclosure and automated WCAG checks', async ({ page }) => {
  await page.route('**/api/sessions/current', route => route.fulfill({ status: 503, json: { error: 'unavailable' } }));
  await page.goto('/');
  await page.keyboard.press('Tab');
  await expect(page.getByRole('link', { name: 'Pular para o conteúdo' })).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.getByRole('main')).toBeFocused();
  await expect(page.getByRole('heading', { name: 'Um projeto em evolução, com limites explícitos.' })).toBeVisible();
  await expect(page.getByText('Prévia ilustrativa do caixa, com dados fictícios. Não representa uma venda real.')).toBeVisible();
  const result = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze();
  expect(result.violations.map(item => ({ id: item.id, targets: item.nodes.map(node => node.target) }))).toEqual([]);
});
