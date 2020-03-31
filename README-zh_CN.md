# Samaritan

![Samaritan](docs/src/images/logo.png)

[![Build Status](https://travis-ci.org/samaritan-proxy/samaritan.svg?branch=master)](https://travis-ci.org/samaritan-proxy/samaritan)
[![Go Report Card](https://goreportcard.com/badge/github.com/samaritan-proxy/samaritan)](https://goreportcard.com/report/github.com/samaritan-proxy/samaritan)
[![codecov](https://codecov.io/gh/samaritan-proxy/samaritan/branch/master/graph/badge.svg)](https://codecov.io/gh/samaritan-proxy/samaritan)
[![Docs](https://img.shields.io/badge/docs-latest-green.svg)](https://samaritan-proxy.github.io/docs/)
[![LICENSE](https://img.shields.io/github/license/samaritan-proxy/samaritan.svg?style=flat-square)](https://github.com/samaritan-proxy/samaritan/blob/master/LICENSE)

[English](./README.md) | 简体中文

**Samaritan (səˈmerətn)** 是一个使用 Go 开发的，工作在客户端模式的透明代理，能够同时支持四层和七层, 目标是提供负载均衡和高的可用性。为了简单，你也可以叫它 Sam(sam)。

*我们以 Samaritan 命令该项目，是希望它能够把我们从痛苦的运维工作中解救出来:*

> A charitable or helpful person (with reference to Luke 10:33).
>
> "suddenly, miraculously, a Good Samaritan leaned over and handed the cashier a dollar bill on my behalf"
>
> https://www.lexico.com/definition/samaritan

## 特性

- 轻量、高性能以及工作在客户端

- 支持热更新，无需停服

- 运行时配置动态变更，无需重启

- 良好的观测性

- 一流的 Redis 集群支持

    - 命令级别的数据埋点
    - 集群级别 scan
    - 大 key 的透明(解)压缩
    - 热 key 的实时统计

## 状态

它部署在饿了么生产环境的每台容器和虚拟机上, 代理了到基础组件的所有流量包括 MySQL、Redis、MQ等，总的部署实例数有 6w 个。

## 文档

- 详细的文档都可以在 [docs](https://samaritan-proxy.github.io/docs/) 找到, 包含介绍、快速上手和架构等信息。

### 相关文章

- [详解 Samaritan——饿了么最新开源的透明代理](https://mp.weixin.qq.com/s?__biz=MzA4ODg0NDkzOA==&mid=2247487045&amp;idx=1&amp;sn=846c3fd05a52378cb22f623cc05d564c&source=41)
- [如何快速定位 Redis 热点 key](https://mp.weixin.qq.com/s/rZs-oWBGGYtNKLMpI0-tXw)

## 许可协议

Samaritan 使用 Apache 2.0 协议，点击 [LICENSE](LICENSE) 查看详情。

