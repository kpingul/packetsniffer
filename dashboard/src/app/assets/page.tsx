'use client';

import { useEffect, useState } from 'react';

interface Device {
  id: number;
  mac: string;
  ips: string[];
  vendor: string | null;
  hostname: string | null;
  os_guess: string | null;
  os_confidence: number | null;
  discovery_source: string | null;
  last_seen: string;
}

export default function AssetsPage() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('');
  const [osFilter, setOsFilter] = useState('all');

  useEffect(() => {
    async function fetchDevices() {
      try {
        const res = await fetch('/api/devices');
        const data = await res.json();
        setDevices(Array.isArray(data) ? data : []);
      } catch (error) {
        console.error('Failed to fetch devices:', error);
      } finally {
        setLoading(false);
      }
    }

    fetchDevices();
  }, []);

  const filteredDevices = devices.filter((device) => {
    const matchesSearch =
      filter === '' ||
      device.mac.toLowerCase().includes(filter.toLowerCase()) ||
      device.hostname?.toLowerCase().includes(filter.toLowerCase()) ||
      device.vendor?.toLowerCase().includes(filter.toLowerCase()) ||
      device.ips.some((ip) => ip.includes(filter));

    const matchesOS =
      osFilter === 'all' ||
      (osFilter === 'unknown' && !device.os_guess) ||
      device.os_guess?.toLowerCase().includes(osFilter.toLowerCase());

    return matchesSearch && matchesOS;
  });

  const uniqueOSGuesses = [...new Set(devices.map((d) => d.os_guess).filter(Boolean))];

  const getOSBadgeClass = (os: string | null) => {
    if (!os) return 'tag-muted';
    const lower = os.toLowerCase();
    if (lower.includes('windows')) return 'tag-cyan';
    if (lower.includes('macos') || lower.includes('ios')) return 'tag-muted';
    if (lower.includes('linux')) return 'tag-amber';
    return 'tag-muted';
  };

  const getConfidenceColor = (confidence: number | null) => {
    if (!confidence) return 'text-[rgb(var(--text-muted))]';
    if (confidence >= 0.8) return 'text-emerald-400';
    if (confidence >= 0.5) return 'text-amber-400';
    return 'text-rose-400';
  };

  return (
    <div className="p-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-semibold text-[rgb(var(--text-primary))] mb-2">
          Network Assets
        </h1>
        <p className="text-[rgb(var(--text-muted))]">
          {devices.length} device{devices.length !== 1 ? 's' : ''} discovered on your network
        </p>
      </div>

      {/* Filters */}
      <div className="card p-4 mb-6">
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex-1 min-w-[200px]">
            <div className="relative">
              <svg
                className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[rgb(var(--text-muted))]"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
              <input
                type="text"
                placeholder="Search MAC, IP, hostname, vendor..."
                className="input pl-10 mono text-sm"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
              />
            </div>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-sm text-[rgb(var(--text-muted))]">OS:</span>
            <select
              value={osFilter}
              onChange={(e) => setOsFilter(e.target.value)}
              className="input py-2 px-3 text-sm w-auto min-w-[140px]"
            >
              <option value="all">All</option>
              <option value="unknown">Unknown</option>
              {uniqueOSGuesses.map((os) => (
                <option key={os} value={os!}>
                  {os}
                </option>
              ))}
            </select>
          </div>

          <div className="text-sm text-[rgb(var(--text-muted))]">
            Showing {filteredDevices.length} of {devices.length}
          </div>
        </div>
      </div>

      {/* Device Table */}
      <div className="card overflow-hidden">
        {loading ? (
          <div className="p-8 space-y-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="h-14 skeleton rounded"></div>
            ))}
          </div>
        ) : filteredDevices.length === 0 ? (
          <div className="p-12 text-center">
            <div className="w-12 h-12 mx-auto mb-4 rounded-full bg-[rgb(var(--bg-tertiary))] flex items-center justify-center">
              <svg
                className="w-6 h-6 text-[rgb(var(--text-muted))]"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                />
              </svg>
            </div>
            <p className="text-[rgb(var(--text-muted))]">
              {devices.length === 0
                ? 'No devices found. Import a capture to see devices.'
                : 'No devices match your filters.'}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="data-table">
              <thead>
                <tr>
                  <th>MAC Address</th>
                  <th>IP Addresses</th>
                  <th>Vendor</th>
                  <th>Hostname</th>
                  <th>OS Guess</th>
                  <th>Discovery</th>
                  <th>Last Seen</th>
                </tr>
              </thead>
              <tbody>
                {filteredDevices.map((device) => (
                  <tr key={device.id}>
                    <td>
                      <span className="mono text-cyan-400">{device.mac}</span>
                    </td>
                    <td>
                      <div className="flex flex-wrap gap-1">
                        {device.ips.slice(0, 3).map((ip) => (
                          <span key={ip} className="mono text-xs">
                            {ip}
                          </span>
                        ))}
                        {device.ips.length > 3 && (
                          <span className="text-xs text-[rgb(var(--text-muted))]">
                            +{device.ips.length - 3} more
                          </span>
                        )}
                      </div>
                    </td>
                    <td>
                      <span className="text-[rgb(var(--text-secondary))]">
                        {device.vendor || '—'}
                      </span>
                    </td>
                    <td>
                      <span className="mono text-sm">
                        {device.hostname || '—'}
                      </span>
                    </td>
                    <td>
                      {device.os_guess ? (
                        <div className="flex items-center gap-2">
                          <span className={`tag ${getOSBadgeClass(device.os_guess)}`}>
                            {device.os_guess}
                          </span>
                          {device.os_confidence && (
                            <span
                              className={`mono text-xs ${getConfidenceColor(device.os_confidence)}`}
                            >
                              {Math.round(device.os_confidence * 100)}%
                            </span>
                          )}
                        </div>
                      ) : (
                        <span className="text-[rgb(var(--text-muted))]">Unknown</span>
                      )}
                    </td>
                    <td>
                      <span
                        className={`tag ${
                          device.discovery_source === 'passive'
                            ? 'tag-muted'
                            : 'tag-emerald'
                        }`}
                      >
                        {device.discovery_source || 'passive'}
                      </span>
                    </td>
                    <td>
                      <span className="mono text-xs text-[rgb(var(--text-muted))]">
                        {new Date(device.last_seen).toLocaleString()}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="mt-6 flex flex-wrap items-center gap-6 text-xs text-[rgb(var(--text-muted))]">
        <div className="flex items-center gap-2">
          <span className="tag tag-cyan">Windows</span>
          <span>Microsoft devices</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="tag tag-muted">macOS/iOS</span>
          <span>Apple devices</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="tag tag-amber">Linux</span>
          <span>Linux-based devices</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="tag tag-emerald">Active</span>
          <span>Discovered via active scan</span>
        </div>
      </div>
    </div>
  );
}
