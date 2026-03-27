#!/bin/bash
# ═══════════════════════════════════════════════════════════════════
# zalo-get-token.sh — Lấy Zalo OA Access Token tự động
#
# Script này thực hiện toàn bộ OAuth 2.0 PKCE flow:
#   1. Tạo PKCE pair (code_verifier + code_challenge)
#   2. In authorization URL để user mở browser
#   3. Chạy HTTP server tạm lắng nghe callback
#   4. Nhận authorization code từ Zalo redirect
#   5. Exchange code lấy access_token + refresh_token
#   6. Update docker/data/config.json
#   7. Restart gateway container
#
# Yêu cầu: openssl, curl, python3, jq (optional)
# Nginx phải proxy /auth/zalo/callback → localhost:$CALLBACK_PORT
#
# Sử dụng:
#   cd ~/picoclaw
#   bash scripts/zalo-get-token.sh
#
# Tuỳ chỉnh:
#   APP_ID=xxx APP_SECRET=xxx bash scripts/zalo-get-token.sh
# ═══════════════════════════════════════════════════════════════════

set -euo pipefail

# ── Cấu hình (có thể override bằng environment variable) ──────────
APP_ID="${APP_ID:-}"
APP_SECRET="${APP_SECRET:-}"
REDIRECT_URI="${REDIRECT_URI:-}"

if [ -z "$APP_ID" ] || [ -z "$APP_SECRET" ] || [ -z "$REDIRECT_URI" ]; then
    error "Cần đặt environment variables: APP_ID, APP_SECRET, REDIRECT_URI"
    echo ""
    echo "  Ví dụ:"
    echo "    APP_ID=xxx APP_SECRET=xxx REDIRECT_URI=https://your-domain/auth/zalo/callback bash $0"
    echo ""
    exit 1
fi
CONFIG_PATH="${CONFIG_PATH:-docker/data/config.json}"
CALLBACK_PORT="${CALLBACK_PORT:-9999}"
COMPOSE_FILE="${COMPOSE_FILE:-docker/docker-compose.yml}"

# ── Màu sắc ───────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

info()  { echo -e "${GREEN}[+]${NC} $*"; }
warn()  { echo -e "${YELLOW}[!]${NC} $*"; }
error() { echo -e "${RED}[-]${NC} $*"; }
step()  { echo -e "${CYAN}[*]${NC} $*"; }

# ── Kiểm tra dependencies ─────────────────────────────────────────
for cmd in openssl curl python3; do
    if ! command -v "$cmd" &>/dev/null; then
        error "Cần cài '$cmd' trước khi chạy script này"
        exit 1
    fi
done

# ── Bước 1: Tạo PKCE pair ─────────────────────────────────────────
step "Tạo PKCE code_verifier + code_challenge..."

CODE_VERIFIER=$(openssl rand -base64 32 | tr -d '=/+' | head -c 43)
CODE_CHALLENGE=$(printf '%s' "$CODE_VERIFIER" \
    | openssl dgst -sha256 -binary \
    | openssl base64 -A \
    | tr '+/' '-_' \
    | tr -d '=')

# Lưu verifier
echo "$CODE_VERIFIER" > /tmp/zalo_verifier.txt
info "Code verifier đã lưu: /tmp/zalo_verifier.txt"

# ── Bước 2: In authorization URL ──────────────────────────────────
ENCODED_REDIRECT=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$REDIRECT_URI'))")
AUTH_URL="https://oauth.zaloapp.com/v4/oa/permission?app_id=${APP_ID}&redirect_uri=${ENCODED_REDIRECT}&code_challenge=${CODE_CHALLENGE}&code_challenge_method=S256"

echo ""
echo "══════════════════════════════════════════════════════════"
echo "  Mở link này trong browser để authorize:"
echo ""
echo "  $AUTH_URL"
echo ""
echo "  Sau khi authorize, Zalo sẽ redirect về callback server."
echo "  Script sẽ tự động exchange code lấy token."
echo "══════════════════════════════════════════════════════════"
echo ""

# ── Bước 3+4: HTTP server lắng nghe callback ──────────────────────
step "Chạy callback server trên port $CALLBACK_PORT..."
step "Đang chờ Zalo redirect..."

# Tạo Python callback server inline
CALLBACK_SCRIPT=$(cat <<'PYEOF'
import http.server
import urllib.parse
import urllib.request
import json
import sys
import os

APP_ID = os.environ["APP_ID"]
APP_SECRET = os.environ["APP_SECRET"]
CODE_VERIFIER = os.environ["CODE_VERIFIER"]
PORT = int(os.environ["CALLBACK_PORT"])

class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        pass  # Suppress default logging

    def do_GET(self):
        parsed = urllib.parse.urlparse(self.path)
        if not parsed.path.startswith("/auth/zalo/callback"):
            self.send_response(404)
            self.end_headers()
            return

        params = urllib.parse.parse_qs(parsed.query)
        code = params.get("code", [None])[0]

        if not code:
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            err_msg = f"Không nhận được 'code'. Query: {parsed.query}"
            self.wfile.write(err_msg.encode())
            print(f"\033[0;31m[-]\033[0m {err_msg}", file=sys.stderr)
            return

        print(f"\033[0;32m[+]\033[0m Nhận được authorization code: {code[:20]}...")

        # Exchange code for token
        data = urllib.parse.urlencode({
            "app_id": APP_ID,
            "grant_type": "authorization_code",
            "code": code,
            "code_verifier": CODE_VERIFIER,
        }).encode()

        req = urllib.request.Request(
            "https://oauth.zaloapp.com/v4/oa/access_token",
            data=data,
            headers={
                "Content-Type": "application/x-www-form-urlencoded",
                "secret_key": APP_SECRET,
            },
        )

        try:
            resp = urllib.request.urlopen(req, timeout=15)
            result = json.loads(resp.read())
        except Exception as e:
            result = {"error": -1, "error_description": str(e)}

        # Trả response cho browser
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.end_headers()

        access_token = result.get("access_token", "")
        refresh_token = result.get("refresh_token", "")

        if access_token:
            html = "<h1>&#10004; Zalo OAuth th&agrave;nh c&ocirc;ng!</h1><p>Token &#273;&atilde; l&#432;u. B&#7841;n c&oacute; th&#7875; &#273;&oacute;ng tab n&agrave;y.</p>"
        else:
            html = f"<h1>&#10008; L&#7895;i</h1><pre>{json.dumps(result, indent=2)}</pre>"

        self.wfile.write(html.encode())

        # Ghi kết quả ra file để shell script đọc
        with open("/tmp/zalo_token_result.json", "w") as f:
            json.dump(result, f)

        # Shutdown
        import threading
        threading.Thread(target=self.server.shutdown).start()

try:
    httpd = http.server.HTTPServer(("127.0.0.1", PORT), Handler)
    httpd.serve_forever()
except KeyboardInterrupt:
    pass
PYEOF
)

# Export env cho Python script
export APP_ID APP_SECRET CODE_VERIFIER CALLBACK_PORT

# Chạy callback server (block cho đến khi nhận code)
python3 -c "$CALLBACK_SCRIPT"

# ── Bước 5+6: Đọc kết quả và update config ────────────────────────
if [ ! -f /tmp/zalo_token_result.json ]; then
    error "Không nhận được response từ Zalo"
    exit 1
fi

ACCESS_TOKEN=$(python3 -c "import json; r=json.load(open('/tmp/zalo_token_result.json')); print(r.get('access_token', ''))")
REFRESH_TOKEN=$(python3 -c "import json; r=json.load(open('/tmp/zalo_token_result.json')); print(r.get('refresh_token', ''))")
ERROR_CODE=$(python3 -c "import json; r=json.load(open('/tmp/zalo_token_result.json')); print(r.get('error', 0))")

if [ -z "$ACCESS_TOKEN" ] || [ "$ERROR_CODE" != "0" ]; then
    error "Exchange token thất bại!"
    echo "Response:"
    python3 -m json.tool /tmp/zalo_token_result.json 2>/dev/null || cat /tmp/zalo_token_result.json
    exit 1
fi

info "Access Token:  ${ACCESS_TOKEN:0:30}..."
info "Refresh Token: ${REFRESH_TOKEN:0:30}..."

# Update config.json
step "Cập nhật $CONFIG_PATH..."

if [ ! -f "$CONFIG_PATH" ]; then
    error "Không tìm thấy $CONFIG_PATH"
    error "Chạy script này từ thư mục gốc của project (cd ~/picoclaw)"
    exit 1
fi

python3 -c "
import json, sys
try:
    with open('$CONFIG_PATH') as f:
        cfg = json.load(f)
    cfg['channels']['zalo']['access_token'] = '$ACCESS_TOKEN'
    cfg['channels']['zalo']['refresh_token'] = '$REFRESH_TOKEN'
    with open('$CONFIG_PATH', 'w') as f:
        json.dump(cfg, f, indent=2, ensure_ascii=False)
        f.write('\n')
    print('\033[0;32m[+]\033[0m Config đã cập nhật')
except Exception as e:
    print(f'\033[0;31m[-]\033[0m Lỗi cập nhật config: {e}', file=sys.stderr)
    sys.exit(1)
"

# ── Bước 7: Restart gateway ───────────────────────────────────────
step "Restart gateway..."

if docker compose -f "$COMPOSE_FILE" --profile gateway restart 2>/dev/null; then
    info "Gateway đã restart"
else
    warn "Không restart được gateway. Chạy thủ công:"
    warn "  docker compose -f $COMPOSE_FILE --profile gateway restart"
fi

# Chờ gateway start
sleep 3

# Kiểm tra gateway hoạt động
if docker compose -f "$COMPOSE_FILE" --profile gateway logs --tail=5 2>/dev/null | grep -q "Channels enabled"; then
    info "Gateway đang chạy"
    CHANNELS=$(docker compose -f "$COMPOSE_FILE" --profile gateway logs --tail=5 2>/dev/null | grep "Channels enabled" | tail -1)
    info "$CHANNELS"
fi

# Cleanup
rm -f /tmp/zalo_token_result.json

echo ""
echo "══════════════════════════════════════════════════════════"
info "Hoàn tất! Zalo OA channel đã sẵn sàng."
echo ""
echo "  Test webhook:"
echo "    curl https://db2.moonlight.pro.vn/webhook/zalo?challenge=test"
echo ""
echo "  Xem log:"
echo "    docker compose -f $COMPOSE_FILE --profile gateway logs -f"
echo "══════════════════════════════════════════════════════════"
