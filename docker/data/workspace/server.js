const express = require('express');
const { spawn } = require('child_process');
const path = require('path');

require('dotenv').config({ path: path.join(__dirname, '..', '..', '.env') });

const app = express();
app.use(express.json({ limit: '1mb' }));

const DOCKER_DIR = path.join(__dirname, '..', '..');
const API_USERNAME = process.env.API_USERNAME;
const API_PASSWORD = process.env.API_PASSWORD;

// Auth middleware — accepts Basic <base64> or plain username:password
app.use((req, res, next) => {
  const auth = req.headers.authorization;
  if (!auth) {
    res.setHeader('WWW-Authenticate', 'Basic realm="picoclaw"');
    return res.status(401).json({ error: 'Unauthorized' });
  }

  let username, password;
  if (auth.startsWith('Basic ')) {
    const decoded = Buffer.from(auth.slice(6), 'base64').toString();
    const colon = decoded.indexOf(':');
    username = decoded.slice(0, colon);
    password = decoded.slice(colon + 1);
  } else {
    const colon = auth.indexOf(':');
    username = auth.slice(0, colon);
    password = auth.slice(colon + 1);
  }

  if (username !== API_USERNAME || password !== API_PASSWORD) {
    return res.status(401).json({ error: 'Invalid credentials' });
  }
  next();
});

/**
 * POST /picoclaw/run
 * Body: { "prompt": "your prompt text here" }
 * Prepends the docker compose picoclaw-agent command and streams output back.
 */
app.post('/picoclaw/run', (req, res) => {
  const { prompt } = req.body;

  if (!prompt || typeof prompt !== 'string' || !prompt.trim()) {
    return res.status(400).json({ error: '"prompt" is required' });
  }

  res.setHeader('Content-Type', 'text/plain; charset=utf-8');
  res.setHeader('Transfer-Encoding', 'chunked');
  res.setHeader('X-Content-Type-Options', 'nosniff');

  const proc = spawn(
    'docker',
    ['compose', '--env-file', '.env', '--profile', 'gateway', 'run', '--rm', 'picoclaw-agent', '-m', prompt.trim()],
    {
      cwd: DOCKER_DIR,
      env: { ...process.env },
    }
  );

  proc.stdout.on('data', (data) => res.write(data));
  proc.stderr.on('data', (data) => res.write(data));

  proc.on('close', (code) => {
    res.end(`\n[exit code: ${code}]`);
  });

  proc.on('error', (err) => {
    if (!res.headersSent) {
      res.status(500).json({ error: err.message });
    } else {
      res.end(`\n[error: ${err.message}]`);
    }
  });
});

const PORT = process.env.PORT || 3100;
app.listen(PORT, () => {
  console.log(`Picoclaw server listening on http://localhost:${PORT}`);
});
