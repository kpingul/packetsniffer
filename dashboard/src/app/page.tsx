'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';

interface Capture {
  id: number;
  sensor_hostname: string;
  interface_name: string;
  start_time: string;
  duration_seconds: number;
  packet_count: number;
  device_count: number;
}

interface Stats {
  totalDevices: number;
  totalCaptures: number;
  totalDomains: number;
  totalPackets: number;
}

export default function HomePage() {
  const [captures, setCaptures] = useState<Capture[]>([]);
  const [stats, setStats] = useState<Stats>({
    totalDevices: 0,
    totalCaptures: 0,
    totalDomains: 0,
    totalPackets: 0,
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchData() {
      try {
        const [capturesRes, devicesRes, trafficRes] = await Promise.all([
          fetch('/api/captures'),
          fetch('/api/devices'),
          fetch('/api/traffic'),
        ]);

        const capturesData = await capturesRes.json();
        const devicesData = await devicesRes.json();
        const trafficData = await trafficRes.json();

        setCaptures(Array.isArray(capturesData) ? capturesData.slice(0, 5) : []);

        const totalPackets = Array.isArray(capturesData)
          ? capturesData.reduce((sum: number, c: Capture) => sum + c.packet_count, 0)
          : 0;

        setStats({
          totalDevices: Array.isArray(devicesData) ? devicesData.length : 0,
          totalCaptures: Array.isArray(capturesData) ? capturesData.length : 0,
          totalDomains: Array.isArray(trafficData.dnsDomains) ? trafficData.dnsDomains.length : 0,
          totalPackets,
        });
      } catch (error) {
        console.error('Failed to fetch data:', error);
      } finally {
        setLoading(false);
      }
    }

    fetchData();
  }, []);

  const statCards = [
    {
      label: 'Total Devices',
      value: stats.totalDevices,
      icon: (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
        </svg>
      ),
      color: 'cyan',
    },
    {
      label: 'Captures',
      value: stats.totalCaptures,
      icon: (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
      ),
      color: 'emerald',
    },
    {
      label: 'DNS Domains',
      value: stats.totalDomains,
      icon: (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
        </svg>
      ),
      color: 'amber',
    },
    {
      label: 'Total Packets',
      value: stats.totalPackets.toLocaleString(),
      icon: (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
      ),
      color: 'rose',
    },
  ];

  const colorClasses: Record<string, { bg: string; text: string; glow: string }> = {
    cyan: {
      bg: 'bg-cyan-500/10',
      text: 'text-cyan-400',
      glow: 'shadow-cyan-500/20',
    },
    emerald: {
      bg: 'bg-emerald-500/10',
      text: 'text-emerald-400',
      glow: 'shadow-emerald-500/20',
    },
    amber: {
      bg: 'bg-amber-500/10',
      text: 'text-amber-400',
      glow: 'shadow-amber-500/20',
    },
    rose: {
      bg: 'bg-rose-500/10',
      text: 'text-rose-400',
      glow: 'shadow-rose-500/20',
    },
  };

  return (
    <div className="p-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-semibold text-[rgb(var(--text-primary))] mb-2">
          Network Overview
        </h1>
        <p className="text-[rgb(var(--text-muted))]">
          Monitor your local network assets and traffic patterns
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        {statCards.map((stat) => (
          <div key={stat.label} className="card p-5">
            <div className="flex items-start justify-between mb-4">
              <div className={`p-2.5 rounded-lg ${colorClasses[stat.color].bg} ${colorClasses[stat.color].text}`}>
                {stat.icon}
              </div>
              <span className="status-dot online"></span>
            </div>
            <div className="space-y-1">
              <p className="text-[rgb(var(--text-muted))] text-sm">{stat.label}</p>
              <p className={`text-2xl font-semibold mono stat-value ${colorClasses[stat.color].text}`}>
                {loading ? (
                  <span className="inline-block w-16 h-7 skeleton rounded"></span>
                ) : (
                  stat.value
                )}
              </p>
            </div>
          </div>
        ))}
      </div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Recent Captures */}
        <div className="lg:col-span-2 card">
          <div className="flex items-center justify-between p-5 border-b border-[rgb(var(--border-subtle))]">
            <h2 className="font-medium text-[rgb(var(--text-primary))]">Recent Captures</h2>
            <Link href="/import" className="text-xs text-cyan-400 hover:text-cyan-300 transition-colors">
              View All →
            </Link>
          </div>

          {loading ? (
            <div className="p-5 space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-16 skeleton rounded-lg"></div>
              ))}
            </div>
          ) : captures.length === 0 ? (
            <div className="p-12 text-center">
              <div className="w-12 h-12 mx-auto mb-4 rounded-full bg-[rgb(var(--bg-tertiary))] flex items-center justify-center">
                <svg className="w-6 h-6 text-[rgb(var(--text-muted))]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              </div>
              <p className="text-[rgb(var(--text-muted))] text-sm mb-4">No captures yet</p>
              <Link href="/import" className="btn btn-primary text-sm">
                Import First Capture
              </Link>
            </div>
          ) : (
            <div className="divide-y divide-[rgb(var(--border-subtle))]">
              {captures.map((capture) => (
                <div key={capture.id} className="p-4 hover:bg-[rgb(var(--bg-tertiary))/30] transition-colors">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-lg bg-[rgb(var(--bg-tertiary))] flex items-center justify-center">
                        <svg className="w-5 h-5 text-cyan-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                        </svg>
                      </div>
                      <div>
                        <p className="font-medium text-[rgb(var(--text-primary))] text-sm">
                          {capture.sensor_hostname}
                        </p>
                        <p className="text-xs text-[rgb(var(--text-muted))] mono">
                          {capture.interface_name}
                        </p>
                      </div>
                    </div>
                    <span className="tag tag-cyan">{capture.device_count} devices</span>
                  </div>
                  <div className="flex items-center gap-4 text-xs text-[rgb(var(--text-muted))] mono">
                    <span>{new Date(capture.start_time).toLocaleString()}</span>
                    <span>•</span>
                    <span>{capture.duration_seconds}s duration</span>
                    <span>•</span>
                    <span>{capture.packet_count.toLocaleString()} packets</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Quick Actions */}
        <div className="card">
          <div className="p-5 border-b border-[rgb(var(--border-subtle))]">
            <h2 className="font-medium text-[rgb(var(--text-primary))]">Quick Actions</h2>
          </div>
          <div className="p-5 space-y-3">
            <Link
              href="/import"
              className="flex items-center gap-3 p-3 rounded-lg bg-[rgb(var(--bg-tertiary))] hover:bg-[rgb(var(--bg-elevated))] transition-colors group"
            >
              <div className="w-10 h-10 rounded-lg bg-cyan-500/10 flex items-center justify-center group-hover:bg-cyan-500/20 transition-colors">
                <svg className="w-5 h-5 text-cyan-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                </svg>
              </div>
              <div>
                <p className="font-medium text-[rgb(var(--text-primary))] text-sm">Import Capture</p>
                <p className="text-xs text-[rgb(var(--text-muted))]">Upload summary.json file</p>
              </div>
            </Link>

            <Link
              href="/assets"
              className="flex items-center gap-3 p-3 rounded-lg bg-[rgb(var(--bg-tertiary))] hover:bg-[rgb(var(--bg-elevated))] transition-colors group"
            >
              <div className="w-10 h-10 rounded-lg bg-emerald-500/10 flex items-center justify-center group-hover:bg-emerald-500/20 transition-colors">
                <svg className="w-5 h-5 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                </svg>
              </div>
              <div>
                <p className="font-medium text-[rgb(var(--text-primary))] text-sm">View Assets</p>
                <p className="text-xs text-[rgb(var(--text-muted))]">Browse discovered devices</p>
              </div>
            </Link>

            <Link
              href="/graph"
              className="flex items-center gap-3 p-3 rounded-lg bg-[rgb(var(--bg-tertiary))] hover:bg-[rgb(var(--bg-elevated))] transition-colors group"
            >
              <div className="w-10 h-10 rounded-lg bg-amber-500/10 flex items-center justify-center group-hover:bg-amber-500/20 transition-colors">
                <svg className="w-5 h-5 text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                </svg>
              </div>
              <div>
                <p className="font-medium text-[rgb(var(--text-primary))] text-sm">Network Graph</p>
                <p className="text-xs text-[rgb(var(--text-muted))]">Visualize connections</p>
              </div>
            </Link>
          </div>

          {/* System Status */}
          <div className="p-5 border-t border-[rgb(var(--border-subtle))]">
            <h3 className="text-xs font-medium text-[rgb(var(--text-muted))] uppercase tracking-wider mb-3">
              System Status
            </h3>
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-[rgb(var(--text-secondary))]">Database</span>
                <span className="flex items-center gap-2 text-emerald-400">
                  <span className="status-dot online"></span>
                  Connected
                </span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-[rgb(var(--text-secondary))]">Last Import</span>
                <span className="text-[rgb(var(--text-muted))] mono text-xs">
                  {captures.length > 0
                    ? new Date(captures[0].start_time).toLocaleDateString()
                    : 'Never'}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
