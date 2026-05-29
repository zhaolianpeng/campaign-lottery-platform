#!/usr/bin/env node

const fs = require('node:fs/promises');
const path = require('node:path');
const mysql = require('mysql2/promise');
const { Umzug } = require('umzug');

function loadEnvFile(filePath) {
  try {
    const content = require('node:fs').readFileSync(filePath, 'utf8');
    for (const line of content.split(/\r?\n/)) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith('#')) {
        continue;
      }
      const eq = trimmed.indexOf('=');
      if (eq <= 0) {
        continue;
      }
      const key = trimmed.slice(0, eq).trim();
      let value = trimmed.slice(eq + 1).trim();
      if (
        (value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'"))
      ) {
        value = value.slice(1, -1);
      }
      if (process.env[key] == null) {
        process.env[key] = value;
      }
    }
  } catch {
    // Ignore missing env files.
  }
}

function hydrateEnvFromDotenv(backendRoot) {
  loadEnvFile(path.join(backendRoot, '.env.local'));
  loadEnvFile(path.join(backendRoot, '.env'));
}

function hydrateEnvFromEcosystem(backendRoot, repoRoot) {
  if (process.env.MYSQL_ENABLED) {
    return;
  }

  const ecosystemPath = path.join(repoRoot, 'ecosystem.config.cjs');
  try {
    // eslint-disable-next-line global-require, import/no-dynamic-require
    const ecosystem = require(ecosystemPath);
    const apps = Array.isArray(ecosystem?.apps) ? ecosystem.apps : [];
    const backendApp = apps.find((app) => app?.name === 'campaign-lottery-api') || apps.find((app) => app?.cwd === backendRoot);
    const env = backendApp?.env;
    if (!env || typeof env !== 'object') {
      return;
    }

    for (const [key, value] of Object.entries(env)) {
      if (process.env[key] == null && value != null) {
        process.env[key] = String(value);
      }
    }
  } catch {
    // Ignore missing ecosystem config and fall back to process.env only.
  }
}

function readEnvBoolean(name, defaultValue = false) {
  const value = process.env[name];
  if (typeof value !== 'string') {
    return defaultValue;
  }

  const normalized = value.trim().toLowerCase();
  if (['true', '1', 'yes', 'on'].includes(normalized)) {
    return true;
  }
  if (['false', '0', 'no', 'off', ''].includes(normalized)) {
    return false;
  }

  throw new Error(`${name} must be a boolean-like string`);
}

function loadMysqlConfig() {
  const enabled = readEnvBoolean('MYSQL_ENABLED', false);
  if (!enabled) {
    throw new Error('MYSQL_ENABLED is false; refusing to run migrations.');
  }

  return {
    dsn: process.env.MYSQL_DSN || '',
    host: process.env.MYSQL_HOST || '127.0.0.1',
    port: Number(process.env.MYSQL_PORT || 3306),
    database: process.env.MYSQL_DATABASE || 'campaign_lottery_platform',
    user: process.env.MYSQL_USER || 'campaign_lottery_app',
    password: process.env.MYSQL_PASSWORD || '',
    charset: process.env.MYSQL_CHARSET || 'utf8mb4',
  };
}

function mysqlOptions(config) {
  if (!config.dsn) {
    return {
      host: config.host,
      port: config.port,
      user: config.user,
      password: config.password,
      charset: config.charset,
    };
  }

  if (config.dsn.startsWith('mysql://') || config.dsn.startsWith('mysql2://')) {
    const url = new URL(config.dsn.replace(/^mysql2:\/\//, 'mysql://'));
    return {
      host: url.hostname,
      port: Number(url.port || 3306),
      user: decodeURIComponent(url.username),
      password: decodeURIComponent(url.password),
      charset: url.searchParams.get('charset') || config.charset,
    };
  }

  const match = config.dsn.match(/^([^:]+):(.*)@tcp\(([^):]+)(?::(\d+))?\)\/([^?]+)(?:\?(.*))?$/);
  if (!match) {
    throw new Error('Unsupported MYSQL_DSN format');
  }

  const [, user, password, host, port, database, query = ''] = match;
  const params = new URLSearchParams(query);
  config.database = database;
  return {
    host,
    port: Number(port || 3306),
    user,
    password,
    charset: params.get('charset') || config.charset,
  };
}

function escapeIdentifier(value) {
  return `\`${String(value).replace(/\`/g, '``')}\``;
}

async function pathExists(targetPath) {
  try {
    await fs.access(targetPath);
    return true;
  } catch {
    return false;
  }
}

async function ensureDatabase(connection, config) {
  await connection.query(
    `CREATE DATABASE IF NOT EXISTS ${escapeIdentifier(config.database)} DEFAULT CHARACTER SET ${config.charset} DEFAULT COLLATE utf8mb4_unicode_ci`,
  );
  await connection.query(`USE ${escapeIdentifier(config.database)}`);
}

class MysqlMigrationStorage {
  constructor(connection) {
    this.connection = connection;
    this.ready = false;
  }

  async ensureTable() {
    if (this.ready) {
      return;
    }

    await this.connection.query(`
      CREATE TABLE IF NOT EXISTS _schema_migrations (
        name VARCHAR(255) NOT NULL,
        executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        PRIMARY KEY (name)
      ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    `);
    this.ready = true;
  }

  async executed() {
    await this.ensureTable();
    const [rows] = await this.connection.query('SELECT name FROM _schema_migrations ORDER BY name ASC');
    return rows.map((row) => row.name);
  }

  async logMigration({ name }) {
    await this.ensureTable();
    await this.connection.query('INSERT INTO _schema_migrations (name) VALUES (?)', [name]);
  }

  async unlogMigration({ name }) {
    await this.ensureTable();
    await this.connection.query('DELETE FROM _schema_migrations WHERE name = ?', [name]);
  }
}

async function listMigrationFiles(repoRoot) {
  const migrations = [];
  const schemaPath = path.join(repoRoot, 'sql', 'schema.mysql.sql');
  if (await pathExists(schemaPath)) {
    migrations.push({ name: '000_legacy_schema', filePath: schemaPath });
  }

  const migrationsDir = path.join(repoRoot, 'sql', 'migrations');
  if (!(await pathExists(migrationsDir))) {
    return migrations;
  }

  const entries = await fs.readdir(migrationsDir, { withFileTypes: true });
  for (const entry of entries.sort((left, right) => left.name.localeCompare(right.name))) {
    if (!entry.isFile() || !entry.name.endsWith('.sql')) {
      continue;
    }
    migrations.push({
      name: entry.name.replace(/\.sql$/, ''),
      filePath: path.join(migrationsDir, entry.name),
    });
  }

  return migrations;
}

async function createUmzug(connection, repoRoot) {
  const migrationFiles = await listMigrationFiles(repoRoot);
  return new Umzug({
    migrations: migrationFiles.map((migration) => ({
      name: migration.name,
      up: async () => {
        const sql = await fs.readFile(migration.filePath, 'utf8');
        if (!sql.trim()) {
          return;
        }
        await connection.query(sql);
      },
      down: async () => {
        throw new Error(`Down migration is not supported for ${migration.name}`);
      },
    })),
    context: connection,
    storage: new MysqlMigrationStorage(connection),
    logger: console,
  });
}

async function main() {
  const command = process.argv[2] || 'up';
  const backendRoot = path.resolve(__dirname, '..');
  const repoRoot = path.resolve(backendRoot, '..');
  hydrateEnvFromDotenv(backendRoot);
  hydrateEnvFromEcosystem(backendRoot, repoRoot);
  const config = loadMysqlConfig();
  const connection = await mysql.createConnection({
    ...mysqlOptions(config),
    multipleStatements: true,
  });

  try {
    await ensureDatabase(connection, config);
    const umzug = await createUmzug(connection, repoRoot);

    if (command === 'up') {
      const migrations = await umzug.up();
      if (migrations.length === 0) {
        console.log('No pending migrations.');
        return;
      }
      console.log(`Applied ${migrations.length} migration(s).`);
      return;
    }

    if (command === 'status') {
      const [executed, pending] = await Promise.all([umzug.executed(), umzug.pending()]);
      console.log('Executed migrations:');
      for (const migration of executed) {
        console.log(`- ${migration.name}`);
      }
      if (executed.length === 0) {
        console.log('- none');
      }
      console.log('Pending migrations:');
      for (const migration of pending) {
        console.log(`- ${migration.name}`);
      }
      if (pending.length === 0) {
        console.log('- none');
      }
      return;
    }

    throw new Error(`Unsupported command: ${command}`);
  } finally {
    await connection.end();
  }
}

main().catch((error) => {
  console.error(error instanceof Error ? error.stack || error.message : error);
  process.exitCode = 1;
});