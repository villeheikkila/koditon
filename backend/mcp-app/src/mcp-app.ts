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
  id?: string;
  canonical_id: string;
  source: string;
  kind: string;
  title: string;
  address?: string;
  city?: string;
  postal?: string;
  latitude?: number;
  longitude?: number;
  price?: number;
  area?: number;
  room_layout?: string;
  url?: string;
  external_url_available?: boolean;
  web_url: string;
  thumbnail_url?: string;
  transactions: PriceTransaction[];
  facts?: Record<string, unknown>;
  costs?: Record<string, unknown>;
  location?: Record<string, unknown>;
  market?: Record<string, unknown>;
  data_quality?: { completeness?: number; missing_fields?: string[]; warnings?: string[] };
};
type SearchResult = {
  view?: string;
  mode: "address" | "search" | "transaction";
  web_url: string;
  rows: ListingRow[];
  transactions: PriceTransaction[];
  total: number;
  page: number;
  page_size: number;
  summary: string;
};
type DetailField = {
  label?: string;
  value?: string;
};
type ListingDetail = {
  view?: string;
  title?: string;
  overview?: DetailField[];
  reports?: Array<{ title?: string; items?: string[] }>;
  data_quality?: { completeness?: number; missing_fields?: string[]; warnings?: string[] };
  canonical?: Record<string, unknown>;
  canonical_extra?: DetailField[];
  source_specific?: DetailField[];
  related?: DetailField[];
  normalized?: Record<string, unknown>;
  raw?: Record<string, unknown>;
};
type DetailState = {
  canonicalID: string;
  loading: boolean;
  error?: string;
  detail?: ListingDetail;
};
type ComparisonResult = {
  view: "comparison";
  summary: string;
  rows: ListingRow[];
  ranking: Array<{ id?: string; rank?: number; score?: number; reasons?: string[] }>;
  tradeoffs: string[];
  missing_data_warnings: string[];
};
type MarketContextResult = {
  view: "market_context";
  summary: string;
  subject?: ListingRow;
  market?: Record<string, unknown>;
  data_quality?: { completeness?: number; missing_fields?: string[]; warnings?: string[] };
};
type MapPoint = {
  id: string;
  title: string;
  lng: number;
  lat: number;
  selected?: boolean;
};

const root = document.getElementById("app");
const app = new App({ name: "Koditon Listings", version: "0.1.0" }, {}, { autoResize: true });
let result: SearchResult | undefined;
let detailState: DetailState | undefined;
let comparison: ComparisonResult | undefined;
let marketContext: MarketContextResult | undefined;

app.ontoolresult = (params) => {
  if (params.isError) {
    renderError("Listing search failed.", contentText(params.content));
    return;
  }
  const view = structuredView(params.structuredContent);
  if (view === "comparison") {
    comparison = normalizeComparison(params.structuredContent);
    result = undefined;
    detailState = undefined;
    marketContext = undefined;
    renderComparison();
    return;
  }
  if (view === "market_context") {
    marketContext = normalizeMarketContext(params.structuredContent);
    result = undefined;
    detailState = undefined;
    comparison = undefined;
    renderMarketContext();
    return;
  }
  if (view === "detail" || !isSearchResult(params.structuredContent)) {
    const detail = normalizeDetail(params.structuredContent, params.content);
    if (detail.canonical || detail.normalized) {
      const canonicalID = stringValue(detail.canonical?.canonical_id) || "detail";
      result = undefined;
      detailState = { canonicalID, loading: false, detail };
      comparison = undefined;
      marketContext = undefined;
      renderStandaloneDetail();
      return;
    }
  }
  result = normalizeResult(params.structuredContent);
  detailState = undefined;
  comparison = undefined;
  marketContext = undefined;
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
      ${renderMap(rows.flatMap((row) => rowMapPoint(row, detailState?.canonicalID === row.canonical_id) ?? []), "Map")}
      <div class="workspace">
        ${rows.length > 0 ? `<div class="list">${rows.map(renderRow).join("")}</div>` : `<div class="empty">No listings matched this query.</div>`}
        ${renderDetailPanel()}
      </div>
      ${result.transactions.length > 0 ? `<section class="sales"><h2>Actual sale prices</h2>${result.transactions.slice(0, 8).map(renderTransaction).join("")}</section>` : ""}
    </section>
  `;
  root.querySelectorAll<HTMLButtonElement>("[data-open]").forEach((button) => {
    button.addEventListener("click", () => openURL(button.dataset.open));
  });
  root.querySelectorAll<HTMLButtonElement>("[data-ask]").forEach((button) => {
    button.addEventListener("click", () => askAbout(button.dataset.ask ?? ""));
  });
  root.querySelectorAll<HTMLButtonElement>("[data-detail]").forEach((button) => {
    button.addEventListener("click", () => showDetail(button.dataset.detail ?? ""));
  });
  root.querySelectorAll<HTMLButtonElement>("[data-close-detail]").forEach((button) => {
    button.addEventListener("click", () => {
      detailState = undefined;
      render();
    });
  });
}
function renderComparison(): void {
  if (!root || !comparison) return;
  root.innerHTML = `
    <section class="shell">
      <header class="topbar">
        <div>
          <p class="eyebrow">Comparison</p>
          <h1>Properties</h1>
          <p class="summary">${escapeHTML(comparison.summary)}</p>
        </div>
      </header>
      <div class="metrics">
        <div><span>Rows</span><strong>${formatNumber(comparison.rows.length)}</strong></div>
        <div><span>Ranked</span><strong>${formatNumber(comparison.ranking.length)}</strong></div>
        <div><span>Warnings</span><strong>${formatNumber(comparison.missing_data_warnings.length)}</strong></div>
      </div>
      <div class="workspace workspace--single">
        ${renderMap(comparison.rows.flatMap((row) => rowMapPoint(row) ?? []), "Map")}
        <div class="list">${comparison.rows.map(renderComparisonRow).join("")}</div>
      </div>
      ${comparison.tradeoffs.length > 0 ? `<section class="sales"><h2>Tradeoffs</h2>${comparison.tradeoffs.map((item) => `<div class="sale-row"><div><strong>${escapeHTML(item)}</strong></div></div>`).join("")}</section>` : ""}
    </section>
  `;
  root.querySelectorAll<HTMLButtonElement>("[data-open]").forEach((button) => {
    button.addEventListener("click", () => openURL(button.dataset.open));
  });
}
function renderComparisonRow(row: ListingRow): string {
  const rank = comparison?.ranking.find((item) => item.id === (row.id || row.canonical_id));
  const reasons = rank?.reasons?.join(" · ") ?? "";
  return renderRow({ ...row, transactions: row.transactions ?? [] }).replace("</article>", `${reasons ? `<div class="comparison-note">#${rank?.rank ?? ""} ${escapeHTML(reasons)}</div>` : ""}</article>`);
}
function renderMarketContext(): void {
  if (!root || !marketContext) return;
  const sales = Array.isArray(marketContext.market?.comparable_sales) ? marketContext.market.comparable_sales as PriceTransaction[] : [];
  root.innerHTML = `
    <section class="shell">
      <header class="topbar">
        <div>
          <p class="eyebrow">Market context</p>
          <h1>Comparable sales</h1>
          <p class="summary">${escapeHTML(marketContext.summary)}</p>
        </div>
      </header>
      <div class="metrics">
        <div><span>Comps</span><strong>${formatNumber(sales.length)}</strong></div>
        <div><span>Median EUR/m2</span><strong>${escapeHTML(formatPricePerM2(numberValue(marketContext.market?.median_price_per_m2)) || "n/a")}</strong></div>
        <div><span>Confidence</span><strong>${escapeHTML(stringValue(marketContext.market?.confidence) || "low")}</strong></div>
      </div>
      ${renderMap(marketContext.subject ? [rowMapPoint(marketContext.subject, true)].filter((point): point is MapPoint => Boolean(point)) : [], "Subject location")}
      ${marketContext.subject ? `<div class="list">${renderRow(marketContext.subject)}</div>` : ""}
      ${sales.length > 0 ? `<section class="sales"><h2>Comparable sales</h2>${sales.slice(0, 12).map(renderTransaction).join("")}</section>` : `<div class="empty">No comparable sales returned.</div>`}
    </section>
  `;
  root.querySelectorAll<HTMLButtonElement>("[data-open]").forEach((button) => {
    button.addEventListener("click", () => openURL(button.dataset.open));
  });
}
function renderRow(row: ListingRow): string {
  const location = [row.address, row.city, row.postal].filter(Boolean).join(" / ");
  const facts = [formatEUR(row.price), formatArea(row.area), row.room_layout].filter(Boolean).join(" · ");
  const thumb = row.thumbnail_url ? `<img src="${escapeAttr(row.thumbnail_url)}" alt="">` : `<span>${escapeHTML(sourceLabel(row.source).slice(0, 2))}</span>`;
  const selected = detailState?.canonicalID === row.canonical_id ? " row--selected" : "";
  return `
    <article class="row${selected}">
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
        <button class="icon-btn" title="Show details" data-detail="${escapeAttr(row.canonical_id)}">Detail</button>
        <button class="icon-btn" title="Ask about this listing" data-ask="${escapeAttr(row.canonical_id)}">Ask</button>
        <button class="icon-btn" title="Open in Koditon" data-open="${escapeAttr(row.web_url)}">Open</button>
        ${row.external_url_available && row.url ? `<button class="icon-btn" title="Open source listing" data-open="${escapeAttr(row.url)}">Source</button>` : ""}
      </div>
    </article>
  `;
}
function renderDetailPanel(): string {
  if (!detailState) return `<aside class="detail-panel detail-panel--empty"><strong>Detail</strong><span>Select a listing to inspect source facts, normalized fields, and related records.</span></aside>`;
  const row = result?.rows.find((item) => item.canonical_id === detailState?.canonicalID);
  if (detailState.loading) return `<aside class="detail-panel"><div class="detail-head"><div><span>Loading detail</span><strong>${escapeHTML(row?.title ?? detailState.canonicalID)}</strong></div><button data-close-detail>Close</button></div><div class="loading">Loading listing detail</div></aside>`;
  if (detailState.error) return `<aside class="detail-panel"><div class="detail-head"><div><span>Detail error</span><strong>${escapeHTML(row?.title ?? detailState.canonicalID)}</strong></div><button data-close-detail>Close</button></div><div class="error"><strong>Could not load detail</strong><span>${escapeHTML(detailState.error)}</span></div></aside>`;
  const detail = detailState.detail;
  if (!detail) return "";
  const canonical = detail.canonical ?? {};
  const normalized = detail.normalized ?? {};
  const title = stringValue(canonical.headline) || row?.title || detailState.canonicalID;
  return `
    <aside class="detail-panel">
      <div class="detail-head">
        <div>
          <span>${escapeHTML([stringValue(canonical.source), stringValue(canonical.kind)].filter(Boolean).join(" / ") || "Listing detail")}</span>
          <strong>${escapeHTML(title)}</strong>
        </div>
        <button data-close-detail>Close</button>
      </div>
      ${renderKeyValues("Overview", [
        ["Address", stringValue(canonical.address) || stringValue(normalized.street_address)],
        ["City", stringValue(canonical.city) || stringValue(normalized.city)],
        ["Postal", stringValue(canonical.postal) || stringValue(normalized.postal)],
        ["Coordinates", formatCoordinates(detailCoordinates(detail))],
        ["Price", formatEUR(numberValue(canonical.price) ?? numberValue(normalized.asking_price) ?? numberValue(normalized.debt_free_price))],
        ["Area", formatArea(numberValue(canonical.area) ?? numberValue(normalized.area_m2))],
        ["Rooms", stringValue(canonical.room_layout) || stringValue(normalized.room_layout)]
      ])}
      ${renderMap(detailMapPoints(detail), "Location")}
      ${renderFields("Source facts", detail.source_specific)}
      ${renderFields("Canonical", detail.canonical_extra)}
      ${renderFields("Related", detail.related)}
      ${renderKeyValues("Normalized", normalizedEntries(normalized))}
    </aside>
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
function renderStandaloneDetail(): void {
  if (!root || !detailState) return;
  root.innerHTML = `
    <section class="shell">
      <header class="topbar">
        <div>
          <p class="eyebrow">Listing detail</p>
          <h1>Detail</h1>
          <p class="summary">Canonical source facts and normalized listing fields.</p>
        </div>
      </header>
      <div class="detail-standalone">${renderDetailPanel()}</div>
    </section>
  `;
  root.querySelectorAll<HTMLButtonElement>("[data-close-detail]").forEach((button) => {
    button.addEventListener("click", () => {
      detailState = undefined;
      renderLoading();
    });
  });
}
async function showDetail(canonicalID: string): Promise<void> {
  const id = canonicalID.trim();
  if (!id) return;
  detailState = { canonicalID: id, loading: true };
  render();
  try {
    const response = await app.callServerTool({ name: "koditon_get_property_detail", arguments: { id } });
    if (response.isError) throw new Error(contentText(response.content) || "Detail tool returned an error.");
    const detail = normalizeDetail(response.structuredContent, response.content);
    detailState = { canonicalID: id, loading: false, detail };
    void app.updateModelContext({ structuredContent: { selected_listing: result?.rows.find((row) => row.canonical_id === id), selected_detail: detail } }).catch(() => {});
  } catch (error) {
    detailState = { canonicalID: id, loading: false, error: error instanceof Error ? error.message : String(error) };
  }
  render();
}
function askAbout(canonicalID: string): void {
  const row = result?.rows.find((item) => item.canonical_id === canonicalID);
  if (!row) return;
  const prompt = `Tell me more about ${row.title} (${row.canonical_id}). Compare asking price, area, source listing facts, normalized details, and any linked actual sale prices.`;
  void app.updateModelContext({ structuredContent: { selected_listing: row, selected_detail: detailState?.canonicalID === canonicalID ? detailState.detail : undefined } }).catch(() => {});
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
    const raw = value as Partial<SearchResult>;
    const rows = Array.isArray(raw.rows) ? raw.rows.map(normalizeListingRow) : [];
    const transactions = Array.isArray(raw.transactions) ? raw.transactions : rows.flatMap((row) => row.transactions);
    return {
      view: raw.view,
      mode: raw.mode ?? "search",
      web_url: raw.web_url ?? "",
      rows,
      transactions,
      total: raw.total ?? rows.length,
      page: raw.page ?? 1,
      page_size: raw.page_size ?? rows.length,
      summary: raw.summary ?? "Structured property results returned."
    };
  }
  return { mode: "search", web_url: "", rows: [], transactions: [], total: 0, page: 1, page_size: 25, summary: "No structured result was returned." };
}
function isSearchResult(value: unknown): value is SearchResult {
  return isObject(value) && "rows" in value;
}
function normalizeListingRow(value: unknown): ListingRow {
  const raw = isObject(value) ? value : {};
  const location = isObject(raw.location) ? raw.location : {};
  const facts = isObject(raw.facts) ? raw.facts : {};
  const costs = isObject(raw.costs) ? raw.costs : {};
  const links = isObject(raw.links) ? raw.links : {};
  const media = isObject(raw.media) ? raw.media : {};
  return {
    ...raw,
    id: stringValue(raw.id),
    canonical_id: stringValue(raw.canonical_id) || stringValue(raw.id) || "property",
    source: stringValue(raw.source),
    kind: stringValue(raw.kind),
    title: stringValue(raw.title) || stringValue(raw.id) || "Property",
    address: stringValue(raw.address) || stringValue(location.address),
    city: stringValue(raw.city) || stringValue(location.city),
    postal: stringValue(raw.postal) || stringValue(location.postal),
    latitude: numberValue(raw.latitude) ?? coordinateAt(location.coordinates, 1),
    longitude: numberValue(raw.longitude) ?? coordinateAt(location.coordinates, 0),
    price: numberValue(raw.price) ?? numberValue(facts.price) ?? numberValue(costs.asking_price),
    area: numberValue(raw.area) ?? numberValue(facts.area_m2),
    room_layout: stringValue(raw.room_layout) || stringValue(facts.rooms),
    url: stringValue(raw.url) || stringValue(links.source),
    external_url_available: Boolean(raw.external_url_available || links.source),
    web_url: stringValue(raw.web_url) || stringValue(links.web),
    thumbnail_url: stringValue(raw.thumbnail_url) || stringValue(media.thumbnail_url),
    transactions: Array.isArray(raw.transactions) ? raw.transactions as PriceTransaction[] : [],
    facts,
    costs,
    location,
    market: isObject(raw.market) ? raw.market : {},
    data_quality: isObject(raw.data_quality) ? raw.data_quality as ListingRow["data_quality"] : undefined
  };
}
function normalizeComparison(value: unknown): ComparisonResult {
  const raw = isObject(value) ? value : {};
  return {
    view: "comparison",
    summary: stringValue(raw.summary) || "Property comparison returned.",
    rows: Array.isArray(raw.rows) ? raw.rows.map(normalizeListingRow) : [],
    ranking: Array.isArray(raw.ranking) ? raw.ranking as ComparisonResult["ranking"] : [],
    tradeoffs: Array.isArray(raw.tradeoffs) ? raw.tradeoffs.map(String) : [],
    missing_data_warnings: Array.isArray(raw.missing_data_warnings) ? raw.missing_data_warnings.map(String) : []
  };
}
function normalizeMarketContext(value: unknown): MarketContextResult {
  const raw = isObject(value) ? value : {};
  return {
    view: "market_context",
    summary: stringValue(raw.summary) || "Market context returned.",
    subject: raw.subject ? normalizeListingRow(raw.subject) : undefined,
    market: isObject(raw.market) ? raw.market : {},
    data_quality: isObject(raw.data_quality) ? raw.data_quality as MarketContextResult["data_quality"] : undefined
  };
}
function structuredView(value: unknown): string {
  return isObject(value) ? stringValue(value.view) : "";
}
function normalizeDetail(structuredContent: unknown, content: unknown): ListingDetail {
  if (isObject(structuredContent) && ("canonical" in structuredContent || "normalized" in structuredContent)) return structuredContent as ListingDetail;
  const text = contentText(content);
  if (!text) return {};
  try {
    const parsed = JSON.parse(text);
    return isObject(parsed) ? parsed as ListingDetail : {};
  } catch {
    return {};
  }
}
function renderMap(points: MapPoint[], title: string): string {
  if (points.length === 0) return "";
  const box = mapBounds(points);
  const markers = points.map((point) => {
    const x = project(point.lng, box.west, box.east);
    const y = 100 - project(point.lat, box.south, box.north);
    return `<g class="${point.selected ? "map-marker map-marker--selected" : "map-marker"}" transform="translate(${x.toFixed(2)} ${y.toFixed(2)})"><circle r="${point.selected ? 4.8 : 3.8}"></circle><title>${escapeHTML(point.title)}</title></g>`;
  }).join("");
  const labels = points.slice(0, 4).map((point) => `<li><span></span>${escapeHTML(point.title)} <em>${escapeHTML(formatCoordinates([point.lng, point.lat]))}</em></li>`).join("");
  return `
    <section class="map-panel">
      <div class="map-head"><h2>${escapeHTML(title)}</h2><span>${formatNumber(points.length)} marker${points.length === 1 ? "" : "s"}</span></div>
      <svg viewBox="0 0 100 100" role="img" aria-label="${escapeAttr(title)}">
        <defs><pattern id="map-grid" width="10" height="10" patternUnits="userSpaceOnUse"><path d="M 10 0 L 0 0 0 10" fill="none" stroke="currentColor" stroke-width="0.35"/></pattern></defs>
        <rect width="100" height="100" rx="3" fill="url(#map-grid)"></rect>
        <path d="M8 74 C22 63 30 79 42 66 S65 49 91 57" fill="none" stroke="currentColor" stroke-width="1.1" opacity="0.32"></path>
        <path d="M14 22 C30 30 42 18 55 31 S79 42 90 29" fill="none" stroke="currentColor" stroke-width="0.9" opacity="0.24"></path>
        ${markers}
      </svg>
      <ol>${labels}</ol>
    </section>
  `;
}
function rowMapPoint(row: ListingRow, selected = false): MapPoint | undefined {
  if (row.longitude == null || row.latitude == null) return undefined;
  if (!isValidCoordinate(row.longitude, row.latitude)) return undefined;
  return { id: row.id || row.canonical_id, title: row.title || row.canonical_id, lng: row.longitude, lat: row.latitude, selected };
}
function detailMapPoints(detail: ListingDetail): MapPoint[] {
  const coordinates = detailCoordinates(detail);
  if (!coordinates) return [];
  const canonical = detail.canonical ?? {};
  return [{ id: stringValue(canonical.canonical_id) || "detail", title: detail.title || stringValue(canonical.headline) || "Property", lng: coordinates[0], lat: coordinates[1], selected: true }];
}
function detailCoordinates(detail: ListingDetail): [number, number] | undefined {
  const record = detail as Record<string, unknown>;
  const location = isObject(record.location) ? record.location : {};
  const canonical = detail.canonical ?? {};
  const normalized = detail.normalized ?? {};
  const lng = coordinateAt(location.coordinates, 0) ?? numberValue(canonical.longitude) ?? numberValue(normalized.longitude);
  const lat = coordinateAt(location.coordinates, 1) ?? numberValue(canonical.latitude) ?? numberValue(normalized.latitude);
  if (lng == null || lat == null || !isValidCoordinate(lng, lat)) return undefined;
  return [lng, lat];
}
function mapBounds(points: MapPoint[]): { west: number; south: number; east: number; north: number } {
  const lngs = points.map((point) => point.lng);
  const lats = points.map((point) => point.lat);
  const west = Math.min(...lngs);
  const east = Math.max(...lngs);
  const south = Math.min(...lats);
  const north = Math.max(...lats);
  const lngPad = Math.max((east - west) * 0.15, 0.01);
  const latPad = Math.max((north - south) * 0.15, 0.01);
  return { west: west - lngPad, east: east + lngPad, south: south - latPad, north: north + latPad };
}
function project(value: number, min: number, max: number): number {
  if (max === min) return 50;
  return ((value - min) / (max - min)) * 86 + 7;
}
function coordinateAt(value: unknown, index: number): number | undefined {
  if (!Array.isArray(value)) return undefined;
  return numberValue(value[index]);
}
function isValidCoordinate(lng: number, lat: number): boolean {
  return lng >= -180 && lng <= 180 && lat >= -90 && lat <= 90;
}
function formatCoordinates(value: [number, number] | undefined): string {
  if (!value) return "";
  return `${value[1].toFixed(5)}, ${value[0].toFixed(5)}`;
}
function contentText(content: unknown): string {
  if (!Array.isArray(content)) return "";
  return content.map((item) => typeof item === "object" && item != null && "text" in item ? String(item.text) : "").filter(Boolean).join("\n");
}
function renderFields(title: string, fields: DetailField[] | undefined): string {
  const items = (fields ?? []).filter((field) => field.value?.trim());
  if (items.length === 0) return "";
  return `<section class="detail-section"><h3>${escapeHTML(title)}</h3><dl>${items.map((field) => `<div><dt>${escapeHTML(field.label ?? "Field")}</dt><dd>${escapeHTML(field.value ?? "")}</dd></div>`).join("")}</dl></section>`;
}
function renderKeyValues(title: string, entries: Array<[string, string]>): string {
  const items = entries.filter((entry) => entry[1].trim());
  if (items.length === 0) return "";
  return `<section class="detail-section"><h3>${escapeHTML(title)}</h3><dl>${items.map(([key, value]) => `<div><dt>${escapeHTML(key)}</dt><dd>${escapeHTML(value)}</dd></div>`).join("")}</dl></section>`;
}
function normalizedEntries(value: Record<string, unknown>): Array<[string, string]> {
  return Object.entries(value).filter(([, item]) => item != null && item !== "").slice(0, 18).map(([key, item]) => [labelize(key), formatUnknown(item)]);
}
function labelize(value: string): string {
  return value.replaceAll("_", " ").replace(/\b\w/g, (char) => char.toUpperCase());
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
function formatUnknown(value: unknown): string {
  if (typeof value === "number") return new Intl.NumberFormat("fi-FI", { maximumFractionDigits: 2 }).format(value);
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (typeof value === "string") return value;
  if (value == null) return "";
  return JSON.stringify(value);
}
function stringValue(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}
function numberValue(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}
function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value != null;
}
function escapeHTML(value: string): string {
  return value.replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;", "'": "&#39;" })[char] ?? char);
}
function escapeAttr(value: string | undefined): string {
  return escapeHTML(value ?? "");
}
