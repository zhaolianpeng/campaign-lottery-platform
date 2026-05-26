import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { spawn } from 'node:child_process';

const mode = process.argv[2];
const allowedModes = new Set(['dev', 'start']);

if (!allowedModes.has(mode)) {
  console.error('Usage: node scripts/next-runtime.mjs <dev|start>');
  process.exit(1);
}

function parseEnvFile(filePath) {
  if (!existsSync(filePath)) {
    return {};
  }

  return readFileSync(filePath, 'utf8')
    .split(/\r?\n/)
    .reduce((config, line) => {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith('#')) {
        return config;
      }

      const separatorIndex = trimmed.indexOf('=');
      if (separatorIndex < 0) {
        return config;
      }

      const key = trimmed.slice(0, separatorIndex).trim();
      const rawValue = trimmed.slice(separatorIndex + 1).trim();
      config[key] = rawValue.replace(/^['"]|['"]$/g, '');
      return config;
    }, {});
}

function loadRuntimeConfig() {
  const envExample = parseEnvFile(resolve('.env.example'));
  const env = parseEnvFile(resolve('.env'));
  const envLocal = parseEnvFile(resolve('.env.local'));

  return {
    ...envExample,
    ...env,
    ...envLocal,
    ...process.env,
  };
}

const runtimeConfig = loadRuntimeConfig();
const hostname = runtimeConfig.FRONT_PAGE_HOSTNAME || 'localhost';
const port = Number(runtimeConfig.FRONT_PAGE_PORT || 3000);

if (!Number.isInteger(port) || port <= 0 || port > 65535) {
  console.error(`Invalid FRONT_PAGE_PORT: ${runtimeConfig.FRONT_PAGE_PORT}`);
  process.exit(1);
}

const command = process.platform === 'win32' ? 'next.cmd' : 'next';
const args = [mode, '--hostname', hostname, '--port', String(port)];

console.log(`Starting front-page with ${hostname}:${port}`);

const child = spawn(command, args, {
  stdio: 'inherit',
  shell: true,
});

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 0);
});
