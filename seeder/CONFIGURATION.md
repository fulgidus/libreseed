# Seeder Configuration Reference

Complete configuration reference for LibreSeed seeders.

## Configuration File

Seeders are configured via `seeder.yaml` file. Default locations:

1. `./seeder.yaml` (current directory)
2. `/etc/libreseed/seeder.yaml` (system-wide)
3. `~/.config/libreseed/seeder.yaml` (user-specific)

## Complete Example

```yaml
# DHT Configuration
dht:
  # UDP port for DHT communication
  port: 6881
  
  # Bootstrap nodes for initial DHT network join
  bootstrap_nodes:
    - router.bittorrent.com:6881
    - dht.transmissionbt.com:6881
    - router.utorrent.com:6881
  
  # DHT routing table size
  routing_table_size: 1000
  
  # DHT query timeout
  query_timeout: 30s

# Storage Configuration
storage:
  # Directory for stored manifests and files
  data_dir: /var/lib/libreseed/data
  
  # Maximum storage size (supports KB, MB, GB, TB)
  max_size: 100GB
  
  # Retention period for packages (days)
  # Packages older than this will be garbage collected
  retention_days: 90
  
  # Minimum free space to maintain
  min_free_space: 10GB
  
  # Enable compression for stored manifests
  compress_manifests: true

# Maintenance Configuration
maintenance:
  # Interval for re-announcing stored manifests to DHT
  re_announce_interval: 30m
  
  # Interval for integrity checks
  integrity_check_interval: 24h
  
  # Interval for garbage collection
  gc_interval: 6h
  
  # Enable automatic garbage collection
  auto_gc: true

# API Configuration
api:
  # HTTP API port
  http_port: 8080
  
  # Enable metrics endpoint
  enable_metrics: true
  
  # Enable debug endpoints
  debug: false

# Logging Configuration
logging:
  # Log level: debug, info, warn, error
  level: info
  
  # Log format: json, text
  format: json
  
  # Log file path (empty = stdout)
  file: /var/log/libreseed/seeder.log
  
  # Log rotation
  rotate:
    max_size: 100MB
    max_age: 30
    max_backups: 10

# Identity Configuration
identity:
  # Path to seeder Ed25519 keypair
  # If not exists, will be generated
  keypair_file: /var/lib/libreseed/seeder-keypair.pem

# Performance Tuning
performance:
  # Maximum concurrent DHT queries
  max_concurrent_queries: 100
  
  # Maximum concurrent file downloads
  max_concurrent_downloads: 50
  
  # Read buffer size for file serving
  read_buffer_size: 64KB
  
  # DHT message buffer size
  dht_buffer_size: 1024

# Rate Limiting
rate_limit:
  # Maximum manifest uploads per hour from single IP
  uploads_per_hour: 100
  
  # Maximum file downloads per hour from single IP
  downloads_per_hour: 1000
  
  # Enable rate limiting
  enabled: true
```

## Environment Variable Overrides

Configuration can be overridden via environment variables using the pattern `LIBRESEED_<SECTION>_<KEY>`:

```bash
# Override DHT port
export LIBRESEED_DHT_PORT=7881

# Override storage directory
export LIBRESEED_STORAGE_DATA_DIR=/mnt/storage/libreseed

# Override log level
export LIBRESEED_LOGGING_LEVEL=debug
```

## Validation

Validate your configuration:

```bash
libreseed-seeder validate --config seeder.yaml
```

## Configuration Sections

### DHT Configuration

Controls DHT network participation:

- **port**: UDP port for DHT (default: 6881)
- **bootstrap_nodes**: Initial DHT nodes to contact
- **routing_table_size**: Maximum DHT routing table entries
- **query_timeout**: Timeout for DHT queries

### Storage Configuration

Manages local storage:

- **data_dir**: Storage directory path
- **max_size**: Maximum storage size
- **retention_days**: How long to keep packages
- **min_free_space**: Minimum free space to maintain
- **compress_manifests**: Enable manifest compression

### Maintenance Configuration

Controls automated maintenance:

- **re_announce_interval**: How often to re-announce manifests
- **integrity_check_interval**: How often to verify stored data
- **gc_interval**: How often to run garbage collection
- **auto_gc**: Enable automatic garbage collection

### API Configuration

HTTP API settings:

- **http_port**: HTTP API port
- **enable_metrics**: Expose Prometheus metrics
- **debug**: Enable debug endpoints
- **tls**: Optional TLS configuration
- **auth**: Optional authentication

### Logging Configuration

Logging settings:

- **level**: Log verbosity (debug, info, warn, error)
- **format**: Log format (json, text)
- **file**: Log file path (empty = stdout)
- **rotate**: Log rotation settings

### Identity Configuration

Seeder identity:

- **keypair_file**: Path to Ed25519 keypair (auto-generated if missing)

### Performance Tuning

Performance optimization:

- **max_concurrent_queries**: DHT query concurrency limit
- **max_concurrent_downloads**: File download concurrency limit
- **read_buffer_size**: File serving buffer size
- **dht_buffer_size**: DHT message buffer size

### Rate Limiting

DOS protection:

- **uploads_per_hour**: Max uploads per IP per hour
- **downloads_per_hour**: Max downloads per IP per hour
- **enabled**: Enable rate limiting
