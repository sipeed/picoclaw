import { copyFile, mkdir, rm } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const rootDir = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.join(rootDir, 'src');
const outDir = path.join(rootDir, '..', 'static', 'dist');

await rm(outDir, { recursive: true, force: true });
await mkdir(outDir, { recursive: true });

const result = await Bun.build({
  entrypoints: [path.join(srcDir, 'index.tsx')],
  outdir: outDir,
  target: 'browser',
  format: 'iife',
  sourcemap: 'none',
  packages: 'bundle',
  naming: {
    entry: 'app.[ext]',
  },
});

if (!result.success) {
  for (const log of result.logs) {
    console.error(log);
  }
  process.exit(1);
}

await copyFile(path.join(srcDir, 'map.js'), path.join(outDir, 'map.js'));
