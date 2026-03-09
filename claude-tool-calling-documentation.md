# Claude Programmatic Tool Calling Documentation

## Overview

Claude Tool Use (juga dikenal sebagai function calling) adalah fitur yang memungkinkan Claude untuk berinteraksi dengan tools dan fungsi eksternal di sisi klien. Fitur ini memungkinkan Anda melengkapi Claude dengan tools kustom untuk melakukan berbagai tugas yang lebih luas.

## Table of Contents

1. [Basic Workflow](#basic-workflow)
2. [Tool Definition](#tool-definition)
3. [Tool Choice Options](#tool-choice-options)
4. [Chain of Thought](#chain-of-thought)
5. [Handling Tool Results](#handling-tool-results)
6. [Best Practices](#best-practices)
7. [Pricing](#pricing)
8. [Code Examples](#code-examples)
9. [Stop Reasons](#stop-reasons)
10. [Error Handling](#error-handling)
11. [Advanced Features](#advanced-features)
12. [OpenAI Compatibility](#openai-compatibility)
13. [Computer Use (Anthropic-Defined Tools)](#computer-use-anthropic-defined-tools)
14. [MCP (Model Context Protocol)](#mcp-model-context-protocol)
15. [Resources](#resources)
16. [Summary](#summary)

---

## Basic Workflow

Integrasi external tools dengan Claude melibatkan langkah-langkah berikut:

1. **Define Tools**: Tentukan tools yang tersedia di permintaan API Anda
2. **User Query**: Kirim pesan dari pengguna
3. **Claude Responds**: Claude mungkin meminta untuk menggunakan tool dengan parameter spesifik
4. **Execute Tool**: Jalankan fungsi yang diminta di kode Anda
5. **Return Results** (Opsional): Kirim hasil tool kembali ke Claude
6. **Final Response**: Claude menyintesis hasil menjadi respons bahasa alami

**Catatan**: Langkah 5 adalah opsional. Untuk beberapa workflow, permintaan tool use dari Claude (langkah 3) mungkin sudah cukup, tanpa perlu mengirim hasil kembali ke Claude.

---

## Tool Definition

Tools ditentukan dalam parameter `tools` tingkat atas dari permintaan API. Setiap definisi tool meliputi:

### Parameters

| Parameter | Description |
|-----------|-------------|
| `name` | Nama tool. Harus cocok dengan regex `^[a-zA-Z0-9_-]{1,64}$` |
| `description` | Deskripsi teks mendetail tentang apa yang dilakukan tool, kapan harus digunakan, dan bagaimana perilakunya |
| `input_schema` | Objek JSON Schema yang mendefinisikan parameter yang diharapkan untuk tool |

### Tool Definition Example

```json
{
  "name": "get_weather",
  "description": "Get the current weather for a location",
  "input_schema": {
    "type": "object",
    "properties": {
      "location": {
        "type": "string",
        "description": "City and state, e.g., San Francisco, CA"
      },
      "unit": {
        "type": "string",
        "enum": ["celsius", "fahrenheit"],
        "description": "Temperature unit"
      }
    },
    "required": ["location"]
  }
}
```

### Complete API Request Example

```json
{
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 1024,
  "tools": [
    {
      "name": "get_weather",
      "description": "Get the current weather for a location",
      "input_schema": {
        "type": "object",
        "properties": {
          "location": {
            "type": "string",
            "description": "City and state, e.g., San Francisco, CA"
          }
        },
        "required": ["location"]
      }
    }
  ],
  "messages": [
    {"role": "user", "content": "What's the weather in San Francisco?"}
  ]
}
```

---

## Tool Choice Options

Parameter `tool_choice` mengontrol bagaimana model menggunakan tools yang disediakan.

### Options

#### 1. Auto (Default)
```json
"tool_choice": {"type": "auto"}
```
- Memungkinkan Claude memutuskan apakah akan memanggil tools atau tidak
- Claude akan menggunakan tools jika diperlukan untuk menjawab pertanyaan

#### 2. Any
```json
"tool_choice": {"type": "any"}
```
- Memberitahu Claude bahwa ia harus menggunakan salah satu tools yang tersedia
- Tidak memaksa tool tertentu
- Prefill assistant message untuk memaksa penggunaan tool

#### 3. Tool (Force Specific Tool)
```json
"tool_choice": {"type": "tool", "name": "get_weather"}
```
- Memaksa Claude untuk selalu menggunakan tool tertentu
- Berguna ketika Anda ingin memastikan tool tertentu digunakan

#### 4. None
```json
"tool_choice": {"type": "none"}
```
- Mencegah Claude menggunakan tools sama sekali

### Tool Choice Behavior Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        TOOL CHOICE OPTIONS                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  AUTO                    ANY                    TOOL            │
│  ┌─────┐               ┌─────┐               ┌─────┐           │
│  │Claude│               │Claude│               │Claude│           │
│  │decides│              │must  │              │must │           │
│  │      │               │use a │              │use   │           │
│  │      │               │tool  │              │specific│         │
│  └──┬──┘               └──┬──┘               └──┬──┘           │
│     │                      │                      │              │
│     ▼                      ▼                      ▼              │
│  Tool/No Tool           Any Tool             Specific Tool       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Catatan Penting**: Ketika `tool_choice` adalah `any` atau `tool`, model tidak akan memancarkan chain-of-thought `text` sebelum `tool_use` blocks, bahkan jika diminta secara eksplisit.

---

## Chain of Thought

Ketika menggunakan tools, Claude sering menampilkan "chain of thought"-nya, yaitu penalaran langkah demi langkah yang digunakan untuk memecahkan masalah dan memutuskan tools mana yang akan digunakan.

### Chain of Thought Example

Untuk pertanyaan: *"What's the weather like in San Francisco right now, and what time is it there?"*

```json
{
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "<thinking>To answer this question, I will:\n1. Use the get_weather tool to get the current weather in San Francisco.\n2. Use the get_time tool to get the current time in the America/Los_Angeles timezone.</thinking>"
    },
    {
      "type": "tool_use",
      "id": "toolu_01A09q90qw90lq917835lq9",
      "name": "get_weather",
      "input": {"location": "San Francisco, CA"}
    }
  ]
}
```

### Model Behavior

- **Claude 3 Opus**: Menampilkan chain of thought secara default ketika `tool_choice` adalah `auto`
- **Claude 3 Sonnet**: Kurang umum secara default, tetapi dapat diprompting untuk menampilkan penalarannya
- **Claude 3 Haiku**: Dapat diprompting untuk menampilkan penalaran

### Enabling Chain of Thought

Untuk Sonnet dan Haiku, tambahkan instruksi seperti:
```
"Before answering, explain your reasoning step-by-step in <thinking> tags."
```

---

## Handling Tool Results

### Tool Use Response Structure

Ketika Claude memutuskan untuk menggunakan tool, ia akan mengembalikan respons dengan:
- `stop_reason`: `"tool_use"`
- Satu atau lebih `tool_use` content blocks yang berisi:
  - `id`: Pengenal unik untuk blok penggunaan tool ini
  - `name`: Nama tool yang digunakan
  - `input`: Objek yang berisi input yang diteruskan ke tool

### Processing Steps

1. **Extract Information**: Ambil `name`, `id`, dan `input` dari blok `tool_use`
2. **Execute Tool**: Jalankan tool yang sesuai di kodebase Anda
3. **Return Results** (Opsional): Lanjutkan percakapan dengan mengirim pesan baru

### Tool Result Example

```json
{
  "role": "user",
  "content": [
    {
      "type": "tool_result",
      "tool_use_id": "toolu_01A09q90qw90lq917835lq9",
      "content": "15 degrees"
    }
  ]
}
```

### Tool Result Parameters

| Parameter | Description |
|-----------|-------------|
| `tool_use_id` | ID dari permintaan tool use yang menjadi hasil |
| `content` | Hasil tool, sebagai string atau array dari content blocks |
| `is_error` (Opsional) | Set ke `true` jika eksekusi tool menghasilkan error |

### Complex Tool Result (with images)

```json
{
  "role": "user",
  "content": [
    {
      "type": "tool_result",
      "tool_use_id": "toolu_01A09q90qw90lq917835lq9",
      "content": [
        {
          "type": "text",
          "text": "Weather is sunny"
        },
        {
          "type": "image",
          "source": {
            "type": "base64",
            "media_type": "image/png",
            "data": "iVBORw0KGgoAAAANSUhEUg..."
          }
        }
      ]
    }
  ]
}
```

---

## Best Practices

### 1. Provide Extremely Detailed Descriptions

Ini adalah faktor terpenting dalam performa tool. Deskripsi harus menjelaskan:

- Apa yang dilakukan tool
- Kapan harus digunakan (dan kapan tidak)
- Apa arti setiap parameter dan bagaimana mempengaruhi perilaku tool
- Setiap caveats atau limitations penting

**Aim for at least 3-4 sentences per tool description, more if the tool is complex.**

### Good vs Bad Descriptions

**❌ Poor Description:**
```
"Gets stock price"
```

**✅ Good Description:**
```
"Get the current stock price for a given ticker symbol. Returns the current
price in USD. Use this when the user asks about stock prices or wants to
know the value of a specific stock. The ticker parameter should be a valid
stock symbol (e.g., AAPL for Apple Inc.)."
```

### 2. Prioritize Descriptions Over Examples

Deskripsi yang jelas dan komprehensif lebih penting daripada contoh penggunaan. Tambahkan contoh hanya setelah deskripsi sepenuhnya terbentuk.

### 3. Use Proper JSON Schema

Definisikan input_schema dengan benar untuk validasi:
- Tentukan tipe data yang tepat
- Gunakan `enum` untuk nilai yang terbatas
- Tandai parameter yang diperlukan dengan `required`

### 4. Model Selection

| Model | Use Case |
|-------|----------|
| **Claude 3 Opus** | Complex tools dan ambiguous queries; menangani multiple tools dengan lebih baik |
| **Claude 3 Sonnet** | Balance between speed and capability |
| **Claude 3 Haiku** | Straightforward tools, lebih cepat |

### 5. Handle Errors Gracefully

```json
{
  "type": "tool_result",
  "tool_use_id": "toolu_01A09q90qw90lq917835lq9",
  "content": "Error: API rate limit exceeded",
  "is_error": true
}
```

---

## Pricing

Tool use requests dikenakan biaya sama seperti permintaan API Claude lainnya, berdasarkan total jumlah input tokens dan output tokens.

### Additional Token Sources

Tokens tambahan dari tool use berasal dari:
- Parameter `tools` dalam permintaan API (nama tool, deskripsi, dan skema)
- Blok konten `tool_use` dalam permintaan dan respons API
- Blok konten `tool_result` dalam permintaan API

### Tool Use System Prompt Token Count

| Model | Tool Choice: auto | Tool Choice: any/tool |
|-------|-------------------|----------------------|
| Claude 3.5 Sonnet | 294 tokens | 261 tokens |
| Claude 3 Opus | 530 tokens | 281 tokens |
| Claude 3 Sonnet | 159 tokens | 235 tokens |
| Claude 3 Haiku | 264 tokens | 340 tokens |

Token count ini ditambahkan ke input dan output tokens normal untuk menghitung total biaya permintaan.

---

## Code Examples

### Example 1: Simple Weather Tool

```python
import anthropic

client = anthropic.Anthropic(api_key="your-api-key")

response = client.messages.create(
    model="claude-sonnet-4-20250514",
    max_tokens=1024,
    tools=[{
        "name": "get_weather",
        "description": "Get the current weather for a location",
        "input_schema": {
            "type": "object",
            "properties": {
                "location": {
                    "type": "string",
                    "description": "City and state, e.g. San Francisco, CA"
                },
                "unit": {
                    "type": "string",
                    "enum": ["celsius", "fahrenheit"],
                    "description": "Temperature unit"
                }
            },
            "required": ["location"]
        }
    }],
    messages=[{
        "role": "user",
        "content": "What's the weather in San Francisco?"
    }]
)

# Check if Claude wants to use a tool
if response.stop_reason == "tool_use":
    for block in response.content:
        if block.type == "tool_use":
            # Execute the tool
            tool_result = execute_get_weather(block.input)

            # Send result back to Claude
            final_response = client.messages.create(
                model="claude-sonnet-4-20250514",
                max_tokens=1024,
                messages=[
                    {"role": "user", "content": "What's the weather in San Francisco?"},
                    {"role": "assistant", "content": response.content},
                    {
                        "role": "user",
                        "content": [{
                            "type": "tool_result",
                            "tool_use_id": block.id,
                            "content": str(tool_result)
                        }]
                    }
                ]
            )
            print(final_response.content[0].text)
```

### Example 2: Forcing Tool Use

```python
response = client.messages.create(
    model="claude-sonnet-4-20250514",
    max_tokens=1024,
    tools=[weather_tool],
    tool_choice={
        "type": "tool",
        "name": "get_weather"
    },
    messages=[{
        "role": "user",
        "content": "Tell me about the weather"
    }]
)
# Claude will be forced to use get_weather tool
```

### Example 3: JSON Output Tool

Tools tidak harus berupa fungsi di sisi klien - Anda dapat menggunakan tools kapan saja Anda ingin model mengembalikan output JSON yang mengikuti skema yang disediakan.

```python
tools=[{
    "name": "record_summary",
    "description": "Record a summary of information",
    "input_schema": {
        "type": "object",
        "properties": {
            "title": {"type": "string"},
            "summary": {"type": "string"},
            "key_points": {
                "type": "array",
                "items": {"type": "string"}
            }
        },
        "required": ["title", "summary", "key_points"]
    }
}]
```

---

## Stop Reasons

| Stop Reason | Description |
|-------------|-------------|
| `end_turn` | Model mencapai stopping point yang natural |
| `max_tokens` | Melebihi `max_tokens` yang diminta |
| `stop_sequence` | Salah satu `stop_sequences` kustom Anda dihasilkan |
| `tool_use` | Model memanggil satu atau lebih tools |
| `pause_turn` | Turn yang lama dijeda |
| `refusal` | Classifier streaming mengintervensi untuk penyalahgunaan kebijakan |

---

## Error Handling

Berbagai jenis error yang dapat terjadi:

### Common Errors

1. **Invalid Tool Definition**: Skema input tidak valid
2. **Tool Execution Failure**: Tool gagal dijalankan di sisi klien
3. **Timeout**: Tool memakan waktu terlalu lama
4. **Rate Limiting**: Terlalu banyak permintaan tool

### Handling Errors

```python
try:
    result = execute_tool(tool_name, tool_input)
except Exception as e:
    # Send error back to Claude
    error_response = client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=1024,
        messages=[...],
        tools=[...],
        messages=[{
            "role": "user",
            "content": [{
                "type": "tool_result",
                "tool_use_id": tool_use_id,
                "content": f"Error: {str(e)}",
                "is_error": True
            }]
        }]
    )
```

---

## Advanced Features

### Cache Control

Anda dapat mengatur cache control pada tools untuk mengurangi biaya token:

```python
tools=[{
    "name": "get_weather",
    "description": "Get weather information",
    "input_schema": {...},
    "cache_control": {
        "type": "ephemeral",
        "ttl": "5m"  # or "1h"
    }
}]
```

### Parallel Tool Use

Claude dapat memanggil multiple tools secara paralel dalam satu respons:

```python
# Response may contain multiple tool_use blocks
for block in response.content:
    if block.type == "tool_use":
        # Execute tool in parallel
        execute_tool_async(block.name, block.input)
```

Untuk menonaktifkan parallel tool use:
```python
tool_choice={
    "type": "auto",
    "disable_parallel_tool_use": True
}
```

---

## OpenAI Compatibility

Claude dan OpenAI memiliki implementasi function calling yang **tidak sepenuhnya compatible** secara format. Berikut perbedaan utamanya:

### Tool Definition Differences

#### Claude Format
```json
{
  "tools": [
    {
      "name": "get_weather",
      "description": "Get the current weather for a location",
      "input_schema": {
        "type": "object",
        "properties": {
          "location": {"type": "string"}
        },
        "required": ["location"]
      }
    }
  ]
}
```

#### OpenAI Format
```json
{
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get the current weather for a location",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {"type": "string"}
          },
          "required": ["location"]
        }
      }
    }
  ]
}
```

**Perbedaan:**
- OpenAI membungkus definisi dalam `{"type": "function", "function": {...}}`
- OpenAI menggunakan `parameters` bukan `input_schema`

### Response Structure Differences

#### Claude Response
```json
{
  "content": [
    {
      "type": "tool_use",
      "id": "toolu_01A09q90qw90lq917835lq9",
      "name": "get_weather",
      "input": {
        "location": "San Francisco, CA"
      }
    }
  ]
}
```

#### OpenAI Response
```json
{
  "tool_calls": [
    {
      "id": "call_abc123",
      "function": {
        "name": "get_weather",
        "arguments": "{\"location\": \"San Francisco, CA\"}"
      }
    }
  ]
}
```

**Perbedaan:**
- Claude: `input` adalah object
- OpenAI: `arguments` adalah JSON string

### Tool Result Differences

#### Claude Format
```json
{
  "role": "user",
  "content": [
    {
      "type": "tool_result",
      "tool_use_id": "toolu_01A09q90qw90lq917835lq9",
      "content": "15 degrees"
    }
  ]
}
```

#### OpenAI Format
```json
{
  "role": "tool",
  "tool_call_id": "call_abc123",
  "content": "15 degrees"
}
```

**Perbedaan:**
- Claude: `role` adalah `"user"`, menggunakan content block
- OpenAI: `role` adalah `"tool"`, flat structure

### Migration Guide

Jika migrasi dari OpenAI ke Claude:

1. **Convert tool definitions**: Hapus wrapper `{"type": "function", "function": ...}` dan ganti `parameters` dengan `input_schema`
2. **Parse tool responses**: Ganti parsing `tool_calls` dengan parsing content blocks `type: "tool_use"`
3. **Format tool results**: Gunakan `role: "user"` dengan content block `type: "tool_result"`

---

## Computer Use (Anthropic-Defined Tools)

Computer Use adalah fitur beta yang memungkinkan Claude untuk berinteraksi langsung dengan komputer desktop. Berbeda dengan tools kustom yang Anda definisikan sendiri, Computer Use menggunakan **3 special tools** yang telah didefinisikan oleh Anthropic.

### Perbedaan Utama dengan Custom Tools

| Aspect | Custom Tools | Anthropic-Defined Tools (Computer Use) |
|--------|-------------|---------------------------------------|
| Definisi | Anda mendefinisikan sendiri | Sudah didefinisikan oleh Anthropic |
| Struktur | Menggunakan `description` dan `input_schema` | Menggunakan field `type` khusus |
| Jumlah | Bisa unlimited | Hanya 3 tools tersedia |
| Fungsi | Sesuai kebutuhan Anda | Khusus untuk komputer interaksi |

### Tiga Anthropic-Defined Tools

#### 1. Computer Tool

**Tool Name:** `computer`
**API Name:** `computer_20241022`

Deskripsi: Memungkinkan Claude untuk melihat dan berinteraksi dengan desktop melalui screenshot.

```json
{
  "type": "computer_20241022",
  "name": "computer",
  "display_width_px": 1024,
  "display_height_px": 768
}
```

**Token Cost:** 683 additional input tokens

**Kemampuan:**
- Mengambil screenshot desktop
- Menggerakkan mouse
- Mengklik (klik kiri, klik kanan, double click)
- Mengetik keyboard
- Menunggu (untuk animasi/loading)

#### 2. Text Editor Tool

**Tool Name:** `str_replace_editor`
**API Name:** `text_editor_20241022`

Deskripsi: Memungkinkan Claude untuk mengedit file teks dengan operasi str_replace.

```json
{
  "type": "text_editor_20241022",
  "name": "str_replace_editor"
}
```

**Token Cost:** 700 additional input tokens

**Kemampuan:**
- Membaca file
- Menulis file baru
- Mengedit file dengan str_replace (presisi)
- View file dalam editor

#### 3. Bash Tool

**Tool Name:** `bash`
**API Name:** `bash_20241022`

Deskripsi: Memungkinkan Claude untuk menjalankan perintah shell.

```json
{
  "type": "bash_20241022",
  "name": "bash"
}
```

**Token Cost:** 245 additional input tokens

**Kemampuan:**
- Menjalankan perintah bash/shell
- Mengambil output command
- Menjalankan script
- Interaksi dengan system

### Contoh Penggunaan Computer Use

```python
import anthropic

client = anthropic.Anthropic(api_key="your-api-key")

response = client.messages.create(
    model="claude-sonnet-4-20250514",
    max_tokens=1024,
    tools=[
        {
            "type": "computer_20241022",
            "name": "computer",
            "display_width_px": 1024,
            "display_height_px": 768
        },
        {
            "type": "text_editor_20241022",
            "name": "str_replace_editor"
        },
        {
            "type": "bash_20241022",
            "name": "bash"
        }
    ],
    messages=[{
        "role": "user",
        "content": "Please open a text editor and write a Python script that says 'Hello World'"
    }]
)
```

### Total Token Cost untuk Computer Use

Jika menggunakan ketiga tools sekaligus:
- Computer Use system prompt: ~3,600 tokens (base)
- Computer tool: +683 tokens
- Text editor tool: +700 tokens
- Bash tool: +245 tokens
- **Total: ~5,228 additional input tokens** (per request)

### Limitasi Computer Use

1. **Beta Feature**: Masih dalam pengembangan, mungkin ada perubahan
2. **High Token Cost**: Menggunakan ~5,000+ tokens tambahan per request
3. **Platform Dependent**: Memerlukan integrasi dengan sistem operasi
4. **Model Availability**: Hanya tersedia untuk model tertentu (Claude 3.5 Sonnet+)

---

## MCP (Model Context Protocol)

MCP (Model Context Protocol) adalah standar open-source yang menghubungkan AI assistant ke sistem dan data source eksternal. Dapat dianggap sebagai "**USB-C untuk aplikasi AI**".

### Apa itu MCP?

MCP adalah protokol yang memungkinkan:
- Menghubungkan Claude ke data source eksternal (database, API, file systems)
- Menambahkan tools kustom tanpa mengubah implementasi Claude
- Membangun integrasi yang dapat digunakan kembali

### Arsitektur MCP

```
┌─────────────────────────────────────────────────────────────────┐
│                        MCP ARCHITECTURE                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────┐         ┌────────────┐         ┌──────────────┐   │
│  │ Claude  │◄────────┤   MCP      │◄────────┤ MCP Hosts    │   │
│  │ Client  │         │  Protocol  │         │              │   │
│  └─────────┘         └────────────┘         └──────────────┘   │
│                                   ▲              │              │
│                                   │              ▼              │
│                            ┌────────────┐   ┌──────────────┐  │
│                            │ MCP Tools  │   │ Data Sources │  │
│                            │            │   │              │  │
│                            │ - Search   │   │ - Databases  │  │
│                            │ - Read     │   │ - APIs       │  │
│                            │ - Write    │   │ - File Sys   │  │
│                            └────────────┘   └──────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### MCP vs Tool Use Biasa

| Aspect | Programmatic Tool Use | MCP |
|--------|---------------------|-----|
| Integrasi | Direct di API call | Melalui MCP host/server |
Use Case | Sederhana, langsung | Kompleks, reusable |
Setup | Minimal | Perlu MCP host |
Tool Discovery | Manual | Automatic |
Scalability | Terbatas | Tinggi |

### Komponen MCP

#### 1. MCP Host
Program yang menjalankan Claude dan mengimplementasikan MCP protocol.

#### 2. MCP Client
Klien yang terhubung ke MCP host untuk menyediakan tools.

#### 3. MCP Tools
Tools yang diekspos oleh MCP client ke Claude.

### Contoh MCP Tools

- **Filesystem MCP**: Akses file system lokal
- **Database MCP**: Query database langsung
- **API MCP**: Panggil API eksternal
- **Git MCP**: Interaksi dengan repository Git

### Menggunakan MCP dengan Claude

```python
# MCP biasanya diimplementasikan di level host/infrastruktur
# Bukan langsung di API call seperti tool use biasa

# Contoh konsep (implementasi sebenarnya tergantung MCP host):
tools = mcp_host.get_available_tools()
# tools akan otomatis includes MCP tools yang terdaftar

response = client.messages.create(
    model="claude-sonnet-4-20250514",
    max_tokens=1024,
    tools=tools,  # MCP tools akan di-include di sini
    messages=[...]
)
```

### MCP Resources

- [MCP Introduction](https://modelcontextprotocol.io/introduction)
- [MCP GitHub](https://github.com/modelcontextprotocol)

---

## Resources

- [Official Anthropic Documentation](https://docs.anthropic.com/en/docs/tool-use)
- [Messages API Reference](https://docs.anthropic.com/en/api/messages)
- [Anthropic Cookbooks](https://github.com/anthropics/anthropic-cookbook)
- [OpenAI to Claude Migration Guide](https://docs.anthropic.com/en/api/openai-sdk-compatibility)

---

## Summary

Claude Tool Use menyediakan cara yang powerful untuk memperluas kemampuan Claude dengan tools eksternal. Kunci sukses adalah:

1. **Deskripsi yang mendetail** untuk setiap tool
2. **Skema input yang jelas** dengan JSON Schema yang valid
3. **Penanganan error yang tepat**
4. **Pemilihan model yang sesuai** dengan kompleksitas task
5. **Memahami token cost** dari tool definitions
6. **Memahami perbedaan dengan OpenAI** jika melakukan migrasi
7. **Memahami Computer Use** untuk interaksi desktop (3 Anthropic-defined tools)
8. **Memahami MCP** untuk integrasi sistem eksternal yang scalable

### Jenis Tool dalam Claude

| Jenis Tool | Definisi | Token Cost | Use Case |
|-----------|---------|-----------|----------|
| **Custom Tools** | Anda definisikan sendiri | Variabel | Use case spesifik bisnis Anda |
| **Anthropic-Defined** | Didefinisikan Anthropic | ~5,200 total | Computer Use (desktop interaction) |
| **MCP Tools** | Melalui MCP protocol | Variabel | Integrasi sistem eksternal |

Dengan mengikuti best practices ini, Anda dapat membangun aplikasi yang memanfaatkan kekuatan Claude dikombinasikan dengan tools kustom Anda sendiri.
