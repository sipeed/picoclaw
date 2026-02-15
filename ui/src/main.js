import './style.css'

const app = document.querySelector('#app')

app.innerHTML = `
  <div class="wrap">
    <header class="header">
      <div class="title">PicoClaw</div>
      <div class="status" id="status">disconnected</div>
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
const form = document.querySelector('#form')
const input = document.querySelector('#input')

const chatId = 'browser'
const pageToken = new URLSearchParams(location.search).get('token') || ''

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
  const tokenPart = pageToken ? `&token=${encodeURIComponent(pageToken)}` : ''
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
