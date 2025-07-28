# Traefik + CoreDNS Investigation

## Motivation

While our Traefik exploration concluded that nginx/dnsmasq was superior, there's a production best practices concern: **dnsmasq is often considered inappropriate for production environments**.

Industry preference:

- **dnsmasq**: Development/testing DNS server
- **CoreDNS**: Production-grade DNS server (used by Kubernetes)

## Traefik + CoreDNS Approach

### Concept

- **CoreDNS**: Handle wildcard DNS resolution (`*.traefik` → Traefik IP)
- **Traefik**: Handle HTTP routing and header transformation (`Host: namespace.traefik` → `Host: namespace.db`)

### Advantages Over dnsmasq

1. **Production-ready**: CoreDNS is designed for production workloads
2. **Cloud-native**: Standard DNS server in Kubernetes environments
3. **Extensible**: Plugin architecture for advanced DNS features
4. **Monitoring**: Better observability and metrics
5. **Performance**: Optimized for high-throughput scenarios

### Configuration Approach

#### CoreDNS Configuration (Corefile)

```conf
.:53 {
    # Wildcard DNS for *.traefik domains
    template IN A traefik {
        match "^([^.]+)\.traefik\.$"
        answer "{{ .Name }} 60 IN A 172.20.0.3"
    }

    # Forward other queries to upstream DNS
    forward . 1.1.1.1 1.0.0.1

    # Logging and health
    log
    health
}
```

#### Traefik Configuration (same as before)

```yaml
labels:
  - "traefik.http.routers.libsql-http.rule=HostRegexp(`{subdomain:[a-z0-9-]+}\\.traefik`)"
  - "traefik.http.routers.libsql-http.middlewares=subdomain-to-db-header"
  - "traefik.http.middlewares.subdomain-to-db-header.headers.customrequestheaders.Host={subdomain}.db"
```

## Potential Benefits

### Production Readiness

- **Scalability**: CoreDNS handles high query volumes better than dnsmasq
- **Reliability**: Better error handling and recovery mechanisms
- **Security**: More security features and regular security updates
- **Compliance**: Often required in enterprise/regulated environments

### Operational Benefits

- **Metrics**: Prometheus metrics out of the box
- **Logging**: Structured logging for better observability
- **Health checks**: Built-in health endpoints
- **Configuration reloading**: Hot reload without container restart

### Architecture Cleanliness

- **Separation of concerns**: DNS server vs HTTP proxy are distinct, production-grade tools
- **Standard tooling**: Both Traefik and CoreDNS are CNCF projects
- **Kubernetes compatibility**: Easy migration to K8s later if needed

## Implementation Questions

1. **CoreDNS template plugin**: Does the wildcard template syntax work as expected?
2. **Variable substitution**: Do Traefik's HostRegexp variables work reliably in production?
3. **Performance**: Is the DNS → HTTP routing overhead acceptable?
4. **Complexity**: Is the operational overhead worth the production benefits?

## Implementation Results

### ✅ CoreDNS Success

**DNS Resolution:** CoreDNS template plugin works perfectly for wildcard DNS

**Configuration that works:**

```conf
.:53 {
    template IN A traefik {
        match "^([^.]+)\.traefik\.$"
        answer "{{ .Name }} 60 IN A 219.219.219.125"
        fallthrough
    }
    forward . 1.1.1.1 1.0.0.1
}
```

**Test results:**

```bash
$ dig @127.0.0.1 -p 9053 metadata.traefik
metadata.traefik. 60 IN A 219.219.219.125

$ dig @127.0.0.1 -p 9053 test.traefik
test.traefik. 60 IN A 219.219.219.125
```

### ❌ Traefik Variable Substitution Still Fails

**Backend connectivity:** ✅ DNS resolution works, backend connects to Traefik
**Header transformation:** ❌ Variable substitution `{subdomain}.db` doesn't work reliably

**Backend logs:**

```bash
ERROR Failed creating metadata namespace Error="unexpected status: 404 Not Found"
```

This indicates Traefik is receiving requests but the `Host: {subdomain}.db` transformation isn't working.

## Production Viability Assessment

### What We Proved

1. **✅ CoreDNS is production-ready**: Wildcard DNS resolution works perfectly
2. **✅ CoreDNS template plugin is suitable**: Handles dynamic subdomain → IP mapping
3. **❌ Traefik variable substitution remains unreliable**: Same core issue as before
4. **✅ Architecture separation is clean**: DNS server and HTTP proxy are distinct

### Core Problem Remains Unsolved

The fundamental issue persists: **Traefik's HostRegexp variable substitution in middleware headers doesn't work reliably**. This is the same limitation we discovered in our original Traefik investigation.

## Alternative Production-Grade Solutions

### Option 1: CoreDNS + nginx (Recommended)

**Concept:** Use CoreDNS for production DNS, keep nginx for proven header transformation

**Benefits:**

- ✅ Production-grade DNS server (CoreDNS)
- ✅ Proven header transformation (nginx regex)
- ✅ Best of both worlds approach
- ✅ Minimal changes to working system

### Option 2: Bind9 + nginx

**Concept:** Use Bind9 (enterprise DNS server) instead of dnsmasq

**Benefits:**

- ✅ Enterprise-grade DNS server
- ✅ Proven nginx header transformation
- ❌ More complex configuration than CoreDNS

### Option 3: Accept dnsmasq for simplicity

**Concept:** Document why dnsmasq is acceptable for this use case

**Benefits:**

- ✅ Simplest configuration
- ✅ Proven to work reliably
- ✅ Minimal resource usage
- ❌ Not considered "production-grade"

## Final Recommendation

### Use CoreDNS + nginx instead of dnsmasq + nginx

This provides:

1. **Production-grade DNS:** CoreDNS is the industry standard
2. **Proven routing:** nginx regex capture and header transformation
3. **Minimal risk:** Only DNS server changes, routing stays the same
4. **Easy migration:** Drop-in replacement for dnsmasq

## Configuration Example

```yaml
# CoreDNS replaces dnsmasq
coredns:
  image: coredns/coredns:1.11.1
  volumes:
    - ./coredns/Corefile:/etc/coredns/Corefile:ro

# nginx stays the same (proven to work)
nginx:
  image: nginx:alpine
  volumes:
    - ./proxy/nginx.conf:/etc/nginx/nginx.conf:ro
```

This gives you production-grade infrastructure while keeping the working nginx proxy layer.
