import { App, applyDocumentTheme, applyHostFonts, applyHostStyleVariables } from "@modelcontextprotocol/ext-apps";
import "./mcp-app.css";

type PriceTransaction = {
  transaction_id?: string;
  id?: string;
  description?: string;
  category?: string;
  type?: string;
  area?: number;
  price?: number;
  price_per_square_meter?: number;
  period_identifier?: string;
  city?: string;
  postal?: string;
  postal_code_code?: string;
  municipality_name_fi?: string;
  confidence?: string;
};
type ListingRow = {
  canonical_id: string;
  source: string;
  kind: string;
  title: string;
  address?: string;
  city?: string;
  postal?: string;
  price?: number;
  area?: number;
  room_layout?: string;
  url?: string;
  web_url: string;
  thumbnail_url?: string;
  transactions: PriceTransaction[];
};
type SearchResult = {
  mode: "address" | "search";
  web_url: string;
  rows: ListingRow[];
  transactions: PriceTransaction[];
  total: number;
  page: number;
  page_size: number;
  summary: string;
};

const root = document.getElementById("app");
const app = new App({ name: "Koditon Listings", version: "0.1.0" }, {}, { autoResize: true });
let result: SearchResult | undefined;

app.ontoolresult = (params) => {
  if (params.isError) {
    renderError("Listing search failed.", contentText(params.content));
    return;
  }
  result = normalizeResult(params.structuredContent);
  render();
};
app.ontoolinput = () => renderLoading();
app.onhostcontextchanged = (ctx) => applyHostContext(ctx);
app.connect().then(() => {
  applyHostContext(app.getHostContext());
  renderLoading();
}).catch((error) => renderError("Unable to connect to MCP host.", error instanceof Error ? error.message : String(error)));

function render(): void {
  if (!root || !result) return;
  const rows = result.rows ?? [];
  root.innerHTML = `
    <section class="shell">
      <header class="topbar">
        <div>
          <p class="eyebrow">${result.mode === "address" ? "Address lookup" : "Listing search"}</p>
          <h1>Listings</h1>
          <p class="summary">${escapeHTML(result.summary)}</p>
        </div>
        <button class="secondary" data-open="${escapeAttr(result.web_url)}">Open web app</button>
      </header>
      <div class="metrics">
        <div><span>Rows</span><strong>${formatNumber(result.total)}</strong></div>
        <div><span>Linked</span><strong>${formatNumber(rows.filter((row) => row.transactions.length > 0).length)}</strong></div>
        <div><span>Sales</span><strong>${formatNumber(result.transactions.length)}</strong></div>
      </div>
      ${rows.length > 0 ? `<div class="list">${rows.map(renderRow).join("")}</div>` : `<div class="empty">No listings matched this query.</div>`}
      ${result.transactions.length > 0 ? `<section class="sales"><h2>Actual sale prices</h2>${result.transactions.slice(0, 8).map(renderTransaction).join("")}</section>` : ""}
    </section>
  `;
  root.querySelectorAll<HTMLButtonElement>("[data-open]").forEach((button) => {
    button.addEventListener("click", () => openURL(button.dataset.open));
  });
  root.querySelectorAll<HTMLButtonElement>("[data-ask]").forEach((button) => {
    button.addEventListener("click", () => askAbout(button.dataset.ask ?? ""));
  });
}
function renderRow(row: ListingRow): string {
  const location = [row.address, row.city, row.postal].filter(Boolean).join(" / ");
  const facts = [formatEUR(row.price), formatArea(row.area), row.room_layout].filter(Boolean).join(" · ");
  const thumb = row.thumbnail_url ? `<img src="${escapeAttr(row.thumbnail_url)}" alt="">` : `<span>${escapeHTML(sourceLabel(row.source).slice(0, 2))}</span>`;
  return `
    <article class="row">
      <div class="thumb thumb--${escapeAttr(row.source)}">${thumb}</div>
      <div class="row-main">
        <div class="badges">
          <span>${escapeHTML(sourceLabel(row.source))}</span>
          <span>${escapeHTML(row.kind)}</span>
          ${row.transactions.length > 0 ? `<strong>${row.transactions.length} sale link${row.transactions.length === 1 ? "" : "s"}</strong>` : ""}
        </div>
        <h2>${escapeHTML(row.title)}</h2>
        <p>${escapeHTML(location || row.canonical_id)}</p>
        <div class="facts">${escapeHTML(facts || "No key facts available")}</div>
        ${row.transactions.length > 0 ? `<div class="inline-sales">${row.transactions.slice(0, 2).map(renderTransaction).join("")}</div>` : ""}
      </div>
      <div class="actions">
        <button class="icon-btn" title="Ask about this listing" data-ask="${escapeAttr(row.canonical_id)}">Ask</button>
        <button class="icon-btn" title="Open in Koditon" data-open="${escapeAttr(row.web_url)}">Open</button>
        ${row.url ? `<button class="icon-btn" title="Open source listing" data-open="${escapeAttr(row.url)}">Source</button>` : ""}
      </div>
    </article>
  `;
}
function renderTransaction(transaction: PriceTransaction): string {
  const location = [transaction.postal ?? transaction.postal_code_code, transaction.city ?? transaction.municipality_name_fi].filter(Boolean).join(" / ");
  const details = [transaction.category, transaction.type, formatArea(transaction.area), formatPricePerM2(transaction.price_per_square_meter), transaction.period_identifier].filter(Boolean).join(" · ");
  return `
    <div class="sale-row">
      <div>
        <strong>${escapeHTML(transaction.description || transaction.transaction_id || transaction.id || "Sale")}</strong>
        <span>${escapeHTML(location || details || "prices")}</span>
      </div>
      <div>
        <strong>${escapeHTML(formatEUR(transaction.price) || "n/a")}</strong>
        <span>${escapeHTML(details)}</span>
      </div>
    </div>
  `;
}
function renderLoading(): void {
  if (!root) return;
  root.innerHTML = `<section class="shell"><div class="loading">Loading listing widget</div></section>`;
}
function renderError(title: string, detail: string): void {
  if (!root) return;
  root.innerHTML = `<section class="shell"><div class="error"><strong>${escapeHTML(title)}</strong><span>${escapeHTML(detail)}</span></div></section>`;
}
function askAbout(canonicalID: string): void {
  const row = result?.rows.find((item) => item.canonical_id === canonicalID);
  if (!row) return;
  const prompt = `Tell me more about ${row.title} (${row.canonical_id}). Compare asking price, area, source listing facts, and any linked actual sale prices.`;
  void app.updateModelContext({ structuredContent: { selected_listing: row } }).catch(() => {});
  void app.sendMessage({ role: "user", content: [{ type: "text", text: prompt }] });
}
function openURL(url: string | undefined): void {
  if (!url) return;
  void app.openLink({ url });
}
function applyHostContext(ctx: ReturnType<typeof app.getHostContext>): void {
  if (!ctx) return;
  if (ctx.theme) applyDocumentTheme(ctx.theme);
  if (ctx.styles?.variables) applyHostStyleVariables(ctx.styles.variables);
  if (ctx.styles?.css?.fonts) applyHostFonts(ctx.styles.css.fonts);
}
function normalizeResult(value: unknown): SearchResult {
  if (typeof value === "object" && value != null && "rows" in value) {
    return value as SearchResult;
  }
  return { mode: "search", web_url: "", rows: [], transactions: [], total: 0, page: 1, page_size: 25, summary: "No structured result was returned." };
}
function contentText(content: unknown): string {
  if (!Array.isArray(content)) return "";
  return content.map((item) => typeof item === "object" && item != null && "text" in item ? String(item.text) : "").filter(Boolean).join("\n");
}
function sourceLabel(source: string): string {
  if (source === "shortcut") return "Shortcut";
  if (source === "frontdoor") return "Frontdoor";
  return source || "Source";
}
function formatEUR(value?: number): string {
  if (value == null) return "";
  return `${new Intl.NumberFormat("fi-FI").format(value)} EUR`;
}
function formatArea(value?: number): string {
  if (value == null) return "";
  return `${new Intl.NumberFormat("fi-FI", { maximumFractionDigits: 1 }).format(value)} m2`;
}
function formatPricePerM2(value?: number): string {
  if (value == null) return "";
  return `${new Intl.NumberFormat("fi-FI").format(value)} EUR/m2`;
}
function formatNumber(value: number): string {
  return new Intl.NumberFormat("fi-FI").format(value);
}
function escapeHTML(value: string): string {
  return value.replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;", "'": "&#39;" })[char] ?? char);
}
function escapeAttr(value: string | undefined): string {
  return escapeHTML(value ?? "");
}
