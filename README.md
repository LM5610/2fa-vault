# 2FA Vault (双因素认证管理工具)

<p align="center">
  <img src="internal/web/dist/icon.svg" width="100" height="100" alt="2FA Vault Logo" />
</p>

<p align="center">
  <strong>基于 Go 语言开发的高安全性、现代化、模块化 2FA (双因素认证) 令牌管理工具</strong><br>
  全面支持 <strong>Windows</strong> 桌面环境与 <strong>iOS</strong> (iPhone / iPad PWA & 原生 GoMobile 框架)
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat-square&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/Platform-Windows%20%7C%20iOS%20%7C%20macOS%20%7C%20Linux-brightgreen?style=flat-square" alt="Platform" />
  <img src="https://img.shields.io/badge/Encryption-Argon2id%20%2B%20AES--256--GCM-blueviolet?style=flat-square" alt="Security" />
  <img src="https://img.shields.io/badge/CGo-Zero%20Dependency-orange?style=flat-square" alt="Zero CGo" />
  <img src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" alt="License" />
</p>

---

## 🌟 核心特性

- 🔒 **端到端零知识加密**：主密码通过 **Argon2id** (内存硬抗暴力破解) 派生密钥，核心数据采用 **AES-256-GCM** 带认证加密，支持闲置自动锁定与敏感内存清空。
- 📱 **深度跨平台支持 (Windows & iOS)**：
  - **Windows**：开箱即用单文件可执行程序，启动后自动拉起默认浏览器窗口，极低内存占用（< 15MB）。
  - **iOS**：支持以 **PWA (Progressive Web App)** 方式一键“添加到主屏幕”，具备原生全屏沉浸体验、安全区（刘海屏/灵动岛）边距适配、震动触感反馈（Haptics）与 Service Worker 离线缓存；亦提供 **GoMobile** iOS Framework 源码导出接口，可无缝嵌入原生 Swift/SwiftUI 工程。
- 🎨 **现代化 UI 设计**：
  - 玻璃拟态（Glassmorphism）与暗色/亮色主题自由切换。
  - 实时 SVG 动态环形倒计时（绿 ➔ 黄 ➔ 红 动态色彩预警）。
  - 单击即可复制验证码并伴随浮动轻量 Toast 提示与震动反馈。
- ⚙️ **多协议与多类型支持**：
  - **RFC 6238 TOTP**：支持 30s/60s 周期、6 位/8 位、SHA1/SHA256/SHA512。
  - **RFC 4226 HOTP**：支持计数器递增与即时计算。
  - **Steam Guard 专属令牌**：支持 Steam 5 位字符验证码（字母+数字）。
- 📷 **扫码与智能录入**：
  - 调用系统摄像头实时扫描屏幕或纸质二维码。
  - 支持相册二维码图片解析。
  - 支持直接粘贴标准 `otpauth://` 格式 URI。
- 🔄 **备份、导入与导出**：
  - 兼容导入 **Aegis Authenticator** 备份与标准 `otpauth://` 列表。
  - 支持一键导出未加密/加密 JSON 备份以及 URI 列表。
  - 任意账户均可直接在屏幕上生成分享二维码，便于用手机扫码同步。
- 🧩 **高内聚低耦合模块化架构**：业务算法、核心加密、数据持久化、迁移转换、API 与前端视图彻底解耦。

---

## 🏗️ 架构与模块划分

```text
2FA/
├── cmd/
│   └── server/             # 程序主入口：CLI 参数解析、LAN 局域网 IP 嗅探、自动打开浏览器、优雅停机
├── pkg/
│   ├── totp/               # OTP 核心计算：RFC 6238 TOTP、RFC 4226 HOTP、Steam Guard、URI 编解码
│   ├── crypto/             # 安全加密模块：Argon2id 密钥派生、AES-256-GCM 封套加解密
│   ├── vault/              # 保险库状态机：CRUD、置顶、标签检索、自动锁定时钟、原子安全落盘
│   ├── migration/          # 导入导出引擎：Aegis JSON 格式适配、标准 otpauth 批量转换
│   └── mobile/             # GoMobile 原生桥接层：提供 iOS/Android gomobile bind 导出接口
├── internal/
│   ├── api/                # RESTful API 与 SSE (Server-Sent Events) 实时推送端点
│   └── web/                # 现代化 Web 前端：通过 go:embed 打包内置，单二进制零外部资源依赖
│       ├── embed.go
│       └── dist/           # SPA/PWA 单页应用源码 (HTML5, CSS3, JS, SVG, Manifest, SW)
├── go.mod / go.sum
└── README.md
```

---

## 🚀 Windows 快速上手

### 1. 编译构建
```bash
# 编译生成单个无任何 C 依赖的 Windows 独立执行程序
go build -o 2fa-vault.exe ./cmd/server
```

### 2. 运行
直接双击 `2fa-vault.exe`，或者在终端中运行：
```powershell
.\2fa-vault.exe
```

程序启动后会自动在终端打印访问信息，并在 0.3 秒内自动调用 Windows 默认浏览器打开：
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

### 3. 可选启动参数
| 参数 | 默认值 | 说明 |
| :--- | :--- | :--- |
| `-port` | `2233` | HTTP 服务监听端口 |
| `-host` | `0.0.0.0` | 监听地址（`0.0.0.0` 允许手机等局域网设备连接） |
| `-data` | `data/vault.enc`| 加密数据库持久化文件路径 |
| `-no-browser` | `false` | 启动时不自动唤起浏览器 |
| `-headless` | `false` | 以无界面后台服务模式运行 |

---

## 📱 iOS (iPhone / iPad) 使用指南

我们为 iOS 用户提供了两种使用模式：

### 模式一：PWA 沉浸式应用（推荐，无需 Mac 与开发者证书）

1. 确保电脑与 iPhone 处于同一个 Wi-Fi 局域网下（或通过 Tailscale / 内网穿透连接）。
2. 在电脑启动 `2fa-vault.exe`，控制台会输出局域网 IP，例如 `http://192.168.1.100:2233`。
3. 在 iPhone 上打开 **Safari 浏览器**，输入上述地址并进入。
4. 点击 Safari 底部工具栏的 **“分享” (Share)** 按钮 ➔ 向上滑动选择 **“添加到主屏幕” (Add to Home Screen)** ➔ 点击右上角“添加”。
5. 返回 iPhone 桌面，点击新生成的 **2FA Vault** 图标：
   - 🌟 无 Safari 地址栏与底部导航条，全屏沉浸如同原生 App。
   - 🌟 完美避开 iPhone 顶部灵动岛/刘海屏及底部小白条的安全区域（Safe Area）。
   - 🌟 支持摄像头扫码录入与触觉震动反馈。

### 模式二：GoMobile 编译为 iOS 原生 Framework

本项目在 `pkg/mobile` 下封装了适配 `gomobile bind` 的全套 GoMobile 绑定接口：

```bash
# 1. 安装 gomobile 工具链
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init

# 2. 编译生成 iOS XCFramework (需在 macOS 上执行)
gomobile bind -target=ios -o TwoFAEngine.xcframework ./pkg/mobile
```

在 Xcode 的 Swift 项目中直接拖入 `TwoFAEngine.xcframework` 即可调用原生接口：
```swift
import TwoFAEngine

// 初始化保险库
try TwoFAEngineInitVault(vaultPath, "MyMasterPassword")

// 获取账户列表 JSON
let jsonString = try TwoFAEngineListAccountsJSON("", "")
```

---

## 🔐 安全机制解析

1. **主密码派生 (KDF)**：
   采用最新标准 **Argon2id**（Time=1, Memory=64MB, Threads=4, Salt=16字节高强度随机数），杜绝 GPU/ASIC 离线暴力破解彩虹表。
2. **认证对称加密 (AEAD)**：
   采用标准 **AES-256-GCM**，每次写入均重新生成专属 12 字节加密随机 Nonce，并自带 16 字节完整性认证标签（Auth Tag），任何篡改均会导致解密直接拒绝。
3. **安全原子写入**：
   数据先写入临时文件（`vault.enc.tmp`），写入并刷盘成功后通过原子重命名替换原有数据库，防止意外断电导致数据库损坏。
4. **内存自动擦除与超时锁定**：
   支持可配置的闲置超时自动上锁（1/5/15/30分钟），锁定瞬间即时清空内存中保存的主密码字符串与明文缓存。

---

## 🧪 自动化测试套件

本项目严格遵循高质量测试规范，所有核心模块均编写了自动化单元测试：

```powershell
# 运行全部单元测试
go test -v ./...
```

**测试覆盖内容**：
- `pkg/totp`：
  - RFC 6238 官方时间步长测试向量比对通过
  - RFC 4226 官方 HOTP 计数测试向量比对通过
  - Steam Guard 专属 Base32 编码计算验证通过
  - Base32 密钥自动去空格/连字符与自动补齐验证通过
  - `otpauth://` URI 双向解析与序列化验证通过
- `pkg/crypto`：
  - 加解密 Roundtrip 一致性验证
  - 错误主密码拒绝测试通过
  - 密文被窜改检测测试通过
- `pkg/vault`：
  - 保险库初始化、锁定、解锁生命周期验证
  - 账户增删改查、排序、搜索与标签过滤验证
  - HOTP 计数自增验证
- `pkg/migration`：
  - Aegis Authenticator 与标准 URI 批量解析验证
- `internal/api`：
  - RESTful 接口全流程 Mock 测试与 PNG 格式二维码生成测试通过

---

## 💻 快捷键与使用技巧

| 快捷键 | 功能 |
| :--- | :--- |
| `Ctrl + F` / `⌘ + F` | 快速聚焦搜索栏 |
| `Ctrl + L` / `⌘ + L` | 立即锁定保险库 |
| `Esc` | 关闭当前弹出的所有模态窗口 |
| **单击验证码框** | 复制对应 2FA 验证码到剪贴板 |

---

## 📄 开源许可证

本项目基于 [MIT License](LICENSE) 开源。
