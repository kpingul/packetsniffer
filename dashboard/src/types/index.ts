// Summary JSON types (matches Go sensor output)
export interface Summary {
  sensor: SensorInfo;
  capture: CaptureInfo;
  devices: DeviceInfo[];
  traffic: TrafficInfo;
}

export interface SensorInfo {
  os: string;
  hostname: string;
  interface: string;
  localIP: string;
}

export interface CaptureInfo {
  startTime: string;
  duration: number;
  packetCount: number;
}

export interface DeviceInfo {
  mac: string;
  ips: string[];
  vendor?: string;
  hostname?: string;
  osGuess?: string;
  confidence?: number;
  signalsUsed?: string[];
  discoverySource?: string;
  firstSeen?: string;
  lastSeen?: string;
}

export interface TrafficInfo {
  protocolCounts: Record<string, number>;
  topPorts: PortCount[];
  topTalkers: TalkerInfo[];
  dnsDomains: DNSDomainInfo[];
  destinations: DestinationInfo[];
}

export interface PortCount {
  port: number;
  protocol: string;
  count: number;
}

export interface TalkerInfo {
  ip: string;
  bytesSent: number;
  bytesReceived: number;
  packetsSent: number;
  packetsReceived: number;
}

export interface DNSDomainInfo {
  domain: string;
  queryCount: number;
  queryingIPs?: string[];
}

export interface DestinationInfo {
  address: string;
  connectionCount: number;
  bytesTotal: number;
}

// Database types
export interface Capture {
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
}

export interface Device {
  id: number;
  capture_id: number;
  mac: string;
  vendor: string | null;
  hostname: string | null;
  os_guess: string | null;
  os_confidence: number | null;
  signals_used: string | null;
  discovery_source: string | null;
  first_seen: string;
  last_seen: string;
  ips?: string[];
}

// Graph types
export interface GraphNode {
  id: string;
  type: 'device' | 'domain' | 'external';
  label: string;
  mac?: string;
  ip?: string;
  vendor?: string;
  os?: string;
  x?: number;
  y?: number;
  fx?: number | null;
  fy?: number | null;
}

export interface GraphLink {
  source: string | GraphNode;
  target: string | GraphNode;
  weight: number;
  protocol?: string;
}

export interface GraphData {
  nodes: GraphNode[];
  links: GraphLink[];
}
