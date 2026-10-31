import pg from 'pg';

export function validateTestEnvironment() {
  const connection = new URL(process.env.E2E_DATABASE_URL || 'invalid:');
  const base = new URL(process.env.E2E_BASE_URL || 'invalid:');
  const local = host => ['localhost', '127.0.0.1', '[::1]', '::1'].includes(host);
  if (!['postgres:', 'postgresql:'].includes(connection.protocol) || !local(connection.hostname) ||
      !/^techsnacks_test_[a-z0-9_]+$/.test(decodeURIComponent(connection.pathname.slice(1))) ||
      !/^test_[a-z0-9_]+$/.test(process.env.E2E_SCHEMA || '') || base.protocol !== 'http:' || !local(base.hostname) || !process.env.E2E_PASSWORD) {
    throw new Error('E2E requires an explicit loopback test database, owned test_* schema, local base URL and generated test credentials.');
  }
  return { connectionString: connection.href, schema: process.env.E2E_SCHEMA };
}

export async function resetFixtures() {
  const { connectionString, schema } = validateTestEnvironment();
  const client = new pg.Client({ connectionString, connectionTimeoutMillis: 5000, query_timeout: 15000 });
  await client.connect();
  try {
    const tables = ['user_sessions', 'auth_login_limits', 'checkout_idempotency_records', 'order_item_models', 'order_models', 'products'];
    await client.query(`TRUNCATE ${tables.map(table => `"${schema}"."${table}"`).join(', ')} RESTART IDENTITY`);
    await client.query(`INSERT INTO "${schema}".products (id, name, price) VALUES ($1,$2,$3),($4,$5,$6)`, ['p1', 'Coxinha de frango', 650, 'p2', 'Pão de queijo', 350]);
  } finally { await client.end(); }
}
