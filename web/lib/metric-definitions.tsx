import React from "react";

interface Definition {
  label: string;
  definition: string;
  interpretation: string;
}

const DEFINITIONS: Record<string, Definition> = {
  // ── Profitabilitas ──────────────────────────────────────────────────────────
  revenue: {
    label: "Pendapatan (Revenue)",
    definition: "Total uang yang diterima perusahaan dari penjualan produk atau jasa sebelum dikurangi biaya apapun.",
    interpretation: "Pertumbuhan konsisten = bisnis berkembang. Penurunan > 10% YoY perlu dicermati.",
  },
  revenue_growth: {
    label: "Pertumbuhan Pendapatan YoY",
    definition: "Persentase kenaikan atau penurunan pendapatan dibanding periode yang sama tahun lalu.",
    interpretation: "> 10% bagus, > 20% sangat baik, < 0% perlu perhatian.",
  },
  gross_margin: {
    label: "Gross Margin",
    definition: "Sisa pendapatan setelah dikurangi biaya langsung produksi (HPP). Rumus: (Pendapatan − HPP) / Pendapatan.",
    interpretation: "Semakin tinggi = produk punya daya saing harga yang kuat. > 40% umumnya baik untuk tech/pharma.",
  },
  operating_margin: {
    label: "Operating Margin",
    definition: "Laba operasional dibagi pendapatan. Mencerminkan efisiensi inti bisnis sebelum bunga & pajak.",
    interpretation: "> 15% baik untuk kebanyakan industri. Margin negatif = operasional merugi.",
  },
  ebitda_margin: {
    label: "EBITDA Margin",
    definition: "EBITDA (laba sebelum bunga, pajak, depresiasi, amortisasi) dibagi pendapatan. Ukuran profitabilitas operasional tanpa efek akuntansi.",
    interpretation: "Digunakan untuk membandingkan perusahaan lintas industri. > 20% umumnya sehat.",
  },
  net_margin: {
    label: "Net Margin",
    definition: "Laba bersih yang tersisa untuk pemegang saham dari setiap rupiah/dolar pendapatan. Rumus: Laba Bersih / Pendapatan.",
    interpretation: "< 0% = merugi, 5–10% = tipis, > 20% = sangat baik (biasanya tech atau bank besar).",
  },
  eps: {
    label: "EPS (Earnings Per Share)",
    definition: "Laba bersih perusahaan dibagi jumlah saham beredar. Menunjukkan berapa laba yang dihasilkan per lembar saham.",
    interpretation: "EPS positif = perusahaan untung. Pertumbuhan EPS konsisten = tanda kesehatan bisnis yang baik.",
  },
  eps_growth: {
    label: "Pertumbuhan EPS YoY",
    definition: "Persentase kenaikan EPS dibanding tahun sebelumnya.",
    interpretation: "> 10% baik. Pertumbuhan EPS lebih cepat dari revenue = efisiensi meningkat.",
  },
  roe: {
    label: "ROE (Return on Equity)",
    definition: "Laba bersih dibagi ekuitas pemegang saham. Mengukur seberapa efisien perusahaan menggunakan modal yang ditanamkan investor.",
    interpretation: "> 15% baik, > 25% sangat baik. < 5% atau negatif = modal tidak produktif.",
  },
  roa: {
    label: "ROA (Return on Assets)",
    definition: "Laba bersih dibagi total aset. Mengukur efisiensi penggunaan seluruh aset perusahaan.",
    interpretation: "> 5% baik untuk kebanyakan industri. Industri padat modal (manufaktur) wajar lebih rendah.",
  },
  roic: {
    label: "ROIC (Return on Invested Capital)",
    definition: "Laba operasional setelah pajak dibagi modal yang diinvestasikan (utang + ekuitas − kas). Ukuran terbaik efisiensi modal.",
    interpretation: "ROIC > WACC (biaya modal ~8–10%) = perusahaan menciptakan nilai. > 15% sangat baik.",
  },

  // ── Likuiditas ──────────────────────────────────────────────────────────────
  current_ratio: {
    label: "Current Ratio",
    definition: "Aset lancar (kas, piutang, persediaan) dibagi liabilitas lancar (utang jangka pendek). Mengukur kemampuan bayar utang jangka pendek.",
    interpretation: "< 1.0 = berbahaya (utang > aset lancar), 1.0–1.5 = cukup, > 1.5 = sehat.",
  },
  quick_ratio: {
    label: "Quick Ratio",
    definition: "Versi lebih ketat dari Current Ratio — persediaan (inventory) dikeluarkan karena sulit dicairkan cepat. Rumus: (Aset Lancar − Persediaan) / Liabilitas Lancar.",
    interpretation: "< 1.0 = perlu waspada, > 1.0 = likuiditas cepat aman.",
  },
  cash_position: {
    label: "Posisi Kas",
    definition: "Total kas dan setara kas yang dimiliki perusahaan saat ini.",
    interpretation: "Kas besar = fleksibilitas tinggi untuk investasi, akuisisi, atau bertahan saat krisis.",
  },

  // ── Leverage ────────────────────────────────────────────────────────────────
  debt_to_equity: {
    label: "Debt / Equity",
    definition: "Total utang dibagi ekuitas pemegang saham. Mengukur seberapa besar perusahaan dibiayai utang vs modal sendiri.",
    interpretation: "< 0.5 = konservatif, 0.5–1.5 = moderat, > 2.0 = agresif. Industri berbeda wajar berbeda (bank bisa > 10).",
  },
  net_debt: {
    label: "Net Debt",
    definition: "Total utang dikurangi kas yang dimiliki. Nilai negatif berarti perusahaan punya lebih banyak kas dari utang (net cash).",
    interpretation: "Net cash = sangat sehat finansial. Net debt tinggi = beban bunga besar.",
  },
  net_debt_to_ebitda: {
    label: "Net Debt / EBITDA",
    definition: "Berapa tahun yang dibutuhkan perusahaan untuk melunasi utang bersihnya jika seluruh EBITDA digunakan. Ukuran leverage paling umum.",
    interpretation: "< 0 = net cash, 0–2x = aman, 2–3x = perlu dipantau, > 4x = leverage tinggi.",
  },
  interest_coverage: {
    label: "Interest Coverage",
    definition: "Laba operasional (EBIT) dibagi beban bunga. Menunjukkan berapa kali perusahaan mampu membayar bunga dari laba operasionalnya.",
    interpretation: "< 1.5x = berbahaya (hampir tidak bisa bayar bunga), > 3x = aman, > 10x = sangat nyaman.",
  },

  // ── Arus Kas ────────────────────────────────────────────────────────────────
  operating_cash_flow: {
    label: "Operating Cash Flow (OCF)",
    definition: "Kas nyata yang dihasilkan dari operasional bisnis inti. Lebih susah dimanipulasi dibanding laba bersih.",
    interpretation: "OCF positif = bisnis menghasilkan kas nyata. OCF < Net Income = laba tidak berkualitas.",
  },
  capex: {
    label: "CapEx (Capital Expenditure)",
    definition: "Pengeluaran untuk membeli atau memperbarui aset tetap (mesin, gedung, infrastruktur). Diperlukan untuk pertumbuhan bisnis.",
    interpretation: "CapEx tinggi = bisnis padat modal (semiconductor, telco). CapEx rendah = bisnis asset-light (software).",
  },
  free_cash_flow: {
    label: "Free Cash Flow (FCF)",
    definition: "Kas bersih yang benar-benar bebas setelah semua pengeluaran operasional dan investasi. Rumus: OCF − CapEx.",
    interpretation: "FCF positif = perusahaan bisa bayar dividen, buyback, atau berekspansi tanpa hutang baru.",
  },
  fcf_margin: {
    label: "FCF Margin",
    definition: "Free Cash Flow dibagi pendapatan. Mengukur berapa persen dari setiap pendapatan yang benar-benar jadi kas bebas.",
    interpretation: "> 10% baik, > 20% sangat baik. Negatif = butuh modal dari luar terus-menerus.",
  },
  fcf_conversion: {
    label: "FCF Conversion",
    definition: "Free Cash Flow dibagi laba bersih. Mengukur seberapa besar laba yang tercatat benar-benar berubah menjadi kas nyata.",
    interpretation: "> 80% = kualitas laba tinggi. < 50% = laba banyak yang masih berupa piutang atau akrual, bukan kas.",
  },

  // ── Valuasi ─────────────────────────────────────────────────────────────────
  pe_ratio: {
    label: "P/E Ratio (Price/Earnings)",
    definition: "Harga saham dibagi EPS. Menunjukkan berapa kali lipat laba yang investor bayar untuk memiliki saham tersebut.",
    interpretation: "< 15x = murah, 15–25x = wajar, > 35x = mahal (kecuali growth stock). Bandingkan dengan industri sejenis.",
  },
  ps_ratio: {
    label: "P/S Ratio (Price/Sales)",
    definition: "Market cap dibagi total pendapatan. Berguna untuk perusahaan yang belum menghasilkan laba.",
    interpretation: "< 1x = sangat murah, 1–3x = wajar untuk perusahaan mapan, > 10x = ekspektasi pertumbuhan tinggi.",
  },
  pb_ratio: {
    label: "P/B Ratio (Price/Book)",
    definition: "Harga saham dibagi nilai buku per saham (total ekuitas / jumlah saham). Mengukur premium di atas nilai aset bersih.",
    interpretation: "< 1x = diperdagangkan di bawah nilai aset (bisa murah atau ada masalah), > 3x = investor ekspektasikan pertumbuhan tinggi.",
  },
  ev_ebitda: {
    label: "EV / EBITDA",
    definition: "Enterprise Value (market cap + utang − kas) dibagi EBITDA. Valuasi yang lebih komprehensif karena memperhitungkan struktur modal.",
    interpretation: "< 10x = relatif murah, 10–20x = wajar, > 25x = mahal. Lebih baik dari P/E untuk perbandingan lintas perusahaan.",
  },

  // ── Skor ────────────────────────────────────────────────────────────────────
  piotroski: {
    label: "Piotroski F-Score",
    definition: "Skor 0–9 yang mengukur kekuatan fundamental dari 9 kriteria: 4 profitabilitas, 3 leverage/likuiditas, 2 efisiensi operasional. Dikembangkan oleh akademisi Joseph Piotroski.",
    interpretation: "7–9 = fundamental kuat (Sehat), 4–6 = netral, 0–3 = lemah (banyak sinyal negatif).",
  },
};

export type MetricKey = keyof typeof DEFINITIONS;

export function getDefinition(key: MetricKey): Definition | undefined {
  return DEFINITIONS[key];
}

export function MetricLabel({ metricKey, children }: { metricKey: MetricKey; children?: React.ReactNode }) {
  const def = DEFINITIONS[metricKey];
  if (!def) return <>{children}</>;

  return (
    <span className="inline-flex items-center gap-1 group">
      <span>{children ?? def.label}</span>
      <span
        className="text-zinc-300 dark:text-zinc-600 group-hover:text-zinc-500 dark:group-hover:text-zinc-400 transition-colors cursor-help"
        title={`${def.definition}\n\nInterpretasi: ${def.interpretation}`}
      >
        ⓘ
      </span>
    </span>
  );
}
