import { NextRequest, NextResponse } from 'next/server';
import { getDB, getAll } from '@/lib/db/client';

interface ProtocolRow {
  protocol: string;
  count: number;
}

interface PortRow {
  port: number;
  protocol: string;
  count: number;
}

interface TalkerRow {
  ip: string;
  bytes_sent: number;
  bytes_received: number;
  packets_sent: number;
  packets_received: number;
}

interface DNSRow {
  domain: string;
  query_count: number;
}

interface DestRow {
  address: string;
  connection_count: number;
  bytes_total: number;
}

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const captureId = searchParams.get('captureId');

    const db = await getDB();

    // Get protocol counts
    let protocolQuery = 'SELECT protocol, SUM(count) as count FROM protocol_counts';
    let protocolParams: number[] = [];

    if (captureId) {
      protocolQuery += ' WHERE capture_id = ?';
      protocolParams.push(parseInt(captureId));
    }
    protocolQuery += ' GROUP BY protocol ORDER BY count DESC';

    const protocols = getAll<ProtocolRow>(db, protocolQuery, protocolParams);

    // Get top ports
    let portsQuery = 'SELECT port, protocol, SUM(count) as count FROM top_ports';
    let portsParams: number[] = [];

    if (captureId) {
      portsQuery += ' WHERE capture_id = ?';
      portsParams.push(parseInt(captureId));
    }
    portsQuery += ' GROUP BY port, protocol ORDER BY count DESC LIMIT 20';

    const ports = getAll<PortRow>(db, portsQuery, portsParams);

    // Get top talkers
    let talkersQuery = `
      SELECT ip,
        SUM(bytes_sent) as bytes_sent,
        SUM(bytes_received) as bytes_received,
        SUM(packets_sent) as packets_sent,
        SUM(packets_received) as packets_received
      FROM top_talkers
    `;
    let talkersParams: number[] = [];

    if (captureId) {
      talkersQuery += ' WHERE capture_id = ?';
      talkersParams.push(parseInt(captureId));
    }
    talkersQuery += ' GROUP BY ip ORDER BY (bytes_sent + bytes_received) DESC LIMIT 20';

    const talkers = getAll<TalkerRow>(db, talkersQuery, talkersParams);

    // Get DNS domains
    let dnsQuery = `
      SELECT domain, SUM(query_count) as query_count
      FROM dns_domains
    `;
    let dnsParams: number[] = [];

    if (captureId) {
      dnsQuery += ' WHERE capture_id = ?';
      dnsParams.push(parseInt(captureId));
    }
    dnsQuery += ' GROUP BY domain ORDER BY query_count DESC LIMIT 50';

    const dnsDomains = getAll<DNSRow>(db, dnsQuery, dnsParams);

    // Get destinations
    let destQuery = `
      SELECT address, SUM(connection_count) as connection_count, SUM(bytes_total) as bytes_total
      FROM destinations
    `;
    let destParams: number[] = [];

    if (captureId) {
      destQuery += ' WHERE capture_id = ?';
      destParams.push(parseInt(captureId));
    }
    destQuery += ' GROUP BY address ORDER BY bytes_total DESC LIMIT 20';

    const destinations = getAll<DestRow>(db, destQuery, destParams);

    return NextResponse.json({
      protocols,
      ports,
      talkers,
      dnsDomains,
      destinations,
    });
  } catch (error) {
    console.error('Traffic fetch error:', error);
    return NextResponse.json({ error: 'Failed to fetch traffic data' }, { status: 500 });
  }
}
