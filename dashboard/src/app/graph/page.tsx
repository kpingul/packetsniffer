'use client';

import { useEffect, useRef, useState } from 'react';
import * as d3 from 'd3';
import type { GraphData, GraphNode, GraphLink } from '@/types';

export default function GraphPage() {
  const svgRef = useRef<SVGSVGElement>(null);
  const [data, setData] = useState<GraphData | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);

  useEffect(() => {
    async function fetchGraph() {
      try {
        const res = await fetch('/api/graph');
        const graphData = await res.json();
        setData(graphData);
      } catch (error) {
        console.error('Failed to fetch graph:', error);
      } finally {
        setLoading(false);
      }
    }

    fetchGraph();
  }, []);

  useEffect(() => {
    if (!svgRef.current || !data || data.nodes.length === 0) return;

    const svg = d3.select(svgRef.current);
    svg.selectAll('*').remove();

    const width = svgRef.current.clientWidth;
    const height = svgRef.current.clientHeight;

    // Colors
    const colors = {
      device: '#06b6d4',    // cyan
      domain: '#10b981',    // emerald
      external: '#f59e0b',  // amber
    };

    // Create simulation
    const simulation = d3.forceSimulation<GraphNode>(data.nodes)
      .force('link', d3.forceLink<GraphNode, GraphLink>(data.links)
        .id(d => d.id)
        .distance(100)
        .strength(0.5))
      .force('charge', d3.forceManyBody().strength(-300))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide().radius(35));

    // Create container with zoom
    const container = svg.append('g');

    // Define arrow marker
    svg.append('defs').append('marker')
      .attr('id', 'arrowhead')
      .attr('viewBox', '-0 -5 10 10')
      .attr('refX', 25)
      .attr('refY', 0)
      .attr('orient', 'auto')
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .append('path')
      .attr('d', 'M 0,-5 L 10,0 L 0,5')
      .attr('fill', '#4b5563');

    // Add zoom behavior
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.1, 4])
      .on('zoom', (event) => {
        container.attr('transform', event.transform);
      });

    svg.call(zoom);

    // Draw links
    const link = container.append('g')
      .selectAll('line')
      .data(data.links)
      .join('line')
      .attr('stroke', '#374151')
      .attr('stroke-opacity', 0.6)
      .attr('stroke-width', d => Math.max(1, Math.sqrt(d.weight || 1)))
      .attr('marker-end', 'url(#arrowhead)');

    // Draw nodes
    const node = container.append('g')
      .selectAll<SVGGElement, GraphNode>('g')
      .data(data.nodes)
      .join('g')
      .call(d3.drag<SVGGElement, GraphNode>()
        .on('start', (event, d) => {
          if (!event.active) simulation.alphaTarget(0.3).restart();
          d.fx = d.x;
          d.fy = d.y;
        })
        .on('drag', (event, d) => {
          d.fx = event.x;
          d.fy = event.y;
        })
        .on('end', (event, d) => {
          if (!event.active) simulation.alphaTarget(0);
          d.fx = null;
          d.fy = null;
        }) as unknown as (selection: d3.Selection<SVGGElement, GraphNode, SVGGElement, unknown>) => void);

    // Node circles
    node.append('circle')
      .attr('r', d => d.type === 'device' ? 12 : 8)
      .attr('fill', d => colors[d.type] || colors.external)
      .attr('stroke', '#1f2937')
      .attr('stroke-width', 2)
      .style('cursor', 'pointer')
      .on('click', (event, d) => {
        setSelectedNode(d);
      })
      .on('mouseover', function(event, d) {
        d3.select(this)
          .transition()
          .duration(200)
          .attr('r', d.type === 'device' ? 15 : 10);
      })
      .on('mouseout', function(event, d) {
        d3.select(this)
          .transition()
          .duration(200)
          .attr('r', d.type === 'device' ? 12 : 8);
      });

    // Glow effect for device nodes
    node.filter(d => d.type === 'device')
      .select('circle')
      .style('filter', 'drop-shadow(0 0 4px rgba(6, 182, 212, 0.5))');

    // Node labels
    node.append('text')
      .text(d => d.label.length > 15 ? d.label.substring(0, 15) + '...' : d.label)
      .attr('x', 16)
      .attr('y', 4)
      .attr('font-size', '10px')
      .attr('font-family', 'JetBrains Mono, monospace')
      .attr('fill', '#9ca3af')
      .style('pointer-events', 'none');

    // Simulation tick
    simulation.on('tick', () => {
      link
        .attr('x1', d => (d.source as GraphNode).x!)
        .attr('y1', d => (d.source as GraphNode).y!)
        .attr('x2', d => (d.target as GraphNode).x!)
        .attr('y2', d => (d.target as GraphNode).y!);

      node.attr('transform', d => `translate(${d.x},${d.y})`);
    });

    // Initial zoom to fit
    const initialScale = 0.8;
    svg.call(zoom.transform, d3.zoomIdentity
      .translate(width * (1 - initialScale) / 2, height * (1 - initialScale) / 2)
      .scale(initialScale));

    return () => {
      simulation.stop();
    };
  }, [data]);

  return (
    <div className="p-8 h-screen flex flex-col">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-[rgb(var(--text-primary))] mb-2">
            Network Graph
          </h1>
          <p className="text-[rgb(var(--text-muted))]">
            Interactive visualization of network relationships
          </p>
        </div>

        {/* Legend */}
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded-full bg-cyan-500 shadow-lg shadow-cyan-500/30" />
            <span className="text-sm text-[rgb(var(--text-muted))]">Devices</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-emerald-500" />
            <span className="text-sm text-[rgb(var(--text-muted))]">Domains</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-amber-500" />
            <span className="text-sm text-[rgb(var(--text-muted))]">External IPs</span>
          </div>
        </div>
      </div>

      {/* Graph Container */}
      <div className="flex-1 flex gap-4">
        <div className="flex-1 card overflow-hidden relative">
          {loading ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-center">
                <div className="w-12 h-12 border-2 border-cyan-500 border-t-transparent rounded-full animate-spin mx-auto mb-4" />
                <p className="text-[rgb(var(--text-muted))]">Loading graph data...</p>
              </div>
            </div>
          ) : data?.nodes.length === 0 ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="text-center">
                <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-[rgb(var(--bg-tertiary))] flex items-center justify-center">
                  <svg className="w-8 h-8 text-[rgb(var(--text-muted))]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                  </svg>
                </div>
                <p className="text-[rgb(var(--text-muted))] mb-2">No graph data available</p>
                <p className="text-sm text-[rgb(var(--text-muted))]">Import captures to visualize your network</p>
              </div>
            </div>
          ) : (
            <svg
              ref={svgRef}
              className="w-full h-full"
              style={{ background: 'rgb(var(--bg-tertiary))' }}
            />
          )}

          {/* Controls */}
          <div className="absolute bottom-4 left-4 flex gap-2">
            <button
              className="btn btn-secondary text-xs"
              onClick={() => {
                if (svgRef.current) {
                  const svg = d3.select(svgRef.current);
                  svg.transition().duration(500).call(
                    d3.zoom<SVGSVGElement, unknown>().transform,
                    d3.zoomIdentity.scale(0.8)
                  );
                }
              }}
            >
              Reset View
            </button>
          </div>

          {/* Stats */}
          <div className="absolute top-4 left-4 flex gap-4 text-xs">
            <div className="px-3 py-1.5 rounded bg-[rgb(var(--bg-primary))/80] backdrop-blur">
              <span className="text-[rgb(var(--text-muted))]">Nodes: </span>
              <span className="mono text-cyan-400">{data?.nodes.length || 0}</span>
            </div>
            <div className="px-3 py-1.5 rounded bg-[rgb(var(--bg-primary))/80] backdrop-blur">
              <span className="text-[rgb(var(--text-muted))]">Links: </span>
              <span className="mono text-cyan-400">{data?.links.length || 0}</span>
            </div>
          </div>
        </div>

        {/* Node Details Panel */}
        {selectedNode && (
          <div className="w-80 card">
            <div className="p-5 border-b border-[rgb(var(--border-subtle))] flex items-center justify-between">
              <h2 className="font-medium text-[rgb(var(--text-primary))]">Node Details</h2>
              <button
                onClick={() => setSelectedNode(null)}
                className="text-[rgb(var(--text-muted))] hover:text-[rgb(var(--text-primary))] transition-colors"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="p-5 space-y-4">
              <div>
                <label className="text-xs text-[rgb(var(--text-muted))] uppercase tracking-wider">Type</label>
                <p className="mt-1">
                  <span className={`tag ${
                    selectedNode.type === 'device' ? 'tag-cyan' :
                    selectedNode.type === 'domain' ? 'tag-emerald' :
                    'tag-amber'
                  }`}>
                    {selectedNode.type}
                  </span>
                </p>
              </div>

              <div>
                <label className="text-xs text-[rgb(var(--text-muted))] uppercase tracking-wider">Label</label>
                <p className="mt-1 mono text-sm text-[rgb(var(--text-primary))]">{selectedNode.label}</p>
              </div>

              {selectedNode.mac && (
                <div>
                  <label className="text-xs text-[rgb(var(--text-muted))] uppercase tracking-wider">MAC Address</label>
                  <p className="mt-1 mono text-sm text-cyan-400">{selectedNode.mac}</p>
                </div>
              )}

              {selectedNode.ip && (
                <div>
                  <label className="text-xs text-[rgb(var(--text-muted))] uppercase tracking-wider">IP Address</label>
                  <p className="mt-1 mono text-sm text-[rgb(var(--text-primary))]">{selectedNode.ip}</p>
                </div>
              )}

              {selectedNode.vendor && (
                <div>
                  <label className="text-xs text-[rgb(var(--text-muted))] uppercase tracking-wider">Vendor</label>
                  <p className="mt-1 text-sm text-[rgb(var(--text-secondary))]">{selectedNode.vendor}</p>
                </div>
              )}

              {selectedNode.os && (
                <div>
                  <label className="text-xs text-[rgb(var(--text-muted))] uppercase tracking-wider">OS Guess</label>
                  <p className="mt-1 text-sm text-[rgb(var(--text-secondary))]">{selectedNode.os}</p>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
