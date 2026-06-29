#!/usr/bin/env node
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { registerAppResource, registerAppTool, RESOURCE_MIME_TYPE } from "@modelcontextprotocol/ext-apps/server";
import { readFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { z } from "zod";
import { KoditonBackend } from "./backend.js";

const apiBaseUrl = process.env.KODITON_API_BASE_URL ?? "http://localhost:8080";
const webBaseUrl = process.env.KODITON_WEB_BASE_URL ?? "http://localhost:5173";
const apiToken = process.env.KODITON_API_TOKEN;
const appResourceUri = "ui://koditon/listings.html";
const distDir = dirname(fileURLToPath(import.meta.url));
const appHtmlPath = resolve(distDir, "../dist/mcp-app.html");
const backend = new KoditonBackend(apiBaseUrl, apiToken, webBaseUrl);
const server = new McpServer({ name: "koditon-mcp", version: "0.1.0" });

registerAppTool(
  server,
  "koditon_find_listings",
  {
    title: "Find Listings",
    description: "Search Finnish property listings by address, postal code, city, realtor/office text, price, or area. Returns rows with key facts, links, and available actual sale price data.",
    inputSchema: {
      query: z.string().optional().describe("Free-text search across listing, housing company, address, URL, and realtor/office text where indexed."),
      address: z.string().optional().describe("Street address. Uses the address lookup flow with linked prices sale rows."),
      postal: z.string().optional().describe("Postal code or prefix."),
      city: z.string().optional().describe("City or municipality."),
      realtor: z.string().optional().describe("Realtor, office, seller, or contact text to include in the free-text search."),
      source: z.enum(["shortcut", "frontdoor", "all"]).optional(),
      kind: z.enum(["ad", "announcement", "building", "all"]).optional(),
      min_price: z.number().int().positive().optional(),
      max_price: z.number().int().positive().optional(),
      min_area: z.number().positive().optional(),
      max_area: z.number().positive().optional(),
      sort: z.enum(["price_asc", "price_desc", "area_asc", "area_desc", "seen_desc"]).optional(),
      page: z.number().int().positive().optional(),
      page_size: z.union([z.literal(25), z.literal(50), z.literal(100)]).optional(),
      include_prices: z.boolean().optional().describe("When searching by postal/city, also include recent actual sale price rows from prices."),
      transaction_limit: z.number().int().positive().max(100).optional()
    },
    _meta: { ui: { resourceUri: appResourceUri } }
  },
  async (input) => {
    const result = await backend.searchApp(input);
    return {
      content: [{ type: "text", text: textSummary(result) }],
      structuredContent: result
    };
  }
);

registerAppResource(
  server,
  "Koditon Listings",
  appResourceUri,
  {
    description: "Interactive listing search results with linked sales prices and web app actions."
  },
  async () => ({
    contents: [
      {
        uri: appResourceUri,
        mimeType: RESOURCE_MIME_TYPE,
        text: await readFile(appHtmlPath, "utf8"),
        _meta: {
          ui: {
            csp: {
              connectDomains: [apiBaseUrl, webBaseUrl],
              resourceDomains: [apiBaseUrl, webBaseUrl]
            },
            prefersBorder: true
          }
        }
      }
    ]
  })
);

server.tool(
  "koditon_search_listings",
  "Search Finnish property listings with filters. Returns canonical IDs that can be passed to koditon_get_listing_detail.",
  {
    query: z.string().optional().describe("Free-text search."),
    address: z.string().optional().describe("Address text. Used as the search query when query is omitted."),
    source: z.enum(["shortcut", "frontdoor", "all"]).optional(),
    kind: z.enum(["ad", "announcement", "building", "all"]).optional(),
    city: z.string().optional(),
    postal: z.string().optional(),
    min_price: z.number().int().positive().optional(),
    max_price: z.number().int().positive().optional(),
    min_area: z.number().positive().optional(),
    max_area: z.number().positive().optional(),
    sort: z.enum(["price_asc", "price_desc", "area_asc", "area_desc", "seen_desc"]).optional(),
    page: z.number().int().positive().optional(),
    page_size: z.union([z.literal(25), z.literal(50), z.literal(100)]).optional()
  },
  async (input) => jsonResult(await backend.searchListings(input))
);

server.tool(
  "koditon_get_listing_detail",
  "Get normalized listing detail by canonical ID or source URL.",
  { id: z.string().min(1).describe("Canonical ID or source URL.") },
  async (input) => jsonResult(await backend.getListingDetail(input))
);

server.tool(
  "koditon_lookup_address",
  "Lookup source listings, price links, canonical offering links, and match candidates for an address.",
  {
    address: z.string().min(1),
    city: z.string().optional(),
    postal: z.string().optional(),
    source: z.enum(["shortcut", "frontdoor", "all"]).optional(),
    page_size: z.number().int().positive().max(100).optional()
  },
  async (input) => jsonResult(await backend.lookupAddress(input))
);

server.tool(
  "koditon_search_transactions",
  "Search Finnish property price transactions with optional municipality, postal code, category, type, and area filters.",
  {
    municipality_ids: z.array(z.string().uuid()).optional(),
    postal_code_ids: z.array(z.string().uuid()).optional(),
    categories: z.array(z.string().min(1)).optional(),
    types: z.array(z.string().min(1)).optional(),
    min_area: z.number().positive().optional(),
    max_area: z.number().positive().optional(),
    limit: z.number().int().positive().max(500).optional()
  },
  async (input) => jsonResult(await backend.searchTransactions(input))
);

server.tool(
  "koditon_list_cities",
  "List all Finnish municipalities and postal codes known to Koditon.",
  {},
  async () => jsonResult(await backend.listCities())
);

server.tool(
  "koditon_list_available_locations",
  "List municipalities and postal codes that have property price transaction data.",
  {},
  async () => jsonResult(await backend.listAvailableLocations())
);

server.tool(
  "koditon_list_categories",
  "List available property building categories for transaction filters.",
  {},
  async () => jsonResult(await backend.listCategories())
);

await server.connect(new StdioServerTransport());

function jsonResult(value: unknown) {
  return { content: [{ type: "text" as const, text: JSON.stringify(value, null, 2) }] };
}
function textSummary(value: { summary: string; web_url: string; rows: Array<{ title: string; address?: string; city?: string; postal?: string; price?: number; area?: number; canonical_id: string; transactions: unknown[] }> }) {
  const rows = value.rows.slice(0, 8).map((row, index) => {
    const location = [row.address, row.city, row.postal].filter(Boolean).join(", ");
    const facts = [row.price ? `${row.price} EUR` : "", row.area ? `${row.area} m2` : "", row.transactions.length > 0 ? `${row.transactions.length} sale price link(s)` : ""].filter(Boolean).join(" · ");
    return `${index + 1}. ${row.title} (${row.canonical_id})${location ? ` — ${location}` : ""}${facts ? ` — ${facts}` : ""}`;
  });
  return [value.summary, `Open in Koditon: ${value.web_url}`, ...rows].join("\n");
}
