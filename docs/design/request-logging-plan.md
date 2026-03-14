# 规划：请求日志收集与统计系统

## 一、需求分析

| 需求 | 说明 |
|------|------|
| 1. 请求记录 | 记录来自不同channel的请求，以文件方式存储 |
| 2. Web查看 | 从web界面查看某时间段的请求量、来源、channel等信息 |
| 3. 日志规则 | 配置日志规则，定期归档压缩/删除 |

---

## 二、系统架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         Gateway Process                          │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │   Channel    │    │     Bus      │    │  Request    │      │
│  │   (Telegram)  │───▶│  (Message)   │───▶│   Logger     │      │
│  │   (Discord)   │    │              │    │              │      │
│  │   (Slack)     │    │              │    │              │      │
│  │   ...         │    │              │    │              │      │
│  └──────────────┘    └──────────────┘    └──────┬───────┘      │
│                                                  │               │
│                                                  ▼               │
│                                    ┌───────────────────────┐   │
│                                    │   Request Log Files   │   │
│                                    │   (JSON Lines)        │   │
│                                    └───────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Web Backend                                │
│  ┌──────────────────┐    ┌─────────────────────────────────┐    │
│  │  Request Stats   │◀───│   Log File Reader/Archiver     │    │
│  │     API          │    │   (Query, Filter, Archive)     │    │
│  └──────────────────┘    └─────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Web Frontend                               │
│  ┌──────────────────┐    ┌─────────────────────────────────┐    │
│  │  Statistics UI   │    │      Request Log Viewer        │    │
│  │  (Charts/Tables) │    │  (Filter by time/channel/user) │    │
│  └──────────────────┘    └─────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 三、详细设计方案

### 3.1 请求日志记录模块

**位置**: `pkg/requestlog/`

| 文件 | 职责 |
|------|------|
| `logger.go` | 核心日志写入器，实现 `bus.Subscriber` 接口 |
| `record.go` | 定义 RequestRecord 结构体 |
| `storage.go` | 文件存储管理，按日期/大小分割文件 |
| `config.go` | 日志配置结构体 |

**核心数据结构**:
```go
// RequestRecord represents a single incoming request
type RequestRecord struct {
    Timestamp     time.Time   `json:"timestamp"`      // 请求时间
    RequestID     string      `json:"request_id"`    // 唯一请求ID
    Channel       string      `json:"channel"`        // 来源channel: telegram, discord, etc.
    SenderID      string      `json:"sender_id"`     // 发送者ID
    SenderInfo    SenderInfo  `json:"sender_info"`   // 发送者详细信息
    ChatID        string      `json:"chat_id"`       // 会话ID
    Content       string      `json:"content"`      // 请求内容 (可截断)
    ContentLength int         `json:"content_len"`  // 内容长度
    Peer          Peer        `json:"peer"`         // 对等方信息
    MessageID     string      `json:"message_id"`    // 平台消息ID
    MediaCount    int         `json:"media_count"`  // 附件数量
    SessionKey    string      `json:"session_key"`  // 会话Key
    ProcessingTime int64      `json:"proc_time_ms"` // 处理耗时(ms)
}
```

**日志文件格式**:
- 文件格式: JSON Lines (每行一个JSON对象)
- 文件命名: `requests-2024-01-15.jsonl`, `requests-2024-01-15.jsonl.1.gz`
- 存储路径: `{data_dir}/logs/requests/`

### 3.2 日志归档模块

**位置**: `pkg/requestlog/archiver.go`

| 功能 | 说明 |
|------|------|
| 定时归档 | 按配置的时间间隔归档日志文件 |
| 压缩归档 | 使用gzip压缩归档文件 |
| 自动清理 | 按保留天数自动删除过期日志 |
| 配置项 | `retention_days`, `archive_interval`, `max_file_size_mb` |

**配置结构**:
```go
type LogConfig struct {
    Enabled         bool   `json:"enabled"`
    LogDir          string `json:"log_dir"`           // 日志目录
    MaxFileSizeMB   int    `json:"max_file_size_mb"` // 单文件大小上限
    MaxFiles        int    `json:"max_files"`        // 保留文件数
    RetentionDays   int    `json:"retention_days"`   // 保留天数
    ArchiveInterval  string `json:"archive_interval"` // 归档间隔: "1h", "24h"
    CompressArchive  bool   `json:"compress_archive"` // 是否压缩归档
}
```

### 3.3 后端API设计

**新增API端点**:

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/stats/requests` | 获取请求统计数据 |
| GET | `/api/stats/requests/channels` | 按channel统计请求量 |
| GET | `/api/stats/requests/timeline` | 时间线统计(按小时/天) |
| GET | `/api/logs/requests` | 查询请求日志(分页) |
| GET | `/api/logs/requests/export` | 导出日志(支持过滤) |
| GET | `/api/config/requestlog` | 获取日志配置 |
| PUT | `/api/config/requestlog` | 更新日志配置 |
| POST | `/api/logs/requests/archive-now` | 手动触发归档 |

**统计API响应示例**:
```json
// GET /api/stats/requests?start=2024-01-01&end=2024-01-31
{
  "total": 1250,
  "by_channel": {
    "telegram": 450,
    "discord": 380,
    "slack": 280,
    "feishu": 140
  },
  "by_day": [
    {"date": "2024-01-01", "count": 45},
    {"date": "2024-01-02", "count": 52}
  ],
  "top_senders": [
    {"sender": "user:123", "channel": "telegram", "count": 28}
  ]
}
```

### 3.4 前端界面设计

**新增页面/组件**:

| 路径 | 组件 | 功能 |
|------|------|------|
| `/stats` | ` 请求统计概览 |
| `/statsStatsPage` |/channels` | `ChannelStats` | 按channel统计 |
| `/stats/timeline` | `TimelineChart` | 时间线图表 |
| `/logs/requests` | `RequestLogViewer` | 请求日志查看器 |
| `/settings/logs` | `LogSettingsPanel` | 日志配置面板 |

**Stats页面设计**:
- 顶部: 关键指标卡片 (总请求量, 今日请求, 最活跃channel, 平均响应时间)
- 中部: 时间线图表 (支持按小时/天/周切换)
- 底部: Channel分布饼图 + Top用户列表

**Request Log Viewer设计**:
- 顶部: 时间范围选择器 + Channel过滤器 + 用户搜索
- 中部: 日志表格 (可排序, 可分页)
- 功能: 点击行展开详情, 支持导出CSV/JSON

---

## 四、实现步骤

### 阶段一: 基础功能

| 序号 | 任务 | 涉及文件 |
|------|------|----------|
| 1.1 | 创建 `pkg/requestlog/` 模块 | 新建目录 |
| 1.2 | 实现 RequestRecord 数据结构 | `pkg/requestlog/record.go` |
| 1.3 | 实现日志写入器 (实现 bus.Subscriber) | `pkg/requestlog/logger.go` |
| 1.4 | 集成到Gateway: 订阅 bus.InboundMessage | 修改gateway启动逻辑 |
| 1.5 | 单元测试 | `pkg/requestlog/*_test.go` |

### 阶段二: 归档功能

| 序号 | 任务 | 涉及文件 |
|------|------|----------|
| 2.1 | 实现日志文件管理 (按日期/大小分割) | `pkg/requestlog/storage.go` |
| 2.2 | 实现归档功能 (压缩/移动) | `pkg/requestlog/archiver.go` |
| 2.3 | 实现定时任务调度 | `pkg/requestlog/scheduler.go` |
| 2.4 | 配置管理与持久化 | `pkg/requestlog/config.go` |

### 阶段三: Web API

| 序号 | 任务 | 涉及文件 |
|------|------|----------|
| 3.1 | 添加统计API (按channel/时间) | `web/backend/api/stats.go` |
| 3.2 | 添加日志查询API (分页/过滤) | `web/backend/api/requestlog.go` |
| 3.3 | 添加配置API | `web/backend/api/requestlog.go` |
| 3.4 | 注册新路由 | `web/backend/api/router.go` |

### 阶段四: 前端界面

| 序号 | 任务 | 涉及文件 |
|------|------|----------|
| 4.1 | 创建统计API客户端 | `web/frontend/src/api/stats.ts` |
| 4.2 | 创建日志查看API客户端 | `web/frontend/src/api/requestlog.ts` |
| 4.3 | 实现 Stats 页面 | `web/frontend/src/routes/stats.tsx` |
| 4.4 | 实现 Request Log Viewer | `web/frontend/src/routes/logs/requests.tsx` |
| 4.5 | 实现日志配置面板 | `web/frontend/src/routes/settings/logs.tsx` |
| 4.6 | 添加侧边栏导航 | `web/frontend/src/components/app-sidebar.tsx` |

### 阶段五: 完善与优化

| 序号 | 任务 | 涉及文件 |
|------|------|----------|
| 5.1 | 前端图表集成 (Recharts/Chart.js) | package.json |
| 5.2 | 国际化支持 (en.json, zh.json) | `web/frontend/src/i18n/` |
| 5.3 | 性能优化 (大数据量分页/虚拟滚动) | 前端组件 |
| 5.4 | 日志导出功能 (CSV/JSON) | 后端API + 前端 |

---

## 五、配置文件示例

```json
{
  "requestlog": {
    "enabled": true,
    "log_dir": "~/.picoclaw/logs/requests",
    "max_file_size_mb": 100,
    "max_files": 100,
    "retention_days": 30,
    "archive_interval": "24h",
    "compress_archive": true,
    "log_content_max_length": 1000,
    "record_media": false
  }
}
```

---

## 六、注意事项

1. **性能考虑**: 日志写入应异步进行，避免阻塞消息处理
2. **存储空间**: 定期检查磁盘使用情况，设置合理保留策略
3. **敏感信息**: 对日志内容进行脱敏处理，避免记录敏感信息
4. **向后兼容**: 归档文件格式应考虑未来兼容性
5. **配置热更新**: 支持在不重启服务的情况下更新配置

---

## 七、依赖项

| 组件 | 用途 | 建议库 |
|------|------|--------|
| 时间处理 | 时间解析/格式化 | 标准库 `time` |
| 文件压缩 | 归档压缩 | 标准库 `compress/gzip` |
| 定时任务 | 归档调度 | 标准库 `time.Ticker` 或 `github.com/robfig/cron` |
| 前端图表 | 统计可视化 | `recharts` (已在项目中) |
| 前端表格 | 日志展示 | `tanstack/react-table` 或自定义 |
