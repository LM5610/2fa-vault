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
  <img src="https://img.shields.io/badge/Frontend-ES%20Modules%20%2B%20CSS%20Modular-orange?style=flat-square" alt="Modular Frontend" />
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
│   └── server/main.go      # 主程序入口：CLI 参数解析、LAN 局域网 IP 自动嗅探、自动打开浏览器、优雅停机
├── pkg/
│   ├── totp/               # OTP 核心计算：RFC 6238 TOTP、RFC 4226 HOTP、Steam Guard、URI 编解码
│   ├── crypto/             # 安全加密模块：Argon2id 密钥派生、AES-256-GCM 封套加解密
│   ├── vault/              # 保险库状态机：CRUD、置顶、标签检索、自动锁定时钟、原子安全落盘
│   ├── migration/          # 导入导出引擎：Aegis JSON 格式适配、标准 otpauth 批量转换
│   └── mobile/             # GoMobile 原生桥接层：提供 iOS/Android gomobile bind 导出接口
├── internal/
│   ├── api/                # RESTful API 服务路由、二维码 URL/文件服务端解码与 SSE 实时推送
│   └── web/                # 现代化 Web 前端 (go:embed 打包内置，单二进制零外部资源依赖)
│       ├── embed.go
│       └── dist/
│           ├── index.html  # 响应式结构骨架
│           ├── css/        # 模块化样式表
│           │   ├── variables.css   # 主题色板、暗黑/明亮模式变量、滚动条、选区
│           │   ├── base.css        # 全局排版、网格布局、搜索栏、标签栏、通用按钮
│           │   ├── cards.css       # 2FA 账户卡片、渐变头像、动态环形倒计时 SVG
│           │   └── modals.css      # 弹窗模态框、Tabs 导航、防白边表单控件、二维码暗黑容器
│           └── js/         # 模块化 ES 脚本
│               ├── api.js          # RESTful 与解码请求接口封装
│               ├── state.js        # 应用响应式状态仓库
│               ├── ui.js           # DOM 渲染、主题管理、Toast 气泡通知
│               ├── countdown.js    # 环形倒计时刻度动画与周期边界刷新
│               ├── scanner.js      # 摄像头扫描、图片链接解码、本地相册解析
│               ├── icloud.js       # iOS Web Share 唤起 iCloud Drive 备份适配
│               ├── events.js       # 交互事件分发与全局快捷键调度 (Ctrl+F, Ctrl+L, Esc)
│               └── app.js          # 主启动编排入口
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
```bash
# 编译生成 Windows 独立可执行程序
go build -ldflags="-s -w" -o 2fa-vault.exe ./cmd/server
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
