import React from 'react';
import ReactDOM from 'react-dom/client';
import { ReactFlow, Background, Controls, MiniMap, MarkerType, useNodesState, useEdgesState, Handle, Position } from '@xyflow/react';
import dagre from 'dagre';

const { useState, useEffect, useCallback, memo } = React;

function TableNode({ data }) {
    const isHighlighted = data.isHighlighted;
    const isDimmed = data.isDimmed;
    const hasRelations = data.hasRelations || false;
    const highlightedFields = data.highlightedFields || new Set();
    
    const nodeClass = `table-node-wrapper ${isHighlighted ? 'highlighted' : ''} ${isDimmed ? 'dimmed' : ''} ${!hasRelations ? 'no-relations' : ''}`;
    
    return React.createElement('div', { 
        className: nodeClass,
        style: { minWidth: '220px', cursor: 'pointer' }, 
        onClick: () => window.openTableEdit(data.label)
    },
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
                
                const isFieldHighlighted = highlightedFields.has(col.name);
                const fieldClass = `table-field ${isFieldHighlighted ? 'field-highlighted' : ''}`;
                
                return React.createElement('div', { 
                    key: idx, 
                    className: fieldClass,
                    style: { position: 'relative' }
                },
                    col.isPrimary && React.createElement(Handle, {
                        type: 'target',
                        position: Position.Left,
                        id: col.name,
                        className: 'field-handle',
                        style: { top: '50%', left: '-4px' }
                    }),
                    col.isForeign && React.createElement(Handle, {
                        type: 'source',
                        position: Position.Right,
                        id: col.name,
                        className: 'field-handle',
                        style: { top: '50%', right: '-4px' }
                    }),
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

const MemoizedTableNode = memo(TableNode);
const nodeTypes = { table: MemoizedTableNode };

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
                    
                    const nodeRelations = new Map();
                    rawEdges.forEach(edge => {
                        nodeRelations.set(edge.source, true);
                        nodeRelations.set(edge.target, true);
                    });
                    
                    const formattedNodes = layoutedNodes.map(node => ({
                        id: node.id,
                        type: 'table',
                        position: node.position,
                        data: { 
                            ...node.data, 
                            isHighlighted: false, 
                            isDimmed: false,
                            hasRelations: nodeRelations.has(node.id) || false,
                            highlightedFields: new Set()
                        },
                        draggable: true,
                        sourcePosition: Position.Right,
                        targetPosition: Position.Left
                    }));
                    
                    const formattedEdges = rawEdges.map((edge, idx) => ({
                        id: edge.id,
                        source: edge.source,
                        target: edge.target,
                        sourceHandle: edge.sourceHandle || null,
                        targetHandle: edge.targetHandle || null,
                        label: edge.label || '',
                        data: { originalLabel: edge.label || '' },
                        labelStyle: { fill: '#6897bb', opacity: 1 },
                        labelBgStyle: { fill: '#2b2b2b', opacity: 0.8 },
                        type: 'smoothstep',
                        animated: false,
                        markerEnd: {
                            type: MarkerType.ArrowClosed,
                            color: '#6897bb'
                        },
                        style: { 
                            stroke: '#6897bb', 
                            strokeWidth: 2,
                            strokeDasharray: '5 5'
                        },
                        className: 'custom-edge'
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

    const handleNodeMouseEnter = useCallback((event, node) => {
        if (!node.data.hasRelations) return;
        
        const connectedEdges = edges.filter(edge => 
            edge.source === node.id || edge.target === node.id
        );
        
        if (connectedEdges.length === 0) return;
        
        const connectedNodeIds = new Set();
        const nodeFieldMap = new Map(); 
        
        connectedEdges.forEach(edge => {
            connectedNodeIds.add(edge.source);
            connectedNodeIds.add(edge.target);
            
            if (edge.sourceHandle) {
                if (!nodeFieldMap.has(edge.source)) {
                    nodeFieldMap.set(edge.source, new Set());
                }
                nodeFieldMap.get(edge.source).add(edge.sourceHandle);
            }
            if (edge.targetHandle) {
                if (!nodeFieldMap.has(edge.target)) {
                    nodeFieldMap.set(edge.target, new Set());
                }
                nodeFieldMap.get(edge.target).add(edge.targetHandle);
            }
        });
        
        setNodes(nds => nds.map(n => {
            const isConnected = connectedNodeIds.has(n.id);
            const shouldDim = n.data.hasRelations && !isConnected;
            
            return {
                ...n,
                data: {
                    ...n.data,
                    isHighlighted: isConnected,
                    isDimmed: shouldDim,
                    highlightedFields: nodeFieldMap.get(n.id) || new Set()
                }
            };
        }));
        
        setEdges(eds => eds.map(edge => {
            const isConnected = edge.source === node.id || edge.target === node.id;
            return {
                ...edge,
                animated: isConnected,
                label: isConnected ? edge.label : '',
                labelStyle: {
                    fill: isConnected ? '#4a9eff' : '#6897bb',
                    opacity: isConnected ? 1 : 0.3
                },
                labelBgStyle: {
                    fill: isConnected ? '#1a3a5a' : '#2b2b2b',
                    opacity: isConnected ? 1 : 0.3
                },
                style: {
                    ...edge.style,
                    stroke: isConnected ? '#4a9eff' : '#3a3a3a',
                    strokeWidth: isConnected ? 3 : 2,
                    opacity: isConnected ? 1 : 0.3
                },
                markerEnd: {
                    ...edge.markerEnd,
                    color: isConnected ? '#4a9eff' : '#3a3a3a'
                }
            };
        }));
    }, [edges, setNodes, setEdges]);

    const handleNodeMouseLeave = useCallback(() => {
        // Reset all nodes
        setNodes(nds => nds.map(n => ({
            ...n,
            data: {
                ...n.data,
                isHighlighted: false,
                isDimmed: false,
                highlightedFields: new Set()
            }
        })));
        
        // Reset all edges
        setEdges(eds => eds.map(edge => {
            const originalLabel = edge.data?.originalLabel || edge.label;
            return {
                ...edge,
                animated: false,
                label: originalLabel,
                labelStyle: { fill: '#6897bb', opacity: 1 },
                labelBgStyle: { fill: '#2b2b2b', opacity: 0.8 },
                style: {
                    ...edge.style,
                    stroke: '#6897bb',
                    strokeWidth: 2,
                    opacity: 1
                },
                markerEnd: {
                    ...edge.markerEnd,
                    color: '#6897bb'
                }
            };
        }));
    }, [setNodes, setEdges]);

    // Handle edge hover
    const handleEdgeMouseEnter = useCallback((event, edge) => {
        const connectedNodeIds = new Set([edge.source, edge.target]);
        const nodeFieldMap = new Map();
        if (edge.sourceHandle) {
            nodeFieldMap.set(edge.source, new Set([edge.sourceHandle]));
        }
        if (edge.targetHandle) {
            nodeFieldMap.set(edge.target, new Set([edge.targetHandle]));
        }
        
        setNodes(nds => nds.map(n => {
            const isConnected = connectedNodeIds.has(n.id);
            const shouldDim = n.data.hasRelations && !isConnected;
            
            return {
                ...n,
                data: {
                    ...n.data,
                    isHighlighted: isConnected,
                    isDimmed: shouldDim,
                    highlightedFields: nodeFieldMap.get(n.id) || new Set()
                }
            };
        }));
        
        setEdges(eds => eds.map(e => {
            const isHovered = e.id === edge.id;
            return {
                ...e,
                animated: isHovered,
                label: isHovered ? e.label : '',
                labelStyle: {
                    fill: isHovered ? '#4a9eff' : '#6897bb',
                    opacity: isHovered ? 1 : 0.3
                },
                labelBgStyle: {
                    fill: isHovered ? '#1a3a5a' : '#2b2b2b',
                    opacity: isHovered ? 0.3 : 0.1
                },
                style: {
                    ...e.style,
                    stroke: isHovered ? '#4a9eff' : '#3a3a3a',
                    strokeWidth: isHovered ? 3 : 2,
                    opacity: isHovered ? 1 : 0.3
                },
                markerEnd: {
                    ...e.markerEnd,
                    color: isHovered ? '#4a9eff' : '#3a3a3a'
                }
            };
        }));
    }, [setNodes, setEdges]);

    const handleEdgeMouseLeave = useCallback(() => {
        setNodes(nds => nds.map(n => ({
            ...n,
            data: {
                ...n.data,
                isHighlighted: false,
                isDimmed: false,
                highlightedFields: new Set()
            }
        })));
        
        setEdges(eds => eds.map(edge => {
            const originalLabel = edge.data?.originalLabel || edge.label;
            return {
                ...edge,
                animated: false,
                label: originalLabel,
                labelStyle: { fill: '#6897bb', opacity: 1 },
                labelBgStyle: { fill: '#2b2b2b', opacity: 0.8 },
                style: {
                    ...edge.style,
                    stroke: '#6897bb',
                    strokeWidth: 2,
                    opacity: 1
                },
                markerEnd: {
                    ...edge.markerEnd,
                    color: '#6897bb'
                }
            };
        }));
    }, [setNodes, setEdges]);

    if (loading) {
        return null;
    }

    return React.createElement(ReactFlow, {
        nodes,
        edges,
        onNodesChange,
        onEdgesChange,
        onNodeMouseEnter: handleNodeMouseEnter,
        onNodeMouseLeave: handleNodeMouseLeave,
        onEdgeMouseEnter: handleEdgeMouseEnter,
        onEdgeMouseLeave: handleEdgeMouseLeave,
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
        preventScrolling: false,
        nodesFocusable: false,
        edgesFocusable: false,
        autoPanOnNodeDrag: true,
        defaultViewport: { x: 0, y: 0, zoom: 0.6 },
        style: { background: '#2b2b2b' },
        proOptions: { hideAttribution: true }
    },
        React.createElement(Background, { color: '#555555', gap: 20, size: 2.5 }),
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
