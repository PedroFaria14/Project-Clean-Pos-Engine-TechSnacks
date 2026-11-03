import { spawn } from 'node:child_process';
import { randomBytes } from 'node:crypto';
import { mkdir, readFile, stat } from 'node:fs/promises';
import { createWriteStream } from 'node:fs';
import { createServer as createHTTPServer, request } from 'node:http';
import { createServer as createTCPServer } from 'node:net';
import { dirname, extname, join, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import pg from 'pg';

const frontend = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const backend = resolve(frontend, '../backend');
const binary = name => process.platform === 'win32' ? `${name}.exe` : name;
const suffix = randomBytes(8).toString('hex');
const runtime = join(frontend, 'test-results', `runtime-${suffix}`);
const schema = `test_e2e_${suffix}`;
const supplied = new URL(process.env.TECHSNACKS_E2E_DATABASE_URL || 'invalid:');
if (!['postgres:', 'postgresql:'].includes(supplied.protocol) || !['127.0.0.1', 'localhost', '[::1]'].includes(supplied.hostname) || !/^techsnacks_test_[a-z0-9_]+$/.test(decodeURIComponent(supplied.pathname.slice(1)))) {
  throw new Error('Set TECHSNACKS_E2E_DATABASE_URL to an explicitly disposable local techsnacks_test_* database.');
}
const admin = new pg.Client({ connectionString: supplied.href, connectionTimeoutMillis: 5000, query_timeout: 15000 });
let connected = false;
let created = false;
let api;
let proxy;
let logs;
let startupError;
let apiClosed;

async function run(command, args, options = {}) {
  const child = spawn(command, args, { stdio: 'inherit', ...options });
  return new Promise((resolveRun, reject) => { child.once('error', reject); child.once('exit', code => code === 0 ? resolveRun() : reject(new Error(`${command} failed with exit ${code}`))); });
}
async function freePort() {
  const socket = createTCPServer();
  await new Promise((resolvePort, reject) => { socket.once('error', reject); socket.listen(0, '127.0.0.1', resolvePort); });
  const port = socket.address().port;
  await new Promise(resolveClose => socket.close(resolveClose));
  return port;
}
async function waitFor(url) {
  for (let i = 0; i < 150; i++) {
    if (startupError) throw new Error('API process could not start.');
    if (api?.exitCode !== null) throw new Error('API exited during E2E startup; inspect its redacted runtime log.');
    try { if ((await fetch(url, { signal: AbortSignal.timeout(1000) })).ok) return; } catch { /* bounded startup polling */ }
    await new Promise(resolveWait => setTimeout(resolveWait, 200));
  }
  throw new Error('E2E readiness deadline exceeded');
}

try {
  await mkdir(runtime, { recursive: true });
  await admin.connect(); connected = true;
  await admin.query(`CREATE SCHEMA "${schema}"`); created = true;
  const apiPort = await freePort();
  const webPort = await freePort();
  const baseURL = `http://127.0.0.1:${webPort}`;
  const env = { ...process.env, APP_ENV: 'development', APP_ALLOWED_ORIGINS: baseURL,
    DATABASE_URL: '', Data_Base: '',
    DB_HOST: supplied.hostname.replace(/^\[|\]$/g, ''), DB_PORT: supplied.port || '5432', DB_USER: decodeURIComponent(supplied.username), DB_PASSWORD: decodeURIComponent(supplied.password),
    DB_PASSWORD_FILE: '', DB_SSLMODE: supplied.searchParams.get('sslmode') || 'disable', DB_NAME: decodeURIComponent(supplied.pathname.slice(1)), DB_SEARCH_PATH: schema, HTTP_ADDR: '', PORT: String(apiPort) };
  for (const name of ['api', 'migrate', 'create_user', 'healthcheck']) await run('go', ['build', '-o', join(runtime, binary(name)), `./cmd/${name}`], { cwd: backend });
  await run(join(runtime, binary('migrate')), [], { env });
  const password = randomBytes(20).toString('hex');
  for (const [username, role] of [['e2e.admin', 'admin'], ['e2e.operator', 'operator'], ['e2e.other', 'operator']]) {
    await run(join(runtime, binary('create_user')), ['--username', username, '--role', role], { env: { ...env, TECHSNACKS_USER_PASSWORD: password } });
  }
  logs = createWriteStream(join(runtime, 'api.log'));
  api = spawn(join(runtime, binary('api')), [], { env, stdio: ['ignore', 'pipe', 'pipe'] });
  apiClosed = new Promise(resolveClose => api.once('close', resolveClose));
  api.once('error', error => { startupError = error; });
  api.stdout.pipe(logs, { end: false }); api.stderr.pipe(logs, { end: false });
  await waitFor(`http://127.0.0.1:${apiPort}/health/ready`);
  await run(join(runtime, binary('healthcheck')), [], { env });
  const dist = join(frontend, 'dist');
  const mime = { '.html': 'text/html', '.js': 'application/javascript', '.css': 'text/css', '.svg': 'image/svg+xml', '.png': 'image/png', '.ico': 'image/x-icon' };
  proxy = createHTTPServer(async (req, res) => {
    const path = new URL(req.url, baseURL).pathname;
    if (path.startsWith('/api/')) {
      const upstream = request({ host: '127.0.0.1', port: apiPort, path: req.url.slice(4), method: req.method,
        headers: { ...req.headers, host: `127.0.0.1:${apiPort}` } }, response => { res.writeHead(response.statusCode, response.headers); response.pipe(res); });
      upstream.on('error', () => { if (!res.headersSent) res.writeHead(502); res.end(); });
      req.on('aborted', () => upstream.destroy());
      res.on('close', () => { if (!res.writableEnded) upstream.destroy(); });
      req.pipe(upstream);
      return;
    }
    try {
      if (req.method !== 'GET' && req.method !== 'HEAD') { res.writeHead(405); res.end(); return; }
      let file = resolve(dist, '.' + decodeURIComponent(path));
      if (file !== dist && !file.startsWith(dist + sep)) { res.writeHead(404); res.end(); return; }
      try { if (!(await stat(file)).isFile()) throw new Error(); } catch {
        if (path.startsWith('/assets/')) { res.writeHead(404); res.end(); return; }
        file = join(dist, 'index.html');
      }
      const data = await readFile(file);
      res.writeHead(200, { 'Content-Type': mime[extname(file)] || 'application/octet-stream', 'Cache-Control': 'no-cache',
        'Content-Security-Policy': "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'" });
      res.end(req.method === 'HEAD' ? undefined : data);
    } catch { res.writeHead(500); res.end(); }
  });
  await new Promise((resolveListen, reject) => { proxy.once('error', reject); proxy.listen(webPort, '127.0.0.1', resolveListen); });
  await run(process.execPath, [join(frontend, 'node_modules/@playwright/test/cli.js'), 'test', '--config=playwright.config.ts'], {
    cwd: frontend, env: { ...process.env, E2E_BASE_URL: baseURL, E2E_DATABASE_URL: supplied.href, E2E_SCHEMA: schema, E2E_PASSWORD: password },
  });
} finally {
  if (proxy) { proxy.closeAllConnections(); await new Promise(resolveClose => proxy.close(resolveClose)); }
  if (api?.pid && api.exitCode === null) {
    api.kill('SIGTERM');
    const timer = setTimeout(() => { if (api.exitCode === null) api.kill('SIGKILL'); }, 10_000);
    await apiClosed; clearTimeout(timer);
  }
  if (apiClosed) await apiClosed;
  logs?.end();
  if (created) await admin.query(`DROP SCHEMA "${schema}" CASCADE`);
  if (connected) await admin.end();
}
