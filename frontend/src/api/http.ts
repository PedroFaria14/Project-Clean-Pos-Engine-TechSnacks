let csrfToken = "";

export function setCsrfToken(value: string): void { csrfToken = value; }

// Authentication material is held in memory only; the browser owns the HttpOnly cookie.
export async function apiRequest(path: string, options: RequestInit = {}): Promise<Response> {
  const headers = new Headers(options.headers);
  const method = (options.method ?? "GET").toUpperCase();
  const requestSession = csrfToken;
  if (method !== "GET" && method !== "HEAD" && csrfToken) headers.set("X-CSRF-Token", csrfToken);
  const response = await fetch(`/api${path}`, { ...options, headers, credentials: "same-origin" });
  if (response.status === 401 && path !== "/sessions" && requestSession === csrfToken && !options.signal?.aborted) {
    csrfToken = "";
    window.dispatchEvent(new Event("techsnacks:session-expired"));
  }
  return response;
}
