import "server-only";

const INTERNAL_KEY = process.env.INTERNAL_API_KEY!;

function goApiUrl(path: string): string {
  // Read per call: on Vercel Services this binding exists only at runtime, not during builds.
  const base = (process.env.API_BASE_URL ?? "").replace(/\/+$/, "");
  return `${base}${path}`;
}

export async function goFetch(
  path: string,
  userEmail: string,
  options: RequestInit = {}
): Promise<Response> {
  return fetch(goApiUrl(path), {
    ...options,
    headers: {
      "Content-Type": "application/json",
      "X-Internal-Key": INTERNAL_KEY,
      "X-User-Email": userEmail,
      ...(options.headers as Record<string, string>),
    },
    cache: "no-store",
  });
}
