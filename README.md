# dns-sync

### Database structure:

```json
{
    "name": "example-host.example.com",
    "content": "127.1.33.7"
}
```

### ENV parameters:

```yaml
DEBUG: true|false
DNS_FILTER: Regex to test and validate all new DNS entries against
CF_API_TOKEN: Cloudflare
CF_DOMAIN: Cloudflare
MONGODB_URI: Full URI with auth
MONGODB_DATABASE: Database name
MONGODB_COLLECTION: Database collection name
```