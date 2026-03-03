# PicoClaw 26M2W3 社区开发者会议

> **PicoClaw的设计目标**：轻量高效，任意部署；简单易用，普惠大众；
> **致PicoClaw开发者**：让我们携手加速AI奇点的到来，共同创造并见证历史。

---

## 26M2W3 概况

### 成果
* **Github 表现**：Star 17K+，Merge 100+ PR，Contributors 70+
* **用户规模**：微信群 1600+，Discord 1300+
* **开发者规模**：微信群 ~50，Discord ~40
* **生态进展**：PicoClaw 进入 Homebrew
* **工程进展**：Provider 完成重构
* **特别鸣谢**：daming, lxowalle 在假期的努力！

### 暴露的问题
* 第一次开展大规模社区协同开发，又是在假期期间，响应速度、社区协调、工程架构方面都暴露出了很多不足。
* PicoClaw 早期 vibe-coding 的快速实现架构在蜂拥而至的 PR 面前会迅速变成“屎山”和冲突地狱。
* 为尽快合并 PR，未充分验证社区开发者的能力，也没有提供合并指导规范，过早给予 write 权限，在上面架构问题下更暴露出问题。
* 忙于以上 PR 协调问题，拖后了文档和宣发进度。特别是宣发问题，被不放春节假的海外开发者项目 zeroclaw 趁虚而入。
* ⚠️ **警惕币圈！** 尤其是 pump.fun 空气币，不要认领参与！

> **会议核心任务**：本次周会主要需要划分项目板块，认领板块负责人，制订下周计划。以下内容社区开发者可以继续添加遗漏的地方。

---

## 开发板块

### 仓库管理
* 新建 `dev` 分支，`main` 分支推送严格化。
* 完善 `CONTRIBUTING.md`。
* **时区审核分工**：
  * GMT+8 附近时区审核（中国）
  * GMT+0 附近时区审核（欧洲）：**Huaaudio**
  * GMT-8 附近时区审核（美洲）
* 仓库权限申请：联系 **zepan** 审核。
* Readme 中公布本次会议的分工人员表格，方便开发者找寻对应人员审核。

### Provider（负责人：daming）
* **进度**：已重构完成。
* **计划**：
  * 梳理支持和计划支持的 provider 协议列表及进度计划。
  * **插件系统探索**：go 原生插件？(参考 [hashicorp/go-plugin](https://github.com/hashicorp/go-plugin))
  * **优化思路**：现在各种系统的 LLM provider 都在重复造轮子，而且每新增一个 provider 都得再改代码、重新发版才能支持。应该把专业的事交给专业的组件来负责。我开了个新的开源项目——`open-next-router`，采用 nginx 原子化配置的思想，新增 provider 无需改代码，新增配置文件即可支持，提供了 go 的 sdk 包，可快速接入项目。PicoClaw 接入后可更聚焦于 agent 的实现而不是各种上游 provider 的适配，就能快其它 claw 一步。

### Channels（负责人：daming）
* **进度**：正在重构。
* **计划**：
  * 梳理支持和计划支持的 channel 协议列表及进度计划。
  * **附件支持讨论**：音频、视频、文件。
    * 附件的生命周期应该由谁管理？channel 应该只负责下载文件，然后交由 Agent 消费完成后管理生命周期？
    * 音频转文字是否要迁移到 agent 层？或者说附件应该在哪一层被处理？
    * 发送附件的方法如何拓展？添加新的方法？拓展原有 Message？
  * 群友建议的 **skill加channel**？(参考 [nanoclaw skill](https://github.com/qwibitai/nanoclaw/blob/main/.claude/skills/add-telegram/SKILL.md))
  * **插件系统讨论**。
  * **架构优化**：
    * 抽离公共的 HTTP 服务器，采用 WebHook 通信的 channel 通过复用公共的服务器来节省资源和端口。
    * Websocket 支持。
    * 将路由相关字段（`peer_kind`、`peer_id`）从 metadata 中提升为 `InboundMessage` 的结构体字段。
  * **状态管理**：聊天记录应该由 channel 管理还是 agent 管理？

### Agent（负责人：学欧）
* Agent Loop 机制优化。
* **记忆系统**：引入 SQLite。
* **Multi-Agent / Swarm** 支持。
* **模型能力回退链**：在主模型不支持多模态时，使用多模态模型进行辅助。

### Tools（负责人：学欧）
* 整理规范。
* 插件系统探索。

### Heartbeat / Status / Log 等（负责人：daming）
* 完善心跳、状态和日志监控。

### Skill
* 搜索 skill 的 skill，已合并 PR：[PR #332](https://github.com/sipeed/picoclaw/pull/332)。
* **安全与维护**：探讨 skill 的维护和安全性问题，防范目前常见的投毒现象。

### MCP（负责人：evo）
* **功能实现**：已有 PR [#376](https://github.com/sipeed/picoclaw/pull/376)、[#282](https://github.com/sipeed/picoclaw/pull/282)。
* 安卓手机操作支持。
* 浏览器操作 (`webmcp?` `action book?`)：已有相关 PR ([agent-browser-tool](https://github.com/sipeed/picoclaw/tree/feat/agent-browser-tool))。

### 占用/效率优化（负责人：学欧）
* **目标**：优化内存占用与执行效率，希望控制在 **20M 以内**。
* **分析**：分析各个版本之间的内存占用变化，分析各个模块的内存占用情况。
* **裁剪**：裁剪出最小版本，用于宣发。

### Security
* 响应并修复安全机构发送的漏洞警示。
* 参考 openclaw 等现有仓库的安全措施，加固 PicoClaw。

### AI CI（负责人：政宇）
* 完善仓库的 CI 流程。
* 加入 AI review 等自动化流程。
* 完善发布流程、测试项目、release note、breaking change 记录。
* 根目录加上 `CLAUDE.md`？
* 增加 `loongarch` & `deb/rpm` 支持。

### UX Testing
* 对 release 版进行一般性测试。
* 站在小白用户角度对使用交互提出意见建议，比如完善 PicoClaw onboard 流程。
* 展示性优化：比如启动时刷屏 ascii-art 的 PicoClaw 标识，增加用户拍摄视频时的辨识度。

### 文档工作
* 仓库 Readme 美化，仓库文档整理、规范。
* 整理所有 Channel、Provider 的实现支持列表。
* 针对小白用户的各个 Provider、Channel 详细手把手教程文档。
* 建设 Wiki 页面（deepwiki?）。

---

## Release 待办事项 (Checklist)
- [ ] Provider
- [ ] Channel
- [ ] Agent
- [ ] Swarm
- [ ] Security
- [ ] MCP：浏览器
- [ ] 文档
- [ ] Logo
- [ ] Metadata 问题解决

---

## 关于插件系统测试方案（补充记录）
测试了以下几种方案：
1. **内置的 plugin 模块**：不考虑。不支持 Windows 等平台 ([plugin](https://pkg.go.dev/plugin@go1.26.0))。
2. **hashicorp/go-plugin**：不考虑。占用资源过大，固件都增加了 20～30M。
3. **net/rpc**（client-server 模式）：
   * **优点**：支持热加载，插件可以保存运行状态。
   * **缺点**：资源消耗较多（内存约增加 5M+，每个插件大小 10+M），每个插件占用一个端口，不太优雅。
4. **encoding/gob**（编译为可执行程序，由主程序调用并获取返回值）：
   * **优点**：支持热加载，消耗资源相对较少（测试固件大小增加了 376KB，内存消耗增加了 640KB）。
   * **缺点**：无法保存运行状态（应该可以用 socket 等方法来优化支持）。

---

## 宣发板块

### 社区运营
* **宣发物料/策划**：负责人 **zepan**，再寻求 1~2 位有网感的社区成员。
  * 制作标准 Logo, Slogan。
  * 制作具有传播性的图文/视频等。
  * 策划互动性、传播性强的用户活动，产生用户内容。
  * KOL 建联等其它宣发手段。
* **微信群运营**：负责人 **zepan**。
* **推特运营**：负责人 **zepan**。
* **Discord运营**：负责人 **OsmiumOP**；需要再找一个国内开发者盯一下，会给予 admin 权限。
* **其他渠道开拓**：小红书、知乎、Reddit？
* **Go社区联络大使**：负责人 **卓**。

---

## 中期 TODO

* **桌面应用 / 安卓 APP**
  * 架构讨论：C/S 还是单程序？接口文档规范？
* **配套硬件**
