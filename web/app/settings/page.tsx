import { auth } from "@/auth";
import { goFetch } from "@/lib/go-api";
import { Nav } from "@/components/layout/nav";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { TelegramLink } from "@/components/settings/telegram-link";

export const instant = false;

export default async function SettingsPage() {
  const session = await auth();
  const email = session?.user?.email ?? "";

  let telegramLinked = false;
  try {
    const res = await goFetch("/v1/me", email);
    if (res.ok) {
      const user = await res.json();
      telegramLinked = user.telegram_chat_id != null;
    }
  } catch {
    //
  }

  return (
    <>
      <Nav />
      <main className="min-h-screen bg-zinc-50 dark:bg-zinc-950 px-4 py-8">
        <div className="max-w-2xl mx-auto space-y-6">
          <div>
            <h1 className="text-2xl font-bold text-zinc-900 dark:text-zinc-50">Pengaturan</h1>
            <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-1">Kelola akun dan notifikasi</p>
          </div>

          {/* Account info */}
          <Card>
            <CardHeader>
              <p className="text-xs font-semibold text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">Akun</p>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-zinc-700 dark:text-zinc-300">{email}</p>
            </CardContent>
          </Card>

          {/* Telegram */}
          <Card>
            <CardHeader>
              <p className="text-xs font-semibold text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">
                Notifikasi Telegram
              </p>
            </CardHeader>
            <CardContent className="space-y-2">
              <p className="text-sm text-zinc-600 dark:text-zinc-400">
                Hubungkan akun Telegram untuk menerima alert berita penting dari watchlist kamu.
              </p>
              <TelegramLink alreadyLinked={telegramLinked} />
            </CardContent>
          </Card>
        </div>
      </main>
    </>
  );
}
