import { NextRequest, NextResponse } from 'next/server';
import { getDB, getAll } from '@/lib/db/client';
import type { GraphNode, GraphLink, GraphData } from '@/types';

interface DeviceRow {
  id: number;
  mac: string;
  vendor: string | null;
  hostname: string | null;
  os_guess: string | null;
  ips: string | null;
}

interface DNSRow {
  domain: string;
}

interface DestRow {
  address: string;
  bytes: number;
}

interface DNSLinkRow {
  domain: string;
  querying_ips: string;
}

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const captureId = searchParams.get('captureId');

    const db = await getDB();

    // Get devices with their IPs
    let deviceQuery = `
      SELECT
        d.id, d.mac, d.vendor, d.hostname, d.os_guess,
        GROUP_CONCAT(di.ip) as ips
      FROM devices d
      LEFT JOIN device_ips di ON d.id = di.device_id
    `;
    const deviceParams: number[] = [];

    if (captureId) {
      deviceQuery += ' WHERE d.capture_id = ?';
      deviceParams.push(parseInt(captureId));
    }
    deviceQuery += ' GROUP BY d.id';

    const devices = getAll<DeviceRow>(db, deviceQuery, deviceParams);

    // Get DNS domains
    let dnsQuery = 'SELECT DISTINCT domain FROM dns_domains';
    const dnsParams: number[] = [];

    if (captureId) {
      dnsQuery += ' WHERE capture_id = ?';
      dnsParams.push(parseInt(captureId));
    }
    dnsQuery += ' ORDER BY query_count DESC LIMIT 50';

    const dnsDomains = getAll<DNSRow>(db, dnsQuery, dnsParams);

    // Get destinations (external IPs)
    let destQuery = 'SELECT DISTINCT address, SUM(bytes_total) as bytes FROM destinations';
    const destParams: number[] = [];

    if (captureId) {
      destQuery += ' WHERE capture_id = ?';
      destParams.push(parseInt(captureId));
    }
    destQuery += ' GROUP BY address ORDER BY bytes DESC LIMIT 30';

    const destinations = getAll<DestRow>(db, destQuery, destParams);

    // Build nodes
    const nodes: GraphNode[] = [];
    const nodeIds = new Set<string>();

    // Add device nodes
    for (const device of devices) {
      const ips = device.ips ? device.ips.split(',') : [];
      const label = device.hostname || ips[0] || device.mac;

      nodes.push({
        id: `device:${device.mac}`,
        type: 'device',
        label,
        mac: device.mac,
        ip: ips[0],
        vendor: device.vendor || undefined,
        os: device.os_guess || undefined,
      });
      nodeIds.add(`device:${device.mac}`);
    }

    // Add domain nodes
    for (const domain of dnsDomains) {
      const id = `domain:${domain.domain}`;
      if (!nodeIds.has(id)) {
        nodes.push({
          id,
          type: 'domain',
          label: domain.domain,
        });
        nodeIds.add(id);
      }
    }

    // Add external IP nodes
    for (const dest of destinations) {
      const id = `external:${dest.address}`;
      if (!nodeIds.has(id)) {
        nodes.push({
          id,
          type: 'external',
          label: dest.address,
          ip: dest.address,
        });
        nodeIds.add(id);
      }
    }

    // Build links from DNS data
    const links: GraphLink[] = [];

    // Get DNS queries with source IPs
    let dnsLinkQuery = `
      SELECT domain, querying_ips
      FROM dns_domains
      WHERE querying_ips IS NOT NULL
    `;
    const dnsLinkParams: number[] = [];

    if (captureId) {
      dnsLinkQuery += ' AND capture_id = ?';
      dnsLinkParams.push(parseInt(captureId));
    }

    const dnsLinks = getAll<DNSLinkRow>(db, dnsLinkQuery, dnsLinkParams);

    // Create a map of IP to device MAC
    const ipToDevice = new Map<string, string>();
    for (const device of devices) {
      const ips = device.ips ? device.ips.split(',') : [];
      for (const ip of ips) {
        ipToDevice.set(ip, device.mac);
      }
    }

    // Add DNS query links
    for (const dns of dnsLinks) {
      try {
        const queryingIPs = JSON.parse(dns.querying_ips) as string[];
        for (const ip of queryingIPs) {
          const deviceMac = ipToDevice.get(ip);
          if (deviceMac) {
            const sourceId = `device:${deviceMac}`;
            const targetId = `domain:${dns.domain}`;

            if (nodeIds.has(sourceId) && nodeIds.has(targetId)) {
              links.push({
                source: sourceId,
                target: targetId,
                weight: 1,
              });
            }
          }
        }
      } catch {
        // Skip invalid JSON
      }
    }

    // Add destination links (connect devices to external IPs they communicated with)
    // This is simplified - in a real implementation you'd have flow data
    for (const dest of destinations) {
      const targetId = `external:${dest.address}`;
      if (nodeIds.has(targetId) && devices.length > 0) {
        // Connect to first device as a placeholder
        // In reality, you'd use flow data to determine actual connections
        links.push({
          source: `device:${devices[0].mac}`,
          target: targetId,
          weight: Math.log10(dest.bytes + 1),
        });
      }
    }

    const graphData: GraphData = { nodes, links };
    return NextResponse.json(graphData);
  } catch (error) {
    console.error('Graph fetch error:', error);
    return NextResponse.json({ error: 'Failed to fetch graph data' }, { status: 500 });
  }
}
