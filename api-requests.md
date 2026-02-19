# LootStash Marketplace API Reference

Base URL: `http://localhost:8081`

## Authentication

All authenticated endpoints require a JWT token in the Authorization header:

```
Authorization: Bearer <supabase_jwt_token>
```

The token is obtained from Supabase Auth after user login.

---

## Health Check

### GET /health

Check if the API is running.

**Headers:** None required

**Response:**
```json
{
  "status": "ok",
  "service": "lootstash-marketplace-api"
}
```

---

## Profiles

### GET /api/v1/profiles/:id

Get a user's public profile.

**Headers:** None required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | User profile ID (UUID) or username |

**Response:**
```json
{
  "id": "uuid",
  "username": "string",
  "displayName": "string",
  "avatarUrl": "string",
  "totalTrades": 0,
  "averageRating": 4.5,
  "ratingCount": 10,
  "isPremium": true,
  "isAdmin": false,
  "profileFlair": "gold",
  "usernameColor": "#FF5733",
  "createdAt": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404` - Profile not found

---

### GET /api/v1/me

Get the current authenticated user's profile.

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": "uuid",
  "username": "string",
  "displayName": "string",
  "avatarUrl": "string",
  "totalTrades": 0,
  "averageRating": 4.5,
  "ratingCount": 10,
  "isPremium": true,
  "isAdmin": false,
  "profileFlair": "gold",
  "usernameColor": "#FF5733",
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `401` - Unauthorized
- `404` - Profile not found

---

### PATCH /api/v1/me

Update the current user's profile.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "displayName": "string (optional, 1-50 chars)",
  "avatarUrl": "string (optional, valid URL)"
}
```

**Response:**
```json
{
  "id": "uuid",
  "username": "string",
  "displayName": "string",
  "avatarUrl": "string",
  "totalTrades": 0,
  "averageRating": 4.5,
  "ratingCount": 10,
  "isPremium": true,
  "isAdmin": false,
  "profileFlair": "gold",
  "usernameColor": "#FF5733",
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400` - Validation error
- `401` - Unauthorized
- `404` - Profile not found

---

### POST /api/v1/me/picture

Upload a profile picture.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

**Request Body:**
| Field | Type | Description |
|-------|------|-------------|
| picture | file | Image file (PNG, JPEG, WebP; max 2MB) |

**Response:**
```json
{
  "avatarUrl": "https://..."
}
```

**Error Responses:**
- `400` - Invalid file type / File too large / No file provided
- `401` - Unauthorized
- `404` - Profile not found
- `500` - Upload failed

---

### PATCH /api/v1/me/username-color

Update the current user's username color (premium only). Color is cleared on subscription cancellation.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "color": "#FF5733"
}
```

| Field | Type | Description |
|-------|------|-------------|
| color | string | A hex color code (e.g., `#FF5733`) or `"none"` to clear |

**Validation:**
- Must be `"none"` or a valid 6-digit hex color matching `^#[0-9A-Fa-f]{6}$`

**Response:**
```json
{
  "success": true,
  "message": "Username color updated"
}
```

**Error Responses:**
- `400` - Invalid color value
- `401` - Unauthorized
- `403` - Premium subscription required

---

## Listings

### GET /api/v1/listings

List and filter item listings.

**Note:** Listings with active trades are automatically hidden from public results.

**Headers:** None required (optional auth for personalized results)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| q | string | Text search in item name |
| game | string | Game filter (e.g., "diablo2") |
| ladder | boolean | Filter by ladder (true/false) |
| hardcore | boolean | Filter by hardcore mode |
| platform | string | Platform filter (pc, xbox, playstation, switch) |
| region | string | Region filter (americas, europe, asia) |
| category | string | Item category (helm, armor, weapon, etc.) |
| rarity | string | Rarity filter (normal, magic, rare, unique, set, runeword) |
| affixFilters | json | JSON array of affix filters (see below) |
| sortBy | string | Sort field (created_at, name, asking_price) |
| sortOrder | string | Sort direction (asc, desc) |
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Affix Filter Format:**
```json
[
  {"code": "all_skills", "minValue": 2},
  {"code": "fire_res", "minValue": 30, "maxValue": 45}
]
```

**Example Request:**
```
GET /api/v1/listings?game=diablo2&ladder=true&category=helm&rarity=unique&page=1&perPage=20
```

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "sellerId": "uuid",
      "seller": {
        "id": "uuid",
        "username": "string",
        "displayName": "string",
        "avatarUrl": "string",
        "totalTrades": 0,
        "averageRating": 4.5,
        "ratingCount": 10,
        "isPremium": true,
        "isAdmin": false,
        "profileFlair": "gold",
        "usernameColor": "#FF5733",
        "createdAt": "2024-01-01T00:00:00Z"
      },
      "name": "Harlequin Crest",
      "itemType": "Shako",
      "rarity": "unique",
      "imageUrl": "string",
      "category": "helm",
      "stats": [
        {"code": "all_skills", "value": 2, "displayText": "+2 To All Skills"},
        {"code": "life", "value": 141, "displayText": "+141 To Life"}
      ],
      "suffixes": [],
      "runes": [],
      "runeOrder": "",
      "baseItemCode": "",
      "baseItemName": "",
      "askingFor": [
        {"type": "rune", "name": "Ist"}
      ],
      "askingPrice": "1.5 Ist",
      "notes": "Perfect roll",
      "game": "diablo2",
      "ladder": true,
      "hardcore": false,
      "platform": "pc",
      "region": "americas",
      "status": "active",
      "views": 234,
      "createdAt": "2024-01-01T00:00:00Z",
      "expiresAt": "2024-01-31T00:00:00Z"
    }
  ],
  "page": 1,
  "perPage": 20,
  "totalCount": 150,
  "totalPages": 8
}
```

**Stat Object:**
| Field | Type | Description |
|-------|------|-------------|
| code | string | Affix code (e.g., "all_skills", "dmg%") |
| value | number/null | Numeric value for filtering (nullable) |
| displayText | string | Human-readable stat text (e.g., "+2 To All Skills") |

**Rune Object (for runewords):**
| Field | Type | Description |
|-------|------|-------------|
| code | string | Rune code (e.g., "r08", "r03") |
| name | string | Rune name (e.g., "Ral", "Tir") |
| imageUrl | string | URL to rune image |

---

### GET /api/v1/listings/:id

Get detailed information about a specific listing.

**Headers:** None required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Listing ID |

**Response:**
```json
{
  "id": "uuid",
  "sellerId": "uuid",
  "seller": { ... },
  "name": "Harlequin Crest",
  "itemType": "Shako",
  "rarity": "unique",
  "imageUrl": "string",
  "category": "helm",
  "stats": [
    {"code": "all_skills", "value": 2, "displayText": "+2 To All Skills"}
  ],
  "suffixes": [],
  "runes": [],
  "runeOrder": "",
  "baseItemCode": "",
  "baseItemName": "",
  "askingFor": [...],
  "askingPrice": "1.5 Ist",
  "notes": "Perfect roll",
  "game": "diablo2",
  "ladder": true,
  "hardcore": false,
  "platform": "pc",
  "region": "americas",
  "status": "active",
  "views": 234,
  "createdAt": "2024-01-01T00:00:00Z",
  "expiresAt": "2024-01-31T00:00:00Z",
  "updatedAt": "2024-01-01T00:00:00Z",
  "tradeCount": 3
}
```

**Example Response (Runeword Listing):**
```json
{
  "id": "uuid",
  "name": "Insight",
  "itemType": "Polearm",
  "rarity": "runeword",
  "category": "runeword",
  "stats": [
    {"code": "dmg%", "value": 235, "displayText": "+235% Enhanced Damage"},
    {"code": "cast2", "value": 35, "displayText": "+35% Faster Cast Rate"},
    {"code": "aura", "value": 14, "displayText": "Level 14 Meditation Aura When Equipped"}
  ],
  "runes": [
    {"code": "r08", "name": "Ral", "imageUrl": "http://.../ral.png"},
    {"code": "r03", "name": "Tir", "imageUrl": "http://.../tir.png"},
    {"code": "r07", "name": "Tal", "imageUrl": "http://.../tal.png"},
    {"code": "r12", "name": "Sol", "imageUrl": "http://.../sol.png"}
  ],
  "runeOrder": "r08r03r07r12",
  "baseItemCode": "9vo",
  "baseItemName": "Voulge",
  ...
}
```

**Error Responses:**
- `404` - Listing not found

---

### POST /api/v1/listings

Create a new listing.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Harlequin Crest (required)",
  "itemType": "Shako (required)",
  "rarity": "unique (required: normal|magic|rare|unique|legendary|set|runeword)",
  "imageUrl": "https://... (optional)",
  "category": "helm (required)",
  "stats": [
    {"code": "all_skills", "value": 2, "displayText": "+2 To All Skills"}
  ],
  "suffixes": [],
  "runes": ["r08", "r03", "r07", "r12"],
  "runeOrder": "r08r03r07r12 (optional, for runewords)",
  "baseItemCode": "9vo (optional, base item code for runewords)",
  "baseItemName": "Voulge (optional, base item name for runewords)",
  "askingFor": [
    {"type": "rune", "name": "Ist"}
  ],
  "askingPrice": "1.5 Ist (optional)",
  "notes": "Perfect roll (optional, max 500 chars)",
  "game": "diablo2 (required)",
  "ladder": true,
  "hardcore": false,
  "platform": "pc (required: pc|xbox|playstation|switch)",
  "region": "americas (required: americas|europe|asia)"
}
```

**Stat Input Format:**
| Field | Type | Description |
|-------|------|-------------|
| code | string | Affix code (required) |
| value | number/string | The actual rolled value (required) |
| displayText | string | Display text from catalog-api (recommended, preserved as-is) |

**Runeword Fields:**
| Field | Type | Description |
|-------|------|-------------|
| runes | string[] | Array of rune codes in socket order (e.g., ["r08", "r03"]) |
| runeOrder | string | Concatenated rune codes (e.g., "r08r03r07r12") |
| baseItemCode | string | Game code for the base item |
| baseItemName | string | Display name of the base item |

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "sellerId": "uuid",
  "name": "Harlequin Crest",
  ...
}
```

**Error Responses:**
- `400` - Validation error
- `401` - Unauthorized

---

### PATCH /api/v1/listings/:id

Update an existing listing (owner only).

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Listing ID |

**Request Body:**
```json
{
  "askingFor": [...],
  "askingPrice": "2 Ist (optional)",
  "notes": "Updated notes (optional)",
  "status": "cancelled (optional: active|cancelled)"
}
```

**Response:**
```json
{
  "id": "uuid",
  ...
}
```

**Error Responses:**
- `400` - Validation error
- `401` - Unauthorized
- `403` - Forbidden (not owner)
- `404` - Listing not found

---

### DELETE /api/v1/listings/:id

Cancel a listing (owner only).

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Listing ID |

**Response:**
```json
{
  "success": true,
  "message": "Listing cancelled"
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not owner)
- `404` - Listing not found

---

### GET /api/v1/my/listings

Get the current user's listings.

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| status | string | Filter by status (active, pending, completed, cancelled) |
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Response:**
```json
{
  "data": [...],
  "page": 1,
  "perPage": 20,
  "totalCount": 5,
  "totalPages": 1
}
```

**Error Responses:**
- `401` - Unauthorized

---

## Services

Services are standalone entities (not listings) where providers offer in-game services. Services are permanent until the provider cancels them. Providers can also **pause** a service to temporarily hide it from search, and **resume** it later. The marketplace shows one card per provider with all their active services, sorted by premium status and rating. Paused and cancelled services are hidden from public search but still visible in the provider's own "my services" list.

### GET /api/v1/services

List service providers (grouped by provider, sorted by premium + rating).

**Headers:** None required (optional auth)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| serviceType | string | Comma-separated service types (rush, crush, grush, sockets, waypoints, ubers, colossal_ancients) |
| game | string | Game filter (e.g., "diablo2") |
| ladder | boolean | Filter by ladder |
| hardcore | boolean | Filter by hardcore mode |
| platforms | string | Comma-separated platforms (pc, xbox, playstation, switch) |
| region | string | Region filter (americas, europe, asia) |
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Response:**
```json
{
  "data": [
    {
      "provider": {
        "id": "uuid",
        "username": "string",
        "displayName": "string",
        "avatarUrl": "string",
        "isPremium": true,
        "averageRating": 4.8,
        "ratingCount": 25,
        ...
      },
      "services": [
        {
          "id": "uuid",
          "serviceType": "rush",
          "name": "Hell Rush - Fast",
          "description": "Full hell rush, all waypoints included",
          "askingPrice": "Ist",
          "askingFor": [...],
          "game": "diablo2",
          "ladder": true,
          "hardcore": false,
          "platforms": ["pc"],
          "region": "americas",
          "notes": "Available evenings EST",
          "status": "active",
          "createdAt": "2024-01-01T00:00:00Z",
          "updatedAt": "2024-01-01T00:00:00Z"
        }
      ]
    }
  ],
  "page": 1,
  "perPage": 20,
  "totalCount": 15,
  "totalPages": 1
}
```

---

### GET /api/v1/services/providers/:id

Get a specific provider's services.

**Headers:** None required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Provider profile ID |

**Response:**
```json
{
  "provider": { ... },
  "services": [...]
}
```

**Error Responses:**
- `404` - Provider not found or has no services

---

### POST /api/v1/services

Create a new service (auth required). One service per type per provider per game.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "serviceType": "rush (required: rush|crush|grush|sockets|waypoints|ubers|colossal_ancients)",
  "name": "Hell Rush - Fast (required, max 100 chars)",
  "description": "Full hell rush (optional, max 2000 chars)",
  "askingFor": [{"type": "rune", "name": "Ist"}],
  "askingPrice": "Ist (optional, max 100 chars)",
  "notes": "Available evenings EST (optional, max 500 chars)",
  "game": "diablo2 (required)",
  "ladder": true,
  "hardcore": false,
  "platforms": ["pc"],
  "region": "americas (required: americas|europe|asia)"
}
```

**Response:** `201 Created`

**Error Responses:**
- `400` - Validation error
- `401` - Unauthorized
- `409` - Already have a service of this type for this game

---

### PATCH /api/v1/services/:id

Update a service (owner only).

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body (all fields optional):**
```json
{
  "name": "Updated name",
  "description": "Updated description",
  "askingPrice": "2 Ist",
  "askingFor": [...],
  "notes": "Updated notes",
  "platforms": ["pc", "xbox"],
  "region": "europe"
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not owner)
- `404` - Service not found

---

### DELETE /api/v1/services/:id

Cancel a service (owner only). This is a permanent soft delete — cancelled services cannot be resumed.

**Headers:**
```
Authorization: Bearer <token>
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not owner)
- `404` - Service not found

---

### POST /api/v1/services/:id/pause

Pause an active service (owner only). Paused services are hidden from public search results but can be resumed later. The provider's card will not show paused services.

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Service ID |

**Response:**
```json
{
  "success": true,
  "message": "Service paused"
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not owner)
- `404` - Service not found
- `409` - Only active services can be paused

---

### POST /api/v1/services/:id/resume

Resume a paused service (owner only). The service becomes visible again in public search results.

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Service ID |

**Response:**
```json
{
  "success": true,
  "message": "Service resumed"
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not owner)
- `404` - Service not found
- `409` - Only paused services can be resumed

---

### GET /api/v1/my/services

List the current user's services.

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Response:** Paginated list of `ServiceResponse` objects.

---

## Service Runs

Service runs are created when a service offer is accepted. Unlike trades, the service stays active after a service run completes — providers can accept multiple concurrent service runs.

### GET /api/v1/service-runs

List the current user's service runs.

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| status | string | Filter by status (active, completed, cancelled) |
| role | string | Filter by role (provider, client, all) |
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "serviceId": "uuid",
      "service": { ... },
      "offerId": "uuid",
      "providerId": "uuid",
      "provider": { ... },
      "clientId": "uuid",
      "client": { ... },
      "offeredItems": [...],
      "status": "active",
      "chatId": "uuid",
      "transactionId": null,
      "canRate": false,
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z",
      "completedAt": null,
      "cancelledAt": null
    }
  ],
  "page": 1,
  "perPage": 20,
  "totalCount": 5,
  "totalPages": 1
}
```

---

### GET /api/v1/service-runs/:id

Get detailed information about a service run.

**Response includes additional fields:**
- `canComplete` - Whether the current user can complete this service run
- `canCancel` - Whether the service run can be cancelled
- `canMessage` - Whether messaging is available

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not a participant)
- `404` - Service run not found

---

### POST /api/v1/service-runs/:id/complete

Mark a service run as completed (either party). Creates a transaction record for rating.

**Note:** The service stays active — the provider can continue accepting new service runs.

**Error Responses:**
- `400` - Service run not active
- `401` - Unauthorized
- `403` - Forbidden (not a participant)
- `404` - Service run not found

---

### POST /api/v1/service-runs/:id/cancel

Cancel an active service run (either party).

**Request Body (optional):**
```json
{
  "reason": "Changed my mind (optional)"
}
```

**Error Responses:**
- `400` - Service run not active
- `401` - Unauthorized
- `403` - Forbidden (not a participant)
- `404` - Service run not found

---

## Offers

Offers represent initial trade proposals on listings or services. When an offer is accepted, a Trade (for items) or Service Run (for services) and Chat are created.

### GET /api/v1/offers

List the current user's offers.

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| type | string | Filter by type (item, service, all) |
| status | string | Filter by status (pending, accepted, rejected, cancelled) |
| role | string | Filter by role (buyer, seller, all) |
| listingId | uuid | Filter by listing ID (get all offers on a specific listing - seller only) |
| serviceId | uuid | Filter by service ID (get all offers on a specific service - provider only) |
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Example Requests:**
```
# Get all my offers (as buyer or seller)
GET /api/v1/offers

# Get only offers I made (as buyer)
GET /api/v1/offers?role=buyer

# Get all offers on my listing (as seller)
GET /api/v1/offers?listingId=<listing-uuid>

# Get pending offers on my listing
GET /api/v1/offers?listingId=<listing-uuid>&status=pending
```

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "type": "item",
      "listingId": "uuid",
      "listing": { ... },
      "serviceId": null,
      "service": null,
      "requesterId": "uuid",
      "requester": { ... },
      "offeredItems": [
        {"type": "rune", "name": "Ist"},
        {"type": "rune", "name": "Mal"}
      ],
      "message": "Willing to add more if needed",
      "status": "pending",
      "declineReason": null,
      "declineNote": "",
      "tradeId": null,
      "serviceRunId": null,
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z",
      "acceptedAt": null
    }
  ],
  "page": 1,
  "perPage": 20,
  "totalCount": 10,
  "totalPages": 1
}
```

**Error Responses:**
- `401` - Unauthorized

---

### GET /api/v1/offers/:id

Get detailed information about an offer.

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Offer ID |

**Response:**
```json
{
  "id": "uuid",
  "type": "item",
  "listingId": "uuid",
  "listing": { ... },
  "serviceId": null,
  "service": null,
  "requesterId": "uuid",
  "requester": { ... },
  "offeredItems": [...],
  "message": "string",
  "status": "pending",
  "declineReason": null,
  "declineNote": "",
  "tradeId": null,
  "serviceRunId": null,
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-01T00:00:00Z",
  "acceptedAt": null
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not a participant)
- `404` - Offer not found

---

### POST /api/v1/offers

Create a new offer on a listing or service.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "type": "item (required: item|service)",
  "listingId": "uuid (required for type=item)",
  "serviceId": "uuid (required for type=service)",
  "offeredItems": [
    {"type": "rune", "name": "Ist"},
    {"type": "rune", "name": "Mal"}
  ],
  "message": "I can add more runes if needed (optional, max 500 chars)"
}
```

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "type": "item",
  "listingId": "uuid",
  "serviceId": null,
  "requesterId": "uuid",
  "offeredItems": [...],
  "message": "string",
  "status": "pending",
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400` - Validation error / Cannot offer on own listing/service / Listing/service not available
- `401` - Unauthorized
- `404` - Listing or service not found

---

### POST /api/v1/offers/:id/accept

Accept an offer (listing/service owner only). For item offers, this creates a Trade and Chat. For service offers, this creates a Service Run and Chat.

**Note:** When an item offer is accepted, the listing is hidden from public search results until the trade is completed or cancelled. For service offers, the service stays active.

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Offer ID |

**Response:**
```json
{
  "offer": {
    "id": "uuid",
    "status": "accepted",
    "acceptedAt": "2024-01-01T00:00:00Z",
    "tradeId": "uuid",
    ...
  },
  "tradeId": "uuid (for item offers)",
  "serviceRunId": "uuid (for service offers)",
  "chatId": "uuid"
}
```

**Error Responses:**
- `400` - Offer not pending / Listing already has active trade
- `401` - Unauthorized
- `403` - Forbidden (not listing/service owner)
- `404` - Offer not found

---

### POST /api/v1/offers/:id/reject

Reject an offer (listing owner only).

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Offer ID |

**Request Body:**
```json
{
  "declineReasonId": 1,
  "declineNote": "Looking for higher offer (optional, max 200 chars)"
}
```

**Response:**
```json
{
  "id": "uuid",
  "status": "rejected",
  "declineReason": {
    "id": 1,
    "code": "price_too_low",
    "message": "The offer is too low"
  },
  "declineNote": "Looking for higher offer",
  ...
}
```

**Error Responses:**
- `400` - Validation error / Offer not pending
- `401` - Unauthorized
- `403` - Forbidden (not listing owner)
- `404` - Offer or decline reason not found

---

### POST /api/v1/offers/:id/cancel

Cancel a pending offer (requester/buyer only).

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Offer ID |

**Response:**
```json
{
  "id": "uuid",
  "status": "cancelled",
  "listingId": "uuid",
  "requesterId": "uuid",
  ...
}
```

**Error Responses:**
- `400` - Only pending offers can be cancelled
- `401` - Unauthorized
- `403` - Forbidden (only the requester can cancel their offer)
- `404` - Offer not found

---

### GET /api/v1/decline-reasons

Get all available decline reasons.

**Headers:** None required

**Response:**
```json
[
  {"id": 1, "code": "price_too_low", "message": "The offer is too low"},
  {"id": 2, "code": "item_sold", "message": "The item has already been sold"},
  {"id": 3, "code": "changed_mind", "message": "I changed my mind about selling"},
  {"id": 4, "code": "wrong_offer", "message": "The offer doesn't match what I'm looking for"},
  {"id": 5, "code": "other", "message": "Other reason"}
]
```

---

## Trades

Trades represent active negotiations after an offer is accepted. Each trade has an associated Chat for communication.

### GET /api/v1/trades

List the current user's trades.

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| status | string | Filter by status (active, completed, cancelled) |
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "offerId": "uuid",
      "listingId": "uuid",
      "listing": { ... },
      "sellerId": "uuid",
      "seller": { ... },
      "buyerId": "uuid",
      "buyer": { ... },
      "status": "active",
      "cancelReason": "",
      "cancelledBy": "",
      "chatId": "uuid",
      "transactionId": null,
      "canRate": false,
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z",
      "completedAt": null,
      "cancelledAt": null
    }
  ],
  "page": 1,
  "perPage": 20,
  "totalCount": 5,
  "totalPages": 1
}
```

**Response Fields for Rating:**
| Field | Type | Description |
|-------|------|-------------|
| transactionId | string/null | Transaction ID for completed trades (used for rating) |
| canRate | boolean | Whether the current user can rate this trade (true if completed and not yet rated) |

**Error Responses:**
- `401` - Unauthorized

---

### GET /api/v1/trades/:id

Get detailed information about a trade.

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Trade ID |

**Response:**
```json
{
  "id": "uuid",
  "offerId": "uuid",
  "listingId": "uuid",
  "listing": { ... },
  "sellerId": "uuid",
  "seller": { ... },
  "buyerId": "uuid",
  "buyer": { ... },
  "status": "active",
  "cancelReason": "",
  "cancelledBy": "",
  "chatId": "uuid",
  "transactionId": null,
  "canRate": false,
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-01T00:00:00Z",
  "completedAt": null,
  "cancelledAt": null,
  "canComplete": true,
  "canCancel": true,
  "canMessage": true
}
```

**Response Fields for Rating:**
| Field | Type | Description |
|-------|------|-------------|
| transactionId | string/null | Transaction ID for completed trades (used for rating) |
| canRate | boolean | Whether the current user can rate this trade (true if completed and not yet rated) |

**Example Response (Completed Trade):**
```json
{
  "id": "uuid",
  "status": "completed",
  "transactionId": "uuid",
  "canRate": true,
  "completedAt": "2024-01-01T00:00:00Z",
  ...
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not a participant)
- `404` - Trade not found

---

### POST /api/v1/trades/:id/complete

Mark a trade as completed (seller only). Creates a transaction record.

**Note:** When a trade is completed, the listing status is set to "completed" and removed from public listings.

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Trade ID |

**Response:**
```json
{
  "trade": {
    "id": "uuid",
    "status": "completed",
    "completedAt": "2024-01-01T00:00:00Z",
    ...
  },
  "transactionId": "uuid"
}
```

**Error Responses:**
- `400` - Trade not active
- `401` - Unauthorized
- `403` - Forbidden (not seller)
- `404` - Trade not found

---

### POST /api/v1/trades/:id/cancel

Cancel an active trade (either party).

**Note:** When a trade is cancelled, the listing becomes visible again in public search results.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Trade ID |

**Request Body (optional):**
```json
{
  "reason": "Changed my mind (optional, max 500 chars)"
}
```

**Response:**
```json
{
  "id": "uuid",
  "status": "cancelled",
  "cancelReason": "Changed my mind",
  "cancelledBy": "uuid",
  "cancelledAt": "2024-01-01T00:00:00Z",
  "transactionId": null,
  "canRate": false,
  ...
}
```

**Error Responses:**
- `400` - Only active trades can be cancelled
- `401` - Unauthorized
- `403` - Forbidden (not a participant)
- `404` - Trade not found

---

## Chats

Chats are created when an offer is accepted. They are linked to either a Trade (for item offers) or a Service Run (for service offers).

### GET /api/v1/chats/:id

Get chat details.

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Chat ID |

**Response:**
```json
{
  "id": "uuid",
  "tradeId": "uuid",
  "serviceRunId": "uuid",
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not a participant)
- `404` - Chat not found

---

### GET /api/v1/chats/:id/messages

Get messages in a chat (paginated).

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Chat ID |

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "chatId": "uuid",
      "senderId": "uuid",
      "sender": { ... },
      "content": "Message content",
      "messageType": "text",
      "readAt": null,
      "createdAt": "2024-01-01T00:00:00Z"
    }
  ],
  "page": 1,
  "perPage": 20,
  "totalCount": 5,
  "totalPages": 1
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not a participant)
- `404` - Chat not found

---

### POST /api/v1/chats/:id/messages

Send a message in a chat.

**Note:** Messaging is only available while the trade is active.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Chat ID |

**Request Body:**
```json
{
  "content": "Message content (required, 1-1000 chars)"
}
```

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "chatId": "uuid",
  "senderId": "uuid",
  "sender": { ... },
  "content": "Message content",
  "messageType": "text",
  "readAt": null,
  "createdAt": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400` - Validation error / Trade not active
- `401` - Unauthorized
- `403` - Forbidden (not a participant)
- `404` - Chat not found

---

### POST /api/v1/chats/:id/read

Mark messages as read. If no message IDs are provided, marks all unread messages in the chat as read.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Chat ID |

**Request Body (optional):**
```json
{
  "messageIds": ["uuid", "uuid", "uuid"]
}
```

If the body is empty or omitted, all unread messages in the chat will be marked as read.

**Response:**
```json
{
  "success": true
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not a participant)
- `404` - Chat not found

---

## Notifications

### GET /api/v1/notifications

List the current user's notifications.

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| unread | boolean | Filter to unread only |
| type | string | Filter by type (trade_request_received, trade_request_accepted, trade_request_rejected, new_message, rating_received, wishlist_match, service_run_created, service_run_completed, service_run_cancelled) |
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "type": "trade_request_received",
      "title": "New Offer",
      "body": "You received a new offer for Harlequin Crest",
      "referenceType": "offer",
      "referenceId": "uuid",
      "read": false,
      "readAt": null,
      "metadata": {},
      "createdAt": "2024-01-01T00:00:00Z"
    }
  ],
  "page": 1,
  "perPage": 20,
  "totalCount": 15,
  "totalPages": 1
}
```

**Error Responses:**
- `401` - Unauthorized

---

### GET /api/v1/notifications/count

Get unread notification count (for bell badge).

**Headers:**
```
Authorization: Bearer <token>
```

**Response:**
```json
{
  "count": 5
}
```

**Error Responses:**
- `401` - Unauthorized

---

### POST /api/v1/notifications/read

Mark notifications as read.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "notificationIds": ["uuid", "uuid", "uuid"]
}
```

**Response:**
```json
{
  "success": true
}
```

**Error Responses:**
- `400` - Validation error
- `401` - Unauthorized

---

## Ratings

### POST /api/v1/ratings

Rate a completed trade.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "transactionId": "uuid (required)",
  "stars": 5,
  "comment": "Great trader, fast and friendly! (optional, max 500 chars)"
}
```

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "transactionId": "uuid",
  "raterId": "uuid",
  "rater": { ... },
  "ratedId": "uuid",
  "stars": 5,
  "comment": "Great trader, fast and friendly!",
  "createdAt": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400` - Validation error
- `401` - Unauthorized
- `403` - Forbidden (not a participant)
- `404` - Transaction not found
- `409` - Already rated this transaction

---

### GET /api/v1/profiles/:id/ratings

Get ratings for a user.

**Headers:** None required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | User profile ID |

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "transactionId": "uuid",
      "raterId": "uuid",
      "rater": { ... },
      "ratedId": "uuid",
      "stars": 5,
      "comment": "Great trader!",
      "createdAt": "2024-01-01T00:00:00Z"
    }
  ],
  "page": 1,
  "perPage": 20,
  "totalCount": 25,
  "totalPages": 2
}
```

---

## Wishlist (Premium)

Premium users can create wishlist items to be notified when matching listings are posted. When a new listing matches a wishlist item's criteria (name, game, filters, and stat ranges), the wishlist owner receives a `wishlist_match` notification.

### GET /api/v1/wishlist

List the current user's wishlist items (paginated).

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "userId": "uuid",
      "name": "Harlequin Crest",
      "category": "helm",
      "rarity": "unique",
      "statCriteria": [
        {"code": "all_skills", "name": "All Skills", "minValue": 2},
        {"code": "life", "name": "Life", "minValue": 130, "maxValue": 141}
      ],
      "game": "diablo2",
      "ladder": true,
      "hardcore": null,
      "platform": null,
      "region": null,
      "status": "active",
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z"
    }
  ],
  "page": 1,
  "perPage": 20,
  "totalCount": 3,
  "totalPages": 1
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Premium required

---

### POST /api/v1/wishlist

Create a new wishlist item. Maximum 10 active items per user.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Harlequin Crest (required, max 100 chars)",
  "category": "helm (optional)",
  "rarity": "unique (optional)",
  "statCriteria": [
    {"code": "all_skills", "name": "All Skills", "minValue": 2},
    {"code": "life", "name": "Life", "minValue": 130}
  ],
  "game": "diablo2 (required)",
  "ladder": true,
  "hardcore": false,
  "platform": "pc (optional)",
  "region": "americas (optional)"
}
```

**Stat Criteria:**
| Field | Type | Description |
|-------|------|-------------|
| code | string | Affix code (required, see Affix Codes Reference) |
| name | string | Display name for the stat (optional, from catalog-api) |
| minValue | number | Minimum stat value (optional) |
| maxValue | number | Maximum stat value (optional) |

**Filter Fields (null = match any):**
| Field | Type | Description |
|-------|------|-------------|
| ladder | boolean/null | Match specific ladder mode, or any if null |
| hardcore | boolean/null | Match specific hardcore mode, or any if null |
| platform | string/null | Match specific platform, or any if null |
| region | string/null | Match specific region, or any if null |
| category | string/null | Match specific category, or any if null |
| rarity | string/null | Match specific rarity, or any if null |

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "userId": "uuid",
  "name": "Harlequin Crest",
  "category": "helm",
  "rarity": "unique",
  "statCriteria": [
    {"code": "all_skills", "name": "All Skills", "minValue": 2},
    {"code": "life", "name": "Life", "minValue": 130}
  ],
  "game": "diablo2",
  "ladder": true,
  "hardcore": false,
  "platform": "pc",
  "region": "americas",
  "status": "active",
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400` - Validation error
- `401` - Unauthorized
- `403` - Premium required / Wishlist limit reached (max 10 active items)

---

### PATCH /api/v1/wishlist/:id

Update a wishlist item (owner only).

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Wishlist item ID |

**Request Body (all fields optional):**
```json
{
  "name": "Hellfire Torch",
  "category": "charm",
  "rarity": "unique",
  "statCriteria": [
    {"code": "sorc_skills", "name": "Sorceress Skills", "minValue": 3}
  ],
  "game": "diablo2",
  "ladder": null,
  "hardcore": null,
  "platform": null,
  "region": null,
  "status": "paused"
}
```

**Status Values:**
| Status | Description |
|--------|-------------|
| active | Actively matching against new listings |
| paused | Temporarily disabled, no matching |

**Response:**
```json
{
  "id": "uuid",
  "userId": "uuid",
  "name": "Hellfire Torch",
  ...
}
```

**Error Responses:**
- `400` - Validation error
- `401` - Unauthorized
- `403` - Forbidden (not owner)
- `404` - Wishlist item not found

---

### DELETE /api/v1/wishlist/:id

Soft-delete a wishlist item (owner only).

**Headers:**
```
Authorization: Bearer <token>
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Wishlist item ID |

**Response:**
```json
{
  "success": true,
  "message": "Wishlist item deleted"
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Forbidden (not owner)
- `404` - Wishlist item not found

---

## Marketplace Stats

### GET /api/v1/marketplace/stats

Get marketplace statistics including active listings, trades today, and online sellers.

**Headers:** None required

**Response:**
```json
{
  "activeListings": 12847,
  "tradesToday": 1234,
  "onlineSellers": 567,
  "avgResponseTimeMinutes": 5.2,
  "lastUpdated": "2026-01-30T10:30:00Z"
}
```

**Response Fields:**
| Field | Type | Description |
|-------|------|-------------|
| activeListings | number | Total number of active listings |
| tradesToday | number | Number of completed trades in the last 24 hours |
| onlineSellers | number | Number of sellers active in the last 15 minutes |
| avgResponseTimeMinutes | number | Average time (in minutes) for sellers to respond to offers |
| lastUpdated | string | ISO 8601 timestamp of when the stats were last calculated |

**Notes:**
- Stats are cached for 5 minutes for performance
- `avgResponseTimeMinutes` is calculated from trades completed in the last 7 days

---

## Games

### GET /api/v1/games/:game/categories

Get item categories for a specific game.

**Headers:** None required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| game | string | Game code (e.g., "diablo2") |

**Response:**
```json
[
  {"code": "helm", "name": "Helms"},
  {"code": "armor", "name": "Body Armor"},
  {"code": "weapon", "name": "Weapons"},
  {"code": "shield", "name": "Shields"},
  {"code": "gloves", "name": "Gloves"},
  {"code": "boots", "name": "Boots"},
  {"code": "belt", "name": "Belts"},
  {"code": "amulet", "name": "Amulets"},
  {"code": "ring", "name": "Rings"},
  {"code": "charm", "name": "Charms"},
  {"code": "jewel", "name": "Jewels"},
  {"code": "rune", "name": "Runes"},
  {"code": "gem", "name": "Gems"},
  {"code": "misc", "name": "Miscellaneous"}
]
```

**Error Responses:**
- `404` - Game not found

---

### GET /api/v1/games/:game/service-types

Get available service types for a specific game. See the [Services](#services) section for full documentation.

---

## Bug Reports

### POST /api/v1/bug-reports

Submit a bug report. Any authenticated user can submit.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "title": "Button not working on mobile (required, 5-200 chars)",
  "description": "When I tap the submit button on the offer page on my iPhone, nothing happens. (required, 10-5000 chars)"
}
```

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "title": "Button not working on mobile",
  "description": "When I tap the submit button on the offer page on my iPhone, nothing happens.",
  "status": "open",
  "createdAt": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400` - Validation error
- `401` - Unauthorized

---

### GET /api/v1/bug-reports

List all bug reports (admin only, paginated).

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| status | string | Filter by status (open, resolved, closed) |
| page | number | Page number (default: 1) |
| perPage | number | Items per page (default: 20, max: 100) |

**Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "title": "Button not working on mobile",
      "description": "When I tap the submit button...",
      "status": "open",
      "reporterId": "uuid",
      "reporterUsername": "trader123",
      "reporterAvatar": "https://...",
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z"
    }
  ],
  "page": 1,
  "perPage": 20,
  "totalCount": 5,
  "totalPages": 1
}
```

**Error Responses:**
- `401` - Unauthorized
- `403` - Admin access required

---

### PATCH /api/v1/bug-reports/:id

Update a bug report's status (admin only).

**Headers:**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Bug report ID |

**Request Body:**
```json
{
  "status": "resolved (required: open|resolved|closed)"
}
```

**Response:**
```json
{
  "id": "uuid",
  "title": "Button not working on mobile",
  "description": "When I tap the submit button...",
  "status": "resolved",
  "reporterId": "uuid",
  "reporterUsername": "trader123",
  "reporterAvatar": "https://...",
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-02T00:00:00Z"
}
```

**Error Responses:**
- `400` - Validation error
- `401` - Unauthorized
- `403` - Admin access required
- `404` - Bug report not found

---

## Error Response Format

All error responses follow this format:

```json
{
  "error": "error_code",
  "message": "Human readable error message",
  "code": 400
}
```

**Common Error Codes:**
| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | bad_request | Invalid request parameters |
| 400 | validation_error | Request body validation failed |
| 401 | unauthorized | Missing or invalid auth token |
| 403 | forbidden | User doesn't have permission |
| 404 | not_found | Resource not found |
| 409 | conflict | Resource already exists |
| 429 | rate_limit_exceeded | Too many requests |
| 500 | internal_error | Server error |

---

## Rate Limiting

The API implements rate limiting:
- **Default:** 100 requests per minute per user/IP
- **Strict endpoints:** 20 requests per minute (for sensitive operations)

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704067200
```

---

## Trade Flow Summary

The trading system supports two flows: item trades and service runs.

### Item Flow

```
1. Buyer creates offer:     POST /api/v1/offers (type: "item") -> Offer(pending)
2. Seller accepts:          POST /api/v1/offers/:id/accept -> Offer(accepted) + Trade(active) + Chat created
                            Listing is hidden from public search results
3. Chat happens:            POST /api/v1/chats/:id/messages -> Messages in Chat
4. Either cancels:          POST /api/v1/trades/:id/cancel -> Trade(cancelled), Listing visible again
   OR
5. Seller completes:        POST /api/v1/trades/:id/complete -> Trade(completed) + Transaction created
                            Returns transactionId for rating
6. Both can rate:           POST /api/v1/ratings -> Rating created (bilateral ratings)
                            Each party can rate once per transaction
```

### Service Flow

```
1. Client creates offer:    POST /api/v1/offers (type: "service") -> Offer(pending)
2. Provider accepts:        POST /api/v1/offers/:id/accept -> Offer(accepted) + ServiceRun(active) + Chat created
                            Service stays active for new offers
3. Chat happens:            POST /api/v1/chats/:id/messages
4. Either cancels:          POST /api/v1/service-runs/:id/cancel -> ServiceRun(cancelled)
   OR
5. Either completes:        POST /api/v1/service-runs/:id/complete -> ServiceRun(completed) + Transaction created
6. Both can rate:           POST /api/v1/ratings
```

**Rating System:**
- When a trade is completed, a `transactionId` is returned and stored on the trade
- Both buyer and seller can rate each other using the same `transactionId`
- Each user can only rate once per transaction (409 Conflict if duplicate)
- Use `GET /api/v1/trades` or `GET /api/v1/trades/:id` to check `canRate` status
- Rating updates the rated user's `averageRating` and `ratingCount` automatically

**Key Behaviors:**
- Listings with active trades are **hidden** from public listing results
- When a trade is cancelled, the listing becomes **visible** again
- When a trade is completed, the listing is marked as **completed** (removed from listings)
- Messaging is only available while the trade is **active**

---

## Affix Codes Reference (Diablo 2)

Both simplified (canonical) codes and game data codes are accepted for filtering. The API automatically expands queries to match either code system.

### Aliased Stat Codes

These stats have both a canonical (user-friendly) code and game data code variants. Use either when filtering:

| Canonical Code | Game Code(s) | Description |
|----------------|--------------|-------------|
| mf | mag% | +X% Better Chance of Getting Magic Items |
| gf | gold% | +X% Extra Gold from Monsters |
| fcr | cast1, cast2, cast3 | +X% Faster Cast Rate |
| ias | swing1, swing2, swing3 | +X% Increased Attack Speed |
| fhr | balance1, balance2, balance3 | +X% Faster Hit Recovery |
| frw | move1, move2, move3 | +X% Faster Run/Walk |
| ed | dmg% | +X% Enhanced Damage |
| ar | att, att% | +X To Attack Rating |
| fire_res | res-fire | Fire Resist +X% |
| cold_res | res-cold | Cold Resist +X% |
| light_res | res-ltng | Lightning Resist +X% |
| poison_res | res-pois | Poison Resist +X% |
| all_res | res-all | All Resistances +X |
| life_steal | lifesteal | +X% Life Stolen Per Hit |
| mana_steal | manasteal | +X% Mana Stolen Per Hit |
| crushing_blow | crush | +X% Crushing Blow |
| deadly_strike | deadly | +X% Deadly Strike |
| open_wounds | openwounds | +X% Chance of Open Wounds |

### Other Common Stat Codes

These codes are used as-is (no aliases):

| Code | Description |
|------|-------------|
| all_skills | +X To All Skills |
| amazon_skills | +X To Amazon Skill Levels |
| assassin_skills | +X To Assassin Skill Levels |
| barbarian_skills | +X To Barbarian Skill Levels |
| druid_skills | +X To Druid Skill Levels |
| necro_skills | +X To Necromancer Skill Levels |
| paladin_skills | +X To Paladin Skill Levels |
| sorc_skills | +X To Sorceress Skill Levels |
| life | +X To Life |
| mana | +X To Mana |
| strength | +X To Strength |
| dexterity | +X To Dexterity |
| vitality | +X To Vitality |
| energy | +X To Energy |
| sockets | Socketed (X) |

See `internal/games/d2/statcodes.go` for the alias map and `internal/games/d2/constants.go` for categories.

---

## Rune Codes Reference (Diablo 2)

Rune codes for runeword listings:

| Code | Name | Number |
|------|------|--------|
| r01 | El | 1 |
| r02 | Eld | 2 |
| r03 | Tir | 3 |
| r04 | Nef | 4 |
| r05 | Eth | 5 |
| r06 | Ith | 6 |
| r07 | Tal | 7 |
| r08 | Ral | 8 |
| r09 | Ort | 9 |
| r10 | Thul | 10 |
| r11 | Amn | 11 |
| r12 | Sol | 12 |
| r13 | Shael | 13 |
| r14 | Dol | 14 |
| r15 | Hel | 15 |
| r16 | Io | 16 |
| r17 | Lum | 17 |
| r18 | Ko | 18 |
| r19 | Fal | 19 |
| r20 | Lem | 20 |
| r21 | Pul | 21 |
| r22 | Um | 22 |
| r23 | Mal | 23 |
| r24 | Ist | 24 |
| r25 | Gul | 25 |
| r26 | Vex | 26 |
| r27 | Ohm | 27 |
| r28 | Lo | 28 |
| r29 | Sur | 29 |
| r30 | Ber | 30 |
| r31 | Jah | 31 |
| r32 | Cham | 32 |
| r33 | Zod | 33 |

See `internal/games/d2/runes.go` for the complete list with image URL generation.
