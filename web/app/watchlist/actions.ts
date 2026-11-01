"use server";

import { revalidatePath } from "next/cache";
import { auth } from "@/auth";
import { goFetch } from "@/lib/go-api";

async function getEmail(): Promise<string> {
  const session = await auth();
  if (!session?.user?.email) throw new Error("not authenticated");
  return session.user.email;
}

export async function addToWatchlist(ticker: string, name: string, status: string) {
  const email = await getEmail();
  const res = await goFetch("/v1/watchlist", email, {
    method: "POST",
    body: JSON.stringify({ ticker, name, status }),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || "failed to add to watchlist");
  }
  revalidatePath("/watchlist");
}

export async function removeFromWatchlist(ticker: string) {
  const email = await getEmail();
  const res = await goFetch(`/v1/watchlist/${ticker}`, email, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error("failed to remove from watchlist");
  revalidatePath("/watchlist");
}
