# Compression

**Transparent compression** is to reduce the use of memory and
network bandwidth by big keys and improve request performance.

## Protocol Design

Mark the data to be compressed by adding a header before the
compressed data, the following diagram depicts the protocol:

```text
                                                +
                                                |
                     header                           compressed data
                                                |
 1byte                                          |
+------+-------+-------+-------+--------+----------------------------------+
|      |       |       |       |        |       |         ...
+--+---+-------+--+----+---+---+----+---+--+-------------------------------+
   |              |        |        |      |    |
   |              |    algorithm    +--+---+    |
   +-------+------+                    +        |
           +                         \r\n       |
      magic number                              |
                                                |
                                                +
```

### Examle(snappy)

- Before

`echo -n $(redis-cli -h ${BACKEND_HOST} GET TEST_KEY) | xxd`

```text
00000000: 3030 3030 3030 3030 3030 3030 3030 3030  0000000000000000
00000010: 3030 3030 3030 3030 3030 3030 3030 3030  0000000000000000
00000020: 3030 3030 3030 3030 3030 3030 3030 3030  0000000000000000
00000030: 3030 3030 3030 3030 3030 3030 3030 3030  0000000000000000
00000040: 3030 3030 3030 3030 3030 3030 3030 3030  0000000000000000
00000050: 3030 3030 3030 3030 3030 3030 3030       00000000000000
```

- After

`echo -n $(redis-cli -h ${BACKEND_HOST} GET TEST_KEY) | xxd`

```text
00000000: 2850 2401 0d0a ff06 0000 734e 6150 7059  (P$.......sNaPpY
00000010: 000d 0000 faca b4a7 5e00 30fe 0100 7201  ........^.0...r.
00000020: 00                                       .
```

## Prameters

[Reference](/proto-ref/#redisoptioncompression)

## Compatibility

Now support `Strings` and `Hashes`.
When compression is enabled, commands such as `GETBIT` are disabled.
Other commands are not processed and passed directly to the backend.

!!! warning
    Using `GETBIT` on the compressed value after the compressed mode 
    is closed will return unexpected results.

| COMMAND  | Support |
|:--------:|:-------:|
|   SET    |    Y    |
|   MSET   |    Y    |
|  GETSET  |    Y    |
|  SETNX   |    Y    |
|   HSET   |    Y    |
|  HMSET   |    Y    |
|  PSETEX  |    Y    |
|  SETEX   |    Y    |
|  HSETEX  |    Y    |
|  APPEND  |    N    |
|   EVAL   |    N    |
|  SETBIT  |    N    |
| SETRANGE |    N    |
|  GETBIT  |    N    |
| GETRANGE |    N    |

## Compress Algorithm

- 1: [snappy](http://google.github.io/snappy/)

- 255: mock, only for unit test

## Performance

TBD
