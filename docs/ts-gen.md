# nr route-info gen-ts — 从 Go typed route 生成 TypeScript 类型

`nr route-info gen-ts` 静态分析项目的类型化 Gin 路由（`core.JSONE/RequestE/NoInputE` 等），
把 handler 的请求/响应契约生成为 TypeScript 类型，替代前端手写接口定义。

## 用法

```sh
# 在项目根目录执行（Go 后端 + web/ 前端）
# 默认输出目录为 <frontend dir>/src/api/generated，frontend dir 取 neter.yml 的 dev.frontend.dir（默认 web）
nr route-info gen-ts --dir .

# 指定输出目录（默认 web/src/api/generated）
nr route-info gen-ts --dir . --output web/src/api/generated

# 只生成某路由包（v1 / app / open）
nr route-info gen-ts --dir . --package v1

# CI 校验生成文件是否过期
nr route-info gen-ts --dir . --check
```

`--output` 为目录时生成**多文件**：

```
web/src/api/generated/
  _common.gen.ts        ApiResponse<T> 响应包装（{ code, msg, data }）
  category.gen.ts       ← internal/routes/v1/category_routes.go
  user.gen.ts           ← internal/routes/v1/user_routes.go
  ...
```

`--output` 以 `.ts` 结尾则保留旧单文件模式（自包含 `ApiResponse`）。

## 生成内容

每个 `.gen.ts` 按路由生成契约，命名 = 方法 + 路径驼峰：

```ts
export interface PostAdminV1CategoryCreateBody { name: string; pid?: number | null; ... }
export type PostAdminV1CategoryCreateResponse = ApiResponse<string>
export interface GetV1CategoryListQuery { pid?: string }
export type GetV1CategoryListResponse = ApiResponse<Array<GetV1CategoryListResponseItem>>
```

- 请求：`{Method}{Path}{Body|Query|Path}`
  - `JSON/JSONE` 包装 → `Body`；`RequestE` 带 form 标签或 GET/DELETE → `Query`；
    URI 参数 → `Path`
  - 请求字段可选 = 后端未标 `binding:"required"`（省略时绑定零值，不会报错）
- 响应：`{Method}{Path}Response = ApiResponse<T>`；`core.ListResponse[T]` →
  `{ list: Array<T>; total: number }`；`EmptyResponse` → `null`
- 响应字段可选 = `omitempty` 或指针；但 `ent.X` 的代码生成字段默认带
  `omitempty`，会视为实体序列化实现细节，不会因此生成 `?`

## 类型映射

| Go | TS |
|---|---|
| `string` / `time.Time` / `rune` / `byte` | `string` |
| `int*` / `uint*` / `float*` | `number` |
| `bool` | `boolean` |
| `[]T` | `Array<T>` |
| `map[K]V` / `gin.H` | `Record<string, unknown>` |
| `*T` | `T \| null`（字段级用 `?`） |
| 未解析类型（枚举、外部包） | `unknown` |

分析器还会：

- 展开嵌入结构体字段（`UpdateXParams` 嵌入 `CreateXParams` 时字段合并进 Body）
- 过滤 `json:"-"` 嵌入结构（ent 的 `config`）与未导出字段（`selectValues` 等），
  避免把 ORM 内部结构泄入 API 类型
- 解析 `ent` 实体（含 `edges` 关联图），字段来自真实序列化标签

## 前端接入模式

生成文件**只含类型**，request client 与 API 模块布局保持项目自有的。接入步骤：

1. 生成：`nr route-info gen-ts --dir .`
2. API 模块中删除手写的 req/res 接口，import 生成类型：

```ts
// src/api/categoryApi.ts
import type { PostAdminV1CategoryCreateBody } from './generated/category.gen.ts'

const postApiCreateCategory = (payload: PostAdminV1CategoryCreateBody) => {
  return request<PostAdminV1CategoryCreateResponse>({ url: '/admin/v1/category/create', method: 'post', data: payload })
}
```

3. 需要补充的少量定义放在 API 模块里（生成器保持"类型只读"）：

   - **共享模型**：若业务接口仅返回 Ent 字段的裁剪子集，仍可补充强类型
     （如 `IUserRtn`）；完整 `ent.X` 展开的非指针字段会直接生成为必填字段
   - **query 宽松输入**：后端 query 字段都是 string，前端常用 number 传参，
     补充 `{ id?: number | string }` 之类的输入类型
   - **枚举联合**：未解析枚举是 `unknown`，可按业务补充字面量联合

4. 校验：`nr route-info gen-ts --check` 可接入 CI，路由变更后强制重新生成。

## 生成器开发

- 分析：`internal/route_info/analyzer.go`（AST 扫描路由注册 + 结构体字段解析）
- 生成：`internal/route_info/tsgen.go`（`GenerateTypeScript` 单文件 / `GenerateTypeScriptFiles` 多文件）
- 命令：`cmd/nr/cmd/route_info.go` 的 `gen-ts` 子命令
- 测试：`internal/route_info/tsgen_test.go`、`analyzer_test.go`

改动生成器后重新安装：

```sh
go install ./cmd/nr   # 安装到 GOBIN（where nr）
```
