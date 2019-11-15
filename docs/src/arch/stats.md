# Stats

Samaritan provide detailed metrics about connection and request which could help us understand the state of network and services, make decisions.

> Types of statistics:
>
> - Counters: Unsigned integers that only increase and never decrease.
>
> - Gauges: Unsigned integers that both increase and decrease.
>
> - Histograms: Unsigned integers that are part of a stream of values that are then aggregated by the collector to ultimately yield summarized percentile values.

## Global

| Name | Type  | Description |
|------|-------|-------------|
| live | Gauge | live status |

## Runtime

| Name                        | Type      | Description                                            |
|-----------------------------|-----------|--------------------------------------------------------|
| runtime.gc_total            | Counter   | Count of GC                                            |
| runtime.gc_pause_us         | Histogram | GC pause duration                                      |
| runtime.goroutines          | Gauge     | Num of goroutines                                      |
| runtime.gc_pause_total_ms   | Gauge     | Total gc pause time                                    |
| runtime.alloc_bytes         | Gauge     | Alloc is bytes of allocated heap objects               |
| runtime.total_alloc_bytes   | Gauge     | Cumulative bytes allocated for heap objects            |
| runtime.sys_bytes           | Gauge     | Total bytes of memory obtained from the OS             |
| runtime.heap_alloc_bytes    | Gauge     | HeapAlloc is bytes of allocated heap objects           |
| runtime.heap_sys_bytes      | Gauge     | Bytes of heap memory obtained from the OS              |
| runtime.heap_idle_bytes     | Gauge     | Bytes in idle (unused) spans                           |
| runtime.heap_inuse_bytes    | Gauge     | Bytes in in-use spans                                  |
| runtime.heap_released_bytes | Gauge     | Bytes of physical memory returned to the OS            |
| runtime.heap_objects        | Gauge     | The number of allocated heap objects                   |
| runtime.stack_inuse_bytes   | Gauge     | Bytes in stack spans                                   |
| runtime.stack_sys_bytes     | Gauge     | Bytes of stack memory obtained from the OS             |
| runtime.lookups             | Gauge     | The number of pointer lookups performed by the runtime |
| runtime.mallocs             | Gauge     | The cumulative count of heap objects allocated         |
| runtime.frees               | Gauge     | The cumulative count of heap objects freed             |

## Service

### Common

#### L4

prefix: service.{service_name}.

| Name                         | Type      | Description                                |
|------------------------------|-----------|--------------------------------------------|
| downstream.cx_total          | Counter   | downstream total connections               |
| downstream.cx_destroy_total  | Counter   | downstream destroyed connections           |
| downstream.cx_active         | Gauge     | downstream active connections              |
| downstream.cx_length_sec     | Histogram | downstream connection length               |
| downstream.cx_rx_bytes_total | Counter   | downstream received connection bytes       |
| downstream.cx_tx_bytes_total | Counter   | downstream sent connection bytes           |
| downstream.cx_restricted     | Counter   | downstream restricted connections          |
| upstream.cx_total            | Counter   | upstream total connections                 |
| upstream.cx_destroy_total    | Counter   | upstream destroyed connections             |
| upstream.cx_active           | Counter   | upstream active connections                |
| upstream.cx_length_sec       | Histogram | upstream connection length                 |
| upstream.cx_rx_bytes_total   | Counter   | upstream received connection bytes         |
| upstream.cx_tx_bytes_total   | Counter   | upstream sent connection bytes             |
| upstream.cx_connect_timeout  | Counter   | upstream total connection connect timeouts |
| upstream.cx_connect_fail     | Counter   | upstream connection failures               |

#### L7

prefix: service.{service_name}.

| Name                          | Type      | Description                              |
|-------------------------------|-----------|------------------------------------------|
| downstream.rq_total           | Counter   | downstream total request                 |
| downstream.rq_success_total   | Counter   | downstream success request               |
| downstream.rq_failure_total   | Counter   | downstream failed request                |
| downstream.rq_active          | Gauge     | downstream active request                |
| downstream.rq_duration_ms     | Histogram | downstream request duration              |
| downstream.rq_rx_bytes_length | Histogram | downstream received request bytes length |
| downstream.rq_tx_bytes_length | Histogram | downstream sent request bytes length     |
| upstream.rq_total             | Counter   | upstream total request                   |
| upstream.rq_success_total     | Counter   | upstream success request                 |
| upstream.rq_failure_total     | Counter   | upstream failed request                  |
| upstream.rq_active            | Gauge     | upstream active request                  |
| upstream.rq_duration_ms       | Histogram | upstream request duration                |
| upstream.rq_rx_bytes_length   | Histogram | upstream received request bytes length   |
| upstream.rq_tx_bytes_length   | Histogram | upstream sent request bytes length       |

### Redis

prefix: service.{service_name}.

#### global

| Name          | Type    | Description                                            |
|---------------|---------|--------------------------------------------------------|
| rq_slow_total | Counter | slow request count, default latency threshold is 50ms |

#### per command

| Name                           | Type      | Description               |
|--------------------------------|-----------|---------------------------|
| redis.{command}.total          | Counter   | command count             |
| redis.{command}.success        | Counter   | command success count     |
| redis.{command}.error          | Counter   | command error count       |
| redis.{command}.latency_micros | Histogram | command latency in micros |

#### upstream

| Name                                 | Type    | Description                 |
|--------------------------------------|---------|-----------------------------|
| upstream.moved                       | Counter | moved request count         |
| upstream.slots_refresh.total         | Counter | slots refresh total count   |
| upstream.slots_refresh.success_total | Counter | slots refresh success count |
| upstream.slots_refresh.failure_total | Counter | slots refresh failure count |
