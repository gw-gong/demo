# Google OAuth Demo

基于 Gin（后端）+ React（前端）的 Google OAuth 2.0 登录 Demo，使用 Session + CORS 实现前后端分离。

## 架构与数据流

```mermaid
sequenceDiagram
    participant User
    participant React as React前端
    participant Gin as Gin后端
    participant Google as Google OAuth

    User->>React: 点击「用 Google 登录」
    React->>Gin: 跳转 GET /auth/login
    Gin->>Google: 重定向到授权页
    User->>Google: 同意授权
    Google->>Gin: 重定向 GET /auth/callback?code=...
    Gin->>Google: 用 code 换 token
    Google->>Gin: access_token
    Gin->>Google: GET userinfo（Bearer token）
    Google->>Gin: email, name, picture 等
    Gin->>Gin: 写入 Session
    Gin->>React: 重定向到前端首页
    React->>Gin: GET /api/me（带 Cookie）
    Gin->>React: JSON 用户信息
    React->>User: 展示邮箱、头像、姓名等
```

## 项目结构

```mermaid
flowchart LR
    subgraph frontend [前端 web]
        React[React + Vite]
        React -->|GET /api/me credentials| API
        React -->|跳转| Login
    end
    subgraph backend [后端 server]
        Gin[Gin]
        Login["/auth/login"]
        Callback["/auth/callback"]
        API["/api/me"]
        Gin --> Login
        Gin --> Callback
        Gin --> API
    end
    subgraph external [外部]
        Google[Google OAuth]
    end
    Login -->|302 授权 URL| Google
    Google -->|302 code| Callback
```

## 目录说明

| 目录 | 技术栈 | 说明 |
|------|--------|------|
| `server/` | Go + Gin + godotenv + oauth2 | 提供 `/auth/login`、`/auth/callback`、`/api/me`，Session 存用户信息 |
| `web/` | React + Vite | 首页拉取 `/api/me` 展示用户信息，未登录时跳转后端登录 |

## 运行方式

- **后端**：`cd server && go run .`（默认 `:8081`，需配置 `.env` 中的 `GOOGLE_CLIENT_SECRET`）
- **前端**：`cd web && npm install && npm run dev`（默认 `http://127.0.0.1:5173`）
- 使用 **http://127.0.0.1:5173** 访问前端，与后端重定向地址一致。
