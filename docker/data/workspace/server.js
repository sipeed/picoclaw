const express = require('express');
const { spawn } = require('child_process');
const { readdirSync, readFileSync } = require('fs');
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

const TEMPLATES_DIR = path.join(__dirname, 'templates');
const AVAILABLE_AREAS = ['auth', 'flow-designer', 'flow-tester', 'knowledge-base', 'logs', 'organization', 'profile', 'settings'];

/**
 * GET /agent/areas
 * Returns the list of available template areas.
 */
app.get('/agent/areas', (_req, res) => {
  res.json({ areas: AVAILABLE_AREAS });
});

/**
 * POST /agent/reference
 * Body: { "area": "auth" }
 * Runs picoclaw to generate the reference document for the given area.
 * Streams output back as plain text.
 */
app.post('/agent/reference', (req, res) => {
  const { area } = req.body;

  if (!area) {
    return res.status(400).json({ error: '"area" is required' });
  }

  if (!AVAILABLE_AREAS.includes(area)) {
    return res.status(400).json({ error: `Unknown area. Available: ${AVAILABLE_AREAS.join(', ')}` });
  }

  const templatePath = path.join(TEMPLATES_DIR, 'reference', `${area}.txt`);
  const prompt = readFileSync(templatePath, 'utf-8');

  res.setHeader('Content-Type', 'text/plain; charset=utf-8');
  res.setHeader('Transfer-Encoding', 'chunked');
  res.setHeader('X-Content-Type-Options', 'nosniff');

  const proc = spawn(
    'docker',
    ['compose', '--env-file', '.env', '--profile', 'gateway', 'run', '--rm', 'picoclaw-agent', '-m', prompt],
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

/**
 * POST /agent/run
 * Body: { "area": "flow-designer", "testFile": "create-new-flow-custom-node", "steps": "...", "expectedResult": "..." }
 * Composes the full prompt from the area template and runs picoclaw via docker compose.
 * Streams output back as plain text.
 */
app.post('/agent/run', (req, res) => {
  const { area, testFile, steps, expectedResult } = req.body;

  if (!area || !testFile || !steps || !expectedResult) {
    return res.status(400).json({ error: '"area", "testFile", "steps", and "expectedResult" are required' });
  }

  if (!AVAILABLE_AREAS.includes(area)) {
    return res.status(400).json({ error: `Unknown area. Available: ${AVAILABLE_AREAS.join(', ')}` });
  }

  const templatePath = path.join(TEMPLATES_DIR, `${area}.txt`);
  const template = readFileSync(templatePath, 'utf-8');

  const prompt = template
    .replace(/\{\{TEST_FILE\}\}/g, testFile)
    .replace(/\{\{STEPS\}\}/g, steps)
    .replace(/\{\{EXPECTED_RESULT\}\}/g, expectedResult);

  res.setHeader('Content-Type', 'text/plain; charset=utf-8');
  res.setHeader('Transfer-Encoding', 'chunked');
  res.setHeader('X-Content-Type-Options', 'nosniff');

  const proc = spawn(
    'docker',
    ['compose', '--env-file', '.env', '--profile', 'gateway', 'run', '--rm', 'picoclaw-agent', '-m', prompt],
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

/**
 * POST /test/autofix
 * Body: { "file": "tests/flow-designer/create-new-flow-custom-node.spec.ts" }
 * Runs picoclaw to auto-fix the spec file until it passes.
 * Streams output back as plain text.
 */
app.post('/test/autofix', (req, res) => {
  const { file } = req.body;

  if (!file) {
    return res.status(400).json({ error: '"file" is required' });
  }

  const available = getTestFiles();
  if (!available.includes(file)) {
    return res.status(400).json({ error: 'Unknown test file', available });
  }

  // Derive area from path: "tests/flow-designer/foo.spec.ts" → "flow-designer"
  const area = file.split('/')[1];
  if (!AVAILABLE_AREAS.includes(area)) {
    return res.status(400).json({ error: `No autofix template for area: ${area}` });
  }

  const templatePath = path.join(TEMPLATES_DIR, 'autofix', `${area}.txt`);
  const prompt = readFileSync(templatePath, 'utf-8').replace(/\{\{SPEC_FILE\}\}/g, file);

  res.setHeader('Content-Type', 'text/plain; charset=utf-8');
  res.setHeader('Transfer-Encoding', 'chunked');
  res.setHeader('X-Content-Type-Options', 'nosniff');

  const proc = spawn(
    'docker',
    ['compose', '--env-file', '.env', '--profile', 'gateway', 'run', '--rm', 'picoclaw-agent', '-m', prompt],
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
