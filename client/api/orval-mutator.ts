const apiBaseUrl = (
  process.env.EXPO_PUBLIC_API_URL ?? "http://localhost:8080"
).replace(/\/+$/, "");

const resolveUrl = (url: string) => {
  if (!apiBaseUrl) {
    return url;
  }
  try {
    const parsed = new URL(url);
    return `${apiBaseUrl}${parsed.pathname}${parsed.search}${parsed.hash}`;
  } catch {
    return url.startsWith("/") ? `${apiBaseUrl}${url}` : `${apiBaseUrl}/${url}`;
  }
};

type HeadersProvider = () => Promise<Record<string, string>>;
let headersProvider: HeadersProvider | null = null;

export const setHeadersProvider = (provider: HeadersProvider | null) => {
  headersProvider = provider;
};

const tryParseJson = (text: string): unknown => {
  try {
    return JSON.parse(text);
  } catch {
    return { detail: text, status: 0 };
  }
};

export const authFetch = async <T>(
  url: string,
  options: RequestInit,
): Promise<T> => {
  const headers = new Headers(options?.headers);
  if (headersProvider) {
    const providedHeaders = await headersProvider();
    for (const [name, value] of Object.entries(providedHeaders)) {
      headers.set(name, value);
    }
  }
  const resolvedUrl = resolveUrl(url);
  const response = await fetch(resolvedUrl, { ...options, headers });
  const body = [204, 205, 304].includes(response.status)
    ? null
    : await response.text();
  const data = body ? tryParseJson(body) : {};
  return { data, status: response.status, headers: response.headers } as T;
};
