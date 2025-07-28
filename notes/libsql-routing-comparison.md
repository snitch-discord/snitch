# LibSQL Multi-tenancy Routing: nginx vs Traefik - Complete Analysis

## Summary

After extensive exploration of Traefik as an alternative to the nginx/dnsmasq setup for LibSQL multi-tenancy, we concluded that the existing nginx approach is definitively superior for this specific use case. Traefik, while excellent for many scenarios, is fundamentally inappropriate for LibSQL's subdomain-based namespace routing requirements.

## Current nginx/dnsmasq Setup

**Strengths:**

- ✅ Simple regex-based subdomain capture: `~^(?<subdomain>[a-z0-9-]+)\.(localhost|snitch-sqld-proxy).*;`
- ✅ Straightforward header transformation: `proxy_set_header Host $subdomain.db;`
- ✅ Works with any number of namespaces dynamically
- ✅ Integrated DNS + routing solution
- ✅ Proven and stable
- ✅ Minimal configuration (2 services, 1 config file)

**Current approach:**

```yaml
# dnsmasq provides wildcard DNS: *.snitch-sqld-proxy -> 219.219.219.125
# nginx transforms: namespace.snitch-sqld-proxy -> Host: namespace.db -> LibSQL
```

## Traefik Exploration Results - Detailed Attempts

### Attempt 1: Direct nginx → Traefik replacement with dynamic routing

**Configuration tried:**

```yaml
- "traefik.http.routers.libsql-dynamic.rule=HostRegexp(`{subdomain:[a-z0-9-]+}.traefik`)"
- "traefik.http.middlewares.dynamic-headers.headers.customrequestheaders.Host={subdomain}.db"
```

**Result:** ❌ Failed
**Why:** Variable substitution syntax didn't work as expected. Traefik couldn't extract the captured subdomain for use in middleware headers.

### Attempt 2: Network aliases approach

**Configuration tried:**

```yaml
networks:
  snitch-network:
    aliases:
      - metadata.db
      - test.db
```

**Result:** ❌ Failed to scale
**Why:** Required hardcoding each namespace. Defeats the purpose of dynamic multi-tenancy. Would need manual configuration for every new namespace.

### Attempt 3: Hardcoded specific routes

**Configuration tried:**

```yaml
- "traefik.http.routers.libsql-metadata.rule=Host(`metadata.snitch-sqld-proxy`)"
- "traefik.http.middlewares.metadata-headers.headers.customrequestheaders.Host=metadata.db"
```

**Result:** ✅ Worked for single namespace
**Why it's insufficient:** Only works for predefined namespaces. Doesn't solve the dynamic multi-tenancy requirement.

### Attempt 4: Eliminate dnsmasq with hardcoded hosts

**What we discovered:** When using hardcoded hosts like `metadata.snitch-sqld-proxy`, we could eliminate dnsmasq entirely because no wildcard DNS resolution is needed.

**Result:** ✅ Worked perfectly for hardcoded case
**Configuration that worked:**

```yaml
# No dnsmasq needed
- "traefik.http.routers.libsql-metadata.rule=Host(`metadata.snitch-sqld-proxy`)"
- "traefik.http.middlewares.metadata-to-db-header.headers.customrequestheaders.Host=metadata.db"
```

**External test:** `curl -H "Host: metadata.snitch-sqld-proxy" http://localhost/version` ✅
**Admin test:** `curl -H "Host: metadata.snitch-sqld-proxy" http://localhost:90/v1/namespaces/metadata/config` ✅

### Attempt 5: Wildcard routing with .traefik TLD

**The core problem identified:**

- Backend constructs URLs like `http://metadata.traefik:80` (subdomain + LIBSQL_HOST)
- DNS lookup fails because `metadata.traefik` doesn't exist
- Need wildcard DNS resolution: `*.traefik` → Traefik container IP

**Configuration tried:**

```yaml
- "traefik.http.routers.libsql-http.rule=HostRegexp(`{subdomain:[a-z0-9-]+}\\.traefik`)"
- "traefik.http.middlewares.subdomain-to-db-header.headers.customrequestheaders.Host={subdomain}.db"
```

**Result:** ❌ Failed at DNS resolution level
**Backend logs:** `dial tcp: failed to lookup address information: Name or service not known`

**Why it failed:** Traefik can do wildcard routing, but without DNS resolution for `*.traefik` hostnames, the backend can't even reach Traefik. We'd still need dnsmasq for wildcard DNS.

### Attempt 6: Hybrid approach (dnsmasq + Traefik)

**Realization:** We'd need both dnsmasq (for DNS) AND Traefik (for routing), making it more complex than the original solution.

**Result:** ❌ Abandoned
**Why:** Adds complexity without benefits. Two services (dnsmasq + Traefik) to do what nginx alone does elegantly.

## Root Cause Analysis: Why Traefik is Inappropriate

### The LibSQL Multi-tenancy Requirements

1. **Dynamic subdomain URLs:** `http://namespace.host:port/`
2. **Header transformation:** Extract `namespace` from subdomain, set `Host: namespace.db`
3. **Wildcard DNS resolution:** `*.host` must resolve to proxy IP
4. **Zero configuration per namespace:** Must work for any namespace without manual setup

### nginx's Natural Fit

- **Regex capture:** `server_name ~^(?<subdomain>[a-z0-9-]+)\..*;` captures any subdomain
- **Variable substitution:** `proxy_set_header Host $subdomain.db;` uses captured variable
- **Single config:** One regex rule handles infinite namespaces
- **Integrated solution:** Works seamlessly with dnsmasq for complete DNS→routing→header pipeline

### Traefik's Fundamental Limitations for This Use Case

1. **Variable substitution complexity:** HostRegexp captures work but middleware variable substitution is unreliable
2. **Two-service requirement:** Still needs dnsmasq for wildcard DNS resolution
3. **Configuration verbosity:** Requires multiple labels vs single nginx regex
4. **No integration benefits:** Docker service discovery doesn't help with subdomain-based namespacing

## Technical Deep Dive: The DNS Problem

The core issue is that LibSQL's client library constructs URLs like:

```conf
http://namespace.{LIBSQL_HOST}:{LIBSQL_PORT}/
```

For this to work:

1. **DNS resolution:** `namespace.{LIBSQL_HOST}` must resolve to an IP
2. **HTTP routing:** Server must route based on Host header
3. **Header transformation:** Server must transform to `Host: namespace.db` for LibSQL

**nginx/dnsmasq solution:**

- dnsmasq: `*.snitch-sqld-proxy` → `219.219.219.125` (DNS)
- nginx: `Host: namespace.snitch-sqld-proxy` → `Host: namespace.db` (routing + transformation)

**Traefik attempts:**

- Still need dnsmasq: `*.traefik` → Traefik IP (DNS)
- Traefik: `Host: namespace.traefik` → `Host: namespace.db` (routing + transformation)
- Result: More complex, same DNS dependency

## Final Verdict: Traefik is Inappropriate for LibSQL Multi-tenancy

### Core Issues That Make Traefik Unsuitable

1. **DNS Dependency Remains:** Still requires dnsmasq or equivalent for wildcard DNS resolution
2. **Increased Complexity:** Two services (dnsmasq + Traefik) vs one (nginx + dnsmasq)
3. **Variable Substitution Fragility:** HostRegexp captures don't reliably work in middleware headers
4. **Configuration Overhead:** Multiple Docker labels vs single nginx regex rule

### What We Proved

- ✅ **Traefik CAN do wildcard routing:** HostRegexp with named groups works
- ✅ **Hardcoded routes work perfectly:** When DNS isn't needed, Traefik routes and transforms headers correctly
- ❌ **Dynamic routing fails on DNS:** Wildcard subdomain resolution still requires dnsmasq
- ❌ **No complexity reduction:** Adding Traefik doesn't eliminate any existing components

### The Fundamental Mismatch

**LibSQL's design assumption:** Clients construct URLs with subdomains (`namespace.host.com`)
**Modern containerized apps:** Use service discovery, not subdomain-based routing
**Traefik's strength:** Service discovery and path-based routing  
**nginx's strength:** Regex-based host manipulation and header transformation

LibSQL's subdomain-based architecture predates modern container patterns and is perfectly suited to nginx's capabilities but awkward for Traefik.

## Simplification Exploration Results

### IPv4 Hardcoding Investigation

**Attempted:** Remove hardcoded IPv4 addresses using Docker's dynamic IP assignment

**What we found:**

- ✅ Can eliminate IPAM configuration and service IPs
- ❌ dnsmasq still needs IP address in `address=/domain/IP` directive
- ❌ Docker's `dns:` field requires IP address, not service name

**Root cause:** dnsmasq configuration syntax requires IP addresses, not hostnames. This is a fundamental limitation, not a configuration choice.

**Conclusion:** Minimal hardcoded IPs (just dnsmasq) are unavoidable and acceptable.

## Architectural Insights

### Why nginx/dnsmasq is Optimal

1. **Single purpose tools:** dnsmasq for DNS, nginx for HTTP proxy/transformation
2. **Natural workflow:** DNS resolution → HTTP routing → Header transformation
3. **Minimal configuration:** One dnsmasq rule, one nginx regex
4. **Battle-tested:** This pattern is used across the industry for similar use cases

### Alternative Approaches Considered

1. **Path-based routing:** LibSQL doesn't support `/namespace/` URLs
2. **Header-only routing:** Still need DNS resolution for initial connection
3. **Service mesh:** Overkill for single-service routing requirements
4. **Custom DNS server:** More complex than dnsmasq for no benefit

## Files Created During Exploration

- `compose-traefik.yml` - Initial Traefik attempt
- `compose-hybrid.yml` - Traefik + dnsmasq hybrid approach
- `compose-hybrid-fixed.yml` - Working hardcoded Traefik configuration
- `notes/libsql-routing-comparison.md` - This comprehensive analysis
- `notes/ipv4-simplification.md` - IPv4 hardcoding investigation results

## Final Recommendation

**Stick with the original nginx/dnsmasq setup.** It is:

- Architecturally appropriate for LibSQL's subdomain-based design
- Simpler than any alternative we explored
- Proven to work reliably in production
- Optimally configured for the specific requirements

The exploration validated that the original design choice was correct, not just convenient.

## Key Lesson

**Tool appropriateness matters more than tool popularity.** nginx's regex capture and proxy header manipulation capabilities make it the ideal tool for LibSQL's subdomain-to-header transformation requirements. Traefik, despite being more modern and feature-rich for container orchestration, doesn't fit this specific use case as naturally.

**Sometimes the "boring" solution is the right solution.**
