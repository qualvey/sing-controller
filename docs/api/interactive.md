# 在线交互接口文档 (Scalar & Swagger)

为了避免每次新增或修改接口都要繁琐地手动更新 Markdown 文档，项目集成了 **OpenAPI 规范驱动的自动化文档系统**。

---

## 访问在线文档

启动 `sing-controller` 后，通过浏览器即可直接访问交互式接口调试台：

- **交互式文档地址**: `http://localhost:8080/api/docs` (由运行中的 Controller 直接提供)
- **原始 OpenAPI 规范 (JSON)**: [下载 swagger.json](/api/swagger.json)
- **原始 OpenAPI 规范 (YAML)**: [下载 swagger.yaml](/api/swagger.yaml)

---

## 特性与优势

1. **零外部重依赖**：
   采用现代化高颜值的 [Scalar](https://scalar.com/) 渲染引擎，单页嵌入，体积极其轻巧，无需在服务端捆绑庞大的静态资源包。
2. **代码级单点真实性**：
   所有路由、请求参数、返回结构体全部通过 Go 代码中的标准 Doc 注释（`@Summary`, `@Param`, `@Success`, `@Router`）声明。
3. **在线调试 (Try it out)**：
   文档页面内建请求发送器，支持直接填入 `X-Secret` 并在网页上直接对各端点发起请求、查看返回 JSON。

---

## 开发者：如何生成与更新接口文档

当在 Go 后端修改或新增接口后，执行一次以下命令即可自动完成文档更新：

::: code-group
```powershell [Windows PowerShell]
# 在 controller 目录下执行 swag 工具
cd controller
go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g internal/api/api.go -o ../docs/public/api
```

```bash [Linux / macOS]
cd controller
go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g internal/api/api.go -o ../docs/public/api
```
:::

生成的 `swagger.json` 会自动同步至服务端与文档站中，真正做到“代码即文档”。
