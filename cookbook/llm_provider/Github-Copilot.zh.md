### 将**Github Copilot**接入**picoclaw**




- [准备工作](#准备工作)
- [为你的claw配置](#为你的claw配置)
- [无法运行或崩溃](#无法运行或崩溃)
  
### 准备工作
 - 安装**github copilot cli**  ([如何安装](https://docs.github.com/zh/copilot/how-tos/copilot-cli/install-copilot-cli))，记得为你的`CLI`登录(使用环境变量/或者使用命令`/login`)，保证`CLI`可以正常使用(调试你的网络环境，api额度等等)
 - 在系统终端中执行`copilot --headless --port 4321` 以启动服务

### 为你的claw配置
```json
"github_copilot": {
      "api_key": "eqweqe",  //可以随便填 ⚠️但是不能留空
      "api_base": "localhost:4321",  //api地址 不需要加`http://`
      "connect_mode": "grpc"  //`grpc` 或者 `stdio` 
    }
```
> [!NOTE]  
> 别忘了更改你的模型提供商
```json
"defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "provider": "copilot",    //⚠️
      "model": "gpt-4.2",       //⚠️
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
```

### 无法运行或崩溃
以下给出常见错误以及应对方案


| 错误信息      | 应对方案         |
|:----------|:-------------|
| `402 状态码` | 检测并充值你的api额度 |




