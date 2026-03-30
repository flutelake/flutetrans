# FluteTrans

<p align="center">
  <img src="assets/flutetrans_icon.png" alt="App Icon" width="160" />
</p>

FluteTrans 是一个跨平台桌面文件传输与远程文件管理工具，支持多种常见协议（FTP/SFTP/WebDAV/SMB/NFS/S3）。项目使用 Wails（Go）作为后端，Svelte + Vite 作为前端，提供连接管理、文件浏览、上传下载与传输队列等能力。

## 功能特性

- 多协议连接管理：保存连接配置，一键连接/断开
- 远程文件浏览：目录列表、基础文件信息展示
- 上传/下载：支持传输进度与传输任务列表
- 本地凭据加密：使用“主密码”对保存的连接凭据进行本机加密存储，可锁定/修改主密码
- 多语言：默认使用操作系统语言；用户手动选择后会在本机持久化

## 支持的协议

- FTP
- SFTP
- WebDAV
- SMB
- NFSv3
- S3（兼容 S3 API 的对象存储）

## Screenshot

<p align="center">
  <img src="assets/screenshot.png" alt="Screenshot" width="600" />
</p>

## 技术栈

- 后端：Go + Wails v2
- 前端：Svelte 5 + Vite 6 + TailwindCSS

## 快速开始（开发）

### 前置依赖

- Go（见 `app/go.mod` 的 `go`/`toolchain` 版本要求）
- Node.js（建议 18+）
- Wails CLI（v2）

### 启动开发模式

```bash
cd app
wails dev
```

该命令会启动前端热更新，并自动与 Go 绑定（Wails runtime）。

## 构建发布包

```bash
cd app
wails build
```

产物位置与平台相关，Wails 会在构建目录下生成可分发的应用包。

## 安全与数据存储

- 连接信息会存储在用户配置目录下的加密文件中（`connections.json.enc`）。
- 加密密钥来源于“主密码”，主密码不会上传到网络。
- 修改主密码会对已保存的连接数据执行“解密 → 使用新密码重新加密”的旋转流程。

## 目录结构

- `app/`：Wails 应用（Go 后端 + 打包配置）
  - `app/internal/`：后端核心逻辑（协议适配、连接/会话、传输管理、加密存储）
  - `app/frontend/`：Svelte 前端

## 贡献

欢迎提交 Issue / PR：

- Bug 修复与协议兼容性改进
- UI/交互优化
- 自动化测试与发布流程完善

## License

本项目采用 GNU Affero General Public License v3.0（AGPL-3.0）开源协议，详见 [LICENSE](file:///Users/flute/Documents/develop/fluteTrans/LICENSE)。
