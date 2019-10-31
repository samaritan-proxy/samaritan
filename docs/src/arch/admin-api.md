# Admin API

## `GET` `/config`

Return all configuration info.

```json
{
  "admin": {
    "bind": {
      "ip": "0.0.0.0",
      "port": 12345
    }
  },
  "services": {
    "example.service": {
      "name": "example.service",
      "config": {
        "listener": {
          "address": {
            "ip": "0.0.0.0",
            "port": 8000
          }
        },
        "healthCheck": {
          "interval": "10s",
          "timeout": "3s",
          "fallThreshold": 3,
          "riseThreshold": 3,
          "tcpChecker": {}
        },
        "connectTimeout": "3s",
        "idleTimeout": "600s",
        "lbPolicy": "LEAST_CONNECTION",
        "protocol": "TCP"
      },
      "endpoints": [
        {
          "address": {
            "ip": "10.0.0.1",
            "port": 8000
          }
        },
        {
          "address": {
            "ip": "10.0.0.2",
            "port": 8000
          }
        }
      ]
    }
  }
}
```

## `GET` `/ops/shutdown`

Immediately shut down the Samaritan.

```json
{"msg": "OK"}
```

## `GET` `/stats`

Return all statistics for local debugging.

```text
live: 1
runtime.alloc_bytes: 2222568
runtime.frees: 23281
runtime.gc_pause_total_ms: 0
runtime.gc_total: 0
runtime.goroutines: 10
runtime.heap_alloc_bytes: 2222568
runtime.heap_idle_bytes: 62750720
...
```

## `GET` `/stats/prometheus`

Return /stats in [Prometheus format](https://prometheus.io/docs/instrumenting/exposition_formats/).

```text
# TYPE samaritan_service_downstream_cx_active gauge
samaritan_service_downstream_cx_active{hostname="localhost",service_name="service_01"} 0
# TYPE samaritan_service_upstream_cx_active gauge
samaritan_service_upstream_cx_active{hostname="localhost",service_name="service_01"} 0
# TYPE samaritan_live gauge
samaritan_live{hostname="localhost"} 1
...
```
