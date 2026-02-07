# Section 1

## Subsection 1
Hello world this has `inline code` in the middle.

> **Output to inspect:**
>
> The cache is content-addressed under `~/.cache/ingestion-auto/bronze/`:
> ```
> ~/.cache/ingestion-auto/bronze/
> ├── metadata.db                        # SQLite index of all cached entries
> ├── tmp/                               # Temporary files during download (empty after success)
> └── objects/
>     └── {hash[:2]}/
>         └── {hash[2:]}                 # The actual cached file (Arrow IPC)
> ```
> The `hash` is the SHA-256 of the file content. The `metadata.db` SQLite
> database has a `cache_entries` table (content_hash, source_id, source_url,
> fetched_at, size_bytes, format, etag, last_modified) and a `source_cache`
> table mapping source IDs to their current hash.
>
> **Quick check:** The fetch command prints the path to the cached file. Note
> this path -- you will pass it to `transform` in step 6.
> ```bash
> # Or query the cache DB directly:
> sqlite3 ~/.cache/ingestion-auto/bronze/metadata.db \
>   "SELECT source_id, content_hash, size_bytes FROM cache_entries;"
> ```
## Subsection 2

# Section 2