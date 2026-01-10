// Pure Canvas Schema Visualizer - No dependencies
class SchemaCanvas {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        this.canvas = document.createElement('canvas');
        this.ctx = this.canvas.getContext('2d', { alpha: false });

        // State
        this.nodes = [];
        this.edges = [];
        this.scale = 0.6;
        this.offset = { x: 0, y: 0 };
        this.hoveredNode = null;
        this.hoveredEdge = null;
        this.draggedNode = null;
        this.isPanning = false;
        this.lastMouse = { x: 0, y: 0 };
        this.dragStartPos = null;
        this.hasDragged = false;
        this.highlightedNodes = new Set();
        this.highlightedFields = new Map();
        this.dashOffset = 0;

        this.init();
    }

    init() {
        this.setupCanvas();
        this.setupEvents();
        this.loadSchema();
        this.startAnimation();
    }

    startAnimation() {
        setInterval(() => {
            if (this.hoveredNode) {
                this.dashOffset += 0.5;
                this.render();
            }
        }, 30);
    }

    setupCanvas() {
        const dpr = window.devicePixelRatio || 1;
        this.canvas.width = this.container.clientWidth * dpr;
        this.canvas.height = this.container.clientHeight * dpr;
        this.canvas.style.width = '100%';
        this.canvas.style.height = '100%';
        this.canvas.style.background = '#1a1a1a';
        this.container.appendChild(this.canvas);

        this.ctx.scale(dpr, dpr);
        this.dpr = dpr;
    }

    async loadSchema() {
        try {
            // Show loading with progress
            this.showMessage('⏳ Loading schema...');

            const res = await fetch('/api/schema');
            const data = await res.json();

            if (data.success && data.data) {
                const rawNodes = data.data.nodes || [];
                const rawEdges = data.data.edges || [];

                if (rawNodes.length === 0) {
                    this.showMessage('⚠️ No tables found in database');
                    return;
                }

                // Show processing message for large schemas
                if (rawNodes.length > 20) {
                    this.showMessage(`📊 Processing ${rawNodes.length} tables...`);
                    // Small delay to show the message
                    await new Promise(resolve => setTimeout(resolve, 100));
                }

                document.getElementById('stats').textContent =
                    `${rawNodes.length} tables • ${rawEdges.length} relationships`;

                this.nodes = this.layoutNodes(rawNodes, rawEdges);
                this.edges = rawEdges;

                // Hide processing message and show schema
                this.hideMessage();

                this.fitView();
                this.render();
            } else {
                this.showMessage(`❌ ${data.message || 'Failed to load schema'}`);
            }
        } catch (err) {
            this.showMessage(`❌ Error: ${err.message}`);
        }
    }

    layoutNodes(nodes, edges) {
        // Build adjacency list
        const adj = new Map();
        nodes.forEach(n => adj.set(n.id, []));
        edges.forEach(e => {
            if (adj.has(e.source)) adj.get(e.source).push(e.target);
            if (adj.has(e.target)) adj.get(e.target).push(e.source);
        });

        // Find connected components
        const visited = new Set();
        const components = [];

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

        // Separate connected and unconnected
        const connected = components.filter(c => c.edges.length > 0);
        const unconnected = components.filter(c => c.edges.length === 0);

        // Sort connected by size
        connected.sort((a, b) => b.nodes.length - a.nodes.length);

        const layouted = [];
        let globalX = 0, globalY = 0;

        // Layout connected components
        connected.forEach((comp, idx) => {
            const positioned = this.layoutComponent(comp);
            positioned.forEach(n => {
                layouted.push({
                    ...n,
                    position: {
                        x: globalX + n.position.x,
                        y: globalY + n.position.y
                    },
                    hasRelations: true
                });
            });

            const maxX = Math.max(...positioned.map(n => n.position.x));
            const maxY = Math.max(...positioned.map(n => n.position.y));

            // Increased spacing between components
            globalX = maxX + 800;
            if (globalX > 3500) {
                globalX = 0;
                globalY += maxY + 600;
            }
        });

        // Layout unconnected in grid at bottom with better spacing
        if (unconnected.length > 0) {
            globalY = layouted.length > 0 ? Math.max(...layouted.map(n => n.position.y)) + 700 : 0;
            globalX = 0;

            const cols = Math.ceil(Math.sqrt(unconnected.length));
            const colSpacing = 500; // Increased column spacing
            const rowSpacing = 400; // Increased row spacing

            unconnected.forEach((comp, idx) => {
                const node = comp.nodes[0];
                layouted.push({
                    ...node,
                    position: {
                        x: (idx % cols) * colSpacing,
                        y: globalY + Math.floor(idx / cols) * rowSpacing
                    },
                    hasRelations: false
                });
            });
        }

        return layouted;
    }

    layoutComponent(comp) {
        // Simple hierarchical layout with better spacing to prevent overlaps
        const levels = new Map();
        const visited = new Set();

        // Find root (node with most connections)
        const root = comp.nodes.reduce((max, n) => {
            const connections = comp.edges.filter(e =>
                e.source === n.id || e.target === n.id
            ).length;
            return connections > (max.connections || 0) ?
                { node: n, connections } : max;
        }, {}).node;

        // BFS to assign levels
        const queue = [{ node: root, level: 0 }];
        visited.add(root.id);

        while (queue.length > 0) {
            const { node, level } = queue.shift();

            if (!levels.has(level)) levels.set(level, []);
            levels.get(level).push(node);

            comp.edges.forEach(e => {
                const nextId = e.source === node.id ? e.target :
                    e.target === node.id ? e.source : null;
                if (nextId && !visited.has(nextId)) {
                    visited.add(nextId);
                    const nextNode = comp.nodes.find(n => n.id === nextId);
                    queue.push({ node: nextNode, level: level + 1 });
                }
            });
        }

        // Position nodes with adaptive spacing to prevent overlaps
        const positioned = [];
        const levelSpacing = 550; // Increased horizontal spacing

        levels.forEach((nodes, level) => {
            // Calculate max node height in this level
            const maxNodeHeight = Math.max(...nodes.map(n => {
                const colCount = n.data.columns?.length || 0;
                return 36 + colCount * 28 + 4;
            }));

            // Use adaptive vertical spacing based on max height
            const nodeSpacing = Math.max(maxNodeHeight + 80, 400);
            const totalHeight = (nodes.length - 1) * nodeSpacing;
            const startY = -totalHeight / 2;

            nodes.forEach((node, idx) => {
                positioned.push({
                    ...node,
                    position: {
                        x: level * levelSpacing,
                        y: startY + idx * nodeSpacing
                    }
                });
            });
        });

        return positioned;
    }

    fitView() {
        if (this.nodes.length === 0) return;

        const padding = 100;
        const minX = Math.min(...this.nodes.map(n => n.position.x)) - padding;
        const minY = Math.min(...this.nodes.map(n => n.position.y)) - padding;
        const maxX = Math.max(...this.nodes.map(n => n.position.x + 250)) + padding;
        const maxY = Math.max(...this.nodes.map(n => n.position.y + 200)) + padding;

        const width = maxX - minX;
        const height = maxY - minY;

        const scaleX = (this.canvas.width / this.dpr) / width;
        const scaleY = (this.canvas.height / this.dpr) / height;
        this.scale = Math.min(scaleX, scaleY, 1) * 0.9;

        this.offset.x = -minX + ((this.canvas.width / this.dpr) / this.scale - width) / 2;
        this.offset.y = -minY + ((this.canvas.height / this.dpr) / this.scale - height) / 2;
    }

    render() {
        const w = this.canvas.width / this.dpr;
        const h = this.canvas.height / this.dpr;

        // Clear
        this.ctx.fillStyle = '#1a1a1a';
        this.ctx.fillRect(0, 0, w, h);

        // Transform
        this.ctx.save();
        this.ctx.translate(this.offset.x * this.scale, this.offset.y * this.scale);
        this.ctx.scale(this.scale, this.scale);

        // Draw grid pattern
        this.drawGrid();

        // Draw edges first
        this.edges.forEach(e => this.drawEdge(e));

        // Draw nodes
        this.nodes.forEach(n => this.drawNode(n));

        this.ctx.restore();
    }

    drawGrid() {
        if (this.scale < 0.3) return; // Don't draw grid when zoomed out too much

        const gridSize = 50;
        const opacity = Math.min(this.scale, 0.3);

        // Get visible area in world coordinates
        const startX = Math.floor(-this.offset.x / gridSize) * gridSize;
        const startY = Math.floor(-this.offset.y / gridSize) * gridSize;
        const endX = startX + (this.canvas.width / this.dpr / this.scale) + gridSize;
        const endY = startY + (this.canvas.height / this.dpr / this.scale) + gridSize;

        this.ctx.strokeStyle = `rgba(255, 255, 255, ${opacity * 0.1})`;
        this.ctx.lineWidth = 1 / this.scale;

        // Draw vertical lines
        for (let x = startX; x < endX; x += gridSize) {
            this.ctx.beginPath();
            this.ctx.moveTo(x, startY);
            this.ctx.lineTo(x, endY);
            this.ctx.stroke();
        }

        // Draw horizontal lines
        for (let y = startY; y < endY; y += gridSize) {
            this.ctx.beginPath();
            this.ctx.moveTo(startX, y);
            this.ctx.lineTo(endX, y);
            this.ctx.stroke();
        }

        // Draw stronger grid lines every 5 units
        this.ctx.strokeStyle = `rgba(255, 255, 255, ${opacity * 0.15})`;
        this.ctx.lineWidth = 1.5 / this.scale;

        for (let x = startX; x < endX; x += gridSize * 5) {
            this.ctx.beginPath();
            this.ctx.moveTo(x, startY);
            this.ctx.lineTo(x, endY);
            this.ctx.stroke();
        }

        for (let y = startY; y < endY; y += gridSize * 5) {
            this.ctx.beginPath();
            this.ctx.moveTo(startX, y);
            this.ctx.lineTo(endX, y);
            this.ctx.stroke();
        }
    }

    drawNode(node) {
        const { x, y } = node.position;
        const w = 280; // Increased from 220 to prevent text overlap
        const colCount = node.data.columns?.length || 0;
        const h = 36 + colCount * 28 + 4;

        const isHovered = this.hoveredNode === node;
        // Also highlight when hovered directly (for tables without relations)
        const isHighlighted = this.highlightedNodes.has(node.id) || isHovered;
        // Only dim tables with relations when another table is hovered
        const isDimmed = this.hoveredNode && !isHighlighted && node.hasRelations;

        // Find fields with relationships
        const fieldsWithRelations = new Set();
        this.edges.forEach(e => {
            if (e.source === node.id && e.sourceHandle) {
                fieldsWithRelations.add(e.sourceHandle);
            }
            if (e.target === node.id && e.targetHandle) {
                fieldsWithRelations.add(e.targetHandle);
            }
        });

        // Shadow
        if (!isDimmed) {
            this.ctx.shadowColor = 'rgba(0,0,0,0.4)';
            this.ctx.shadowBlur = 8;
            this.ctx.shadowOffsetY = 2;
        }

        // Background with gradient for depth
        const gradient = this.ctx.createLinearGradient(x, y, x, y + h);
        if (isHovered) {
            gradient.addColorStop(0, '#3a3f41');
            gradient.addColorStop(1, '#2d3538');
        } else if (isHighlighted) {
            gradient.addColorStop(0, '#2d3a4a');
            gradient.addColorStop(1, '#243044');
        } else if (isDimmed) {
            gradient.addColorStop(0, '#1a1a1a');
            gradient.addColorStop(1, '#151515');
        } else {
            gradient.addColorStop(0, '#2b2b2b');
            gradient.addColorStop(1, '#252525');
        }
        this.ctx.fillStyle = gradient;
        this.ctx.fillRect(x, y, w, h);

        // Border with glow effect for highlighted
        if (isHighlighted) {
            this.ctx.shadowColor = '#4a9eff';
            this.ctx.shadowBlur = 12;
            this.ctx.strokeStyle = '#4a9eff';
            this.ctx.lineWidth = 2;
        } else {
            this.ctx.strokeStyle = '#555';
            this.ctx.lineWidth = 1;
        }
        this.ctx.strokeRect(x, y, w, h);

        this.ctx.shadowBlur = 0;
        this.ctx.shadowOffsetY = 0;

        // Header
        this.ctx.fillStyle = '#1e1e1e';
        this.ctx.fillRect(x, y, w, 36);

        // Table name - truncate if too long
        // Use bright color when hovered or highlighted
        this.ctx.fillStyle = isDimmed ? '#666' : (isHovered || isHighlighted) ? '#fff' : '#e0e0e0';
        this.ctx.font = 'bold 14px Inter, sans-serif';
        this.ctx.textBaseline = 'middle';
        this.ctx.fillText('📋', x + 8, y + 18);
        const truncatedTableName = this.truncateText(node.data.label, w - 50);
        this.ctx.fillText(truncatedTableName, x + 30, y + 18);

        // Columns
        const highlightedFields = this.highlightedFields.get(node.id) || new Set();
        const maxNameWidth = w * 0.55; // 55% for column name, 45% for type
        const maxTypeWidth = w * 0.35; // Leave some padding

        node.data.columns?.forEach((col, i) => {
            const cy = y + 36 + i * 28 + 2;
            const isFieldHighlighted = highlightedFields.has(col.name);
            const hasRelation = fieldsWithRelations.has(col.name);

            // Subtle background for fields with relationships
            if (hasRelation) {
                this.ctx.fillStyle = 'rgba(74, 158, 255, 0.08)';
                this.ctx.fillRect(x + 2, cy, w - 4, 26);
            }

            // Field background on hover
            if (isFieldHighlighted) {
                this.ctx.fillStyle = 'rgba(74, 158, 255, 0.25)';
                this.ctx.fillRect(x + 2, cy, w - 4, 26);
            }

            // Icon
            const icon = col.isPrimary ? '🔑' : col.isForeign ? '🔗' : '•';
            this.ctx.fillStyle = isDimmed ? '#444' :
                col.isPrimary ? '#ffd700' :
                    col.isForeign ? '#4a9eff' :
                        (isHovered || isHighlighted) ? '#aaa' : '#888';
            this.ctx.font = '12px Inter, sans-serif';
            this.ctx.fillText(icon, x + 8, cy + 13);

            // Column name - truncated to prevent overlap
            // Brighter text when hovered or highlighted
            this.ctx.fillStyle = isDimmed ? '#555' :
                isFieldHighlighted ? '#4a9eff' :
                    (isHovered || isHighlighted) ? '#fff' : '#e0e0e0';
            const truncatedName = this.truncateText(col.name, maxNameWidth);
            this.ctx.fillText(truncatedName, x + 28, cy + 13);

            // Type - truncated and right-aligned
            this.ctx.fillStyle = isDimmed ? '#444' : (isHovered || isHighlighted) ? '#bbb' : '#888';
            this.ctx.textAlign = 'right';
            const truncatedType = this.truncateText(col.type, maxTypeWidth);
            this.ctx.fillText(truncatedType, x + w - 10, cy + 13);
            this.ctx.textAlign = 'left';
        });
    }

    drawEdge(edge) {
        const src = this.nodes.find(n => n.id === edge.source);
        const tgt = this.nodes.find(n => n.id === edge.target);
        if (!src || !tgt) return;

        const isHighlighted = this.hoveredEdge === edge ||
            (this.hoveredNode &&
                (this.hoveredNode.id === edge.source ||
                    this.hoveredNode.id === edge.target));
        const isDimmed = this.hoveredNode && !isHighlighted;

        // Find field positions
        let srcY = src.position.y + 18;
        let tgtY = tgt.position.y + 18;

        if (edge.sourceHandle) {
            const srcCol = src.data.columns?.findIndex(c => c.name === edge.sourceHandle);
            if (srcCol >= 0) srcY = src.position.y + 36 + srcCol * 28 + 15;
        }

        if (edge.targetHandle) {
            const tgtCol = tgt.data.columns?.findIndex(c => c.name === edge.targetHandle);
            if (tgtCol >= 0) tgtY = tgt.position.y + 36 + tgtCol * 28 + 15;
        }

        // Start from right edge of source, end at left edge of target
        const srcX = src.position.x + 280; // Match new node width
        const tgtX = tgt.position.x;

        // Calculate node heights
        const srcH = 36 + (src.data.columns?.length || 0) * 28 + 4;
        const tgtH = 36 + (tgt.data.columns?.length || 0) * 28 + 4;

        // Draw line with glow effect for highlighted
        if (isHighlighted) {
            // Draw glow
            this.ctx.shadowColor = '#4a9eff';
            this.ctx.shadowBlur = 15;
            this.ctx.strokeStyle = '#4a9eff';
            this.ctx.lineWidth = 3;
        } else {
            this.ctx.shadowBlur = 0;
            this.ctx.strokeStyle = isDimmed ? '#3a3a3a' : '#6897bb';
            this.ctx.lineWidth = isDimmed ? 1.5 : 2;
        }

        this.ctx.globalAlpha = isDimmed ? 0.3 : 1;

        // Animated dashes on hover with smoother animation
        if (isHighlighted) {
            this.ctx.setLineDash([10, 6]);
            this.ctx.lineDashOffset = -this.dashOffset;
        } else {
            this.ctx.setLineDash([6, 4]);
        }

        this.ctx.beginPath();
        this.ctx.moveTo(srcX, srcY);

        // Calculate control points to route around nodes
        const dx = tgtX - srcX;
        const dy = tgtY - srcY;

        // Extend control points beyond node boundaries
        const controlDist = Math.max(Math.abs(dx) * 0.4, 80);
        const cp1x = srcX + controlDist;
        const cp2x = tgtX - controlDist;

        // Adjust Y control points to avoid overlapping nodes
        let cp1y = srcY;
        let cp2y = tgtY;

        // If line goes through source node, route around it
        if (cp1x < src.position.x + 220 + 50) {
            if (tgtY < src.position.y) {
                cp1y = src.position.y - 20;
            } else if (tgtY > src.position.y + srcH) {
                cp1y = src.position.y + srcH + 20;
            }
        }

        // If line goes through target node, route around it
        if (cp2x > tgt.position.x - 50) {
            if (srcY < tgt.position.y) {
                cp2y = tgt.position.y - 20;
            } else if (srcY > tgt.position.y + tgtH) {
                cp2y = tgt.position.y + tgtH + 20;
            }
        }

        this.ctx.bezierCurveTo(cp1x, cp1y, cp2x, cp2y, tgtX, tgtY);
        this.ctx.stroke();

        // Reset shadow for arrow
        this.ctx.shadowBlur = 0;

        // Arrow
        const angle = Math.atan2(cp2y - tgtY, cp2x - tgtX) + Math.PI;
        this.ctx.setLineDash([]);
        this.ctx.beginPath();
        this.ctx.moveTo(tgtX, tgtY);
        this.ctx.lineTo(
            tgtX + 10 * Math.cos(angle - Math.PI / 6),
            tgtY + 10 * Math.sin(angle - Math.PI / 6)
        );
        this.ctx.lineTo(
            tgtX + 10 * Math.cos(angle + Math.PI / 6),
            tgtY + 10 * Math.sin(angle + Math.PI / 6)
        );
        this.ctx.closePath();
        this.ctx.fillStyle = this.ctx.strokeStyle;
        this.ctx.fill();

        // Label
        if (isHighlighted && edge.label) {
            const midX = (srcX + tgtX) / 2;
            const midY = (srcY + tgtY) / 2;

            this.ctx.fillStyle = '#1a3a5a';
            this.ctx.globalAlpha = 0.9;
            const labelW = this.ctx.measureText(edge.label).width + 12;
            const labelX = midX - labelW / 2;
            const labelY = midY - 10;
            this.ctx.fillRect(labelX, labelY, labelW, 20);

            this.ctx.fillStyle = '#4a9eff';
            this.ctx.globalAlpha = 1;
            this.ctx.font = '11px Inter, sans-serif';
            this.ctx.textAlign = 'center';
            this.ctx.fillText(edge.label, midX, labelY + 13);
            this.ctx.textAlign = 'left';
        }

        this.ctx.globalAlpha = 1;
        this.ctx.setLineDash([]);
    }

    setupEvents() {
        // Search
        const searchInput = document.getElementById('table-search');
        if (searchInput) {
            searchInput.addEventListener('input', e => {
                const query = e.target.value.toLowerCase();
                if (!query) return;

                const node = this.nodes.find(n =>
                    n.data.label.toLowerCase().includes(query)
                );

                if (node) {
                    this.focusNode(node);
                }
            });
        }

        // Context menu
        this.canvas.addEventListener('contextmenu', e => {
            e.preventDefault();
            const menu = document.getElementById('context-menu');
            menu.style.display = 'block';
            menu.style.left = e.clientX + 'px';
            menu.style.top = e.clientY + 'px';
        });

        document.addEventListener('click', () => {
            document.getElementById('context-menu').style.display = 'none';
        });

        // Mouse move
        this.canvas.addEventListener('mousemove', e => {
            const rect = this.canvas.getBoundingClientRect();
            const x = (e.clientX - rect.left) / this.scale - this.offset.x;
            const y = (e.clientY - rect.top) / this.scale - this.offset.y;

            if (this.draggedNode) {
                this.draggedNode.position.x = x - this.dragOffset.x;
                this.draggedNode.position.y = y - this.dragOffset.y;

                // Track if actually dragged
                if (this.dragStartPos) {
                    const dx = Math.abs(this.draggedNode.position.x - this.dragStartPos.x);
                    const dy = Math.abs(this.draggedNode.position.y - this.dragStartPos.y);
                    if (dx > 5 || dy > 5) this.hasDragged = true;
                }

                this.render();
                return;
            }

            if (this.isPanning) {
                this.offset.x += (e.clientX - this.lastMouse.x) / this.scale;
                this.offset.y += (e.clientY - this.lastMouse.y) / this.scale;
                this.lastMouse = { x: e.clientX, y: e.clientY };
                this.render();
                return;
            }

            // Check hover
            const prevHovered = this.hoveredNode;
            const prevHoveredEdge = this.hoveredEdge;

            this.hoveredNode = this.nodes.find(n => {
                const nodeH = 36 + (n.data.columns?.length || 0) * 28 + 4;
                return x >= n.position.x && x <= n.position.x + 280 &&
                    y >= n.position.y && y <= n.position.y + nodeH;
            });

            // Check edge hover if no node hovered
            if (!this.hoveredNode) {
                this.hoveredEdge = this.edges.find(e => {
                    const src = this.nodes.find(n => n.id === e.source);
                    const tgt = this.nodes.find(n => n.id === e.target);
                    if (!src || !tgt) return false;

                    let srcY = src.position.y + 18;
                    let tgtY = tgt.position.y + 18;

                    if (e.sourceHandle) {
                        const srcCol = src.data.columns?.findIndex(c => c.name === e.sourceHandle);
                        if (srcCol >= 0) srcY = src.position.y + 36 + srcCol * 28 + 15;
                    }
                    if (e.targetHandle) {
                        const tgtCol = tgt.data.columns?.findIndex(c => c.name === e.targetHandle);
                        if (tgtCol >= 0) tgtY = tgt.position.y + 36 + tgtCol * 28 + 15;
                    }

                    const srcX = src.position.x + 280;
                    const tgtX = tgt.position.x;

                    // Check if mouse is near the edge line (simple distance check)
                    const midX = (srcX + tgtX) / 2;
                    const midY = (srcY + tgtY) / 2;
                    const dist = Math.sqrt((x - midX) ** 2 + (y - midY) ** 2);
                    return dist < 30;
                });
            } else {
                this.hoveredEdge = null;
            }

            if (this.hoveredNode !== prevHovered || this.hoveredEdge !== prevHoveredEdge) {
                this.updateHighlights();
                this.render();
            }

            this.canvas.style.cursor = (this.hoveredNode || this.hoveredEdge) ? 'pointer' : 'default';
        });

        // Mouse down
        this.canvas.addEventListener('mousedown', e => {
            const rect = this.canvas.getBoundingClientRect();
            const x = (e.clientX - rect.left) / this.scale - this.offset.x;
            const y = (e.clientY - rect.top) / this.scale - this.offset.y;

            const node = this.nodes.find(n => {
                const nodeH = 36 + (n.data.columns?.length || 0) * 28 + 4;
                return x >= n.position.x && x <= n.position.x + 280 &&
                    y >= n.position.y && y <= n.position.y + nodeH;
            });

            if (node) {
                this.draggedNode = node;
                this.dragStartPos = { x: node.position.x, y: node.position.y };
                this.hasDragged = false;
                this.dragOffset = {
                    x: x - node.position.x,
                    y: y - node.position.y
                };
            } else {
                this.isPanning = true;
                this.lastMouse = { x: e.clientX, y: e.clientY };
                this.canvas.style.cursor = 'grabbing';
            }
        });

        // Mouse up
        this.canvas.addEventListener('mouseup', () => {
            this.draggedNode = null;
            this.dragStartPos = null;
            this.isPanning = false;
            this.canvas.style.cursor = this.hoveredNode ? 'pointer' : 'default';
        });

        // Wheel zoom
        this.canvas.addEventListener('wheel', e => {
            e.preventDefault();

            const rect = this.canvas.getBoundingClientRect();
            const canvasW = this.canvas.width / this.dpr;
            const canvasH = this.canvas.height / this.dpr;

            // Mouse position in canvas space
            const mouseCanvasX = e.clientX - rect.left;
            const mouseCanvasY = e.clientY - rect.top;

            // Mouse position in world space (before zoom)
            const mouseWorldX = (mouseCanvasX / this.scale) - this.offset.x;
            const mouseWorldY = (mouseCanvasY / this.scale) - this.offset.y;

            // Slower zoom speed
            const delta = e.deltaY < 0 ? 1.05 : 0.95;
            const oldScale = this.scale;
            this.scale = Math.max(0.05, Math.min(3, this.scale * delta));

            // Adjust offset so mouse position stays fixed
            this.offset.x = (mouseCanvasX / this.scale) - mouseWorldX;
            this.offset.y = (mouseCanvasY / this.scale) - mouseWorldY;

            this.render();
        });

        // Click
        this.canvas.addEventListener('click', e => {
            // Only open editor if not dragged
            if (!this.hasDragged && this.hoveredNode) {
                window.openTableEdit?.(this.hoveredNode.data.label);
            }
            this.hasDragged = false;
        });

        // Resize
        window.addEventListener('resize', () => {
            this.setupCanvas();
            this.render();
        });

        // Mouse leave - clear hover states
        this.canvas.addEventListener('mouseleave', () => {
            if (this.hoveredNode || this.hoveredEdge) {
                this.hoveredNode = null;
                this.hoveredEdge = null;
                this.updateHighlights();
                this.render();
                this.canvas.style.cursor = 'default';
            }
        });
    }

    updateHighlights() {
        this.highlightedNodes.clear();
        this.highlightedFields.clear();

        if (this.hoveredEdge) {
            // Highlight nodes connected by hovered edge
            this.highlightedNodes.add(this.hoveredEdge.source);
            this.highlightedNodes.add(this.hoveredEdge.target);

            if (this.hoveredEdge.sourceHandle) {
                if (!this.highlightedFields.has(this.hoveredEdge.source)) {
                    this.highlightedFields.set(this.hoveredEdge.source, new Set());
                }
                this.highlightedFields.get(this.hoveredEdge.source).add(this.hoveredEdge.sourceHandle);
            }

            if (this.hoveredEdge.targetHandle) {
                if (!this.highlightedFields.has(this.hoveredEdge.target)) {
                    this.highlightedFields.set(this.hoveredEdge.target, new Set());
                }
                this.highlightedFields.get(this.hoveredEdge.target).add(this.hoveredEdge.targetHandle);
            }
            return;
        }

        if (!this.hoveredNode || !this.hoveredNode.hasRelations) return;

        const connectedEdges = this.edges.filter(e =>
            e.source === this.hoveredNode.id || e.target === this.hoveredNode.id
        );

        connectedEdges.forEach(e => {
            this.highlightedNodes.add(e.source);
            this.highlightedNodes.add(e.target);

            if (e.sourceHandle) {
                if (!this.highlightedFields.has(e.source)) {
                    this.highlightedFields.set(e.source, new Set());
                }
                this.highlightedFields.get(e.source).add(e.sourceHandle);
            }

            if (e.targetHandle) {
                if (!this.highlightedFields.has(e.target)) {
                    this.highlightedFields.set(e.target, new Set());
                }
                this.highlightedFields.get(e.target).add(e.targetHandle);
            }
        });
    }

    showMessage(msg) {
        // Create overlay instead of replacing innerHTML to preserve canvas
        let overlay = this.container.querySelector('.schema-overlay');
        if (!overlay) {
            overlay = document.createElement('div');
            overlay.className = 'schema-overlay';
            this.container.appendChild(overlay);
        }
        overlay.innerHTML = `<div class="loading"><div>${msg}</div></div>`;
        overlay.style.display = 'flex';
    }

    hideMessage() {
        const overlay = this.container.querySelector('.schema-overlay');
        if (overlay) {
            overlay.style.display = 'none';
        }
        // Also remove original loading div if present
        const loading = this.container.querySelector('.loading:not(.schema-overlay .loading)');
        if (loading) loading.remove();
    }

    // Truncate text to fit within maxWidth
    truncateText(text, maxWidth) {
        if (!text) return '';
        const measured = this.ctx.measureText(text).width;
        if (measured <= maxWidth) return text;

        let truncated = text;
        while (truncated.length > 0 && this.ctx.measureText(truncated + '…').width > maxWidth) {
            truncated = truncated.slice(0, -1);
        }
        return truncated + '…';
    }

    // Public API
    zoomIn() {
        this.scale = Math.min(3, this.scale * 1.2);
        this.render();
    }

    zoomOut() {
        this.scale = Math.max(0.05, this.scale / 1.2);
        this.render();
    }

    resetView() {
        this.fitView();
        this.render();
    }

    focusNode(node) {
        const w = this.canvas.width / this.dpr;
        const h = this.canvas.height / this.dpr;

        this.scale = 1;
        this.offset.x = (w / 2) / this.scale - node.position.x - 110;
        this.offset.y = (h / 2) / this.scale - node.position.y - 100;

        this.render();
    }

    organizeLayout(type) {
        const rawNodes = this.nodes.map(n => ({
            id: n.id,
            data: n.data,
            position: { x: 0, y: 0 }
        }));

        if (type === 'grid') {
            const cols = Math.ceil(Math.sqrt(rawNodes.length));
            const colSpacing = 500; // Increased spacing
            const rowSpacing = 400; // Increased spacing
            rawNodes.forEach((n, i) => {
                n.position = {
                    x: (i % cols) * colSpacing,
                    y: Math.floor(i / cols) * rowSpacing
                };
            });
        } else if (type === 'circular') {
            const radius = Math.max(500, rawNodes.length * 70); // Increased radius
            const angleStep = (2 * Math.PI) / rawNodes.length;
            rawNodes.forEach((n, i) => {
                const angle = i * angleStep - Math.PI / 2; // Start from top
                n.position = {
                    x: Math.cos(angle) * radius,
                    y: Math.sin(angle) * radius
                };
            });
        } else if (type === 'force') {
            // Optimized force-directed layout with better spacing
            rawNodes.forEach((n, i) => {
                n.position = {
                    x: Math.random() * 2000 - 1000,
                    y: Math.random() * 2000 - 1000
                };
            });

            // Adaptive iterations for better performance
            const iterations = Math.min(80, Math.max(50, 200 - rawNodes.length));

            for (let iter = 0; iter < iterations; iter++) {
                // Repulsion between all nodes
                for (let i = 0; i < rawNodes.length; i++) {
                    for (let j = i + 1; j < rawNodes.length; j++) {
                        const dx = rawNodes[j].position.x - rawNodes[i].position.x;
                        const dy = rawNodes[j].position.y - rawNodes[i].position.y;
                        const dist = Math.sqrt(dx * dx + dy * dy) || 1;
                        const force = 8000 / (dist * dist); // Increased repulsion

                        rawNodes[i].position.x -= (dx / dist) * force;
                        rawNodes[i].position.y -= (dy / dist) * force;
                        rawNodes[j].position.x += (dx / dist) * force;
                        rawNodes[j].position.y += (dy / dist) * force;
                    }
                }

                // Attraction for connected nodes
                this.edges.forEach(e => {
                    const src = rawNodes.find(n => n.id === e.source);
                    const tgt = rawNodes.find(n => n.id === e.target);
                    if (!src || !tgt) return;

                    const dx = tgt.position.x - src.position.x;
                    const dy = tgt.position.y - src.position.y;
                    const dist = Math.sqrt(dx * dx + dy * dy) || 1;
                    const force = dist * 0.015; // Slightly increased attraction

                    src.position.x += (dx / dist) * force;
                    src.position.y += (dy / dist) * force;
                    tgt.position.x -= (dx / dist) * force;
                    tgt.position.y -= (dy / dist) * force;
                });
            }
        }

        this.nodes = rawNodes.map(n => ({ ...n, hasRelations: true }));
        this.fitView();
        this.render();
    }
}

// Initialize
let schemaCanvas;
document.addEventListener('DOMContentLoaded', () => {
    schemaCanvas = new SchemaCanvas('root');
});
