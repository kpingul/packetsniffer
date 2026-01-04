'use client';

import { useEffect, useState } from 'react';

interface TrafficData {
  protocols: { protocol: string; count: number }[];
  ports: { port: number; protocol: string; count: number }[];
  talkers: {
    ip: string;
    bytes_sent: number;
    bytes_received: number;
    packets_sent: number;
    packets_received: number;
  }[];
  dnsDomains: { domain: string; query_count: number }[];
  destinations: { address: string; connection_count: number; bytes_total: number }[];
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export default function NetworkPage() {
  const [data, setData] = useState<TrafficData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchTraffic() {
      try {
        const res = await fetch('/api/traffic');
        const trafficData = await res.json();
        setData(trafficData);
      } catch (error) {
        console.error('Failed to fetch traffic:', error);
      } finally {
        setLoading(false);
      }
    }

    fetchTraffic();
  }, []);

  const totalTraffic = data?.protocols.reduce((sum, p) => sum + p.count, 0) || 0;

  const protocolColors: Record<string, string> = {
    TCP: 'bg-cyan-500',
    UDP: 'bg-amber-500',
    ICMP: 'bg-emerald-500',
    ARP: 'bg-rose-500',
    Other: 'bg-slate-500',
  };

  return (
    <div className="p-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-semibold text-[rgb(var(--text-primary))] mb-2">
          Network Traffic
        </h1>
        <p className="text-[rgb(var(--text-muted))]">
          Protocol distribution and traffic analysis
        </p>
      </div>

      {loading ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="card h-80 skeleton"></div>
          ))}
        </div>
      ) : (
        <div className="space-y-6">
          {/* Protocol Distribution */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="card">
              <div className="p-5 border-b border-[rgb(var(--border-subtle))]">
                <h2 className="font-medium text-[rgb(var(--text-primary))]">Protocol Distribution</h2>
              </div>
              <div className="p-5">
                {data?.protocols && data.protocols.length > 0 ? (
                  <div className="space-y-4">
                    {/* Bar chart */}
                    <div className="h-8 flex rounded-lg overflow-hidden">
                      {data.protocols.map((p) => {
                        const percentage = (p.count / totalTraffic) * 100;
                        return (
                          <div
                            key={p.protocol}
                            className={`${protocolColors[p.protocol] || protocolColors.Other} first:rounded-l-lg last:rounded-r-lg transition-all hover:opacity-80`}
                            style={{ width: `${percentage}%` }}
                            title={`${p.protocol}: ${percentage.toFixed(1)}%`}
                          />
                        );
                      })}
                    </div>

                    {/* Legend */}
                    <div className="grid grid-cols-2 gap-3">
                      {data.protocols.map((p) => (
                        <div key={p.protocol} className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <div
                              className={`w-3 h-3 rounded ${protocolColors[p.protocol] || protocolColors.Other}`}
                            />
                            <span className="mono text-sm text-[rgb(var(--text-secondary))]">
                              {p.protocol}
                            </span>
                          </div>
                          <div className="text-right">
                            <span className="mono text-sm text-[rgb(var(--text-primary))]">
                              {((p.count / totalTraffic) * 100).toFixed(1)}%
                            </span>
                            <span className="text-xs text-[rgb(var(--text-muted))] ml-2">
                              ({p.count.toLocaleString()})
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : (
                  <p className="text-center text-[rgb(var(--text-muted))] py-8">
                    No protocol data available
                  </p>
                )}
              </div>
            </div>

            {/* Top Ports */}
            <div className="card">
              <div className="p-5 border-b border-[rgb(var(--border-subtle))]">
                <h2 className="font-medium text-[rgb(var(--text-primary))]">Top Ports</h2>
              </div>
              <div className="p-5">
                {data?.ports && data.ports.length > 0 ? (
                  <div className="space-y-3">
                    {data.ports.slice(0, 8).map((port, index) => {
                      const maxCount = data.ports[0].count;
                      const percentage = (port.count / maxCount) * 100;
                      return (
                        <div key={`${port.protocol}-${port.port}`}>
                          <div className="flex items-center justify-between mb-1">
                            <div className="flex items-center gap-2">
                              <span className="mono text-sm text-cyan-400">{port.port}</span>
                              <span className="tag tag-muted text-[10px]">{port.protocol}</span>
                            </div>
                            <span className="mono text-xs text-[rgb(var(--text-muted))]">
                              {port.count.toLocaleString()}
                            </span>
                          </div>
                          <div className="h-1.5 bg-[rgb(var(--bg-tertiary))] rounded-full overflow-hidden">
                            <div
                              className="h-full bg-gradient-to-r from-cyan-500 to-cyan-400 rounded-full transition-all"
                              style={{ width: `${percentage}%` }}
                            />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <p className="text-center text-[rgb(var(--text-muted))] py-8">
                    No port data available
                  </p>
                )}
              </div>
            </div>
          </div>

          {/* Top Talkers & DNS */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Top Talkers */}
            <div className="card">
              <div className="p-5 border-b border-[rgb(var(--border-subtle))]">
                <h2 className="font-medium text-[rgb(var(--text-primary))]">Top Talkers</h2>
              </div>
              <div className="overflow-x-auto">
                {data?.talkers && data.talkers.length > 0 ? (
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>IP Address</th>
                        <th className="text-right">Sent</th>
                        <th className="text-right">Received</th>
                        <th className="text-right">Total</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.talkers.slice(0, 10).map((talker) => (
                        <tr key={talker.ip}>
                          <td>
                            <span className="mono text-cyan-400">{talker.ip}</span>
                          </td>
                          <td className="text-right">
                            <span className="mono text-emerald-400">
                              {formatBytes(talker.bytes_sent)}
                            </span>
                          </td>
                          <td className="text-right">
                            <span className="mono text-amber-400">
                              {formatBytes(talker.bytes_received)}
                            </span>
                          </td>
                          <td className="text-right">
                            <span className="mono">
                              {formatBytes(talker.bytes_sent + talker.bytes_received)}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                ) : (
                  <p className="text-center text-[rgb(var(--text-muted))] p-8">
                    No talker data available
                  </p>
                )}
              </div>
            </div>

            {/* DNS Domains */}
            <div className="card">
              <div className="p-5 border-b border-[rgb(var(--border-subtle))]">
                <h2 className="font-medium text-[rgb(var(--text-primary))]">DNS Domains</h2>
              </div>
              <div className="p-5">
                {data?.dnsDomains && data.dnsDomains.length > 0 ? (
                  <div className="space-y-2 max-h-80 overflow-y-auto">
                    {data.dnsDomains.slice(0, 20).map((domain, index) => (
                      <div
                        key={domain.domain}
                        className="flex items-center justify-between p-2 rounded hover:bg-[rgb(var(--bg-tertiary))] transition-colors"
                      >
                        <span className="mono text-sm text-[rgb(var(--text-secondary))] truncate max-w-[200px]">
                          {domain.domain}
                        </span>
                        <span className="tag tag-muted ml-2">
                          {domain.query_count} queries
                        </span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-center text-[rgb(var(--text-muted))] py-8">
                    No DNS data available
                  </p>
                )}
              </div>
            </div>
          </div>

          {/* External Destinations */}
          <div className="card">
            <div className="p-5 border-b border-[rgb(var(--border-subtle))]">
              <h2 className="font-medium text-[rgb(var(--text-primary))]">External Destinations</h2>
            </div>
            <div className="overflow-x-auto">
              {data?.destinations && data.destinations.length > 0 ? (
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>Address</th>
                      <th className="text-right">Connections</th>
                      <th className="text-right">Total Bytes</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.destinations.slice(0, 15).map((dest) => (
                      <tr key={dest.address}>
                        <td>
                          <span className="mono text-amber-400">{dest.address}</span>
                        </td>
                        <td className="text-right">
                          <span className="mono">{dest.connection_count.toLocaleString()}</span>
                        </td>
                        <td className="text-right">
                          <span className="mono">{formatBytes(dest.bytes_total)}</span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <p className="text-center text-[rgb(var(--text-muted))] p-8">
                  No destination data available
                </p>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
