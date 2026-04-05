# Windows 开发环境配置文档（PicoClaw 项目）

本文档详细说明如何在 Windows 系统上配置 PicoClaw 项目的开发环境，包括安装 **Go**、**nvm-windows**、**Node.js 22.22.1** 以及 **make** 工具，并配置相关加速镜像。

------

## 📋 环境要求

- Windows 10 / 11（64 位）
- 稳定的网络连接（部分下载可能需要代理，已提供国内镜像配置）

------

## 1. 安装基础工具

### 1.1 Git（可选但推荐）

Git 用于版本控制和克隆代码库。如果未安装，请前往 [Git 官网](https://git-scm.com/download/win) 下载安装程序，按默认选项安装即可。

安装完成后，打开 Git Bash 验证：

bash

```
git --version
```



------

## 2. 安装 Golang

### 2.1 下载安装包

访问 [Go 官网下载页](https://golang.org/dl/)（国内用户可使用 [Go 中文网镜像](https://studygolang.com/dl)），选择最新稳定版的 Windows 安装包（例如 `go1.23.4.windows-amd64.msi`）。

### 2.2 安装

运行下载的 `.msi` 文件，按提示安装。默认安装路径为 `C:\Go`，建议保持默认。

安装程序会自动将 `C:\Go\bin` 添加到系统 PATH 环境变量。完成后，**重启命令提示符**或 **重新加载环境变量**。

### 2.3 验证安装

打开 **命令提示符（CMD）** 或 **PowerShell**，执行：

bash

```
go version
```



若显示类似 `go version go1.23.4 windows/amd64`，表示安装成功。

------

## 3. 配置 Go 模块代理（加速）

为了加快 Go 模块下载速度，推荐设置国内代理。

### 3.1 设置环境变量

在命令提示符或 PowerShell 中执行以下命令（永久生效）：

bash

```
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOPRIVATE=*.corp.example.com   # 如果使用私有仓库，可配置
```



也可通过系统环境变量手动添加：

- 变量名：`GOPROXY`
- 变量值：`https://goproxy.cn,direct`

### 3.2 验证

bash

```
go env GOPROXY
```



应输出 `https://goproxy.cn,direct`。

------

## 4. 安装 nvm-windows（Node 版本管理工具）

nvm-windows 可以让你方便地安装和切换 Node.js 版本。

### 4.1 下载安装

- 前往 [nvm-windows 发布页](https://github.com/coreybutler/nvm-windows/releases)
- 下载最新版本的 `nvm-setup.exe`（例如 `nvm-setup.zip` 解压后运行）
- 运行安装程序，按默认选项安装（安装路径可自定义，但避免包含空格）

### 4.2 验证安装

重新打开命令提示符，执行：

bash

```
nvm version
```



若显示版本号，如 `1.1.12`，表示安装成功。

------

## 5. 安装 Node.js 22.22.1

使用 nvm 安装指定版本。

### 5.1 安装 Node.js 22.22.1

在命令提示符中执行：

bash

```
nvm install 22.22.1
```



等待下载完成，输出类似 `Downloading node.js version 22.22.1 (64-bit)... Complete` 表示成功。

### 5.2 切换到该版本

bash

```
nvm use 22.22.1
```



### 5.3 验证

bash

```
node --version   # 应显示 v22.22.1
npm --version    # 应显示 10.x
```



------

## 6. 配置 npm 镜像加速（可选）

为了加快 npm 包下载速度，建议配置淘宝镜像。

bash

```
npm config set registry https://registry.npmmirror.com
```



验证：

bash

```
npm config get registry
```



输出 `https://registry.npmmirror.com` 即成功。

------

## 7. 安装 make 工具

Windows 默认没有 `make` 命令，需手动安装。推荐使用 **MSYS2** 获得完整的 Linux 命令行工具集。

### 7.1 安装 MSYS2

- 前往 [MSYS2 官网](https://www.msys2.org/) 下载安装程序 `msys2-x86_64-xxxx.exe`
- 运行安装程序，按默认选项安装（建议路径：`C:\msys64`）
- 安装完成后，在开始菜单找到 **MSYS2 MinGW 64-bit** 快捷方式，打开

### 7.2 更新 MSYS2 核心包

在打开的 MSYS2 终端中执行：

bash

```
pacman -Syu
```



如果提示关闭终端，请关闭后重新打开 MSYS2 终端，再次运行：

bash

```
pacman -Su
```



### 7.3 安装 make 和其他常用工具

bash

```
pacman -S make gcc base-devel
```



等待安装完成。

### 7.4 将 MSYS2 的 bin 目录添加到系统 PATH（可选）

如果你希望在普通 CMD 或 PowerShell 中也使用 `make`，需要将 `<path>\msys\usr\bin` 添加到系统 PATH 环境变量。

- 右键“此电脑” → “属性” → “高级系统设置” → “环境变量”
- 在“系统变量”中找到 `Path`，点击“编辑” → “新建”，添加 `C:\msys64\mingw64\bin`
- 点击确定保存，重启命令提示符生效。

### 7.5 验证 make

在任意终端（CMD、PowerShell 或 MSYS2 终端）执行：

bash

```
make --version
```



若显示类似 `GNU Make 4.4.1`，则安装成功。

------

## 8. 安装 pnpm（可选，用于构建 PicoClaw 前端）

PicoClaw 的 Web 前端构建依赖 pnpm。

bash

```
npm install -g pnpm
```



验证：

bash

```
pnpm --version
```



------

## 9. 完整验证所有工具

在命令提示符中依次运行以下命令，检查版本：

bash

```
go version
node --version
npm --version
pnpm --version
make --version
git --version   # 如果安装了 Git
```



所有命令均应正常显示版本号。

------

## 10. 常见问题

### Q1: `nvm install 22.22.1` 提示“下载失败”或速度慢

- 可能是网络问题，可以尝试从 [Node.js 官网](https://nodejs.org/dist/v22.22.1/) 手动下载对应系统版本的安装包，然后使用 `nvm install <path-to-exe>` 安装。
- 或者先配置 npm 镜像后再试。

### Q2: `make` 命令在 CMD 中找不到

- 检查是否已将 `C:\msys64\mingw64\bin` 添加到系统 PATH，并重启终端。

### Q3: Go 命令在 MSYS2 终端中找不到

- 确保 Go 安装目录（`C:\Go\bin`）已添加到系统 PATH，并重启 MSYS2 终端。

### Q4: 使用 Git Bash 作为默认终端

- 如果习惯了 Git Bash，可以将 MSYS2 的 `make` 也添加到 Git Bash 的 PATH 中，或在 Git Bash 中直接使用 MSYS2 的 `make` 路径（如 `/c/msys64/mingw64/bin/make`）。

### Q5: 代理冲突

- 如果公司网络有代理，请确保 `GOPROXY` 和 `npm registry` 配置为可访问的镜像。

------

## 📚 附录：环境变量参考

| 变量名         | 推荐值                           | 说明             |
| :------------- | :------------------------------- | :--------------- |
| `GOPROXY`      | `https://goproxy.cn,direct`      | Go 模块代理      |
| `GOPRIVATE`    | `*.corp.example.com`             | 私有仓库（如有） |
| `npm registry` | `https://registry.npmmirror.com` | npm 镜像         |

------

配置完成后，你就可以在 PicoClaw 项目目录下执行 `make build-launcher` 等命令进行开发构建了。如有其他问题，欢迎查阅项目文档或提交 Issue。