# 2FA Vault (双因素认证管理工具)

<p align="center">
  <img src="internal/web/dist/icon.svg" width="100" height="100" alt="2FA Vault Logo" />
</p>

<p align="center">
  <strong>基于 Go 语言开发的高安全性、现代化、模块化 2FA (双因素认证) 令牌管理工具</strong><br>
  全面支持 <strong>Windows</strong> 桌面环境与 <strong>iOS</strong> (iPhone / iPad PWA & 原生 GoMobile 框架)，深度集成 <strong>iCloud 云盘备份</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat-square&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/Platform-Windows%20%7C%20iOS%20%7C%20macOS%20%7C%20Linux-brightgreen?style=flat-square" alt="Platform" />
  <img src="https://img.shields.io/badge/Encryption-Argon2id%20%2B%20AES--256--GCM-blueviolet?style=flat-square" alt="Security" />
  <img src="https://img.shields.io/badge/Sync-iCloud%20Drive%20Ready-blue?style=flat-square" alt="iCloud" />
  <img src="https://img.shields.io/badge/Frontend-Vue%203%20%2B%20TypeScript%20%2B%20Tailwind-42b883?style=flat-square&logo=vuedotjs" alt="Vue 3 Frontend" />
  <img src="https://img.shields.io/badge/Build-Vite%20%2B%20Zero%20CGo-646CFF?style=flat-square&logo=vite" alt="Vite Build" />
  <img src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" alt="License" />
</p>

---

## 🌟 核心特性

- 🔒 **端到端零知识加密**：主密码通过 **Argon2id** (内存硬抗暴力破解) 派生密钥，核心数据采用 **AES-256-GCM** 带认证加密，支持闲置自动锁定与敏感内存清空。
- 📱 **深度跨平台支持 (Windows & iOS)**：
  - **Windows**：开箱即用单文件可执行程序，启动后自动拉起默认浏览器窗口，极低内存占用（< 15MB）。
  - **iOS**：支持以 **PWA (Progressive Web App)** 方式一键“添加到主屏幕”，具备原生全屏沉浸体验、安全区（刘海屏/灵动岛）边距适配、震动触感反馈（Haptics）与 Service Worker 离线缓存；亦提供 **GoMobile** iOS Framework 源码导出接口，可无缝嵌入原生 Swift/SwiftUI 工程。
- ☁️ **iOS iCloud 云盘备份与同步**：
  - **PWA / Safari 一键备份到 iCloud**：利用 iOS 系统的 `navigator.share` 文件分享能力，一键唤起原生分享面板，直接选择 **“存储到‘文件’ ➔ iCloud 云盘”**，实现跨 Apple 设备（iPhone/iPad/Mac）的云端安全备份。
  - **从 iCloud 云盘导入恢复**：导入时可直接唤起 iOS 原生文件选择器访问 iCloud 云盘中的备份文件。
  - **Windows ➔ iOS 联动**：在 Windows 上安装“iCloud 客户端”后，可将数据文件存放在 iCloudDrive 目录，实现跨系统云同步。
- 📷 **全能二维码识别与智能录入**：
  - 🌐 **图片链接解析**：直接粘贴网络图片链接（HTTP / HTTPS / Data URI），后端自动拉取并毫秒级解码二维码，彻底解决浏览器跨域（CORS）与画布污染问题。
  - 📷 **实时摄像头扫码**：调用设备后置摄像头实时扫描屏幕或纸质二维码。
  - 🖼️ **相册/本地图片导入**：支持从设备相册选择二维码截图自动解析。
  - 📋 **标准 URI 导入**：支持粘贴标准 `otpauth://` 格式 URI。
- 🎨 **现代化模块化 UI 设计**：
  - 前后端**彻底模块化**：CSS 按职责划分为主题变量、基础排版、卡片组件、模态弹窗；JS 按功能拆分为 API 通信、状态仓库、UI 渲染、倒计时刻、扫描解析、iCloud 备份与事件调度。
  - 杜绝视觉白色瑕疵：表单 `<select>` 下拉菜单、浏览器 Autofill 自动填充、滚动条均经过定制，全界面保持纯净深邃的深色/浅色主题。
  - 实时 SVG 动态环形倒计时（绿 ➔ 黄 ➔ 红 动态色彩预警）。
  - 单击即可复制验证码并伴随浮动轻量 Toast 提示与震动反馈。
- ⚙️ **多协议与多类型支持**：
  - **RFC 6238 TOTP**：支持 30s/60s 周期、6 位/8 位、SHA1/SHA256/SHA512。
  - **RFC 4226 HOTP**：支持计数器递增与即时计算。
  - **Steam Guard 专属令牌**：支持 Steam 5 位字符验证码（字母+数字）。
- 🔄 **数据迁移兼容**：
  - 兼容导入 **Aegis Authenticator** 备份与标准 `otpauth://` 批量列表。
  - 任意账户均可直接在屏幕上生成分享二维码，便于用手机扫码同步。

---

## 🏗️ 模块化架构

```text
2FA/
├── cmd/
│   └── server/main.go          # 主程序入口：CLI 参数解析、LAN 局域网 IP 自动嗅探、自动打开浏览器
├── pkg/
│   ├── totp/                   # OTP 核心计算：RFC 6238 TOTP、RFC 4226 HOTP、Steam Guard、URI 编解码
│   ├── crypto/                 # 安全加密模块：Argon2id 密钥派生、AES-256-GCM 封套加解密
│   ├── vault/                  # 保险库状态机：CRUD、置顶、标签检索、自动锁定时钟、原子安全落盘
│   ├── migration/              # 导入导出引擎：Aegis JSON 格式适配、标准 otpauth 批量转换
│   └── mobile/                 # GoMobile 原生桥接层：提供 iOS/Android gomobile bind 导出接口
├── internal/
│   ├── api/                    # 彻底模块化的 RESTful API 控制器
│   │   ├── server.go           # 服务初始化、统一路由注册与 JSON 响应封装
│   │   ├── vault_handlers.go   # 保险库生命周期：状态探测、初始化、解锁、锁定、改密、自动锁定
│   │   ├── account_handlers.go # 2FA 令牌核心业务：CRUD、标签检索、HOTP 递增、置顶、导出二维码
│   │   ├── qr_handlers.go      # 二维码与图像处理：服务端解码、跨域图片代理、PNG 二维码生成
│   │   ├── otp_handlers.go     # 实时 OTP 运算：动态验证码计算预览、SSE 实时时间刻推送
│   │   └── migration_handlers.go # 迁移数据管道：Aegis JSON 与批量 URI 导入导出
│   └── web/                    # 编译产物托管层 (go:embed all:dist 内置入单一可执行程序)
│       ├── embed.go
│       └── dist/               # Vite 编译生成的超轻量生产级 SPA 资源包
├── frontend/                   # 现代化 Vue 3 + TypeScript 前端工程 (Vite + Tailwind CSS)
│   ├── src/
│   │   ├── api/client.ts       # 强类型 RESTful 客户端封装
│   │   ├── types/vault.ts      # 2FA 令牌、算法、加密与状态 TypeScript 类型定义
│   │   ├── composables/        # 响应式逻辑组合函数 (useVault, useTheme, useToast)
│   │   ├── components/         # 组件化 UI 视图
│   │   │   ├── Navbar.vue      # 导航栏、实时搜索 (Ctrl+F)、标签分类筛选条、快速操作
│   │   │   ├── AccountCard.vue # 2FA 令牌卡片、平滑 SVG 环形倒计时、点击一键复制
│   │   │   ├── LivePreview.vue # 实时动态验证码预览组件 (密钥有效性校验、秒级倒数)
│   │   │   ├── AddModal.vue    # 添加/编辑弹窗 (截屏粘贴 Ctrl+V、网页图片拖拽、URL 解析、相册选择)
│   │   │   ├── QRModal.vue     # 令牌二维码展示弹窗 (含实时动态码与 URI 复制)
│   │   │   ├── SettingsModal.vue # 自动锁定时间、暗色/亮色模式分段切换、主密码修改
│   │   │   ├── ImportExportModal.vue # Aegis JSON 导出、iCloud 原生分享、批量导入
│   │   │   ├── DragOverlay.vue # 全局拖拽悬停视觉指示层
│   │   │   └── ToastContainer.vue # 悬浮气泡通知
│   │   ├── App.vue             # 根布局与全局拖拽/截屏粘贴/键盘快捷键监听
│   │   └── main.ts             # 应用装配与 Service Worker (PWA 离线支持) 注册
│   ├── public/                 # PWA 图标、Web Manifest 与 Service Worker
│   ├── vite.config.ts          # Vite 构建配置 (自动构建输出至 ../internal/web/dist)
│   └── tailwind.config.js      # 高对比度暗色/亮色主题与现代化调色板配置
├── go.mod / go.sum
└── README.md
```

---

## ☁️ iOS 是否能备份到 iCloud 中？

**答：完全支持！针对不同使用形态，提供了三种原生/无缝的 iCloud 备份与同步方案：**

### 方案 1：PWA 网页应用 ➔ 一键备份到 iCloud 云盘（推荐，开箱即用）
1. 在 iPhone / iPad 上通过 Safari 打开 2FA Vault（或点击桌面 PWA 图标）。
2. 点击顶部 **“导入/备份”** 按钮。
3. 点击 **“一键备份到 iCloud 云盘 (iOS) / 导出 JSON”**。
4. 系统会自动调起 iOS 原生分享面板，直接选择 **“存储到‘文件’” ➔ “iCloud 云盘”**（支持在 iCloud 中新建 `2FA Vault` 文件夹存放）。
5. **恢复导入**：在任何一台 Apple 设备上点击“从 iCloud/本地文件导入”，即可直接打开 iOS“文件”App 选取 iCloud 云盘中的备份还原。

### 方案 2：Windows 与 iPhone 跨设备自动同步（通过 Windows 版 iCloud）
1. 在 Windows 电脑上安装 Apple 官方提供的 **iCloud for Windows**。
2. 启动 2FA Vault 时指定数据文件存放在 iCloud 盘路径：
   ```powershell
   .\2fa-vault.exe -data "C:\Users\<用户名>\iCloudDrive\2FA\vault.enc"
   ```
3. 任何在 Windows 上保存的令牌会自动实时同步至 iCloud 云端。

### 方案 3：GoMobile 原生 iOS App（iCloud Documents 自动无缝同步）
如果将 `pkg/mobile/bridge.go` 编译为原生 iOS App：
1. 在 Xcode 工程中开启 `Signing & Capabilities ➔ iCloud ➔ iCloud Documents`。
2. 将 `vault.enc` 保存至 App 的 Ubiquity 容器目录：
   ```swift
   let icloudURL = FileManager.default.url(forUbiquityContainerIdentifier: nil)?.appendingPathComponent("Documents/vault.enc")
   try TwoFAEngineInitVault(icloudURL!.path, "MasterPassword")
   ```
3. 数据库将由 iOS 系统底层在所有登录相同 Apple ID 的设备间透明自动漫游。

---

## 🚀 启动与使用指南

### 1. 编译构建

项目支持一键编译前端与打包单文件二进制：
```powershell
# 步骤 1：编译 Vue 3 前端工程 (产物自动同步至 internal/web/dist)
cd frontend
pnpm build     # 或 npm run build
cd ..

# 步骤 2：编译生成 Windows 独立单文件可执行程序 (无外部依赖，仅单个 .exe)
go build -ldflags="-s -w" -o 2fa-vault.exe ./cmd/server
```

前端开发模式 (支持热重载 HMR)：
```powershell
# 终端 1 启动后端服务
go run ./cmd/server

# 终端 2 启动前端热重载开发服务器
cd frontend
pnpm dev       # 访问 http://localhost:5173，自动代理 /api 请求
```

### 2. 运行
```powershell
.\2fa-vault.exe
```

程序启动后会自动在终端打印访问信息，并在 0.3 秒内自动调用默认浏览器打开：
```text
================================================================
  2FA Vault v1.0.0 - Secure Cross-Platform Authenticator
  Modular Go Backend · AES-256-GCM + Argon2id · Windows & iOS PWA
================================================================
  • Local Access (Windows) : http://127.0.0.1:2233
  • iOS / Mobile Access    : http://192.168.1.100:2233
  • Encrypted Vault File   : data/vault.enc
----------------------------------------------------------------
```

### 3. 解析图片链接添加 2FA 令牌
1. 点击顶部 **“+ 添加令牌”**。
2. 切换到 **“图片链接解析”** 标签页。
3. 粘贴任意包含二维码的图片 URL（例如 `https://example.com/2fa.png` 或 `data:image/png;base64,...`）。
4. 点击 **“解析链接中的二维码”**，系统会自动拉取图像、识别并填充服务商、账号、密钥等信息，点击保存即可完成录入。

---

## 🧪 自动化测试套件

```powershell
# 运行全部单元测试
go test -v ./...
```

测试已覆盖：
- RFC 6238 TOTP 与 RFC 4226 HOTP 官方向量校验
- Steam Guard 专属令牌生成测试
- Argon2id + AES-256-GCM 加解密与篡改拦截测试
- 保险库增删改查、排序、标签、自动锁定测试
- 二维码图片网络链接服务端解码测试
- Aegis 与标准 URI 批量迁移测试

---

## 📄 开源许可证

本项目基于 [MIT License](LICENSE) 开源。
