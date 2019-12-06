# Getting Started

This section describes how to get started quickly and provides some sample configurations.

## Installation

### Pre-built binaries

It's the fastest way, you can choose the appropriate binaries from [release page].

Assuming your os system is darwin and platform is amd64, you can use the following command to download and install.

```shell
$ VERSION=1.0.0 OS=darwin ARCH=amd64 curl -fsSL \
    https://github.com/samaritan-proxy/samaritan/releases/download/v$VERSION/samaritan$VERSION.$OS-$ARCH.tar.gz \
    -o /tmp/samaritan-$VERSION.tar.gz && \
    tar -zxvf /tmp/samaritan-$VERSION.tar.gz && \
    install /tmp/samaritan-$VERSION/samaritan /usr/local/bin
```

!!! note
    You should replace the version, os system and arch information in download link as needed.

### Manually build

If the pre-built binaries doesn't meet your needs or want to try the lastest features which are not released, you could build
it from the source code. This first needs *[Go]* installed(version 1.13+ is required).

!!! warning
    The master branch is in continuous development and has not undergone the extensive testing and verification. There may be
    some unknown problems, please do not use it directly for production.

```shell
$ git clone https://github.com/samaritan-proxy/samaritan.git
$ cd samaritan && make
$ install ./bin/samaritan /usr/local/bin
```

## Running simple example

The following is an example of using Sam as a L4 layer TCP proxy to access [example.com]

Terminal 1
```shell
$ # resolve example.com, get ip
$ ip=$(dig +short example.com | head -1)
$ # generate config file
$ sudo bash -c 'cat > /etc/samaritan.yaml' <<EOF
admin:
  bind:
    ip: 127.0.0.1
    port: 12345
log:
  level: INFO
static_services:
  - name: exmaple.com
    config:
      listener:
        address:
          ip: 0.0.0.0
          port: 8080
      protocol: TCP
    endpoints:
      - address:
          ip: $ip
          port: 80
EOF
$ # start samaritan
$ samaritan
I 2019/10/18 15:10:38.067913 samaritan.go:60: stats config is empty
I 2019/10/18 15:10:38.068501 samaritan.go:184: Using CPUs: 4
I 2019/10/18 15:10:38.068513 samaritan.go:185: Version: 1.0.0
D 2019/10/18 15:10:38.068579 config.go:114: Disable dynamic config
I 2019/10/18 15:10:38.070397 server.go:39: Starting HTTP server at 0.0.0.0:12345
I 2019/10/18 15:10:38.070534 samaritan.go:89: PID: 60289
I 2019/10/18 15:10:38.072735 monitor.go:48: health check config is null, healthy check will disable
I 2019/10/18 15:10:38.074646 controller.go:110: Add processor exmaple
I 2019/10/18 15:10:38.093603 listener.go:105: [exmaple] start serving at 0.0.0.0:8080
```

!!! tip
    You could store the config file anywhere you like and can find.

Terminal 2
```shell
$ curl -v -I -H "Host: example.com" http://127.0.0.1:8080
* Rebuilt URL to: http://127.0.0.1:8080/
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 8080 (#0)
> HEAD / HTTP/1.1
> Host: example.com
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
HTTP/1.1 200 OK
< Accept-Ranges: bytes
Accept-Ranges: bytes
< Cache-Control: max-age=604800
Cache-Control: max-age=604800
< Content-Type: text/html; charset=UTF-8
Content-Type: text/html; charset=UTF-8
< Date: Fri, 18 Oct 2019 07:26:11 GMT
Date: Fri, 18 Oct 2019 07:26:11 GMT
< Etag: "3147526947+gzip+ident"
Etag: "3147526947+gzip+ident"
< Expires: Fri, 25 Oct 2019 07:26:11 GMT
Expires: Fri, 25 Oct 2019 07:26:11 GMT
< Last-Modified: Thu, 17 Oct 2019 07:18:26 GMT
Last-Modified: Thu, 17 Oct 2019 07:18:26 GMT
< Server: ECS (sjc/4E74)
Server: ECS (sjc/4E74)
< X-Cache: HIT
X-Cache: HIT
< Content-Length: 1256
Content-Length: 1256

<
* Connection #0 to host 127.0.0.1 left intact
```

As you can see, we visited [example.com] via Sam.

## Configuration

In the aboved section, we generate a simple configuration file and start Sam with it. But you may have no idea about
it, you may be overwhelmed. For better understanding, we will show you the specific parts of the configuration file.

### instance

The [instance message] is used to specify the environment in which the instance is located, and will be reported to
the management server.

```yaml
instance:
  id: ip_port
  belong: sash
```

!!! warning
    If you want to use the dynamic config feature, you must add this section to configuration file.

### log

The [log message] is used to specify the log level and output destination.

```yaml
log:
  level: INFO
  output:
    type: SYSLOG
    target: 127.0.0.1:514
```

If it is omitted, the log level will be *INFO* and the message will be printed to terminal.

!!! question "Why does the output not support files"
    - If the disk io hang, the whole process will probably hang when not handled correctly, we don't want this to happen.
    - There are a lot mature toolkit to process logs in linux system, such as ryslog and logrotate.
      We should reuse them instead of creating ourselves.

### stats

The [stats message] is used to specify the stats config include internal metrics config and sinks.

```yaml
stats:
  sinks:
    - type: STATSD
      endpoint: example.com:8125
```

If it is omitted, the internal metrics wouldn't flush to any *TSDB* except [Prometheus] which could still be visited via `/amin/stats/prometheus`.
[Prometheus] is sufficient for most users which means you could omit the stats config directly.

### admin

The [admin message] is used to specify the configuration for local admin api.

```yaml
admin:
  bind:
    ip: 127.0.0.1
    port: 12345
```

It's required, otherwise the entire process cannot be started. More details about admin api, you can view [doc][admin api]

### static services

The [static service message] is used to specify the static service including name, proxy strategy and endpoints.

```yaml
static_services:
  - name: example.com
    config:
      listener:
        address:
          ip: 127.0.0.1
          port: 8080
       protocol: TCP
    endpoints:
      - address:
          ip: 93.184.216.34
          port: 80
```

You can use it to configure the proxy's services in a local development environment, but it is not recommended.
Here are the reasons:

1. Contains a lot configuraton items, easy to make mistakes when writing by hand.
2. Unable to support large-scale operation and maintenance, imagine how to update the configuration if deployed to a 10k machines. It's so horrible.

The correct and recommended way is to use dynamic source, which will be described in detail below.

### dynamic source

The [dynamic source message] is used to specify the dynamic source configuration.

```yaml
dynamic_source_config:
  endpoint: example.com:80
```

Once the dynamic source is configured, Sam could know and apply the changes of instance's dependent services, service's proxy policy and service's endpoints in real time.
The communication way between them is grpc, and [here][discovery service message] is the specific interface definition.

!!! tip
    As the management server, **Sash** will implement all the interface required by dynamic source.
    It is still private now, but we will open source as soon as possible.

## Examples

We have created some sandboxes using [Docker] and [Docker Compose] to show how to use Samaritan. To run them, please make sure you have docker and docker-compose installed.

### TCP

In this example, we show how to use Samaritan as a L4 proxy.

**Step 1: Start all containers including http-server and proxy**

```sh
$ git clone --depth 1 https://github.com/samaritan-proxy/samaritan.git /tmp/samaritan
$ cd /tmp/samaritan/examples/tcp
$ docker-compose up
```

**Step 2: View http://127.0.0.1 using curl or browser**

```sh
$ curl http://127.0.0.1
```

You could see the response: `hello, world!`

### Redis

TBD


[release page]: https://github.com/samaritan-proxy/samaritan/releases
[Go]: https://golang.org
[example.com]: http://example.com
[Prometheus]: https://prometheus.io
[gRPC]: https://grpc.io
[Docker]: https://docs.docker.com/
[Docker Compose]: https://docs.docker.com/compose/

[admin api]:

[instance message]: proto-ref.md#instance
[log message]: proto-ref.md#log
[stats message]: proto-ref.md#stats
[admin message]: proto-ref.md#admin
[static service message]: proto-ref.md#staticservice
[dynamic source message]: proto-ref.md#bootstrap.ConfigSource
[discovery service message]: proto-ref.md#discoveryservice
