================================
  PicoClaw Windows 版使用说明
================================

【快速开始】

第一步：配置
  双击 setup.bat，按照提示输入：
    1. API Key（支持 OpenAI 兼容 API）
    2. API Base URL（默认：https://api.openai.com/v1）
    3. 模型名称（默认：gpt-4）
    4. 飞书 App ID
    5. 飞书 App Secret

第二步：使用
  配置完成后，服务会自动启动。
  打开飞书，找到你的机器人，开始聊天！

第三步：停止服务
  在运行 setup.bat 的窗口按 Ctrl+C 停止服务。


【常见问题】

Q: 如何更换配置？
A: 再次双击 setup.bat 重新配置即可。新配置会覆盖旧配置。

Q: 配置文件保存在哪里？
A: 配置文件保存在 C:\Users\你的用户名\.picoclaw\config.json

Q: 如何查看日志？
A: 日志保存在 C:\Users\你的用户名\.picoclaw\logs\

Q: 启动时提示"PowerShell 执行策略限制"怎么办？
A: setup.bat 已经自动处理了此问题，如果仍然遇到，请以管理员身份运行 PowerShell，执行：
   Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

Q: 防火墙拦截怎么办？
A: 首次启动时，Windows 防火墙可能会询问是否允许访问，请选择"允许"。

Q: 杀毒软件报警怎么办？
A: 这是正常现象（程序未签名），请添加信任或暂时禁用杀毒软件。


【支持的 API】

PicoClaw 支持所有 OpenAI 兼容的 API，包括但不限于：
  - OpenAI 官方 API (https://api.openai.com/v1)
  - Azure OpenAI
  - 智谱 AI (GLM-4)
  - 月之暗面 (Moonshot)
  - 零一万物 (Yi)
  - 百川智能 (Baichuan)
  - MiniMax
  - 阿里云百炼
  - 豆包（火山引擎）
  - 其他 OpenAI 兼容 API


【获取帮助】

GitHub: https://github.com/sipeed/picoclaw
问题反馈: https://github.com/sipeed/picoclaw/issues


【许可证】

本项目采用 MIT 许可证
