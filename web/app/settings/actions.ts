"use server";

import { auth } from "@/auth";
import { goFetch } from "@/lib/go-api";

async function getEmail() {
  const session = await auth();
  if (!session?.user?.email) throw new Error("not authenticated");
  return session.user.email;
}

export async function generateTelegramLink(): Promise<{ link?: string; already_linked?: boolean }> {
  const email = await getEmail();
  const res = await goFetch("/v1/telegram/link-token", email, { method: "POST" });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error ?? `HTTP ${res.status}`);
  }
  return res.json();
}

export async function checkTelegramLinked(): Promise<boolean> {
  const email = await getEmail();
  const res = await goFetch("/v1/me", email);
  if (!res.ok) return false;
  const user = await res.json();
  return user.telegram_chat_id != null;
}
