// node -e "require('./server.js')" & sleep 2 && curl -s -X POST http://localhost:3100/test/run -H 'Content-Type: application/json' -d '{}' && kill %1 2>/dev/null; true

const express = require('express');
const { spawn } = require('child_process');
const { readdirSync } = require('fs');
const path = require('path');

const app = express();
app.use(express.json());

const TESTS_DIR = path.join(__dirname, 'tests');

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

// docker/ directory is two levels up from workspace/
const DOCKER_DIR = path.join(__dirname, '..', '..');

function getPromptFiles() {
  const results = [];
  function walk(dir) {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        walk(full);
      } else if (entry.name.endsWith('-prompt.txt')) {
        results.push(path.relative(__dirname, full));
      }
    }
  }
  walk(TESTS_DIR);
  return results;
}

/**
 * GET /picoclaw/prompts
 * Returns the list of available prompt .txt files.
 */
app.get('/picoclaw/prompts', (_req, res) => {
  res.json({ files: getPromptFiles() });
});

/**
 * POST /picoclaw/run
 * Body: { "file": "tests/knowledge-base/schedule-kb-full-sync-advanced-prompt.txt" }
 * Executes the prompt file as a bash script from the docker/ directory.
 * Streams output back as plain text.
 */
app.post('/picoclaw/run', (req, res) => {
  const { file } = req.body;

  if (!file) {
    return res.status(400).json({ error: '"file" is required' });
  }

  const available = getPromptFiles();
  if (!available.includes(file)) {
    return res.status(400).json({ error: 'Unknown prompt file', available });
  }

  const absFile = path.join(__dirname, file);

  res.setHeader('Content-Type', 'text/plain; charset=utf-8');
  res.setHeader('Transfer-Encoding', 'chunked');
  res.setHeader('X-Content-Type-Options', 'nosniff');

  const proc = spawn('bash', [absFile], {
    cwd: DOCKER_DIR,
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

const PORT = process.env.PORT || 3100;
app.listen(PORT, () => {
  console.log(`Test server listening on http://localhost:${PORT}`);
});
