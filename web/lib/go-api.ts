import "server-only";

const BASE_URL = process.env.API_BASE_URL!;
const INTERNAL_KEY = process.env.INTERNAL_API_KEY!;

export async function goFetch(
  path: string,
  userEmail: string,
  options: RequestInit = {}
): Promise<Response> {
  return fetch(`${BASE_URL}${path}`, {
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
