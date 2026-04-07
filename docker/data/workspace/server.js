const express = require('express');
const { spawn } = require('child_process');
const { readdirSync } = require('fs');
const path = require('path');

const app = express();
app.use(express.json({ limit: '1mb' }));

const TESTS_DIR = path.join(__dirname, 'tests');
// docker/ directory is two levels up from workspace/
const DOCKER_DIR = path.join(__dirname, '..', '..');

function getTestFiles() {
  const results = [];
  function walk(dir) {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        walk(full);
      } else if (entry.name.endsWith('.spec.ts')) {
        results.push(path.relative(__dirname, full));
      }
    }
  }
  walk(TESTS_DIR);
  return results;
}

/**
 * GET /test/files
 * Returns the list of available test spec files.
 */
app.get('/test/files', (_req, res) => {
  res.json({ files: getTestFiles() });
});

/**
 * POST /test/run
 * Body: { "file": "tests/flow-designer/create-new-flow-knowledge-base-node.spec.ts", "reporter": "line" }
 * Streams playwright output back as plain text.
 */
app.post('/test/run', (req, res) => {
  const { file, reporter = 'line' } = req.body;

  if (!file) {
    return res.status(400).json({ error: '"file" is required' });
  }

  const available = getTestFiles();
  if (!available.includes(file)) {
    return res.status(400).json({ error: 'Unknown test file', available });
  }

  const args = ['playwright', 'test', file, `--reporter=${reporter}`];

  res.setHeader('Content-Type', 'text/plain; charset=utf-8');
  res.setHeader('Transfer-Encoding', 'chunked');
  res.setHeader('X-Content-Type-Options', 'nosniff');

  const proc = spawn('npx', args, {
    cwd: __dirname,
    env: { ...process.env },
  });

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

/**
 * POST /agent/run
 * Body: { "prompt": "your prompt text here" }
 * Runs picoclaw via docker compose with the given prompt, streams output back.
 */
app.post('/agent/run', (req, res) => {
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
  console.log(`Test server listening on http://localhost:${PORT}`);
});
