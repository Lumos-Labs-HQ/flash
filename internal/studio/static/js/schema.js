import React from 'react';
import ReactDOM from 'react-dom/client';
import { ReactFlow, Background, Controls, MiniMap, MarkerType, useNodesState, useEdgesState, Handle, Position } from '@xyflow/react';
import dagre from 'dagre';

const { useState, useEffect } = React;

function TableNode({ data }) {
    return React.createElement('div', { style: { minWidth: '220px', cursor: 'pointer' }, onClick: () => window.openTableEdit(data.label) },
        React.createElement(Handle, { type: 'target', position: Position.Left }),
        React.createElement(Handle, { type: 'source', position: Position.Right }),
        React.createElement('div', { className: 'table-header' },
            React.createElement('span', null, '📋'),
            data.label
        ),
        React.createElement('div', { className: 'table-body' },
            data.columns && data.columns.map((col, idx) => {
                let icon = '•';
                let iconClass = '';
                if (col.isPrimary) {
                    icon = '🔑';
                    iconClass = 'pk-icon';
                } else if (col.isForeign) {
                    icon = '🔗';
                    iconClass = 'fk-icon';
                }
                
                return React.createElement('div', { key: idx, className: 'table-field' },
                    React.createElement('div', { className: 'field-left' },
                        React.createElement('span', { className: `field-icon ${iconClass}` }, icon),
                        React.createElement('span', { className: 'field-name' }, col.name)
                    ),
                    React.createElement('span', { className: 'field-type' }, col.type)
                );
            })
        )
    );
}

const nodeTypes = { table: TableNode };

function SchemaFlow() {
    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        fetch('/api/schema')
            .then(res => res.json())
            .then(data => {
                if (data.success && data.data) {
                    const rawNodes = data.data.nodes || [];
                    const rawEdges = data.data.edges || [];
                    
                    if (rawNodes.length === 0) {
                        document.getElementById('root').innerHTML = '<div class="loading"><div>⚠️ No tables found in database</div></div>';
                        return;
                    }
                    
                    document.getElementById('stats').textContent = 
                        `${rawNodes.length} tables • ${rawEdges.length} relationships`;
                    
                    const layoutedNodes = getLayoutedElements(rawNodes, rawEdges);
                    
                    const formattedNodes = layoutedNodes.map(node => ({
                        id: node.id,
                        type: 'table',
                        position: node.position,
                        data: node.data,
                        draggable: true,
                        sourcePosition: Position.Right,
                        targetPosition: Position.Left
                    }));
                    
                    const formattedEdges = rawEdges.map((edge, idx) => ({
                        id: edge.id,
                        source: edge.source,
                        target: edge.target,
                        label: edge.label || '',
                        type: 'smoothstep',
                        animated: false,
                        markerEnd: {
                            type: MarkerType.ArrowClosed,
                            color: '#6897bb'
                        },
                        style: { 
                            stroke: '#6897bb', 
                            strokeWidth: 2
                        }
                    }));
                    
                    setNodes(formattedNodes);
                    setEdges(formattedEdges);
                    setLoading(false);
                } else {
                    document.getElementById('root').innerHTML = 
                        `<div class="loading"><div>❌ ${data.message || 'Failed to load schema'}</div></div>`;
                }
            })
            .catch(err => {
                document.getElementById('root').innerHTML = 
                    `<div class="loading"><div>❌ Error: ${err.message}</div></div>`;
            });
    }, []);

    if (loading) {
        return null;
    }

    return React.createElement(ReactFlow, {
        nodes,
        edges,
        onNodesChange,
        onEdgesChange,
        nodeTypes,
        nodesDraggable: true,
        nodesConnectable: false,
        elementsSelectable: true,
        fitView: true,
        minZoom: 0.05,
        maxZoom: 3,
        zoomOnScroll: true,
        zoomOnPinch: true,
        zoomOnDoubleClick: true,
        panOnScroll: false,
        defaultViewport: { x: 0, y: 0, zoom: 0.6 },
        style: { background: '#2b2b2b' }
    },
        React.createElement(Background, { color: '#3a3a3a', gap: 20, size: 1 }),
        React.createElement(Controls, { 
            style: { 
                background: '#3c3f41', 
                border: '1px solid #555'
            } 
        }),
        React.createElement(MiniMap, { 
            nodeColor: '#5a5d5f',
            nodeStrokeColor: '#6897bb',
            nodeStrokeWidth: 2,
            maskColor: 'rgba(104, 151, 187, 0.2)',
            style: { 
                background: '#3c3f41', 
                border: '1px solid #555'
            },
            zoomable: true,
            pannable: true
        })
    );
}

function getLayoutedElements(nodes, edges) {
    const visited = new Set();
    const components = [];
    const adj = new Map();
    
    nodes.forEach(n => adj.set(n.id, []));
    edges.forEach(e => {
        if (adj.has(e.source)) adj.get(e.source).push(e.target);
        if (adj.has(e.target)) adj.get(e.target).push(e.source);
    });
    
    nodes.forEach(node => {
        if (visited.has(node.id)) return;
        
        const component = { nodes: [], edges: [] };
        const queue = [node.id];
        
        while (queue.length > 0) {
            const id = queue.shift();
            if (visited.has(id)) continue;
            
            visited.add(id);
            component.nodes.push(nodes.find(n => n.id === id));
            
            adj.get(id).forEach(connId => {
                if (!visited.has(connId)) queue.push(connId);
            });
        }
        
        component.edges = edges.filter(e => 
            component.nodes.some(n => n.id === e.source) &&
            component.nodes.some(n => n.id === e.target)
        );
        
        components.push(component);
    });
    
    components.sort((a, b) => {
        const aHasEdges = a.edges.length > 0 ? 1 : 0;
        const bHasEdges = b.edges.length > 0 ? 1 : 0;
        if (aHasEdges !== bHasEdges) return bHasEdges - aHasEdges;
        return b.nodes.length - a.nodes.length;
    });
    
    const layoutedComponents = [];
    let globalOffsetX = 0;
    let globalOffsetY = 0;
    let rowMaxY = 0;
    const componentsPerRow = 3;
    
    components.forEach((comp, idx) => {
        if (comp.edges.length === 0 && comp.nodes.length === 1) {
            const node = comp.nodes[0];
            layoutedComponents.push({
                ...node,
                position: {
                    x: globalOffsetX,
                    y: globalOffsetY
                }
            });
            
            if ((idx + 1) % componentsPerRow === 0) {
                globalOffsetX = 0;
                globalOffsetY = rowMaxY + 400;
                rowMaxY = 0;
            } else {
                globalOffsetX += 400;
                rowMaxY = Math.max(rowMaxY, globalOffsetY + 200);
            }
            return;
        }
        
        const dagreGraph = new dagre.graphlib.Graph();
        dagreGraph.setDefaultEdgeLabel(() => ({}));
        dagreGraph.setGraph({ 
            rankdir: 'LR',
            nodesep: 250,
            ranksep: 400,
            marginx: 50,
            marginy: 50
        });

        comp.nodes.forEach((node) => {
            dagreGraph.setNode(node.id, { width: 250, height: 200 });
        });

        comp.edges.forEach((edge) => {
            dagreGraph.setEdge(edge.source, edge.target);
        });

        dagre.layout(dagreGraph);

        const layouted = comp.nodes.map((node) => {
            const nodeWithPosition = dagreGraph.node(node.id);
            return {
                ...node,
                position: {
                    x: globalOffsetX + nodeWithPosition.x - 125,
                    y: globalOffsetY + nodeWithPosition.y - 100,
                },
            };
        });
        
        layoutedComponents.push(...layouted);
        
        const maxX = Math.max(...layouted.map(n => n.position.x));
        const compMaxY = Math.max(...layouted.map(n => n.position.y));
        rowMaxY = Math.max(rowMaxY, compMaxY);
        
        if ((idx + 1) % componentsPerRow === 0) {
            globalOffsetX = 0;
            globalOffsetY = rowMaxY + 400;
            rowMaxY = 0;
        } else {
            globalOffsetX = maxX + 500;
        }
    });
    
    return layoutedComponents;
}

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(React.createElement(SchemaFlow));
