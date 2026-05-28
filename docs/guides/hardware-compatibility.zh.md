> 返回 [README](../project/README.zh.md)

# 🖥️ PicoClaw 硬件兼容性列表

PicoClaw 几乎可以在任何 Linux 设备上运行。本页面记录了已验证的芯片、产品和开发板。

**你的硬件不在列表中？** 提交 PR 来添加它！欢迎硬件厂商贡献和联合推广。

---

## 1. 已验证的芯片支持

### x86

| 厂商 | 芯片 | 备注 |
|------|------|------|
| Intel | Any x86 CPU (i386+) | 所有桌面/服务器/笔记本处理器 |
| AMD | Any x86 CPU | 所有桌面/服务器/笔记本处理器 |

### ARM

| 子架构 | 典型芯片 | 备注 |
|--------|----------|------|
| ARMv6 | [BCM2835](https://www.raspberrypi.com/documentation/computers/processors.html#bcm2835) (Raspberry Pi 1/Zero) | 单核 ARM1176JZF-S |
| ARMv7 | [Allwinner V3s](https://linux-sunxi.org/V3s) | 单核 Cortex-A7，用于 LicheePi Zero |
| ARM64 | [Allwinner H618](https://linux-sunxi.org/H618) | 四核 Cortex-A53，用于 Orange Pi Zero 3 |
| ARM64 | [BCM2711](https://www.raspberrypi.com/documentation/computers/processors.html#bcm2711) (Raspberry Pi 4) | 四核 Cortex-A72 |
| ARM64 | [BCM2712](https://www.raspberrypi.com/documentation/computers/processors.html#bcm2712) (Raspberry Pi 5) | 四核 Cortex-A76 |
| ARM64 | [AX630C](https://www.axera-tech.com/) (爱芯元智) | 双核 Cortex-A53 + NPU，用于 NanoKVM-Pro / MaixCAM2 |

### RISC-V (riscv64)

| 厂商 | 芯片 | 核心 | 备注 |
|------|------|------|------|
| [SOPHGO (算能)](https://www.sophgo.com/) | SG2002 | C906 @ 1GHz | 256MB DDR3 片上内存，用于 LicheeRV-Nano / NanoKVM / MaixCAM |
| [Allwinner (全志)](https://www.allwinnertech.com/) | V861 | Dual C907 | 128MB DDR3L 片上内存，1 TOPS NPU，4K AI 摄像头 SiP |
| [Allwinner (全志)](https://www.allwinnertech.com/) | V881 | C907 | RISC-V AI 摄像头系列 |
| [Arterytek (匠芯创)](https://www.arterytek.com/) | D213 | RISC-V | 用于 HaaS506-LD1 工业 RTU |
| [SpacemiT (进迭)](https://www.spacemit.com/) | K1 | 8x X60 @ 1.8GHz | 用于 Milk-V Jupiter, BananaPi BPI-F3 |
| [SpacemiT (进迭)](https://www.spacemit.com/) | K3 | 8x X100 @ 2.5GHz | 符合 RVA23 规范，1024 位 RVV，FP8 AI 推理 |
| [Zhihe (知合)](https://www.zhihe-tech.com/) | A210 | High-perf RISC-V | 8 核，16MB L3 缓存，桌面级 |
| [Canaan (嘉楠)](https://www.canaan-creative.com/) | K230 | Dual C908 @ 1.6GHz | 6 TOPS KPU，用于 CanMV-K230 |

### MIPS

| 厂商 | 芯片 | 备注 |
|------|------|------|
| MediaTek | [MT7620](https://www.mediatek.com/products/home-networking/mt7620) | MIPS24KEc @ 580MHz，用于许多 OpenWrt 路由器（如小米路由器 3G） |

### LoongArch (loong64)

| 厂商 | 芯片 | 备注 |
|------|------|------|
| [Loongson (龙芯)](https://www.loongson.cn/) | 3A5000 | 四核 LA464 @ 2.5GHz，桌面/工作站 |
| [Loongson (龙芯)](https://www.loongson.cn/) | 3A6000 | 四核 4C/8T @ 2.5GHz，IPC 可与 Intel 第十代相媲美 |
| [Loongson (龙芯)](https://www.loongson.cn/) | 2K1000LA | 双核 @ 1GHz，工业/物联网应用 |

---

## 2. 已验证的产品（按发布日期排列）

已通过 PicoClaw 测试的消费产品、路由器和工业设备。

| 年份 | 产品 | 架构 | SoC | 内存 | 类别 |
|------|------|------|-----|------|------|
| 2009 | Nokia N900 | ARM (A8) | OMAP3430 | 256MB | 智能手机 |
| 2012 | Samsung Galaxy Note 10.1 (N8000) | ARM (A9) | Exynos 4412 | 2GB | 平板电脑 |
| 2016 | Xiaomi Router 3G (小米路由器3G) | MIPS | MT7620 | 256MB | 路由器 (OpenWrt) |
| 2018 | Phicomm N1 (斐讯N1) | ARM64 (A53) | S905D | 2GB | 电视盒子 / 家庭服务器 |
| 2019 | Xiaomi AI Speaker (小爱音箱) | ARM64 (A53) | — | 256MB | 智能音箱 |
| 2024 | [NanoKVM](https://wiki.sipeed.com/hardware/en/kvm/NanoKVM/introduction.html) | RISC-V | SG2002 | 256MB | IP-KVM |
| 2025 | HaaS506-LD1 | RISC-V | D213 | 128MB | 工业 RTU |
| 2025 | [NanoKVM-Pro](https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Pro/introduction.html) | ARM64 (A53) | AX630C | 1GB | 专业 IP-KVM |
| 2026 | [MaixCAM2](https://wiki.sipeed.com/hardware/en/maixcam/index.html) | ARM64 (A53) | AX630C | 1/4GB | 4K AI 摄像头 |

---

## 3. 已验证的开发板（按发布日期排列）

| 年份 | 开发板 | 架构 | SoC | 内存 | 购买链接 |
|------|--------|------|-----|------|----------|
| 2012 | [Raspberry Pi 1 Model B](https://www.raspberrypi.com/products/) | ARMv6 | BCM2835 | 512MB | — |
| 2015 | [Raspberry Pi 2 Model B](https://www.raspberrypi.com/products/raspberry-pi-2-model-b/) | ARMv7 (A7) | BCM2836 | 1GB | — |
| 2015 | [Raspberry Pi Zero](https://www.raspberrypi.com/products/raspberry-pi-zero/) | ARMv6 | BCM2835 | 512MB | — |
| 2016 | [Raspberry Pi 3 Model B](https://www.raspberrypi.com/products/raspberry-pi-3-model-b/) | ARM64 (A53) | BCM2837 | 1GB | — |
| 2017 | [LicheePi Zero](https://wiki.sipeed.com/hardware/en/lichee/Zero/Zero.html) | ARMv7 (A7) | Allwinner V3s | 64MB | [Sipeed](https://sipeed.com/) |
| 2019 | [Raspberry Pi 4 Model B](https://www.raspberrypi.com/products/raspberry-pi-4-model-b/) | ARM64 (A72) | BCM2711 | 1~8GB | [RPi](https://www.raspberrypi.com/) |
| 2023 | [Raspberry Pi 5](https://www.raspberrypi.com/products/raspberry-pi-5/) | ARM64 (A76) | BCM2712 | 2~8GB | [RPi](https://www.raspberrypi.com/) |
| 2024 | [LicheeRV-Nano](https://wiki.sipeed.com/hardware/en/lichee/RV_Nano/1_intro.html) | RISC-V | SG2002 | 256MB | [AliExpress](https://www.aliexpress.com/item/1005006519668532.html) |
| 2024 | [MaixCAM-Pro](https://wiki.sipeed.com/hardware/en/maixcam/index.html) | RISC-V | SG2002 | 256MB | [Sipeed](https://sipeed.com/) |
| 2024 | [Milk-V Duo 64M](https://milkv.io/docs/duo/getting-started/duo) | RISC-V | CV1800B | 64MB | [Milk-V](https://milkv.io/) |
| 2024 | [CanMV-K230](https://developer.canaan-creative.com/k230_canmv/en/main/) | RISC-V | K230 | 512MB | [Canaan](https://www.canaan-creative.com/) |

---

## 4. 同样适用于

### Android 手机（通过 Termux）

任何 ARM64 Android 手机（2015 年以后），1GB 以上内存。安装 [Termux](https://github.com/termux/termux-app)，使用 `proot` 运行 PicoClaw。

> 参见 [README：在旧 Android 手机上运行](../project/README.zh.md#-run-on-old-android-phones) 获取设置说明。

### 桌面 / 服务器 / 云

| 平台 | 备注 |
|------|------|
| x86_64 Linux | 原生二进制文件，无依赖 |
| x86_64 Windows | 原生二进制文件 |
| macOS (Intel / Apple Silicon) | 原生二进制文件 |
| Docker (any platform) | `docker compose` 一行命令，参见 [Docker 指南](docker.md) |
| OpenWrt routers | MIPS/ARM 构建，需要 >32MB 可用内存 |
| FreeBSD / NetBSD | 提供 x86_64 和 arm64 构建 |

---

## 5. 最低要求

| 资源 | 最低要求 | 推荐配置 |
|------|----------|----------|
| 内存 | 10MB 可用 | 32MB 以上可用 |
| 存储 | 20MB（二进制文件） | 50MB 以上（含工作区） |
| CPU | 任意（单核 0.6GHz 以上） | — |
| 操作系统 | Linux (kernel 3.x+) | Linux 5.x+ |
| 网络 | 必需（用于 LLM API 调用） | 以太网或 WiFi |

---

## 6. 如何测试与贡献

```bash
# 1. 下载适合你架构的版本
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz

# 2. 初始化
./picoclaw onboard

# 3. 测试
./picoclaw agent -m "Hello, what board am I running on?"
```

可用构建版本：`linux-amd64`, `linux-arm64`, `linux-arm`, `linux-riscv64`, `linux-loong64`, `linux-mipsle`

### 添加你的硬件

1. Fork 本仓库
2. 将你的芯片/产品/开发板添加到相应的表格中
3. 包含：名称、架构、SoC、内存、年份，以及可用的链接
4. 提交 PR

硬件厂商：想要添加官方支持或联合推广？请提交 issue 或通过 [Discord](https://discord.gg/V4sAZ9XWpN) 联系我们。
