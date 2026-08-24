/** Client-side CSV / JSON export helpers for list pages. */

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export function exportJSON(data: unknown, filename: string) {
  const blob = new Blob([JSON.stringify(data, null, 2)], {
    type: "application/json",
  });
  downloadBlob(blob, filename.endsWith(".json") ? filename : `${filename}.json`);
}

export function exportCSV(
  rows: Record<string, unknown>[],
  filename: string,
  columns?: string[],
) {
  if (!rows.length) {
    const empty = new Blob([""], { type: "text/csv" });
    downloadBlob(empty, filename.endsWith(".csv") ? filename : `${filename}.csv`);
    return;
  }
  const cols = columns || Object.keys(rows[0] || {});
  const escape = (v: unknown) => {
    if (v === null || v === undefined) return "";
    const s = typeof v === "object" ? JSON.stringify(v) : String(v);
    if (/[",\n]/.test(s)) return `"${s.replace(/"/g, '""')}"`;
    return s;
  };
  const lines = [
    cols.join(","),
    ...rows.map((row) => cols.map((c) => escape(row[c])).join(",")),
  ];
  const blob = new Blob([lines.join("\n")], { type: "text/csv;charset=utf-8" });
  downloadBlob(blob, filename.endsWith(".csv") ? filename : `${filename}.csv`);
}

export function stampFilename(prefix: string, ext: "csv" | "json") {
  const d = new Date().toISOString().slice(0, 10);
  return `${prefix}_${d}.${ext}`;
}
