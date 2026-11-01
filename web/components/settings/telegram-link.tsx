"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { generateTelegramLink, checkTelegramLinked } from "@/app/settings/actions";

interface Props {
  alreadyLinked: boolean;
}

export function TelegramLink({ alreadyLinked }: Props) {
  const router = useRouter();
  const [link, setLink] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState("");

  if (alreadyLinked) {
    return (
      <div className="flex items-center gap-2">
        <span className="inline-flex items-center gap-1.5 text-sm text-green-600 dark:text-green-400">
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
          Telegram sudah terhubung
        </span>
      </div>
    );
  }

  async function handleGenerate() {
    setLoading(true);
    setError("");
    try {
      const result = await generateTelegramLink();
      if (result.already_linked) {
        router.refresh();
        return;
      }
      if (result.link) setLink(result.link);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Gagal generate link");
    } finally {
      setLoading(false);
    }
  }

  async function handleCheckStatus() {
    setChecking(true);
    setError("");
    try {
      const linked = await checkTelegramLinked();
      if (linked) {
        router.refresh();
        return;
      }
      setError("Telegram belum terhubung. Pastikan kamu sudah klik link dan bot merespons dengan pesan konfirmasi.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Gagal cek status");
    } finally {
      setChecking(false);
    }
  }

  return (
    <div className="space-y-3">
      {!link ? (
        <Button variant="secondary" size="sm" onClick={handleGenerate} disabled={loading}>
          {loading ? "Memuat..." : "Hubungkan Telegram"}
        </Button>
      ) : (
        <div className="space-y-3">
          <div className="space-y-2">
            <p className="text-xs text-zinc-500 dark:text-zinc-400">
              <strong>Langkah 1:</strong> Klik tombol di bawah untuk membuka bot di Telegram
            </p>
            <a
              href={link}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-[#229ED9] hover:bg-[#1a8bbf] text-white text-sm font-medium transition-colors"
            >
              <svg className="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.373 0 0 5.373 0 12s5.373 12 12 12 12-5.373 12-12S18.627 0 12 0zm5.894 8.221l-1.97 9.28c-.145.658-.537.818-1.084.508l-3-2.21-1.447 1.394c-.16.16-.295.295-.605.295l.213-3.053 5.56-5.023c.242-.213-.054-.333-.373-.12L7.527 14.47l-2.9-.91c-.63-.197-.643-.63.136-.93l11.267-4.347c.527-.19.988.13.864.938z"/>
              </svg>
              Buka di Telegram
            </a>
          </div>
          <div className="space-y-1">
            <p className="text-xs text-zinc-500 dark:text-zinc-400">
              <strong>Langkah 2:</strong> Setelah bot Telegram merespons "Berhasil terhubung", klik tombol di bawah
            </p>
            <Button variant="secondary" size="sm" onClick={handleCheckStatus} disabled={checking}>
              {checking ? "Mengecek..." : "Cek Status"}
            </Button>
          </div>
          <p className="text-xs text-zinc-400 dark:text-zinc-500">
            Link berlaku 24 jam. Bot harus aktif (worker berjalan) untuk memproses koneksi.
          </p>
        </div>
      )}
      {error && <p className="text-xs text-red-500">{error}</p>}
    </div>
  );
}
