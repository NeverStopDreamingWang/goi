# Goi 开发文档

**简体中文** | [English](README.en.md) · [项目主页](../README.md)

本目录按功能说明当前框架的配置、使用方式、扩展接口和实现边界。示例导入路径为 `github.com/NeverStopDreamingWang/goi/v3`；具体 API 以当前源码为准。

## 从哪里开始

1. 初次使用先阅读[快速开始](../README.md#快速开始)或 [CLI 使用指南](../cmd/goi/README.md)。
2. 先配置[日志](logging.md)与[服务设置](configuration.md)，再注册[路由](routing.md)和[中间件](middleware.md)。
3. 通过[请求与响应](http.md)、[参数验证](validation.md)完成接口，向数据库操作传递请求 context。
4. 按[生命周期](lifecycle.md)注册后台任务和清理回调，最后调用 `RunServer`。

## 功能指南

| 指南 | 内容 |
| --- | --- |
| [路由、路径转换与静态资源](routing.md) | 路由分组、方法映射、404/405、Converter、文件/目录及嵌入资源 |
| [中间件](middleware.md) | 三个处理入口、执行顺序、短路、异常，以及 CORS、安全响应头等内置实现 |
| [请求与响应](http.md) | 路径/查询/请求体参数、MIME 与 Parser、上传、context、Response、Data、Result、RawHandler |
| [参数验证与类型转换](validation.md) | 字段标签、内置验证器、自定义 Parse、错误响应与反射赋值 |
| [启动、后台任务与停机](lifecycle.md) | Startup、ShutdownCallback、信号和手动停机、错误传播、资源顺序和等待边界 |
| [内存缓存](cache.md) | 读写、TTL、清理模式、容量与淘汰设置，以及当前实现限制 |
| [日志](logging.md) | 默认日志配置、级别、输出定制、访问日志、切割任务和用户管理的关闭 |
| [服务配置、HTTPS、国际化与时间](configuration.md) | 共享设置、证书、自签名工具、全局语言及两种时间表示 |
| [密码哈希与密码学工具](security.md) | PBKDF2-SHA256、随机数据、AES/RSA 格式和使用边界 |
| [数据库与自定义 ORM](../db/README.md) | 连接注册、配置优先级、模型、标识符/占位符/列策略及结果扫描 |
| [CLI 使用说明](../cmd/goi/README.md) | 安装、可选构建、项目/应用路径规则、注册、迁移和运行 |

## 示例与验证边界

完整应用可参考独立的 [goi_example](https://github.com/NeverStopDreamingWang/goi_example) 项目；其配置、数据库和初始化行为与纯框架 API 示例不同。

文档中的完整 Go 示例可作为独立文件编译。含 `RunServer` 的例子会阻塞并监听端口，日志和证书例子会写文件；不要为了校验文档而直接批量运行全部示例。嵌入资源的示例还需要相应资源文件，片段代码应放在注明的上下文中。

这些指南记录现有行为，不把配置项名称或注释视为额外保证。对于定时缓存清理、淘汰策略、代理头、日志并发、XML 和密码学输入等已有边界，请阅读对应指南的限制说明。本轮文档完善不改变生产实现。

## 相关入口

[项目主页](../README.md) · [版本发布](https://github.com/NeverStopDreamingWang/goi/releases) · [源码模块](../go.mod)
