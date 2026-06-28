export type SearchListingsInput = {
  query?: string;
  address?: string;
  source?: "shortcut" | "frontdoor" | "all";
  kind?: "ad" | "announcement" | "building" | "all";
  city?: string;
  postal?: string;
  min_price?: number;
  max_price?: number;
  min_area?: number;
  max_area?: number;
  sort?: "price_asc" | "price_desc" | "area_asc" | "area_desc" | "seen_desc";
  page?: number;
  page_size?: 25 | 50 | 100;
};
export type SearchAppInput = SearchListingsInput & {
  realtor?: string;
  include_prices?: boolean;
  transaction_limit?: number;
};
export type GetListingDetailInput = {
  id: string;
};
export type AddressLookupInput = {
  address: string;
  city?: string;
  postal?: string;
  source?: "shortcut" | "frontdoor" | "all";
  page_size?: number;
};
export type SearchTransactionsInput = {
  municipality_ids?: string[];
  postal_code_ids?: string[];
  categories?: string[];
  types?: string[];
  min_area?: number;
  max_area?: number;
  limit?: number;
};
export type SearchResultRow = {
  canonical_id: string;
  source: string;
  kind: string;
  headline?: string;
  address?: string;
  city?: string;
  postal?: string;
  price?: number;
  area?: number;
  room_layout?: string;
  url?: string;
  external_url_available?: boolean;
  last_seen_at?: string;
};
export type AddressListing = {
  listing_id: string;
  canonical_id: string;
  source: string;
  kind: string;
  native_id: string;
  headline?: string;
  address?: string;
  city?: string;
  postal?: string;
  asking_price?: number;
  debt_free_price?: number;
  area?: number;
  room_layout?: string;
  url?: string;
  external_url_available?: boolean;
  offering_id?: string;
  transactions?: PriceTransactionLink[];
};
export type PriceTransactionLink = {
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
  is_matched?: boolean;
  matched_listing_count?: number;
  matched_offering_count?: number;
  confidence?: string;
};
export type AvailabilityLocations = {
  municipalities?: Array<{ id: string; name_fi: string; code: string }>;
  postal_codes?: Array<{ id: string; municipality_id: string; code: string; name_fi: string }>;
};
export type SearchAppResult = {
  mode: "address" | "search";
  query: SearchAppInput;
  web_url: string;
  rows: AppListingRow[];
  transactions: PriceTransactionLink[];
  total: number;
  page: number;
  page_size: number;
  summary: string;
};
export type AppListingRow = {
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
  external_url_available?: boolean;
  web_url: string;
  thumbnail_url?: string;
  transactions: PriceTransactionLink[];
};
export class KoditonBackend {
  private readonly baseUrl: URL;
  private readonly webBaseUrl: URL;
  private readonly token?: string;
  constructor(baseUrl: string, token?: string, webBaseUrl = baseUrl) {
    this.baseUrl = new URL(baseUrl);
    this.webBaseUrl = new URL(webBaseUrl);
    this.token = token?.trim() || undefined;
  }
  async searchApp(input: SearchAppInput): Promise<SearchAppResult> {
    const normalized = normalizeSearchAppInput(input);
    if (normalized.address) {
      return this.addressSearchApp(normalized);
    }
    const search = await this.searchListings(normalized) as { rows?: SearchResultRow[]; total?: number; page?: number; page_size?: number };
    const rows = (search.rows ?? []).map((row) => appRowFromSearch(row, this.webURL(`/listing/${encodeURIComponent(row.canonical_id)}`)));
    const transactions = normalized.include_prices ? await this.transactionsForInput(normalized) : [];
    return {
      mode: "search",
      query: normalized,
      web_url: this.webURL(searchPath(normalized)),
      rows,
      transactions,
      total: search.total ?? rows.length,
      page: search.page ?? normalized.page ?? 1,
      page_size: search.page_size ?? normalized.page_size ?? 25,
      summary: summarizeAppResult(rows, transactions, search.total ?? rows.length)
    };
  }
  searchListings(input: SearchListingsInput): Promise<unknown> {
    const query = new URLSearchParams();
    setString(query, "q", input.query || input.address);
    setString(query, "source", input.source);
    setString(query, "kind", input.kind);
    setString(query, "city", input.city);
    setString(query, "postal", input.postal);
    setNumber(query, "min_price", input.min_price);
    setNumber(query, "max_price", input.max_price);
    setNumber(query, "min_area", input.min_area);
    setNumber(query, "max_area", input.max_area);
    setString(query, "sort", input.sort);
    setNumber(query, "page", input.page);
    setNumber(query, "page_size", input.page_size);
    return this.getJSON("/api/v1/search", query);
  }
  getListingDetail(input: GetListingDetailInput): Promise<unknown> {
    const query = new URLSearchParams();
    setString(query, "id", input.id);
    return this.getJSON("/api/v1/entity", query);
  }
  lookupAddress(input: AddressLookupInput): Promise<unknown> {
    const query = new URLSearchParams();
    setString(query, "address", input.address);
    setString(query, "city", input.city);
    setString(query, "postal", input.postal);
    setString(query, "source", input.source);
    setNumber(query, "page_size", input.page_size);
    return this.getJSON("/api/v1/address-lookup", query);
  }
  searchTransactions(input: SearchTransactionsInput): Promise<unknown> {
    const query = new URLSearchParams();
    setCSV(query, "municipality_ids", input.municipality_ids);
    setCSV(query, "postal_code_ids", input.postal_code_ids);
    setCSV(query, "categories", input.categories);
    setCSV(query, "types", input.types);
    setNumber(query, "min_area", input.min_area);
    setNumber(query, "max_area", input.max_area);
    setNumber(query, "limit", input.limit);
    return this.getJSON("/api/v1/prices/transactions/filtered", query);
  }
  listCities(): Promise<unknown> {
    return this.getJSON("/api/v1/postal/cities");
  }
  listAvailableLocations(): Promise<unknown> {
    return this.getJSON("/api/v1/availability/locations");
  }
  listCategories(): Promise<unknown> {
    return this.getJSON("/api/v1/availability/categories");
  }
  private async addressSearchApp(input: SearchAppInput): Promise<SearchAppResult> {
    const lookup = await this.lookupAddress({
      address: input.address ?? input.query ?? "",
      city: input.city,
      postal: input.postal,
      source: input.source,
      page_size: input.page_size
    }) as { listings?: AddressListing[] };
    const rows = (lookup.listings ?? []).map((listing) => appRowFromAddress(listing, this.webURL(listing.offering_id ? `/target/offering/${encodeURIComponent(listing.offering_id)}` : `/listing/${encodeURIComponent(listing.canonical_id)}`)));
    const transactions = rows.flatMap((row) => row.transactions);
    return {
      mode: "address",
      query: input,
      web_url: this.webURL(addressPath(input)),
      rows,
      transactions,
      total: rows.length,
      page: 1,
      page_size: input.page_size ?? 50,
      summary: summarizeAppResult(rows, transactions, rows.length)
    };
  }
  private async transactionsForInput(input: SearchAppInput): Promise<PriceTransactionLink[]> {
    if (!input.postal && !input.city) {
      return [];
    }
    const availability = await this.listAvailableLocations() as AvailabilityLocations;
    const postalCodes = availability.postal_codes ?? [];
    const municipalities = availability.municipalities ?? [];
    const selectedPostalIDs = input.postal ? postalCodes.filter((pc) => pc.code.startsWith(input.postal ?? "")).map((pc) => pc.id) : [];
    const selectedMunicipalityIDs = input.city ? municipalities.filter((m) => m.name_fi.localeCompare(input.city ?? "", undefined, { sensitivity: "accent" }) === 0 || m.name_fi.toLowerCase().includes((input.city ?? "").toLowerCase())).map((m) => m.id) : [];
    if (selectedPostalIDs.length === 0 && selectedMunicipalityIDs.length === 0) {
      return [];
    }
    const response = await this.searchTransactions({
      postal_code_ids: selectedPostalIDs,
      municipality_ids: selectedMunicipalityIDs,
      min_area: input.min_area,
      max_area: input.max_area,
      limit: input.transaction_limit ?? 25
    }) as { transactions?: PriceTransactionLink[] };
    return response.transactions ?? [];
  }
  private webURL(path: string): string {
    return new URL(path, this.webBaseUrl).toString();
  }
  private async getJSON(path: string, query?: URLSearchParams): Promise<unknown> {
    const url = new URL(path, this.baseUrl);
    if (query) {
      url.search = query.toString();
    }
    const response = await fetch(url, { headers: this.headers() });
    const text = await response.text();
    if (!response.ok) {
      throw new Error(`Koditon API ${response.status}: ${text || response.statusText}`);
    }
    if (text.trim() === "") {
      return null;
    }
    return JSON.parse(text) as unknown;
  }
  private headers(): Record<string, string> {
    const headers: Record<string, string> = { Accept: "application/json" };
    if (this.token) {
      headers.Authorization = `Bearer ${this.token}`;
    }
    return headers;
  }
}
function setString(query: URLSearchParams, key: string, value: string | undefined): void {
  const trimmed = value?.trim();
  if (trimmed) {
    query.set(key, trimmed);
  }
}
function setNumber(query: URLSearchParams, key: string, value: number | undefined): void {
  if (value !== undefined && Number.isFinite(value)) {
    query.set(key, String(value));
  }
}
function setCSV(query: URLSearchParams, key: string, values: string[] | undefined): void {
  const cleaned = values?.map((value) => value.trim()).filter(Boolean);
  if (cleaned && cleaned.length > 0) {
    query.set(key, cleaned.join(","));
  }
}
function normalizeSearchAppInput(input: SearchAppInput): SearchAppInput {
  const query = [input.query, input.realtor].filter(Boolean).join(" ").trim() || undefined;
  return { ...input, query, source: input.source ?? "all", kind: input.kind ?? "all", page: input.page ?? 1, page_size: input.page_size ?? 25 };
}
function appRowFromSearch(row: SearchResultRow, webURL: string): AppListingRow {
  return {
    canonical_id: row.canonical_id,
    source: row.source,
    kind: row.kind,
    title: row.headline || row.address || row.canonical_id,
    address: row.address,
    city: row.city,
    postal: row.postal,
    price: row.price,
    area: row.area,
    room_layout: row.room_layout,
    url: row.url,
    external_url_available: row.external_url_available,
    web_url: webURL,
    transactions: []
  };
}
function appRowFromAddress(listing: AddressListing, webURL: string): AppListingRow {
  return {
    canonical_id: listing.canonical_id,
    source: listing.source,
    kind: listing.kind,
    title: listing.headline || listing.address || listing.canonical_id,
    address: listing.address,
    city: listing.city,
    postal: listing.postal,
    price: listing.asking_price,
    area: listing.area,
    room_layout: listing.room_layout,
    url: listing.url,
    external_url_available: listing.external_url_available,
    web_url: webURL,
    transactions: listing.transactions ?? []
  };
}
function summarizeAppResult(rows: AppListingRow[], transactions: PriceTransactionLink[], total: number): string {
  const linked = rows.filter((row) => row.transactions.length > 0).length;
  const prices = transactions.filter((transaction) => typeof transaction.price === "number");
  const average = prices.length > 0 ? Math.round(prices.reduce((sum, transaction) => sum + (transaction.price ?? 0), 0) / prices.length) : undefined;
  return `${total} listing${total === 1 ? "" : "s"} found; ${linked} listing${linked === 1 ? "" : "s"} have linked sale prices${average ? `; average shown sale price ${average} EUR` : ""}.`;
}
function searchPath(input: SearchAppInput): string {
  const params = new URLSearchParams();
  setString(params, "q", input.query);
  setString(params, "city", input.city);
  setString(params, "postal", input.postal);
  setString(params, "source", input.source === "all" ? undefined : input.source);
  return `/search${params.size > 0 ? `?${params.toString()}` : ""}`;
}
function addressPath(input: SearchAppInput): string {
  const params = new URLSearchParams();
  setString(params, "address", input.address ?? input.query);
  setString(params, "city", input.city);
  setString(params, "postal", input.postal);
  setString(params, "source", input.source === "all" ? undefined : input.source);
  return `/address${params.size > 0 ? `?${params.toString()}` : ""}`;
}
