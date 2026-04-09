<div align="center">
<img src="assets/logo.webp" alt="PicoClaw" width="512">

<h1>PicoClaw: Go로 작성된 초고효율 AI 어시스턴트</h1>

<h3>$10 하드웨어 · 10MB RAM · ms 부팅 · Let's Go, PicoClaw!</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20MIPS%2C%20RISC--V%2C%20LoongArch-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://docs.picoclaw.io/"><img src="https://img.shields.io/badge/Docs-Official-007acc?style=flat&logo=read-the-docs&logoColor=white" alt="Docs"></a>
    <a href="https://deepwiki.com/sipeed/picoclaw"><img src="https://img.shields.io/badge/Wiki-DeepWiki-FFA500?style=flat&logo=wikipedia&logoColor=white" alt="Wiki"></a>
    <br>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
    <a href="./assets/wechat.png"><img src="https://img.shields.io/badge/WeChat-Group-41d56b?style=flat&logo=wechat&logoColor=white"></a>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Community-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>

[中文](README.zh.md) | [日本語](README.ja.md) | **한국어** | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | [Italiano](README.it.md) | [Bahasa Indonesia](README.id.md) | [Malay](README.my.md) | [English](README.md)

</div>

---

> **PicoClaw**는 [Sipeed](https://sipeed.com)가 시작한 독립적인 오픈소스 프로젝트입니다. 처음부터 끝까지 **Go**로 새로 작성되었으며, OpenClaw, NanoBot, 혹은 다른 어떤 프로젝트의 포크도 아닙니다.

**PicoClaw**는 [NanoBot](https://github.com/HKUDS/nanobot)에서 영감을 받은 초경량 개인용 AI 어시스턴트입니다. **Go**로 처음부터 다시 구현되었고, "셀프 부트스트래핑" 방식으로 만들어졌습니다. 즉, AI 에이전트 자체가 아키텍처 전환과 코드 최적화를 주도했습니다.

**$10 하드웨어에서 10MB 미만 RAM으로 동작**합니다. OpenClaw보다 메모리를 99% 적게 쓰고, Mac mini보다 98% 저렴합니다!

<table align="center">
<tr align="center">
<td align="center" valign="top">
<p align="center">
<img src="assets/picoclaw_mem.gif" width="360" height="240">
</p>
</td>
<td align="center" valign="top">
<p align="center">
<img src="assets/licheervnano.png" width="400" height="240">
</p>
</td>
</tr>
</table>

> [!CAUTION]
> **보안 안내**
>
> * **암호화폐 없음:** PicoClaw는 공식 토큰이나 암호화폐를 **발행한 적이 없습니다**. `pump.fun` 또는 기타 거래 플랫폼에서의 모든 주장은 **사기**입니다.
> * **공식 도메인:** **유일한** 공식 웹사이트는 **[picoclaw.io](https://picoclaw.io)** 이며, 회사 웹사이트는 **[sipeed.com](https://sipeed.com)** 입니다.
> * **주의:** 많은 `.ai/.org/.com/.net/...` 도메인이 제3자에 의해 등록되어 있습니다. 신뢰하지 마세요.
> * **참고:** PicoClaw는 빠르게 초기 개발이 진행 중입니다. 아직 해결되지 않은 보안 문제가 있을 수 있습니다. v1.0 이전에는 프로덕션 배포를 권장하지 않습니다.
> * **참고:** PicoClaw는 최근 많은 PR을 병합했습니다. 최근 빌드는 10~20MB RAM을 사용할 수 있습니다. 기능이 안정화된 뒤 리소스 최적화를 진행할 예정입니다.

## 📢 뉴스

2026-03-31 📱 **Android 지원!** PicoClaw가 이제 Android에서 실행됩니다! APK는 [picoclaw.io](https://picoclaw.io/download)에서 다운로드하세요.

2026-03-25 🚀 **v0.2.4 출시!** 에이전트 아키텍처 전면 개편(SubTurn, Hooks, Steering, EventBus), WeChat/WeCom 통합, 보안 강화(`.security.yml`, 민감 정보 필터링), 새 프로바이더(AWS Bedrock, Azure, Xiaomi MiMo), 그리고 35건의 버그 수정이 포함되었습니다. PicoClaw는 **26K 스타**를 달성했습니다!

2026-03-17 🚀 **v0.2.3 출시!** 시스템 트레이 UI(Windows 및 Linux), 서브에이전트 상태 조회(`spawn_status`), 실험적 게이트웨이 핫 리로드, Cron 보안 게이트, 그리고 2건의 보안 수정이 추가되었습니다. PicoClaw는 **25K 스타**를 달성했습니다!

2026-03-09 🎉 **v0.2.1 — 역대 최대 업데이트!** MCP 프로토콜 지원, 4개의 새 채널(Matrix/IRC/WeCom/Discord Proxy), 3개의 새 프로바이더(Kimi/Minimax/Avian), 비전 파이프라인, JSONL 메모리 저장소, 모델 라우팅이 추가되었습니다.

2026-02-28 📦 **v0.2.0** 이 Docker Compose 및 WebUI 런처 지원과 함께 출시되었습니다.

<details>
<summary>이전 뉴스...</summary>

2026-02-26 🎉 PicoClaw가 단 17일 만에 **20K 스타**를 달성했습니다! 채널 자동 오케스트레이션과 기능 인터페이스가 적용되었습니다.

2026-02-16 🎉 PicoClaw가 1주일 만에 **12K 스타**를 돌파했습니다! 커뮤니티 메인터너 역할과 [로드맵](ROADMAP.md)이 공식적으로 공개되었습니다.

2026-02-13 🎉 PicoClaw가 4일 만에 **5000 스타**를 돌파했습니다! 프로젝트 로드맵과 개발자 그룹이 준비 중입니다.

2026-02-09 🎉 **PicoClaw 출시!** $10 하드웨어와 10MB 미만 RAM에서 동작하는 AI 에이전트를 단 1일 만에 만들었습니다. Let's Go, PicoClaw!

</details>

## ✨ 기능

🪶 **초경량**: 코어 메모리 사용량이 10MB 미만으로 OpenClaw보다 99% 작습니다.*

💰 **최소 비용**: $10짜리 하드웨어에서도 충분히 구동되어 Mac mini보다 98% 저렴합니다.

⚡️ **초고속 부팅**: 시작 속도가 400배 빠릅니다. 0.6GHz 싱글코어 프로세서에서도 1초 미만에 부팅됩니다.

🌍 **진정한 이식성**: RISC-V, ARM, MIPS, x86 아키텍처 전반에 단일 바이너리로 동작합니다. 하나의 바이너리로 어디서나 실행됩니다!

🤖 **AI 부트스트래핑**: 순수 Go 네이티브 구현입니다. 코어 코드의 95%는 에이전트가 생성했고, 사람이 검토하며 다듬었습니다.

🔌 **MCP 지원**: 네이티브 [Model Context Protocol](https://modelcontextprotocol.io/) 통합을 제공하여 어떤 MCP 서버든 연결해 에이전트 기능을 확장할 수 있습니다.

👁️ **비전 파이프라인**: 이미지와 파일을 에이전트에 직접 보낼 수 있으며, 멀티모달 LLM용 base64 인코딩이 자동으로 처리됩니다.

🧠 **스마트 라우팅**: 규칙 기반 모델 라우팅으로 간단한 질의는 경량 모델에 보내 API 비용을 절약합니다.

_*최근 빌드는 급격한 PR 병합으로 인해 10~20MB를 사용할 수 있습니다. 리소스 최적화는 계획되어 있습니다. 부팅 속도 비교는 0.8GHz 싱글코어 벤치마크를 기준으로 합니다(아래 표 참고)._

<div align="center">

|                                | OpenClaw      | NanoBot                  | **PicoClaw**                           |
| ------------------------------ | ------------- | ------------------------ | -------------------------------------- |
| **언어**                       | TypeScript    | Python                   | **Go**                                 |
| **RAM**                        | >1GB          | >100MB                   | **< 10MB***                            |
| **부팅 시간**</br>(0.8GHz 코어) | >500초        | >30초                    | **<1초**                               |
| **비용**                       | Mac Mini $599 | 대부분의 Linux 보드 ~$50 | **모든 Linux 보드**</br>**최저 $10부터** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

</div>

> **[하드웨어 호환 목록](docs/hardware-compatibility.md)** — 테스트된 모든 보드를 확인하세요. $5 RISC-V 보드부터 Raspberry Pi, Android 스마트폰까지 포함됩니다. 사용 중인 보드가 없나요? PR을 보내주세요!

<p align="center">
<img src="assets/hardware-banner.jpg" alt="PicoClaw Hardware Compatibility" width="100%">
</p>

## 🦾 데모

### 🛠️ 표준 어시스턴트 워크플로

<table align="center">
<tr align="center">
<th><p align="center">풀스택 엔지니어 모드</p></th>
<th><p align="center">로깅 및 계획</p></th>
<th><p align="center">웹 검색 및 학습</p></th>
</tr>
<tr>
<td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
</tr>
<tr>
<td align="center">개발 · 배포 · 확장</td>
<td align="center">스케줄링 · 자동화 · 기억</td>
<td align="center">탐색 · 인사이트 · 트렌드</td>
</tr>
</table>

### 🐜 혁신적인 초저사양 배포

PicoClaw는 사실상 거의 모든 Linux 장치에 배포할 수 있습니다!

- 최소형 홈 어시스턴트를 위해 $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) E(이더넷) 또는 W(WiFi6) 에디션
- 서버 자동 운영을 위해 $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html) 또는 $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html)
- 스마트 감시를 위해 $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) 또는 $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera)

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 더 많은 배포 사례가 기다리고 있습니다!

## 📦 설치

### picoclaw.io에서 다운로드(권장)

**[picoclaw.io](https://picoclaw.io)** 를 방문하세요. 공식 웹사이트가 플랫폼을 자동 감지하고 원클릭 다운로드를 제공합니다. 아키텍처를 직접 고를 필요가 없습니다.

### 사전 컴파일된 바이너리 다운로드

또는 [GitHub Releases](https://github.com/sipeed/picoclaw/releases) 페이지에서 플랫폼에 맞는 바이너리를 다운로드할 수 있습니다.

### 소스에서 빌드(개발용)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# 코어 바이너리 빌드
make build

# WebUI 런처 빌드 (WebUI 모드에 필요)
make build-launcher

# 여러 플랫폼용 빌드
make build-all

# Raspberry Pi Zero 2 W용 빌드 (32비트: make build-linux-arm, 64비트: make build-linux-arm64)
make build-pi-zero

# 빌드 후 설치
make install
```

**Raspberry Pi Zero 2 W:** OS에 맞는 바이너리를 사용하세요. 32비트 Raspberry Pi OS는 `make build-linux-arm`, 64비트는 `make build-linux-arm64`입니다. 또는 `make build-pi-zero`로 둘 다 빌드할 수 있습니다.

## 🚀 빠른 시작 가이드

### 🌐 WebUI Launcher (데스크톱 권장)

WebUI Launcher는 설정과 채팅을 위한 브라우저 기반 인터페이스를 제공합니다. 명령줄을 몰라도 가장 쉽게 시작할 수 있는 방법입니다.

**옵션 1: 더블클릭(데스크톱)**

[picoclaw.io](https://picoclaw.io)에서 다운로드한 뒤 `picoclaw-launcher`를 더블클릭하세요(Windows에서는 `picoclaw-launcher.exe`). 브라우저가 자동으로 `http://localhost:18800`을 엽니다.

**옵션 2: 명령줄**

```bash
picoclaw-launcher
# 브라우저에서 http://localhost:18800 열기
```

> [!TIP]
> **원격 접속 / Docker / VM:** 모든 인터페이스에서 수신하려면 `-public` 플래그를 추가하세요.
> ```bash
> picoclaw-launcher -public
> ```

<p align="center">
<img src="assets/launcher-webui.jpg" alt="WebUI Launcher" width="600">
</p>

**시작 방법:**

WebUI를 연 뒤 다음 순서로 진행하세요. **1)** 프로바이더 설정(LLM API 키 추가) -> **2)** 채널 설정(예: Telegram) -> **3)** 게이트웨이 시작 -> **4)** 채팅!

자세한 WebUI 문서는 [docs.picoclaw.io](https://docs.picoclaw.io)를 참고하세요.

<details>
<summary><b>Docker(대안)</b></summary>

```bash
# 1. 이 저장소를 클론
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. 첫 실행 - docker/data/config.json을 자동 생성한 뒤 종료
#    (config.json과 workspace/가 모두 없을 때만 실행됨)
docker compose -f docker/docker-compose.yml --profile launcher up
# 컨테이너가 "First-run setup complete."를 출력하고 종료됩니다.

# 3. API 키 설정
vim docker/data/config.json

# 4. 시작
docker compose -f docker/docker-compose.yml --profile launcher up -d
# http://localhost:18800 열기
```

> **Docker / VM 사용자:** 게이트웨이는 기본적으로 `127.0.0.1`에서 수신합니다. 호스트에서 접근 가능하게 하려면 `PICOCLAW_GATEWAY_HOST=0.0.0.0`을 설정하거나 `-public` 플래그를 사용하세요.

```bash
# 로그 확인
docker compose -f docker/docker-compose.yml logs -f

# 중지
docker compose -f docker/docker-compose.yml --profile launcher down

# 업데이트
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

</details>

<details>
<summary><b>macOS - 첫 실행 보안 경고</b></summary>

macOS에서는 인터넷에서 다운로드한 앱이고 Mac App Store 공증을 거치지 않았기 때문에, 첫 실행 시 `picoclaw-launcher`가 차단될 수 있습니다.

**1단계:** `picoclaw-launcher`를 더블클릭합니다. 그러면 보안 경고가 표시됩니다.

<p align="center">
<img src="assets/macos-gatekeeper-warning.jpg" alt="macOS Gatekeeper warning" width="400">
</p>

> *"picoclaw-launcher"을(를) 열 수 없습니다. Apple에서 이 앱이 악성 소프트웨어가 없으며 Mac이나 개인 정보를 해치지 않는다고 확인할 수 없습니다.*

**2단계:** **시스템 설정** -> **개인정보 보호 및 보안** 으로 이동한 뒤 **보안** 섹션까지 스크롤하여 **그래도 열기(Open Anyway)** 를 클릭하고, 대화상자에서 다시 한 번 **그래도 열기**를 확인합니다.

<p align="center">
<img src="assets/macos-gatekeeper-allow.jpg" alt="macOS Privacy & Security — Open Anyway" width="600">
</p>

이 과정을 한 번만 거치면 이후에는 `picoclaw-launcher`가 정상적으로 열립니다.

</details>

### 💻 TUI Launcher (헤드리스 / SSH 권장)

TUI(Terminal UI) Launcher는 설정과 관리를 위한 모든 기능을 갖춘 터미널 인터페이스를 제공합니다. 서버, Raspberry Pi, 기타 헤드리스 환경에 적합합니다.

```bash
picoclaw-launcher-tui
```

<p align="center">
<img src="assets/launcher-tui.jpg" alt="TUI Launcher" width="600">
</p>

**시작 방법:**

TUI 메뉴를 사용해 다음 순서로 진행하세요. **1)** 프로바이더 설정 -> **2)** 채널 설정 -> **3)** 게이트웨이 시작 -> **4)** 채팅!

자세한 TUI 문서는 [docs.picoclaw.io](https://docs.picoclaw.io)를 참고하세요.

### 📱 Android

오래된 스마트폰에 새 생명을 불어넣어 보세요! PicoClaw를 설치하면 스마트 AI 어시스턴트로 바꿀 수 있습니다.

**옵션 1: APK 설치**

미리보기:

<table>
  <tr>
    <td><img src="assets/fui_main_page.jpg" width="200"></td>
    <td><img src="assets/fui_web_page.jpg" width="200"></td>
    <td><img src="assets/fui_log_page.jpg" width="200"></td>
    <td><img src="assets/fui_setting_page.jpg" width="200"></td>
  </tr>
</table>

[picoclaw.io](https://picoclaw.io/download/)에서 APK를 다운로드해 바로 설치하세요. Termux가 필요 없습니다!

**옵션 2: Termux**

<details>
<summary><b>터미널 런처 (리소스 제약 환경용)</b></summary>

1. [Termux](https://github.com/termux/termux-app)를 설치합니다([GitHub Releases](https://github.com/termux/termux-app/releases)에서 다운로드하거나 F-Droid / Google Play에서 검색).
2. 다음 명령을 실행합니다.

```bash
# 최신 릴리스 다운로드
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard   # chroot가 표준 Linux 파일시스템 레이아웃을 제공합니다
```

그다음 아래의 터미널 런처 섹션을 따라 설정을 마무리하세요.

<img src="assets/termux.jpg" alt="PicoClaw on Termux" width="512">

런처 UI 없이 `picoclaw` 코어 바이너리만 있는 최소 환경에서는 명령줄과 JSON 설정 파일만으로도 모든 설정을 마칠 수 있습니다.

**1. 초기화**

```bash
picoclaw onboard
```

그러면 `~/.picoclaw/config.json`과 워크스페이스 디렉터리가 생성됩니다.

**2. 설정** (`~/.picoclaw/config.json`)

```jsonc
{
  "agents": {
    "defaults": {
      "model_name": "gpt-5.4"
    }
  },
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      // api_key는 이제 .security.yml에서 로드됩니다.
    }
  ]
}
```

> 사용 가능한 모든 옵션이 포함된 전체 설정 템플릿은 저장소의 `config/config.example.json`을 참고하세요.
>
> 참고: `config.example.json` 형식은 버전 0이며 민감 정보가 포함되어 있습니다. 실행 시 자동으로 버전 1+로 마이그레이션되며, 이후 `config.json`에는 비민감 정보만 저장되고 민감 정보는 `.security.yml`에 저장됩니다. 민감 정보를 직접 수정해야 한다면 `docs/security_configuration.md`를 참고하세요.

**3. 채팅**

```bash
# 단발성 질문
picoclaw agent -m "2+2는 얼마야?"

# 대화형 모드
picoclaw agent

# 채팅 앱 연동용 게이트웨이 시작
picoclaw gateway
```

</details>

## 🔌 프로바이더(LLM)

PicoClaw는 `model_list` 설정을 통해 30개 이상의 LLM 프로바이더를 지원합니다. 형식은 `protocol/model`입니다.

| 프로바이더 | 프로토콜 | API Key | 비고 |
|----------|----------|---------|------|
| [OpenAI](https://platform.openai.com/api-keys) | `openai/` | 필수 | GPT-5.4, GPT-4o, o3 등 |
| [Anthropic](https://console.anthropic.com/settings/keys) | `anthropic/` | 필수 | Claude Opus 4.6, Sonnet 4.6 등 |
| [Google Gemini](https://aistudio.google.com/apikey) | `gemini/` | 필수 | Gemini 3 Flash, 2.5 Pro 등 |
| [OpenRouter](https://openrouter.ai/keys) | `openrouter/` | 필수 | 200개 이상의 모델, 통합 API |
| [Zhipu (GLM)](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) | `zhipu/` | 필수 | GLM-4.7, GLM-5 등 |
| [DeepSeek](https://platform.deepseek.com/api_keys) | `deepseek/` | 필수 | DeepSeek-V3, DeepSeek-R1 |
| [Volcengine](https://console.volcengine.com) | `volcengine/` | 필수 | Doubao, Ark 모델 |
| [Qwen](https://dashscope.console.aliyun.com/apiKey) | `qwen/` | 필수 | Qwen3, Qwen-Max 등 |
| [Groq](https://console.groq.com/keys) | `groq/` | 필수 | 빠른 추론(Llama, Mixtral) |
| [Moonshot (Kimi)](https://platform.moonshot.cn/console/api-keys) | `moonshot/` | 필수 | Kimi 모델 |
| [Minimax](https://platform.minimaxi.com/user-center/basic-information/interface-key) | `minimax/` | 필수 | MiniMax 모델 |
| [Mistral](https://console.mistral.ai/api-keys) | `mistral/` | 필수 | Mistral Large, Codestral |
| [NVIDIA NIM](https://build.nvidia.com/) | `nvidia/` | 필수 | NVIDIA 호스팅 모델 |
| [Cerebras](https://cloud.cerebras.ai/) | `cerebras/` | 필수 | 빠른 추론 |
| [Novita AI](https://novita.ai/) | `novita/` | 필수 | 다양한 오픈 모델 |
| [Xiaomi MiMo](https://platform.xiaomimimo.com/) | `mimo/` | 필수 | MiMo 모델 |
| [Ollama](https://ollama.com/) | `ollama/` | 불필요 | 로컬 모델, 셀프 호스팅 |
| [vLLM](https://docs.vllm.ai/) | `vllm/` | 불필요 | 로컬 배포, OpenAI 호환 |
| [LiteLLM](https://docs.litellm.ai/) | `litellm/` | 환경에 따라 다름 | 100개 이상의 프로바이더를 위한 프록시 |
| [Azure OpenAI](https://portal.azure.com/) | `azure/` | 필수 | 엔터프라이즈 Azure 배포 |
| [GitHub Copilot](https://github.com/features/copilot) | `github-copilot/` | OAuth | 디바이스 코드 로그인 |
| [Antigravity](https://console.cloud.google.com/) | `antigravity/` | OAuth | Google Cloud AI |
| [AWS Bedrock](https://console.aws.amazon.com/bedrock)* | `bedrock/` | AWS 자격 증명 | AWS에서 Claude, Llama, Mistral 사용 |

> \* AWS Bedrock은 빌드 태그 `go build -tags bedrock`이 필요합니다. 모든 AWS 파티션(aws, aws-cn, aws-us-gov)에서 엔드포인트를 자동 해석하려면 `api_base`를 리전명(예: `us-east-1`)으로 설정하세요. 전체 엔드포인트 URL을 직접 사용할 경우에는 환경 변수 또는 AWS config/profile을 통해 `AWS_REGION`도 함께 설정해야 합니다.

<details>
<summary><b>로컬 배포(Ollama, vLLM 등)</b></summary>

**Ollama:**
```json
{
  "model_list": [
    {
      "model_name": "local-llama",
      "model": "ollama/llama3.1:8b",
      "api_base": "http://localhost:11434/v1"
    }
  ]
}
```

**vLLM:**
```json
{
  "model_list": [
    {
      "model_name": "local-vllm",
      "model": "vllm/your-model",
      "api_base": "http://localhost:8000/v1"
    }
  ]
}
```

프로바이더 전체 설정은 [프로바이더와 모델](docs/providers.md)을 참고하세요.

</details>

## 💬 채널(채팅 앱)

18개 이상의 메시징 플랫폼을 통해 PicoClaw와 대화할 수 있습니다.

| 채널 | 설정 | 프로토콜 | 문서 |
|---------|------|----------|------|
| **Telegram** | 쉬움(봇 토큰) | Long polling | [가이드](docs/channels/telegram/README.md) |
| **Discord** | 쉬움(봇 토큰 + intents) | WebSocket | [가이드](docs/channels/discord/README.md) |
| **WhatsApp** | 쉬움(QR 스캔 또는 브리지 URL) | Native / Bridge | [가이드](docs/chat-apps.md#whatsapp) |
| **Weixin** | 쉬움(네이티브 QR 스캔) | iLink API | [가이드](docs/chat-apps.md#weixin) |
| **QQ** | 쉬움(AppID + AppSecret) | WebSocket | [가이드](docs/channels/qq/README.md) |
| **Slack** | 쉬움(봇 + 앱 토큰) | Socket Mode | [가이드](docs/channels/slack/README.md) |
| **Matrix** | 중간(homeserver + 토큰) | Sync API | [가이드](docs/channels/matrix/README.md) |
| **DingTalk** | 중간(클라이언트 자격 증명) | Stream | [가이드](docs/channels/dingtalk/README.md) |
| **Feishu / Lark** | 중간(App ID + Secret) | WebSocket/SDK | [가이드](docs/channels/feishu/README.md) |
| **LINE** | 중간(인증 정보 + webhook) | Webhook | [가이드](docs/channels/line/README.md) |
| **WeCom** | 쉬움(QR 로그인 또는 수동 설정) | WebSocket | [가이드](docs/channels/wecom/README.md) |
| **VK** | 쉬움(그룹 토큰) | Long Poll | [가이드](docs/channels/vk/README.md) |
| **IRC** | 중간(서버 + 닉네임) | IRC protocol | [가이드](docs/chat-apps.md#irc) |
| **OneBot** | 중간(WebSocket URL) | OneBot v11 | [가이드](docs/channels/onebot/README.md) |
| **MaixCam** | 쉬움(활성화) | TCP socket | [가이드](docs/channels/maixcam/README.md) |
| **Pico** | 쉬움(활성화) | 네이티브 프로토콜 | 내장 |
| **Pico Client** | 쉬움(WebSocket URL) | WebSocket | 내장 |

> webhook 기반 채널은 모두 하나의 게이트웨이 HTTP 서버(`gateway.host`:`gateway.port`, 기본값 `127.0.0.1:18790`)를 공유합니다. Feishu는 WebSocket/SDK 모드를 사용하며 이 공용 HTTP 서버를 사용하지 않습니다.

> 로그 상세도는 `gateway.log_level`(기본값: `warn`)로 제어됩니다. 지원 값은 `debug`, `info`, `warn`, `error`, `fatal`입니다. `PICOCLAW_LOG_LEVEL` 환경 변수로도 설정할 수 있습니다. 자세한 내용은 [설정 문서](docs/configuration.md#gateway-log-level)를 참고하세요.

자세한 채널 설정 방법은 [채팅 앱 설정 가이드](docs/chat-apps.md)를 참고하세요.

## 🔧 도구

### 🔍 웹 검색

PicoClaw는 최신 정보를 제공하기 위해 웹 검색을 수행할 수 있습니다. `tools.web`에서 설정하세요.

| 검색 엔진 | API Key | 무료 제공량 | 링크 |
|-----------|---------|-------------|------|
| DuckDuckGo | 불필요 | 무제한 | 내장 백업 검색 |
| [Baidu Search](https://cloud.baidu.com/doc/qianfan-api/s/Wmbq4z7e5) | 필수 | 하루 1000회 쿼리 | AI 기반, 중국 시장 최적화 |
| [Tavily](https://tavily.com) | 필수 | 월 1000회 쿼리 | AI 에이전트에 최적화 |
| [Brave Search](https://brave.com/search/api) | 필수 | 월 2000회 쿼리 | 빠르고 프라이빗함 |
| [Perplexity](https://www.perplexity.ai) | 필수 | 유료 | AI 기반 검색 |
| [SearXNG](https://github.com/searxng/searxng) | 불필요 | 셀프 호스팅 | 무료 메타 검색 엔진 |
| [GLM Search](https://open.bigmodel.cn/) | 필수 | 상이함 | Zhipu 웹 검색 |

### ⚙️ 기타 도구

PicoClaw에는 파일 작업, 코드 실행, 스케줄링 등을 위한 내장 도구가 포함되어 있습니다. 자세한 내용은 [도구 설정](docs/tools_configuration.md)을 참고하세요.

## 🎯 스킬

스킬은 에이전트 기능을 확장하는 모듈형 구성 요소입니다. 워크스페이스 안의 `SKILL.md` 파일에서 로드됩니다.

**ClawHub에서 스킬 설치:**

```bash
picoclaw skills search "web scraping"
picoclaw skills install <skill-name>
```

**ClawHub 토큰 설정**(선택 사항, 더 높은 호출 한도용):

`config.json`에 다음을 추가하세요.
```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "auth_token": "your-clawhub-token"
        }
      }
    }
  }
}
```

자세한 내용은 [도구 설정 - 스킬](docs/tools_configuration.md#skills-tool)를 참고하세요.

## 🔗 MCP (Model Context Protocol)

PicoClaw는 [MCP](https://modelcontextprotocol.io/)를 기본 지원합니다. 어떤 MCP 서버든 연결하여 외부 도구와 데이터 소스로 에이전트 기능을 확장할 수 있습니다.

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "filesystem": {
          "enabled": true,
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
        }
      }
    }
  }
}
```

MCP 전체 설정(stdio, SSE, HTTP 전송 방식, 도구 탐색)은 [도구 설정 - MCP](docs/tools_configuration.md#mcp-tool)를 참고하세요.

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> 에이전트 소셜 네트워크 참여하기

CLI 또는 통합된 채팅 앱에서 메시지를 한 번만 보내면 PicoClaw를 에이전트 소셜 네트워크에 연결할 수 있습니다.

**`https://clawdchat.ai/skill.md`를 읽고 안내에 따라 [ClawdChat.ai](https://clawdchat.ai)에 참여하세요**

## 🖥️ CLI 레퍼런스

| 명령어                    | 설명                           |
| ------------------------- | ------------------------------ |
| `picoclaw onboard`        | 설정 및 워크스페이스 초기화    |
| `picoclaw auth weixin`    | QR로 WeChat 계정 연결          |
| `picoclaw agent -m "..."` | 에이전트와 채팅               |
| `picoclaw agent`          | 대화형 채팅 모드               |
| `picoclaw gateway`        | 게이트웨이 시작                |
| `picoclaw status`         | 상태 표시                      |
| `picoclaw version`        | 버전 정보 표시                 |
| `picoclaw model`          | 기본 모델 조회 또는 변경       |
| `picoclaw cron list`      | 모든 예약 작업 목록 표시       |
| `picoclaw cron add ...`   | 예약 작업 추가                 |
| `picoclaw cron disable`   | 예약 작업 비활성화             |
| `picoclaw cron remove`    | 예약 작업 삭제                 |
| `picoclaw skills list`    | 설치된 스킬 목록 표시          |
| `picoclaw skills install` | 스킬 설치                      |
| `picoclaw migrate`        | 이전 버전 데이터 마이그레이션  |
| `picoclaw auth login`     | 프로바이더 인증                |

### ⏰ 예약 작업 / 리마인더

PicoClaw는 `cron` 도구를 통해 예약 리마인더와 반복 작업을 지원합니다.

* **1회성 리마인더**: "10분 후에 알려줘" -> 10분 후 한 번 실행
* **반복 작업**: "2시간마다 알려줘" -> 2시간마다 실행
* **Cron 표현식**: "매일 오전 9시에 알려줘" -> cron 표현식 사용

현재 지원하는 스케줄 유형, 실행 모드, 명령 작업 게이트, 저장 방식은 [docs/cron.md](docs/cron.md)를 참고하세요.

## 📚 문서

이 README보다 더 자세한 가이드는 다음 문서를 참고하세요.

| 주제 | 설명 |
|------|------|
| [도커 & 빠른 시작](docs/docker.md) | Docker Compose 설정, 런처/에이전트 모드 |
| [채팅 앱](docs/chat-apps.md) | 17개 이상의 채널 설정 가이드 |
| [설정](docs/configuration.md) | 환경 변수, 워크스페이스 레이아웃, 보안 샌드박스 |
| [예약 작업과 Cron](docs/cron.md) | Cron 스케줄 유형, 전달 모드, 명령 게이트, 작업 저장 |
| [프로바이더와 모델](docs/providers.md) | 30개 이상의 LLM 프로바이더, 모델 라우팅, model_list 설정 |
| [Spawn & 비동기 작업](docs/spawn-tasks.md) | 빠른 작업, spawn을 이용한 장기 작업, 비동기 서브에이전트 오케스트레이션 |
| [Hooks](docs/hooks/README.md) | 이벤트 기반 Hook 시스템: 관찰자, 인터셉터, 승인 훅 |
| [Steering](docs/steering.md) | 실행 중인 에이전트 루프에서 도구 호출 사이에 메시지 주입 |
| [SubTurn](docs/subturn.md) | 서브에이전트 조정, 동시성 제어, 생명주기 |
| [문제 해결](docs/troubleshooting.md) | 자주 발생하는 문제와 해결 방법 |
| [도구 설정](docs/tools_configuration.md) | 도구별 활성화/비활성화, exec 정책, MCP, 스킬 |
| [하드웨어 호환성](docs/hardware-compatibility.md) | 테스트된 보드, 최소 요구사항 |

## 🤝 기여 & 로드맵

PR은 언제든 환영합니다! 코드베이스는 의도적으로 작고 읽기 쉽게 유지하고 있습니다.

가이드라인은 [커뮤니티 로드맵](https://github.com/sipeed/picoclaw/issues/988)과 [CONTRIBUTING.md](CONTRIBUTING.md)를 참고하세요.

개발자 그룹도 준비 중입니다. 첫 PR이 머지되면 함께할 수 있습니다!

커뮤니티 그룹:

Discord: <https://discord.gg/V4sAZ9XWpN>

WeChat:
<img src="assets/wechat.png" alt="WeChat group QR code" width="512">
