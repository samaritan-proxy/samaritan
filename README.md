# Samaritan

![Samaritan](docs/src/images/logo.png)

[![Build Status](https://travis-ci.org/samaritan-proxy/samaritan.svg?branch=master)](https://travis-ci.org/samaritan-proxy/samaritan)
[![Go Report Card](https://goreportcard.com/badge/github.com/samaritan-proxy/samaritan)](https://goreportcard.com/report/github.com/samaritan-proxy/samaritan)
[![codecov](https://codecov.io/gh/samaritan-proxy/samaritan/branch/master/graph/badge.svg)](https://codecov.io/gh/samaritan-proxy/samaritan)
[![Docs](https://img.shields.io/badge/docs-latest-green.svg)](https://samaritan-proxy.github.io/docs/)
[![LICENSE](https://img.shields.io/github/license/samaritan-proxy/samaritan.svg?style=flat-square)](https://github.com/samaritan-proxy/samaritan/blob/master/LICENSE)

English | [简体中文](./README-zh_CN.md)

**Samaritan (səˈmerətn)** is a client side L4 or L7 proxy, written in Golang, with the aim to provide high availability and load balancing. You can call it Sam (sam) for simplicity.

_We named this project Samaritan as it saves our OPs from extreme misery:_

> A charitable or helpful person (with reference to Luke 10:33).
>
> "suddenly, miraculously, a Good Samaritan leaned over and handed the cashier a dollar bill on my behalf"

## Features

- Fast, efficient, lightweight and works on the client side

- Hot restart, zero downtime

- Hot re-configuration without down time

- Good observability

- First-class Redis cluster support
    - commond level stats
    - cluster scan
    - transparent compression
    - hotkey in real time

## Status

It is deployed on every container and virtual machine in the production environment at ELEME, proxying all traffic to the basic components including Redis, MySQL, MQ, etc.
And the total number of running instances is close to sixty thousand.

## Documentation

- For documentation about specific topics, including introduction, quick start, architecture, etc, see [docs](https://samaritan-proxy.github.io/docs/)
- Examples can be found in the [examples directory](examples/).

## License

Samaritan is licensed under the Apache 2.0 license. See [LICENSE](LICENSE) for the full license text.
