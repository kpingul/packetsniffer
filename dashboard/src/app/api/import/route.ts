import { NextRequest, NextResponse } from 'next/server';
import { getDB, saveDB, runQuery, insert } from '@/lib/db/client';
import { z } from 'zod';
import type { Summary } from '@/types';

// Validation schema for summary JSON
const SummarySchema = z.object({
  sensor: z.object({
    os: z.string(),
    hostname: z.string(),
    interface: z.string(),
    localIP: z.string(),
  }),
  capture: z.object({
    startTime: z.string(),
    duration: z.number(),
    packetCount: z.number(),
  }),
  devices: z.array(z.object({
    mac: z.string(),
    ips: z.array(z.string()),
    vendor: z.string().optional(),
    hostname: z.string().optional(),
    osGuess: z.string().optional(),
    confidence: z.number().optional(),
    signalsUsed: z.array(z.string()).optional(),
    discoverySource: z.string().optional(),
    firstSeen: z.string().optional(),
    lastSeen: z.string().optional(),
  })),
  traffic: z.object({
    protocolCounts: z.record(z.number()),
    topPorts: z.array(z.object({
      port: z.number(),
      protocol: z.string(),
      count: z.number(),
    })),
    topTalkers: z.array(z.object({
      ip: z.string(),
      bytesSent: z.number(),
      bytesReceived: z.number(),
      packetsSent: z.number(),
      packetsReceived: z.number(),
    })),
    dnsDomains: z.array(z.object({
      domain: z.string(),
      queryCount: z.number(),
      queryingIPs: z.array(z.string()).optional(),
    })),
    destinations: z.array(z.object({
      address: z.string(),
      connectionCount: z.number(),
      bytesTotal: z.number(),
    })),
  }),
});

export async function POST(request: NextRequest) {
  try {
    const formData = await request.formData();
    const file = formData.get('file') as File;

    if (!file) {
      return NextResponse.json({ error: 'No file provided' }, { status: 400 });
    }

    const content = await file.text();
    let data: Summary;

    try {
      data = SummarySchema.parse(JSON.parse(content));
    } catch (e) {
      return NextResponse.json({ error: 'Invalid JSON format' }, { status: 400 });
    }

    const db = await getDB();

    // Insert capture
    const captureId = insert(db, `
      INSERT INTO captures (sensor_os, sensor_hostname, interface_name,
        local_ip, start_time, duration_seconds, packet_count, filename)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `, [
      data.sensor.os,
      data.sensor.hostname,
      data.sensor.interface,
      data.sensor.localIP,
      data.capture.startTime,
      data.capture.duration,
      data.capture.packetCount,
      file.name
    ]);

    // Insert devices
    for (const device of data.devices) {
      const deviceId = insert(db, `
        INSERT INTO devices (capture_id, mac, vendor, hostname, os_guess,
          os_confidence, signals_used, discovery_source, first_seen, last_seen)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
      `, [
        captureId,
        device.mac,
        device.vendor || null,
        device.hostname || null,
        device.osGuess || null,
        device.confidence || null,
        device.signalsUsed ? JSON.stringify(device.signalsUsed) : null,
        device.discoverySource || 'passive',
        device.firstSeen || data.capture.startTime,
        device.lastSeen || data.capture.startTime
      ]);

      for (const ip of device.ips) {
        runQuery(db, `INSERT INTO device_ips (device_id, ip) VALUES (?, ?)`, [deviceId, ip]);
      }
    }

    // Insert protocol counts
    for (const [protocol, count] of Object.entries(data.traffic.protocolCounts)) {
      runQuery(db, `INSERT INTO protocol_counts (capture_id, protocol, count) VALUES (?, ?, ?)`,
        [captureId, protocol, count]);
    }

    // Insert top ports
    for (const port of data.traffic.topPorts) {
      runQuery(db, `INSERT INTO top_ports (capture_id, port, protocol, count) VALUES (?, ?, ?, ?)`,
        [captureId, port.port, port.protocol, port.count]);
    }

    // Insert top talkers
    for (const talker of data.traffic.topTalkers) {
      runQuery(db, `
        INSERT INTO top_talkers (capture_id, ip, bytes_sent, bytes_received, packets_sent, packets_received)
        VALUES (?, ?, ?, ?, ?, ?)
      `, [
        captureId,
        talker.ip,
        talker.bytesSent,
        talker.bytesReceived,
        talker.packetsSent,
        talker.packetsReceived
      ]);
    }

    // Insert DNS domains
    for (const domain of data.traffic.dnsDomains) {
      runQuery(db, `INSERT INTO dns_domains (capture_id, domain, query_count, querying_ips) VALUES (?, ?, ?, ?)`,
        [captureId, domain.domain, domain.queryCount, domain.queryingIPs ? JSON.stringify(domain.queryingIPs) : null]);
    }

    // Insert destinations
    for (const dest of data.traffic.destinations) {
      runQuery(db, `INSERT INTO destinations (capture_id, address, connection_count, bytes_total) VALUES (?, ?, ?, ?)`,
        [captureId, dest.address, dest.connectionCount, dest.bytesTotal]);
    }

    // Save to disk
    saveDB();

    return NextResponse.json({
      success: true,
      captureId,
      deviceCount: data.devices.length,
    });
  } catch (error) {
    console.error('Import error:', error);
    return NextResponse.json({ error: 'Import failed' }, { status: 500 });
  }
}
