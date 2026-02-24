# CLI Database Query Tool

Read-only CLI for querying the koditon database. Build and run from `backend/`.

## Build

```
cd backend && go build -o cli ./cmd/cli/
```

## Commands

### search — Find ads, buildings, and announcements

```
./cli search [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--query` | string | | Free text search |
| `--source` | string | all | shortcut, frontdoor, all |
| `--kind` | string | all | ad, building, announcement, all |
| `--type` | string | all | listing, rental, all |
| `--city` | string | | City name (e.g. Helsinki, Espoo, Tampere) |
| `--postal` | string | | Postal code (e.g. 00100) |
| `--min-price` | int | 0 | Minimum price in euros |
| `--max-price` | int | 0 | Maximum price in euros |
| `--min-area` | float | 0 | Minimum area in m² |
| `--max-area` | float | 0 | Maximum area in m² |
| `--sort` | string | seen_desc | price_asc, price_desc, area_asc, area_desc, seen_desc |
| `--limit` | int | 25 | Results per page (25, 50, or 100) |
| `--page` | int | 1 | Page number |
| `--after` | string | | Published after date (YYYY-MM-DD) |
| `--before` | string | | Published before date (YYYY-MM-DD) |

**Examples:**

```bash
# Ads in Helsinki under 300k
./cli search --city Helsinki --kind ad --max-price 300000

# Large apartments in Espoo
./cli search --city Espoo --min-area 80 --sort area_desc --limit 50

# Free text search across all sources
./cli search --query "Kallio" --kind ad

# Page through results
./cli search --city Helsinki --page 2 --limit 25
```

### detail — Show full entity details

Takes a single positional argument: the canonical ID from search results.

```
./cli detail <canonical-id>
```

Canonical ID format: `source:kind:nativeID`

**Examples:**

```bash
./cli detail shortcut:ad:12345
./cli detail frontdoor:building:550e8400-e29b-41d4-a716-446655440000
./cli detail frontdoor:announcement:a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

Output includes: core fields (address, city, price, area, URL), promoted detail fields, source-specific fields, and related entity counts.

### transactions — Search price transactions

```
./cli transactions --city <city> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--city` | string | **required** | City name |
| `--search` | string | | Address, neighborhood, or postal code fragment |
| `--limit` | int | 50 | Maximum results |

**Examples:**

```bash
# All recent transactions in Helsinki
./cli transactions --city Helsinki

# Transactions in a specific area
./cli transactions --city Helsinki --search "Kallio" --limit 20

# Search by postal code
./cli transactions --city Espoo --search "02100"
```

Output columns: period, description, type, area, price, €/m², postal code, neighborhood, condition.

## Workflow Tips

1. Start with a broad `search` to find entities of interest.
2. Copy a canonical ID from the results and use `detail` for the full picture.
3. Use `transactions` to look up comparable sale prices in the same area.

## Requirements

Needs `DATABASE_URL` (and other env vars) set via `.env` / `.env.local` in the `backend/` directory — same config as the TUI and API server.
