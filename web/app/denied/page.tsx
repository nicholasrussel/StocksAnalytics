import Link from "next/link";

export default function DeniedPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-zinc-50 dark:bg-zinc-950">
      <div className="w-full max-w-sm px-8 py-10 text-center">
        <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-red-100 dark:bg-red-900/20 mb-4">
          <svg className="w-6 h-6 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <h1 className="text-xl font-semibold text-zinc-900 dark:text-zinc-50 mb-2">
          Akses Tidak Diizinkan
        </h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400 mb-6">
          Email kamu tidak terdaftar di aplikasi ini. Hubungi admin untuk mendapatkan akses.
        </p>
        <Link
          href="/login"
          className="text-sm font-medium text-zinc-700 dark:text-zinc-300 hover:underline"
        >
          Kembali ke halaman login
        </Link>
      </div>
    </div>
  );
}
