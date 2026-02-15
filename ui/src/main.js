import './style.css'

const app = document.querySelector('#app')

const THEME_STORAGE_KEY = 'picoclaw.theme'

function getStoredTheme() {
  const v = localStorage.getItem(THEME_STORAGE_KEY)
  if (v === 'light' || v === 'dark' || v === 'system') return v
  return 'system'
}

function applyTheme(mode) {
  const root = document.documentElement
  if (mode === 'system') {
    root.removeAttribute('data-theme')
  } else {
    root.setAttribute('data-theme', mode)
  }
}

let themeMode = getStoredTheme()
applyTheme(themeMode)

app.innerHTML = `
  <div class="wrap">
    <header class="header">
      <div class="title">PicoClaw</div>
      <div class="header-right">
        <label class="theme" for="theme">
          <span class="theme-label">Theme</span>
          <select class="theme-select" id="theme">
            <option value="system">System</option>
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </label>
        <div class="status" id="status">disconnected</div>
      </div>
    </header>

    <main class="chat" id="chat"></main>

    <form class="composer" id="form">
      <input class="input" id="input" placeholder="Type a message..." autocomplete="off" />
      <button class="btn" id="send" type="submit">Send</button>
    </form>
  </div>
`

const chat = document.querySelector('#chat')
const statusEl = document.querySelector('#status')
const themeSelect = document.querySelector('#theme')
const form = document.querySelector('#form')
const input = document.querySelector('#input')

const chatId = 'browser'
const TOKEN_STORAGE_KEY = 'picoclaw.gateway_token'
const urlParams = new URLSearchParams(location.search)
const tokenFromUrl = urlParams.get('token') || ''
const tokenFromStorage = localStorage.getItem(TOKEN_STORAGE_KEY) || ''
let gatewayToken = tokenFromUrl || tokenFromStorage
let tokenCameFromUrl = !!tokenFromUrl

function removeTokenFromUrl() {
  const p = new URLSearchParams(location.search)
  if (!p.has('token')) return
  p.delete('token')
  const qs = p.toString()
  const newUrl = `${location.pathname}${qs ? `?${qs}` : ''}${location.hash || ''}`
  history.replaceState(null, '', newUrl)
}

themeSelect.value = themeMode
themeSelect.addEventListener('change', () => {
  themeMode = themeSelect.value
  localStorage.setItem(THEME_STORAGE_KEY, themeMode)
  applyTheme(themeMode)
})

const media = window.matchMedia('(prefers-color-scheme: dark)')
media.addEventListener('change', () => {
  if (themeMode === 'system') applyTheme('system')
})

function addMessage(role, text) {
  const item = document.createElement('div')
  item.className = `msg ${role}`

  const bubble = document.createElement('div')
  bubble.className = 'bubble'
  bubble.textContent = text

  item.appendChild(bubble)
  chat.appendChild(item)
  chat.scrollTop = chat.scrollHeight
}

function wsUrl() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const tokenPart = gatewayToken ? `&token=${encodeURIComponent(gatewayToken)}` : ''
  return `${proto}//${location.host}/ws?chat_id=${encodeURIComponent(chatId)}${tokenPart}`
}

let ws
let reconnectTimer

function setStatus(s) {
  statusEl.textContent = s
  statusEl.dataset.state = s
}

function connect() {
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return

  setStatus('connecting')
  ws = new WebSocket(wsUrl())

  ws.addEventListener('open', () => {
    setStatus('connected')

    if (tokenCameFromUrl && gatewayToken) {
      localStorage.setItem(TOKEN_STORAGE_KEY, gatewayToken)
      removeTokenFromUrl()
      tokenCameFromUrl = false
    }
  })

  ws.addEventListener('close', () => {
    setStatus('disconnected')
    if (!reconnectTimer) {
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null
        connect()
      }, 1000)
    }
  })

  ws.addEventListener('message', (ev) => {
    try {
      const msg = JSON.parse(ev.data)
      if (msg && msg.type === 'message' && typeof msg.content === 'string') {
        addMessage('assistant', msg.content)
      }
    } catch {
    }
  })
}

form.addEventListener('submit', (e) => {
  e.preventDefault()
  const text = input.value.trim()
  if (!text) return
  input.value = ''

  addMessage('user', text)

  if (!ws || ws.readyState !== WebSocket.OPEN) {
    connect()
  }

  const payload = { chat_id: chatId, content: text }
  try {
    ws.send(JSON.stringify(payload))
  } catch {
  }
})

connect()
