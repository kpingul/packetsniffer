import initSqlJs, { Database } from 'sql.js';
import path from 'path';
import fs from 'fs';

const DB_PATH = path.join(process.cwd(), 'data', 'network.db');

let db: Database | null = null;
let SQL: Awaited<ReturnType<typeof initSqlJs>> | null = null;

export async function getDB(): Promise<Database> {
  if (db === null) {
    // Initialize SQL.js
    if (SQL === null) {
      SQL = await initSqlJs();
    }

    // Ensure data directory exists
    const dataDir = path.dirname(DB_PATH);
    if (!fs.existsSync(dataDir)) {
      fs.mkdirSync(dataDir, { recursive: true });
    }

    // Load existing database or create new one
    if (fs.existsSync(DB_PATH)) {
      const fileBuffer = fs.readFileSync(DB_PATH);
      db = new SQL.Database(fileBuffer);
    } else {
      db = new SQL.Database();
    }

    // Run migrations
    await runMigrations(db);

    // Save after migrations
    saveDB();
  }
  return db;
}

function runMigrations(db: Database) {
  const schemaPath = path.join(process.cwd(), 'src/lib/db/schema.sql');

  if (fs.existsSync(schemaPath)) {
    const schema = fs.readFileSync(schemaPath, 'utf-8');
    db.run(schema);
  }
}

export function saveDB() {
  if (db !== null) {
    const data = db.export();
    const buffer = Buffer.from(data);
    fs.writeFileSync(DB_PATH, buffer);
  }
}

export function closeDB() {
  if (db !== null) {
    saveDB();
    db.close();
    db = null;
  }
}

// Helper functions for common operations
export function runQuery(db: Database, sql: string, params: unknown[] = []): void {
  db.run(sql, params as (string | number | Uint8Array | null)[]);
}

export function getAll<T>(db: Database, sql: string, params: unknown[] = []): T[] {
  const stmt = db.prepare(sql);
  stmt.bind(params as (string | number | Uint8Array | null)[]);

  const results: T[] = [];
  while (stmt.step()) {
    results.push(stmt.getAsObject() as T);
  }
  stmt.free();
  return results;
}

export function getOne<T>(db: Database, sql: string, params: unknown[] = []): T | undefined {
  const results = getAll<T>(db, sql, params);
  return results[0];
}

export function insert(db: Database, sql: string, params: unknown[] = []): number {
  db.run(sql, params as (string | number | Uint8Array | null)[]);
  const result = db.exec('SELECT last_insert_rowid() as id');
  return result[0]?.values[0]?.[0] as number || 0;
}
