# Lootstash Marketplace API - Project Guide

## Overview

Go service for a Diablo II peer-to-peer item trading platform. Full trading platform with listings, offers, trades, chat, and premium features. Backed by Supabase (PostgreSQL + Auth + Storage), Redis, Stripe, and Battle.net OAuth.

## Build & Run

```bash
go build ./...
go run . serve          # HTTP server (port from config)
```

Uses Cobra CLI. Loads `.env` via godotenv.

## Architecture

```
Handlers (HTTP/Fiber)
    ↓
Services (business logic, caching, coordination)
    ↓
Repositories (data access, SQL queries)
    ↓
Models (domain entities)
    ↓
PostgreSQL (Bun ORM) + Redis cache
```

Middleware stack: Recovery → Logger → RequestID → CORS → Auth → Handlers

## Key Paths

| Path | Purpose |
|------|---------|
| `main.go` | Entry point, loads .env |
| `cmd/root.go` | CLI config (DB, Redis, Supabase, Stripe, Battle.net) |
| `cmd/serve.go` | HTTP server startup |
| `internal/api/server.go` | Route registration, middleware, dependency wiring |
| `internal/api/middleware/auth.go` | JWT auth (Supabase JWKS + HMAC fallback) |
| `internal/api/handlers/v1/` | All HTTP handlers (13 files) |
| `internal/api/dto/` | Request/response DTOs (9 files) |
| `internal/models/` | Database models (12 files) |
| `internal/service/` | Business logic (9 service files) |
| `internal/repository/` | Data access (10 repo files + interfaces.go) |
| `internal/database/bun.go` | Bun ORM setup (max 25 conn, 5 idle) |
| `internal/cache/` | Redis client, cache keys, invalidation |
| `internal/games/` | Game registry, D2 handler (categories, rarities, stat validation) |

## API Endpoints

### Public (no auth)

```
GET  /api/v1/listings              # List/filter listings (card view)
GET  /api/v1/listings/:id          # Listing detail (full stats)
GET  /api/v1/profiles/:id          # User profile
GET  /api/v1/profiles/:id/ratings  # User ratings
GET  /api/v1/decline-reasons       # Offer decline reasons
GET  /api/v1/marketplace/stats     # Marketplace statistics
GET  /api/v1/games/:game/categories
POST /api/v1/webhooks/stripe       # Stripe webhook
```

### Authenticated

```
# Profile
GET/PATCH /api/v1/me               # Current user profile
POST      /api/v1/me/picture       # Upload avatar
PATCH     /api/v1/me/flair         # Profile flair (premium)

# Battle.net
POST   /api/v1/me/battlenet/link|callback
DELETE /api/v1/me/battlenet

# Listings
GET    /api/v1/my/listings         # User's own listings (card view)
POST   /api/v1/listings            # Create listing
PATCH  /api/v1/listings/:id        # Update listing
DELETE /api/v1/listings/:id        # Cancel listing

# Offers
GET    /api/v1/offers              # User's offers (buyer/seller)
POST   /api/v1/offers              # Create offer
GET    /api/v1/offers/:id
POST   /api/v1/offers/:id/accept|reject|cancel

# Trades
GET    /api/v1/trades              # User's trades
GET    /api/v1/trades/:id
POST   /api/v1/trades/:id/complete|cancel

# Chat (per trade)
GET    /api/v1/chats/:id
GET    /api/v1/chats/:id/messages
POST   /api/v1/chats/:id/messages
POST   /api/v1/chats/:id/read

# Notifications
GET    /api/v1/notifications
GET    /api/v1/notifications/count
POST   /api/v1/notifications/read

# Ratings
POST   /api/v1/ratings

# Subscriptions (Stripe)
GET    /api/v1/subscriptions/me
POST   /api/v1/subscriptions/checkout|cancel
GET    /api/v1/subscriptions/billing-history

# Wishlist (premium)
GET/POST   /api/v1/wishlist
PATCH/DELETE /api/v1/wishlist/:id

# Premium
GET    /api/v1/marketplace/price-history
GET    /api/v1/my/listings/count
```

## Trading Flow

1. Seller creates **Listing** with item details, stats, asking price/items
2. Buyer submits **Offer** on listing with offered items
3. Seller **accepts** (→ creates Trade + Chat + notifications) or **rejects** (with decline reason)
4. Participants coordinate via **Chat** messages within the Trade
5. Trade **completed** → creates **Transaction** record for rating eligibility
6. Both parties can submit **Rating** (1-5 stars) on the Transaction

## Stats Handling (Important)

Listing stats are stored as `json.RawMessage` (JSONB) with this raw structure from frontend:
```json
{"code": "ac%", "value": 163, "displayText": "+163% Enhanced Defense", "isVariable": true}
```

Two transformation methods in `ListingService`:
- **`transformCardStats`**: Returns only `isVariable=true` stats (for list/card views via `ToCardResponse`)
- **`transformAllStats`**: Returns ALL stats with `isVariable` flag preserved (for detail view via `ToResponse`/`ToDetailResponse`)

The `ItemStat` DTO includes an `IsVariable` field so the frontend can style variable stats differently.

## DTOs

Two listing response types:
- **`ListingCardResponse`**: Lightweight for card/list views (id, name, itemType, rarity, imageUrl, variable stats only, askingFor, askingPrice, game metadata, seller, views, createdAt)
- **`ListingResponse`**: Full details (all fields including category, suffixes, runes, baseItem info, notes, status, expiresAt, ALL stats with isVariable)
- **`ListingDetailResponse`**: Extends `ListingResponse` with `updatedAt` and `tradeCount`

## Caching Strategy

Redis cache with key patterns:
- `profile:{id}` — 1 hour TTL
- `listing:{id}` — 15 min TTL
- `filter:results:{hash}` — 20s TTL (listing filter query results, keyed by SHA-256 of filter params)
- `notification:count:{userId}`
- `decline:reasons`
- `ratelimit:{ip}:{endpoint}`
- `marketplace:stats`

Cache invalidation via `cache.Invalidator` on entity updates. Filter result cache is invalidated on listing create/update/delete (belt-and-suspenders with 20s TTL).

### Cache-Control Headers

Public GET endpoints set `Cache-Control: public, max-age=N, s-maxage=N` via middleware (`internal/api/middleware/cache_control.go`):
- Listings list, recent: 15s
- Profiles: 60s
- Listing detail, marketplace stats: 300s
- Game categories, decline reasons, service types: 3600s

## Authentication

Supabase JWT (ES256/ECDSA) via JWKS endpoint with 5-min key cache. Fallback to HMAC (HS256) if configured. Two middleware variants:
- `NewAuthMiddleware()` — strict, 401 on missing token
- `OptionalAuthMiddleware()` — extracts user if present, continues without

User ID stored in Fiber context via `middleware.GetUserID(c)`.

## Database Schema

All tables in `d2` schema with RLS enabled.

### Core Tables

| Table | Key Fields |
|-------|-----------|
| `profiles` | username, display_name, avatar, is_premium, profile_flair, stripe_*, battle_net_*, total_trades, average_rating, preferred_ladder, preferred_hardcore, preferred_platforms (TEXT[]), preferred_region |
| `listings` | seller_id, name, item_type, rarity, category, stats (JSONB), suffixes, runes, asking_for (JSONB), asking_price, game, ladder, hardcore, platform, region, status, views, expires_at |
| `listing_stats` | listing_id, stat_code, stat_value (normalized from listings.stats via DB trigger — used for affix filtering) |
| `offers` | listing_id, requester_id, offered_items (JSONB), status, decline_reason_id |
| `trades` | offer_id, listing_id, seller_id, buyer_id, status, cancel_reason |
| `chats` | trade_id (unique) |
| `messages` | chat_id, sender_id, content, message_type, read_at |
| `transactions` | trade_id, item_name, item_details (JSONB), offered_items (JSONB) |
| `ratings` | transaction_id (unique), rater_id, rated_id, stars (1-5), comment |
| `notifications` | user_id, type (enum), title, metadata (JSONB), reference_type, reference_id, read |
| `wishlist_items` | user_id, name, category, rarity, stat_criteria (JSONB), game, ladder, hardcore, platforms (TEXT[]), region, status |
| `billing_events` | user_id, stripe_event_id (unique), event_type, amount_cents, currency |
| `decline_reasons` | code (unique), message, active |
| `marketplace_stats` | active_listings, trades_today, avg_response_time_minutes |

### Enums

- **Rarity**: normal, magic, rare, unique, legendary, set, runeword
- **Platform**: pc, xbox, playstation, switch
- **Region**: americas, europe, asia
- **Listing status**: active, pending, completed, cancelled
- **Offer status**: pending, accepted, rejected, cancelled
- **Trade status**: active, completed, cancelled
- **Notification type**: trade_request_received, trade_request_accepted, trade_request_rejected, new_message, rating_received, wishlist_match
- **Message type**: text, system, trade_update

### D2 Game Categories

helms, body armor, weapons, shields, gloves, boots, belts, amulets, rings, charms, jewels, runes, gems, misc

## Dependencies (Go 1.22)

- gofiber/fiber/v2 - HTTP framework
- uptrace/bun - PostgreSQL ORM
- golang-jwt/jwt/v5 - JWT handling
- google/uuid - UUID generation
- redis/go-redis/v9 - Redis client
- stripe/stripe-go/v81 - Stripe integration
- spf13/cobra - CLI framework
- go-playground/validator/v10 - Request validation
- joho/godotenv - Environment loading

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection string |
| `SUPABASE_URL` | Supabase project URL |
| `SUPABASE_JWT_SECRET` | Supabase JWT secret for HMAC fallback |
| `LOG_LEVEL` | Logging level (info, debug, error) |
| `LOG_JSON` | Enable JSON logging format |
| `BATTLENET_CLIENT_ID` | Battle.net OAuth client ID |
| `BATTLENET_CLIENT_SECRET` | Battle.net OAuth client secret |
| `BATTLENET_REDIRECT_URI` | Battle.net OAuth redirect URI |
| `STRIPE_SECRET_KEY` | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret |
| `STRIPE_PRICE_ID` | Stripe price ID for premium subscription |
| `STRIPE_SUCCESS_URL` | Redirect URL after successful checkout |
| `STRIPE_CANCEL_URL` | Redirect URL after cancelled checkout |

## Key Patterns

- **Affix filtering**: Standard stat filters query the normalized `d2.listing_stats` table (synced by DB trigger). Skill tab filters (`skilltab` with `param`) still use JSONB `jsonb_array_elements` since `listing_stats` has no `param` column
- **Wishlist matching**: New listings trigger async matching against user wishlists → notifications
- **Premium gating**: Free users limited to 10 active listings. Premium unlocks unlimited listings, wishlist, profile flair, price history
- **Notification system**: Polymorphic references (`reference_type` + `reference_id`) to link any entity
- **Game registry**: Pluggable game handler system (`internal/games/`) — currently only D2 implemented
- **RLS**: All Supabase tables use Row Level Security; service role bypasses for background jobs

## Docker

```bash
# Build
docker build -t lootstash-marketplace-api .

# Run with local Redis
docker-compose up -d redis
docker run -p 8080:8080 --env-file .env lootstash-marketplace-api
```

## Fly.io Deployment

```bash
fly launch --no-deploy  # First time setup
fly deploy              # Deploy
fly secrets set DATABASE_URL=xxx REDIS_URL=xxx STRIPE_SECRET_KEY=xxx  # Set secrets
```

Scaling config (`fly.toml`): `min_machines_running=1` (eliminates cold starts), concurrency limits at 150 soft / 200 hard requests.
