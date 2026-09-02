
[x]编辑节点，如果修改tag,路径tag与body tag不一致（后端）

[x]前端没有添加节点的入口

[x] 订阅功能闭环失败

```shell
2026/09/01 18:58:27 [INFO] [HTTP] PUT /api/providers/13bc2eeb-5beb-4a34-8886-36139a1519ab 200 (438.3µs) - 127.0.0.1 17B
```
后端日志无报错


http://localhost:5174/api/providers/13bc2eeb-5beb-4a34-8886-36139a1519ab
前端请求200
```json
{
    "success": true
}
```

前端notification显示订阅源已更新
解析节点有数字

[x] 订阅源删除后，节点还在