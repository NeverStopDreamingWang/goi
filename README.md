# goi

基于 `net/http` 进行开发的 Web 框架

功能：

1. http web 服务
2. 多级路由：子父路由，内置路由转换器，自定义路由转换器
   * Router.Include(UrlPath) 创建一个子路由
   * Router.UrlPatterns(UrlPath, goi.AsView{GET: Test1}) 注册一个路由
3. 静态路由：静态路由映射，返回文件对象即响应该文件内容
    * Router.StaticUrlPatterns(UrlPath, StaticUrlPath) 注册一个静态路由
4. 中间件：请求中间件，视图中间件，响应中间件
    * Server.MiddleWares.BeforeRequest(RequestMiddleWare) 注册请求中间件
	* Server.MiddleWares.BeforeView(ViewMiddleWare) 注册视图中间件
	* Server.MiddleWares.BeforeResponse(ResponseMiddleWare) 注册响应中间件
4. 日志模块：支持三种日志，日志等级，按照日志大小、日期自动进行日志切割
5. 内置 JWT Token
6. ORM 模型关系映射，MySQL、Sqlite3
7. 缓存
   * 支持过期策略，默认惰性删除
     * PERIODIC（定期删除：默认每隔 1s 就随机抽取一些设置了过期时间的 key，检查其是否过期）
     * SCHEDULED（定时删除：某个设置了过期时间的 key，到期后立即删除）
   * 支持内存淘汰策略
     * NOEVICTION（直接抛出错误）
     * ALLKEYS_RANDOM（随机删除-所有键）
     * ALLKEYS_LRU（删除最近最少使用-所有键）
     * ALLKEYS_LFU（删除最近最不频繁使用-所有键）
     * VOLATILE_RANDOM（随机删除-设置过期时间的键）
     * VOLATILE_LRU（删除最近最少使用-设置过期时间的键）
     * VOLATILE_LFU（删除最近最不频繁使用-设置过期时间的键）
     * VOLATILE_TTL（删除即将过期的键-设置过期时间的键）

[详细示例：example](./example)
