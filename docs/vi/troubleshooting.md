# 🐛 Khắc Phục Sự Cố

> Quay lại [README](../../README.vi.md)

## "model ... not found in model_list" hoặc OpenRouter "free is not a valid model ID"

**Triệu chứng:** Bạn thấy một trong các lỗi sau:

- `Error creating provider: model "openrouter/free" not found in model_list`
- OpenRouter trả về 400: `"free is not a valid model ID"`

**Nguyên nhân:** Trường `model` trong mục `model_list` của bạn là giá trị được gửi đến API. Đối với OpenRouter, bạn phải sử dụng ID mô hình **đầy đủ**, không phải dạng viết tắt.

- **Sai:** `"model": "free"` → OpenRouter nhận được `free` và từ chối.
- **Đúng:** `"model": "openrouter/free"` → OpenRouter nhận được `openrouter/free` (định tuyến tự động tầng miễn phí).

**Cách sửa:** Trong `~/.picoclaw/config.json` (hoặc đường dẫn cấu hình của bạn):

1. **agents.defaults.model_name** phải khớp với một `model_name` trong `model_list` (ví dụ: `"openrouter-free"`).
2. **model** của mục đó phải là ID mô hình OpenRouter hợp lệ, ví dụ:
   - `"openrouter/free"` – tầng miễn phí tự động
   - `"google/gemini-2.0-flash-exp:free"`
   - `"meta-llama/llama-3.1-8b-instruct:free"`

Ví dụ:

```json
{
  "agents": {
    "defaults": {
      "model_name": "openrouter-free"
    }
  },
  "model_list": [
    {
      "model_name": "openrouter-free",
      "model": "openrouter/free",
      "api_key": "sk-or-v1-YOUR_OPENROUTER_KEY",
      "api_base": "https://openrouter.ai/api/v1"
    }
  ]
}
```

Lấy khóa của bạn tại [OpenRouter Keys](https://openrouter.ai/keys).
