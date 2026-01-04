'use client';

import { useCallback, useEffect, useState } from 'react';

interface ImportResult {
  success: boolean;
  captureId?: number;
  deviceCount?: number;
  error?: string;
}

interface Capture {
  id: number;
  sensor_hostname: string;
  interface_name: string;
  start_time: string;
  duration_seconds: number;
  packet_count: number;
  device_count: number;
  imported_at: string;
  filename: string;
}

export default function ImportPage() {
  const [isDragging, setIsDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [captures, setCaptures] = useState<Capture[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchCaptures();
  }, []);

  async function fetchCaptures() {
    try {
      const res = await fetch('/api/captures');
      const data = await res.json();
      setCaptures(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch captures:', error);
    } finally {
      setLoading(false);
    }
  }

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback(async (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);

    const files = e.dataTransfer.files;
    if (files.length > 0) {
      await uploadFile(files[0]);
    }
  }, []);

  const handleFileSelect = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files && files.length > 0) {
      await uploadFile(files[0]);
    }
  }, []);

  async function uploadFile(file: File) {
    if (!file.name.endsWith('.json')) {
      setResult({ success: false, error: 'Please upload a JSON file' });
      return;
    }

    setUploading(true);
    setResult(null);

    try {
      const formData = new FormData();
      formData.append('file', file);

      const res = await fetch('/api/import', {
        method: 'POST',
        body: formData,
      });

      const data = await res.json();

      if (res.ok) {
        setResult({
          success: true,
          captureId: data.captureId,
          deviceCount: data.deviceCount,
        });
        fetchCaptures(); // Refresh the list
      } else {
        setResult({ success: false, error: data.error || 'Import failed' });
      }
    } catch (error) {
      setResult({ success: false, error: 'Network error during upload' });
    } finally {
      setUploading(false);
    }
  }

  return (
    <div className="p-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-semibold text-[rgb(var(--text-primary))] mb-2">
          Import Capture
        </h1>
        <p className="text-[rgb(var(--text-muted))]">
          Upload sensor summary files to analyze network data
        </p>
      </div>

      {/* Upload Zone */}
      <div
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={`
          card relative overflow-hidden mb-8
          ${isDragging ? 'border-cyan-500 bg-cyan-500/5' : ''}
          transition-all duration-200
        `}
      >
        <div className="p-12 text-center">
          {/* Animated border */}
          {isDragging && (
            <div className="absolute inset-0 pointer-events-none">
              <div className="absolute inset-0 border-2 border-cyan-500 rounded-lg animate-pulse" />
            </div>
          )}

          <div className={`
            w-16 h-16 mx-auto mb-6 rounded-full flex items-center justify-center
            ${isDragging ? 'bg-cyan-500/20' : 'bg-[rgb(var(--bg-tertiary))]'}
            transition-colors
          `}>
            {uploading ? (
              <div className="w-8 h-8 border-2 border-cyan-500 border-t-transparent rounded-full animate-spin" />
            ) : (
              <svg
                className={`w-8 h-8 ${isDragging ? 'text-cyan-400' : 'text-[rgb(var(--text-muted))]'}`}
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
                />
              </svg>
            )}
          </div>

          <h3 className="text-lg font-medium text-[rgb(var(--text-primary))] mb-2">
            {uploading ? 'Uploading...' : isDragging ? 'Drop file here' : 'Upload Summary File'}
          </h3>
          <p className="text-[rgb(var(--text-muted))] mb-6">
            Drag and drop a <span className="mono text-cyan-400">summary_*.json</span> file, or click to browse
          </p>

          <label className="btn btn-primary cursor-pointer">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Select File
            <input
              type="file"
              accept=".json"
              onChange={handleFileSelect}
              className="hidden"
              disabled={uploading}
            />
          </label>
        </div>
      </div>

      {/* Result Message */}
      {result && (
        <div
          className={`
            card p-4 mb-8 flex items-center gap-4
            ${result.success ? 'border-emerald-500/30 bg-emerald-500/5' : 'border-rose-500/30 bg-rose-500/5'}
          `}
        >
          {result.success ? (
            <>
              <div className="w-10 h-10 rounded-full bg-emerald-500/20 flex items-center justify-center">
                <svg className="w-5 h-5 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <div>
                <p className="font-medium text-emerald-400">Import Successful</p>
                <p className="text-sm text-[rgb(var(--text-muted))]">
                  Imported {result.deviceCount} devices (Capture #{result.captureId})
                </p>
              </div>
            </>
          ) : (
            <>
              <div className="w-10 h-10 rounded-full bg-rose-500/20 flex items-center justify-center">
                <svg className="w-5 h-5 text-rose-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </div>
              <div>
                <p className="font-medium text-rose-400">Import Failed</p>
                <p className="text-sm text-[rgb(var(--text-muted))]">{result.error}</p>
              </div>
            </>
          )}
        </div>
      )}

      {/* Import History */}
      <div className="card">
        <div className="p-5 border-b border-[rgb(var(--border-subtle))]">
          <h2 className="font-medium text-[rgb(var(--text-primary))]">Import History</h2>
        </div>

        {loading ? (
          <div className="p-5 space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-16 skeleton rounded"></div>
            ))}
          </div>
        ) : captures.length === 0 ? (
          <div className="p-12 text-center">
            <div className="w-12 h-12 mx-auto mb-4 rounded-full bg-[rgb(var(--bg-tertiary))] flex items-center justify-center">
              <svg className="w-6 h-6 text-[rgb(var(--text-muted))]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <p className="text-[rgb(var(--text-muted))]">No captures imported yet</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="data-table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Filename</th>
                  <th>Sensor</th>
                  <th>Devices</th>
                  <th>Packets</th>
                  <th>Duration</th>
                  <th>Captured</th>
                  <th>Imported</th>
                </tr>
              </thead>
              <tbody>
                {captures.map((capture) => (
                  <tr key={capture.id}>
                    <td>
                      <span className="mono text-cyan-400">#{capture.id}</span>
                    </td>
                    <td>
                      <span className="mono text-sm">{capture.filename}</span>
                    </td>
                    <td>
                      <div>
                        <p className="text-[rgb(var(--text-primary))]">{capture.sensor_hostname}</p>
                        <p className="text-xs text-[rgb(var(--text-muted))] mono">{capture.interface_name}</p>
                      </div>
                    </td>
                    <td>
                      <span className="tag tag-cyan">{capture.device_count}</span>
                    </td>
                    <td>
                      <span className="mono">{capture.packet_count.toLocaleString()}</span>
                    </td>
                    <td>
                      <span className="mono">{capture.duration_seconds}s</span>
                    </td>
                    <td>
                      <span className="mono text-xs">
                        {new Date(capture.start_time).toLocaleString()}
                      </span>
                    </td>
                    <td>
                      <span className="mono text-xs text-[rgb(var(--text-muted))]">
                        {new Date(capture.imported_at).toLocaleString()}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Help Section */}
      <div className="mt-8 p-6 rounded-lg bg-[rgb(var(--bg-tertiary))/50] border border-[rgb(var(--border-subtle))]">
        <h3 className="font-medium text-[rgb(var(--text-primary))] mb-4">How to capture data</h3>
        <div className="space-y-3 text-sm text-[rgb(var(--text-muted))]">
          <p>Run the sensor to capture network data:</p>
          <pre className="p-4 rounded bg-[rgb(var(--bg-primary))] mono text-xs overflow-x-auto">
            <code className="text-cyan-400">sudo ./sensor --duration 30 --output ./captures</code>
          </pre>
          <p>Then upload the generated <span className="mono text-cyan-400">summary_*.json</span> file here.</p>
        </div>
      </div>
    </div>
  );
}
