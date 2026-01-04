import { NextResponse } from 'next/server';
import { getDB, getAll } from '@/lib/db/client';

interface CaptureRow {
  id: number;
  sensor_os: string;
  sensor_hostname: string;
  interface_name: string;
  local_ip: string;
  start_time: string;
  duration_seconds: number;
  packet_count: number;
  imported_at: string;
  filename: string;
  device_count: number;
}

export async function GET() {
  try {
    const db = await getDB();

    const captures = getAll<CaptureRow>(db, `
      SELECT
        c.*,
        COUNT(DISTINCT d.id) as device_count
      FROM captures c
      LEFT JOIN devices d ON c.id = d.capture_id
      GROUP BY c.id
      ORDER BY c.start_time DESC
    `);

    return NextResponse.json(captures);
  } catch (error) {
    console.error('Captures fetch error:', error);
    return NextResponse.json({ error: 'Failed to fetch captures' }, { status: 500 });
  }
}
