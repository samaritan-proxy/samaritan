# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [config/bootstrap/bootstrap.proto](#config/bootstrap/bootstrap.proto)
    - [Admin](#bootstrap.Admin)
    - [Bootstrap](#bootstrap.Bootstrap)
    - [ConfigSource](#bootstrap.ConfigSource)
    - [Log](#bootstrap.Log)
    - [Log.Output](#bootstrap.Log.Output)
    - [Sink](#bootstrap.Sink)
    - [StaticService](#bootstrap.StaticService)
    - [Stats](#bootstrap.Stats)
  
    - [Log.Level](#bootstrap.Log.Level)
    - [Log.Output.Type](#bootstrap.Log.Output.Type)
    - [Sink.Type](#bootstrap.Sink.Type)
  
  
  

- [config/hc/hc.proto](#config/hc/hc.proto)
    - [ATCPChecker](#hc.ATCPChecker)
    - [ATCPChecker.Action](#hc.ATCPChecker.Action)
    - [HealthCheck](#hc.HealthCheck)
    - [MySQLChecker](#hc.MySQLChecker)
    - [RedisChecker](#hc.RedisChecker)
    - [TCPChecker](#hc.TCPChecker)
  
  
  
  

- [config/protocol/redis/redis.proto](#config/protocol/redis/redis.proto)
    - [Compression](#redis.Compression)
  
    - [Compression.Method](#redis.Compression.Method)
    - [ReadStrategy](#redis.ReadStrategy)
  
  
  

- [config/protocol/protocol.proto](#config/protocol/protocol.proto)
    - [MySQLOption](#protocol.MySQLOption)
    - [RedisOption](#protocol.RedisOption)
    - [TCPOption](#protocol.TCPOption)
  
    - [Protocol](#protocol.Protocol)
  
  
  

- [config/service/service.proto](#config/service/service.proto)
    - [Endpoint](#service.Endpoint)
    - [Service](#service.Service)
  
    - [Endpoint.State](#service.Endpoint.State)
    - [Endpoint.Type](#service.Endpoint.Type)
  
  
  

- [config/service/config.proto](#config/service/config.proto)
    - [Config](#service.Config)
    - [Listener](#service.Listener)
  
    - [LoadBalancePolicy](#service.LoadBalancePolicy)
  
  
  

- [common/instance.proto](#common/instance.proto)
    - [Instance](#common.Instance)
  
  
  
  

- [common/address.proto](#common/address.proto)
    - [Address](#common.Address)
  
  
  
  

- [api/discovery.proto](#api/discovery.proto)
    - [SvcConfigDiscoveryRequest](#api.SvcConfigDiscoveryRequest)
    - [SvcConfigDiscoveryResponse](#api.SvcConfigDiscoveryResponse)
    - [SvcConfigDiscoveryResponse.UpdatedEntry](#api.SvcConfigDiscoveryResponse.UpdatedEntry)
    - [SvcDiscoveryRequest](#api.SvcDiscoveryRequest)
    - [SvcDiscoveryResponse](#api.SvcDiscoveryResponse)
    - [SvcEndpointDiscoveryRequest](#api.SvcEndpointDiscoveryRequest)
    - [SvcEndpointDiscoveryResponse](#api.SvcEndpointDiscoveryResponse)
  
  
  
    - [DiscoveryService](#api.DiscoveryService)
  

- [Scalar Value Types](#scalar-value-types)



<a name="config/bootstrap/bootstrap.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## config/bootstrap/bootstrap.proto



<a name="bootstrap.Admin"></a>

### Admin
The admin message is required to configure the administration server.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bind | [common.Address](#common.Address) |  | The TCP address that the administration server will listen on, not null. |






<a name="bootstrap.Bootstrap"></a>

### Bootstrap
This message is supplied via &#39;-config&#39; cli flag and act as the
root of configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [common.Instance](#common.Instance) |  | The instance contains the meta information of the current instance. |
| log | [Log](#bootstrap.Log) |  | Configuration for log. |
| stats | [Stats](#bootstrap.Stats) |  | Configuration for stats. |
| admin | [Admin](#bootstrap.Admin) |  | Configuration for the local administration HTTP server. |
| static_services | [StaticService](#bootstrap.StaticService) | repeated | Statically specified services. |
| dynamic_source_config | [ConfigSource](#bootstrap.ConfigSource) |  | Configuration for dynamic source config. |






<a name="bootstrap.ConfigSource"></a>

### ConfigSource
Configuration for dynamic source config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) |  | The gRPC endpoint of dynamic source config service, not null. |






<a name="bootstrap.Log"></a>

### Log
Log config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [Log.Level](#bootstrap.Log.Level) |  | Logging level |
| output | [Log.Output](#bootstrap.Log.Output) |  | Output target configuration, support send log to stdout or syslog, use stdout as default. |






<a name="bootstrap.Log.Output"></a>

### Log.Output



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [Log.Output.Type](#bootstrap.Log.Output.Type) |  | Type of output target. |
| target | [string](#string) |  | Address of server which is required when `SYSLOG` is selected. |






<a name="bootstrap.Sink"></a>

### Sink
Sink is a sink for stats. Each Sink is responsible for writing stats
to a backing store.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [Sink.Type](#bootstrap.Sink.Type) |  |  |
| endpoint | [string](#string) |  | Sink endpoint, not empty. |






<a name="bootstrap.StaticService"></a>

### StaticService
The wrapper of service config and endpoints.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of service. |
| config | [service.Config](#service.Config) |  | The proxy strategy of service. |
| endpoints | [service.Endpoint](#service.Endpoint) | repeated | The endpoints of service. Need at least one endpoint. |






<a name="bootstrap.Stats"></a>

### Stats
Stats config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sinks | [Sink](#bootstrap.Sink) | repeated | Optional set of stats sinks. |





 


<a name="bootstrap.Log.Level"></a>

### Log.Level


| Name | Number | Description |
| ---- | ------ | ----------- |
| INFO | 0 | Print base messages during running. This is in addition to warnings and errors. |
| DEBUG | -1 | Print everything, including debugging information. |
| WARING | 1 | Print all warnings and errors. |
| ERROR | 2 | Print all errors. |
| QUIET | 3 | Print nothing. |



<a name="bootstrap.Log.Output.Type"></a>

### Log.Output.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| STDOUT | 0 | stdout |
| SYSLOG | 1 | syslog |



<a name="bootstrap.Sink.Type"></a>

### Sink.Type
Sink type

| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 |  |
| STATSD | 1 | statsd |


 

 

 



<a name="config/hc/hc.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## config/hc/hc.proto



<a name="hc.ATCPChecker"></a>

### ATCPChecker
ATCP checker config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [ATCPChecker.Action](#hc.ATCPChecker.Action) | repeated | List of actions. All actions will execute during the health check, if one of the actions fails, this health check will be considered as failed. Need at least one action. |






<a name="hc.ATCPChecker.Action"></a>

### ATCPChecker.Action
Action represents a set of requests from Samaritan to the server
and what the expected server returns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| send | [bytes](#bytes) |  | This is used to send a data along with a connection opening. |
| expect | [bytes](#bytes) |  | Expecting content returned from the server. |






<a name="hc.HealthCheck"></a>

### HealthCheck
Configuration of health check.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| interval | [google.protobuf.Duration](#google.protobuf.Duration) |  | The interval between two consecutive health checks. Must be greater than 0s. |
| timeout | [google.protobuf.Duration](#google.protobuf.Duration) |  | The timeout when doing a health check. Must be greater than 0s. |
| fall_threshold | [uint32](#uint32) |  | A server will be considered as dead after # consecutive unsuccessful health checks. Must be greater than 0. |
| rise_threshold | [uint32](#uint32) |  | A server will be considered as operational after # consecutive successful health checks. Must be greater than 0. |
| tcp_checker | [TCPChecker](#hc.TCPChecker) |  |  |
| atcp_checker | [ATCPChecker](#hc.ATCPChecker) |  |  |
| mysql_checker | [MySQLChecker](#hc.MySQLChecker) |  |  |
| redis_checker | [RedisChecker](#hc.RedisChecker) |  |  |






<a name="hc.MySQLChecker"></a>

### MySQLChecker
MySQL checker config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| username | [string](#string) |  | MySQL server username, not null. |






<a name="hc.RedisChecker"></a>

### RedisChecker
Redis checker config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| password | [string](#string) |  | Redis server password, if the password is not empty, the AUTH command will be sent before the PING command. |






<a name="hc.TCPChecker"></a>

### TCPChecker
TCP checker config.





 

 

 

 



<a name="config/protocol/redis/redis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## config/protocol/redis/redis.proto



<a name="redis.Compression"></a>

### Compression
Configuration of compression.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enable | [bool](#bool) |  | Switch of compress, default is off. NOTE: Uncompress will always work. |
| method | [Compression.Method](#redis.Compression.Method) |  | Compression algorithm used in compression filter. |
| threshold | [uint32](#uint32) |  | Value will be ignored when byte length of value is less than the threshold, must be greater than 0. |





 


<a name="redis.Compression.Method"></a>

### Compression.Method


| Name | Number | Description |
| ---- | ------ | ----------- |
| SNAPPY | 0 |  |
| MOCK | 255 |  |



<a name="redis.ReadStrategy"></a>

### ReadStrategy
Strategy of a read only command.

| Name | Number | Description |
| ---- | ------ | ----------- |
| MASTER | 0 | Read from master nodes. |
| SLAVE | 1 | Read from slave nodes. |
| BOTH | 2 | Read from all nodes. |


 

 

 



<a name="config/protocol/protocol.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## config/protocol/protocol.proto



<a name="protocol.MySQLOption"></a>

### MySQLOption
MySQL protocol option.






<a name="protocol.RedisOption"></a>

### RedisOption
Redis protocol option.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| read_strategy | [redis.ReadStrategy](#redis.ReadStrategy) |  | Strategy of a read only command. |
| compression | [redis.Compression](#redis.Compression) |  | Configuration of compression. |






<a name="protocol.TCPOption"></a>

### TCPOption
TCP protocol option.





 


<a name="protocol.Protocol"></a>

### Protocol
Protocol enum.

| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 |  |
| TCP | 1 | TCP |
| MySQL | 2 | MySQL |
| Redis | 3 | Redis |


 

 

 



<a name="config/service/service.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## config/service/service.proto



<a name="service.Endpoint"></a>

### Endpoint
Endpoint represents an endpoint of service.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [common.Address](#common.Address) |  | Address of endpoint, must be set. |
| state | [Endpoint.State](#service.Endpoint.State) |  | Healthy state of endpoint. When state is DOWN, this host will not be selected for load balancing. |
| type | [Endpoint.Type](#service.Endpoint.Type) |  | Type of endpoints. When all hosts whose type is main are in the DOWN state, the host whose type is backup will be selected. |






<a name="service.Service"></a>

### Service
Service represents a service.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of service. |





 


<a name="service.Endpoint.State"></a>

### Endpoint.State


| Name | Number | Description |
| ---- | ------ | ----------- |
| UP | 0 | healthy |
| DOWN | 1 | unhealthy |
| UNKNOWN | 2 | unknown |



<a name="service.Endpoint.Type"></a>

### Endpoint.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| MAIN | 0 | main |
| BACKUP | 1 | backup |


 

 

 



<a name="config/service/config.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## config/service/config.proto



<a name="service.Config"></a>

### Config
Configuration of service,
contains configuration information required for the processor to run.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| listener | [Listener](#service.Listener) |  | Listener config, must be set. |
| health_check | [hc.HealthCheck](#hc.HealthCheck) |  | Health check will be disabled when not defined. |
| connect_timeout | [google.protobuf.Duration](#google.protobuf.Duration) |  | The maximum time to wait for a connection attempt to a server to succeed, default is 3s. |
| idle_timeout | [google.protobuf.Duration](#google.protobuf.Duration) |  | The maximum inactivity time on the client side, default is 10min. |
| lb_policy | [LoadBalancePolicy](#service.LoadBalancePolicy) |  |  |
| protocol | [protocol.Protocol](#protocol.Protocol) |  | Protocol of service, can not be UNKNOWN. |
| tcp_option | [protocol.TCPOption](#protocol.TCPOption) |  |  |
| redis_option | [protocol.RedisOption](#protocol.RedisOption) |  |  |
| mysql_option | [protocol.MySQLOption](#protocol.MySQLOption) |  |  |






<a name="service.Listener"></a>

### Listener
Listener configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [common.Address](#common.Address) |  | The address that the listener should listen on, must be set. |
| connection_limit | [uint32](#uint32) |  | The limit of connections, new connections that exceed this value are immediately be closed. Default value is 0 that the limit will be disable. |





 


<a name="service.LoadBalancePolicy"></a>

### LoadBalancePolicy
Load balance policy.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ROUND_ROBIN | 0 | RoundRobin |
| LEAST_CONNECTION | 1 | LeastConnection |
| RANDOM | 2 | Random |
| CLUSTER_PROVIDED | 3 | Provided by redis cluster |


 

 

 



<a name="common/instance.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## common/instance.proto



<a name="common.Instance"></a>

### Instance



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | The id of this instance, if id is not define, it will generate by ip and admin port. |
| version | [string](#string) |  | Version of this instance running now. This field is automatically populated by Samaritan does not require to specify by user. |
| belong | [string](#string) |  | The service name which this instance belongs to. It&#39;s required when you want to change the behavior of Sam at runtime, such as update proxy policy, update service endpoints, etc. If two instances belong to the same service, it will be treated in one group. |





 

 

 

 



<a name="common/address.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## common/address.proto



<a name="common.Address"></a>

### Address



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ip | [string](#string) |  | IP address, IPv4 or IPv6. |
| port | [uint32](#uint32) |  | Port, [0, 65535]. |





 

 

 

 



<a name="api/discovery.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/discovery.proto



<a name="api.SvcConfigDiscoveryRequest"></a>

### SvcConfigDiscoveryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| svc_names_subscribe | [string](#string) | repeated | All service names that need to subscribe. |
| svc_names_unsubscribe | [string](#string) | repeated | All service names that need to unsubscribe. |






<a name="api.SvcConfigDiscoveryResponse"></a>

### SvcConfigDiscoveryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| updated | [SvcConfigDiscoveryResponse.UpdatedEntry](#api.SvcConfigDiscoveryResponse.UpdatedEntry) | repeated | Update of configuration. |






<a name="api.SvcConfigDiscoveryResponse.UpdatedEntry"></a>

### SvcConfigDiscoveryResponse.UpdatedEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [service.Config](#service.Config) |  |  |






<a name="api.SvcDiscoveryRequest"></a>

### SvcDiscoveryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [common.Instance](#common.Instance) |  | Meta information of the current instance. |






<a name="api.SvcDiscoveryResponse"></a>

### SvcDiscoveryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| added | [service.Service](#service.Service) | repeated | Added service. |
| removed | [service.Service](#service.Service) | repeated | Removed service. |






<a name="api.SvcEndpointDiscoveryRequest"></a>

### SvcEndpointDiscoveryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| svc_names_subscribe | [string](#string) | repeated | All service names that need to subscribe. |
| svc_names_unsubscribe | [string](#string) | repeated | All service names that need to unsubscribe. |






<a name="api.SvcEndpointDiscoveryResponse"></a>

### SvcEndpointDiscoveryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| svc_name | [string](#string) |  | Name of service which endpoints had updated. |
| added | [service.Endpoint](#service.Endpoint) | repeated | Added endpoints. |
| removed | [service.Endpoint](#service.Endpoint) | repeated | Removed endpoints. |





 

 

 


<a name="api.DiscoveryService"></a>

### DiscoveryService
DiscoveryService is a service which is used to discover service, service config
and service endpoints.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| StreamSvcs | [SvcDiscoveryRequest](#api.SvcDiscoveryRequest) | [SvcDiscoveryResponse](#api.SvcDiscoveryResponse) stream |  |
| StreamSvcConfigs | [SvcConfigDiscoveryRequest](#api.SvcConfigDiscoveryRequest) stream | [SvcConfigDiscoveryResponse](#api.SvcConfigDiscoveryResponse) stream |  |
| StreamSvcEndpoints | [SvcEndpointDiscoveryRequest](#api.SvcEndpointDiscoveryRequest) stream | [SvcEndpointDiscoveryResponse](#api.SvcEndpointDiscoveryResponse) stream |  |

 



## Scalar Value Types

| .proto Type | Notes | C++ Type | Java Type | Python Type |
| ----------- | ----- | -------- | --------- | ----------- |
| <a name="double" /> double |  | double | double | float |
| <a name="float" /> float |  | float | float | float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long |
| <a name="bool" /> bool |  | bool | boolean | boolean |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str |

