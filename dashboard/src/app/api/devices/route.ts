import { NextRequest, NextResponse } from 'next/server';
import { getDB, getAll } from '@/lib/db/client';

interface DeviceRow {
  id: number;
  capture_id: number;
  mac: string;
  vendor: string | null;
  hostname: string | null;
  os_guess: string | null;
  os_confidence: number | null;
  signals_used: string | null;
  discovery_source: string | null;
  first_seen: string | null;
  last_seen: string | null;
  ips: string | null;
}

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const captureId = searchParams.get('captureId');
    const vendor = searchParams.get('vendor');
    const os = searchParams.get('os');

    const db = await getDB();

    let query = `
      SELECT
        d.id, d.capture_id, d.mac, d.vendor, d.hostname,
        d.os_guess, d.os_confidence, d.signals_used,
        d.discovery_source, d.first_seen, d.last_seen,
        GROUP_CONCAT(di.ip) as ips
      FROM devices d
      LEFT JOIN device_ips di ON d.id = di.device_id
    `;

    const conditions: string[] = [];
    const params: (string | number)[] = [];

    if (captureId) {
      conditions.push('d.capture_id = ?');
      params.push(parseInt(captureId));
    }

    if (vendor) {
      conditions.push('d.vendor LIKE ?');
      params.push(`%${vendor}%`);
    }

    if (os) {
      conditions.push('d.os_guess LIKE ?');
      params.push(`%${os}%`);
    }

    if (conditions.length > 0) {
      query += ' WHERE ' + conditions.join(' AND ');
    }

    query += ' GROUP BY d.id ORDER BY d.last_seen DESC';

    const devices = getAll<DeviceRow>(db, query, params);

    // Parse IPs from comma-separated string to array
    const result = devices.map((device) => ({
      ...device,
      ips: device.ips ? String(device.ips).split(',') : [],
      signals_used: device.signals_used
        ? JSON.parse(String(device.signals_used))
        : [],
    }));

    return NextResponse.json(result);
  } catch (error) {
    console.error('Devices fetch error:', error);
    return NextResponse.json({ error: 'Failed to fetch devices' }, { status: 500 });
  }
}
