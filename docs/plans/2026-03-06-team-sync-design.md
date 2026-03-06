# Team Sync via Encrypted Cloud Channels

## Problem

Teams need to share collections without setting up a git repo or running infrastructure. The solution should feel like pastebin — simple, no accounts — but with encryption so public storage is safe.

## Decision Summary

- Entire collections are the sync unit (not individual requests)
- Push/pull is manual (no live sync)
- Client-side AES-256-GCM encryption with a shared group passphrase
- Central Cloudflare Worker + KV at `sync.ratatuile.dev` as the storage backend
- One group key per team, namespaces all collections under that team
- Remote wins on pull (overwrites local, no merge)
- TUI sync modal for push/pull operations

## Architecture

```
+---------------+      AES-encrypted JSON       +----------------------+
|  ratatuile   | <-------- push/pull ---------> |  Cloudflare Worker   |
|   (client)    |                                |  + KV store          |
+---------------+                                +----------------------+
       |                                                   |
   group key                                    KV key: {namespace}/{collectionName}
   entered once                                 KV value: nonce || AES-GCM(collectionJSON)
   in settings
```

## Key Derivation & Namespacing

1. User enters a group passphrase (e.g., `"my-team-secret"`)
2. **Namespace** = first 16 hex chars of `sha256(passphrase)` — public, used as KV key prefix
3. **Encryption key** = `scrypt(passphrase, salt="ratatuile")` -> 256-bit AES key
4. KV key format: `{namespace}/{collectionName}`
5. KV value format: `nonce (12 bytes) || AES-GCM ciphertext`

Without the passphrase, you cannot derive the namespace (to locate data) or decrypt it (to read it).

## Cloudflare Worker API

Three endpoints, zero auth:

| Method | Path                        | Body           | Description              |
|--------|-----------------------------|----------------|--------------------------|
| PUT    | `/:namespace/:collection`   | encrypted blob | Push collection          |
| GET    | `/:namespace/:collection`   | -              | Pull collection          |
| GET    | `/:namespace`               | -              | List collection names    |

The list endpoint returns collection names in plaintext. Collection names are not considered sensitive; this enables the sync modal to show what's available remotely.

## Client: `internal/sync/`

New package with:

- `DeriveKey(passphrase string) (namespace string, aesKey []byte)` — scrypt + sha256
- `Encrypt(key []byte, plaintext []byte) ([]byte, error)` — AES-256-GCM
- `Decrypt(key []byte, ciphertext []byte) ([]byte, error)` — AES-256-GCM
- `Push(endpoint, namespace, collectionName string, blob []byte) error` — HTTP PUT
- `Pull(endpoint, namespace, collectionName string) ([]byte, error)` — HTTP GET
- `ListRemote(endpoint, namespace string) ([]string, error)` — HTTP GET

## Settings Persistence

- `~/.ratatuile/group_key` — stores the group passphrase (plaintext, same pattern as `active_env`)
- `~/.ratatuile/sync_endpoint` — optional, defaults to `https://sync.ratatuile.dev`

## TUI: `internal/tui/syncmodal/`

New modal overlay, toggled via keybind (Ctrl+G for "group sync"):

- If group key not configured, prompts user to enter it first
- Two-column view: local collections (left), remote collections (right)
- Push: select a local collection, encrypt, upload
- Pull: select a remote collection, download, decrypt, overwrite local
- Status indicators for each collection (local only / remote only / both)

## Cloudflare Worker Constraints (Free Tier)

- 100k reads/day, 1k writes/day
- 1MB max value size per KV entry
- No expiration — values persist indefinitely
- No auth, no rate limiting beyond Cloudflare defaults

## What This Does NOT Do

- No accounts or tokens
- No versioning or history
- No conflict resolution (remote wins)
- No environment sync (only collections)
- No end-to-end verification (if someone with the key pushes garbage, others pull garbage)
