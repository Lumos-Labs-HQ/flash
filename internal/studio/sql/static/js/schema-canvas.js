// DataGrip-style Schema Visualizer
class SchemaCanvas {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        this.canvas = document.createElement('canvas');
        this.ctx = this.canvas.getContext('2d', { alpha: false });

        // State
        this.nodes = [];
        this.edges = [];
        this.scale = 1;
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

        // Design constants - DataGrip style
        this.nodeWidth = 240;
        this.headerHeight = 32;
        this.rowHeight = 24;
        this.nodePadding = 8;
        this.nodeRadius = 6;

        // Colors - DataGrip dark theme
        this.colors = {
            bg: '#1e1e1e',
            grid: '#2a2a2a',
            gridMajor: '#333333',
            nodeBg: '#2d2d2d',
            nodeHeader: '#3c3f41',
            nodeBorder: '#515151',
            nodeHoverBorder: '#589df6',
            nodeHighlightBorder: '#589df6',
            text: '#bababa',
            textBright: '#ffffff',
            textDim: '#6e6e6e',
            primary: '#cc7832',
            foreign: '#589df6',
            edge: '#589df6',
            edgeDim: '#404040'
        };

        this.init();
    }

    init() {
        this.setupCanvas();
        this.setupEvents();
        this.loadSchema();
    }

    setupCanvas() {
        // Remove any existing loading elements first
        const existingLoading = this.container.querySelector('.loading');
        if (existingLoading) existingLoading.remove();

        // Check if canvas already exists
        let existingCanvas = this.container.querySelector('canvas');
        if (existingCanvas) {
            this.canvas = existingCanvas;
        } else {
            this.container.appendChild(this.canvas);
        }

        const dpr = window.devicePixelRatio || 1;
        const containerW = this.container.clientWidth;
        const containerH = this.container.clientHeight;

        this.canvas.width = containerW * dpr;
        this.canvas.height = containerH * dpr;
        this.canvas.style.width = '100%';
        this.canvas.style.height = '100%';
        this.canvas.style.background = this.colors.bg;
        this.ctx = this.canvas.getContext('2d', { alpha: false });
        this.ctx.scale(dpr, dpr);
        this.dpr = dpr;

        console.log('setupCanvas:', { containerW, containerH, dpr, canvasW: this.canvas.width, canvasH: this.canvas.height });
    }

    async loadSchema() {
        try {
            this.showMessage('Loading schema...');
            const res = await fetch('/api/schema');
            const data = await res.json();

            if (data.success && data.data) {
                const rawNodes = data.data.nodes || [];
                const rawEdges = data.data.edges || [];

                if (rawNodes.length === 0) {
                    this.showMessage('No tables found');
                    return;
                }

                // Debug: log relationships
                console.log('Schema loaded:', rawNodes.length, 'tables,', rawEdges.length, 'relationships');
                console.log('Edges:', rawEdges.map(e => `${e.sourceHandle} -> ${e.targetHandle}`));

                document.getElementById('stats').textContent =
                    `${rawNodes.length} tables • ${rawEdges.length} relationships`;

                this.nodes = this.layoutNodes(rawNodes, rawEdges);
                this.edges = rawEdges;

                // Resolve any overlapping tables
                this.resolveOverlaps();

                // Hide all loading elements
                this.hideMessage();

                this.fitView();
                this.render();
            } else {
                this.showMessage(data.message || 'Failed to load');
            }
        } catch (err) {
            this.showMessage(`Error: ${err.message}`);
        }
    }

    getNodeHeight(node) {
        const cols = node.data?.columns?.length || 0;
        return this.headerHeight + cols * this.rowHeight + this.nodePadding;
    }

    layoutNodes(nodes, edges) {
        // Build relationship maps
        const relations = new Map();
        const inDegree = new Map();
        const outDegree = new Map();

        nodes.forEach(n => {
            relations.set(n.id, new Set());
            inDegree.set(n.id, 0);
            outDegree.set(n.id, 0);
        });

        edges.forEach(e => {
            if (relations.has(e.source)) relations.get(e.source).add(e.target);
            if (relations.has(e.target)) relations.get(e.target).add(e.source);
            outDegree.set(e.source, (outDegree.get(e.source) || 0) + 1);
            inDegree.set(e.target, (inDegree.get(e.target) || 0) + 1);
        });

        // Find connected components
        const visited = new Set();
        const components = [];

        const dfs = (nodeId, component) => {
            if (visited.has(nodeId)) return;
            visited.add(nodeId);
            const node = nodes.find(n => n.id === nodeId);
            if (node) component.push(node);
            relations.get(nodeId)?.forEach(neighbor => dfs(neighbor, component));
        };

        nodes.forEach(node => {
            if (!visited.has(node.id)) {
                const component = [];
                dfs(node.id, component);
                components.push(component);
            }
        });

        // Sort components: connected first (by size), then unconnected
        const connected = components.filter(c => c.length > 1 ||
            edges.some(e => e.source === c[0]?.id || e.target === c[0]?.id));
        const unconnected = components.filter(c => c.length === 1 &&
            !edges.some(e => e.source === c[0]?.id || e.target === c[0]?.id));

        connected.sort((a, b) => b.length - a.length);

        const positioned = [];
        let globalY = 0;

        // Layout connected components with hierarchical approach
        connected.forEach(comp => {
            const compEdges = edges.filter(e =>
                comp.some(n => n.id === e.source) && comp.some(n => n.id === e.target));

            const laid = this.layoutComponent(comp, compEdges, inDegree, outDegree, 0, globalY);

            // Calculate component bounds
            let maxY = globalY;
            laid.forEach(n => {
                maxY = Math.max(maxY, n.position.y + this.getNodeHeight(n));
                positioned.push({ ...n, hasRelations: true });
            });

            globalY = maxY + 40;
        });

        // Layout unconnected tables in a compact grid to the right
        if (unconnected.length > 0) {
            const maxX = positioned.length > 0
                ? Math.max(...positioned.map(n => n.position.x)) + this.nodeWidth + 80
                : 0;

            const cols = Math.min(3, Math.ceil(Math.sqrt(unconnected.length)));
            let col = 0;
            let rowMaxHeight = 0;
            let currentY = 0;

            unconnected.forEach(comp => {
                const node = comp[0];
                const h = this.getNodeHeight(node);

                positioned.push({
                    ...node,
                    position: { x: maxX + col * (this.nodeWidth + 50), y: currentY },
                    hasRelations: false
                });

                rowMaxHeight = Math.max(rowMaxHeight, h);
                col++;
                if (col >= cols) {
                    col = 0;
                    currentY += rowMaxHeight + 40;
                    rowMaxHeight = 0;
                }
            });
        }

        return positioned;
    }

    layoutComponent(nodes, edges, inDegree, outDegree, startX, startY) {
        if (nodes.length === 0) return [];
        if (nodes.length === 1) {
            return [{ ...nodes[0], position: { x: startX, y: startY } }];
        }

        // Build adjacency for BFS level assignment
        const adj = new Map();
        nodes.forEach(n => adj.set(n.id, { targets: [], sources: [] }));

        edges.forEach(e => {
            if (adj.has(e.source)) adj.get(e.source).targets.push(e.target);
            if (adj.has(e.target)) adj.get(e.target).sources.push(e.source);
        });

        // Find the central table (most connections = usually users table)
        const sorted = [...nodes].sort((a, b) => {
            const aConns = (inDegree.get(a.id) || 0) + (outDegree.get(a.id) || 0);
            const bConns = (inDegree.get(b.id) || 0) + (outDegree.get(b.id) || 0);
            return bConns - aConns;
        });

        // BFS to assign levels - center node is level 0
        const nodeLevel = new Map();
        const queue = [{ id: sorted[0].id, level: 0 }];
        nodeLevel.set(sorted[0].id, 0);

        while (queue.length > 0) {
            const { id, level } = queue.shift();
            const connections = adj.get(id);
            if (!connections) continue;

            // Tables that reference this one go to the left (higher level number)
            connections.sources.forEach(srcId => {
                if (!nodeLevel.has(srcId)) {
                    nodeLevel.set(srcId, level + 1);
                    queue.push({ id: srcId, level: level + 1 });
                }
            });

            // Tables this one references go to the right (lower level number)
            connections.targets.forEach(tgtId => {
                if (!nodeLevel.has(tgtId)) {
                    nodeLevel.set(tgtId, level - 1);
                    queue.push({ id: tgtId, level: level - 1 });
                }
            });
        }

        // Handle any unvisited nodes
        nodes.forEach(n => {
            if (!nodeLevel.has(n.id)) nodeLevel.set(n.id, 0);
        });

        // Normalize levels to start from 0
        const allLevels = Array.from(nodeLevel.values());
        const minLevel = Math.min(...allLevels);
        nodeLevel.forEach((level, id) => nodeLevel.set(id, level - minLevel));

        // Group by level
        const levelGroups = new Map();
        nodes.forEach(n => {
            const level = nodeLevel.get(n.id) || 0;
            if (!levelGroups.has(level)) levelGroups.set(level, []);
            levelGroups.get(level).push(n);
        });

        // Calculate positions with improved spacing to avoid overlaps
        const positioned = [];
        const levelSpacing = this.nodeWidth + 80; // Increased spacing between levels
        const nodeSpacing = 35; // Increased spacing between nodes in same level
        const sortedLevels = Array.from(levelGroups.keys()).sort((a, b) => a - b);

        // Calculate the maximum height needed for proper vertical centering
        let maxLevelHeight = 0;
        sortedLevels.forEach(level => {
            const nodesAtLevel = levelGroups.get(level);
            let levelHeight = 0;
            nodesAtLevel.forEach(node => {
                levelHeight += this.getNodeHeight(node) + nodeSpacing;
            });
            maxLevelHeight = Math.max(maxLevelHeight, levelHeight);
        });

        // Position each level with vertical centering
        sortedLevels.forEach(level => {
            const nodesAtLevel = levelGroups.get(level);

            // Calculate total height of this level
            let levelHeight = 0;
            nodesAtLevel.forEach(node => {
                levelHeight += this.getNodeHeight(node) + nodeSpacing;
            });
            levelHeight -= nodeSpacing; // Remove last spacing

            // Sort by connection count for better routing
            nodesAtLevel.sort((a, b) => {
                const aConns = edges.filter(e => e.source === a.id || e.target === a.id).length;
                const bConns = edges.filter(e => e.source === b.id || e.target === b.id).length;
                return bConns - aConns;
            });

            // Start Y position to center this level
            let currentY = startY + (maxLevelHeight - levelHeight) / 2;

            nodesAtLevel.forEach(node => {
                const h = this.getNodeHeight(node);
                positioned.push({
                    ...node,
                    position: { x: startX + level * levelSpacing, y: currentY }
                });
                currentY += h + nodeSpacing;
            });
        });

        return positioned;
    }

    fitView() {
        if (this.nodes.length === 0) return;

        const padding = 50;
        const canvasW = this.canvas.width / this.dpr;
        const canvasH = this.canvas.height / this.dpr;

        // Calculate content bounding box
        let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
        this.nodes.forEach(n => {
            const h = this.getNodeHeight(n);
            minX = Math.min(minX, n.position.x);
            minY = Math.min(minY, n.position.y);
            maxX = Math.max(maxX, n.position.x + this.nodeWidth);
            maxY = Math.max(maxY, n.position.y + h);
        });

        const contentW = maxX - minX;
        const contentH = maxY - minY;

        // Calculate scale to fit content with padding
        const scaleX = (canvasW - padding * 2) / contentW;
        const scaleY = (canvasH - padding * 2) / contentH;
        this.scale = Math.min(scaleX, scaleY, 1.5);
        this.scale = Math.max(0.3, this.scale);

        // Content center in world coordinates
        const contentCenterX = minX + contentW / 2;
        const contentCenterY = minY + contentH / 2;

        // We want content center to appear at screen center
        // With transform: ctx.scale(s) then ctx.translate(ox, oy)
        // Screen pos = (worldPos + offset) * scale
        // So: (contentCenter + offset) * scale = screenCenter
        // offset = screenCenter / scale - contentCenter
        const screenCenterX = canvasW / 2;
        const screenCenterY = canvasH / 2;

        this.offset.x = screenCenterX / this.scale - contentCenterX;
        this.offset.y = screenCenterY / this.scale - contentCenterY;

        console.log('fitView calculated:', {
            canvasW, canvasH,
            contentBounds: { minX, minY, maxX, maxY },
            contentCenter: { x: contentCenterX, y: contentCenterY },
            screenCenter: { x: screenCenterX, y: screenCenterY },
            scale: this.scale,
            offsetX: this.offset.x,
            offsetY: this.offset.y
        });
    }

    render() {
        const w = this.canvas.width / this.dpr;
        const h = this.canvas.height / this.dpr;

        // Clear with background color
        this.ctx.fillStyle = this.colors.bg;
        this.ctx.fillRect(0, 0, w, h);

        // Transform - apply scale first, then translate in world coordinates
        this.ctx.save();
        this.ctx.scale(this.scale, this.scale);
        this.ctx.translate(this.offset.x, this.offset.y);

        // Draw subtle grid
        this.drawGrid();

        // Draw edges first (behind nodes)
        this.edges.forEach(e => this.drawEdge(e));

        // Draw nodes
        this.nodes.forEach(n => this.drawNode(n));

        this.ctx.restore();
    }

    drawGrid() {
        if (this.scale < 0.25) return;

        const gridSize = 40;
        const opacity = Math.min(this.scale * 0.3, 0.15);

        // Calculate visible world bounds
        const viewW = this.canvas.width / this.dpr / this.scale;
        const viewH = this.canvas.height / this.dpr / this.scale;
        const startX = Math.floor(-this.offset.x / gridSize) * gridSize - gridSize;
        const startY = Math.floor(-this.offset.y / gridSize) * gridSize - gridSize;
        const endX = startX + viewW + gridSize * 3;
        const endY = startY + viewH + gridSize * 3;

        // Minor grid
        this.ctx.strokeStyle = this.colors.grid;
        this.ctx.lineWidth = 1 / this.scale;
        this.ctx.globalAlpha = opacity;

        for (let x = startX; x < endX; x += gridSize) {
            this.ctx.beginPath();
            this.ctx.moveTo(x, startY);
            this.ctx.lineTo(x, endY);
            this.ctx.stroke();
        }
        for (let y = startY; y < endY; y += gridSize) {
            this.ctx.beginPath();
            this.ctx.moveTo(startX, y);
            this.ctx.lineTo(endX, y);
            this.ctx.stroke();
        }

        this.ctx.globalAlpha = 1;
    }

    drawNode(node) {
        const { x, y } = node.position;
        const w = this.nodeWidth;
        const cols = node.data.columns || [];
        const h = this.getNodeHeight(node);

        const isHovered = this.hoveredNode === node;
        const isHighlighted = this.highlightedNodes.has(node.id) || isHovered;
        const isDimmed = this.hoveredNode && !isHighlighted && node.hasRelations;

        // Node shadow
        if (!isDimmed) {
            this.ctx.shadowColor = 'rgba(0,0,0,0.3)';
            this.ctx.shadowBlur = 10;
            this.ctx.shadowOffsetY = 2;
        }

        // Node background
        this.ctx.fillStyle = isDimmed ? '#222' : this.colors.nodeBg;
        this.roundRect(x, y, w, h, this.nodeRadius);
        this.ctx.fill();

        // Reset shadow
        this.ctx.shadowBlur = 0;
        this.ctx.shadowOffsetY = 0;

        // Border
        this.ctx.strokeStyle = isHighlighted ? this.colors.nodeHoverBorder :
            (isDimmed ? '#333' : this.colors.nodeBorder);
        this.ctx.lineWidth = isHighlighted ? 2 : 1;
        this.roundRect(x, y, w, h, this.nodeRadius);
        this.ctx.stroke();

        // Header background
        this.ctx.save();
        this.ctx.beginPath();
        this.ctx.moveTo(x + this.nodeRadius, y);
        this.ctx.lineTo(x + w - this.nodeRadius, y);
        this.ctx.quadraticCurveTo(x + w, y, x + w, y + this.nodeRadius);
        this.ctx.lineTo(x + w, y + this.headerHeight);
        this.ctx.lineTo(x, y + this.headerHeight);
        this.ctx.lineTo(x, y + this.nodeRadius);
        this.ctx.quadraticCurveTo(x, y, x + this.nodeRadius, y);
        this.ctx.closePath();
        this.ctx.fillStyle = isDimmed ? '#252525' : this.colors.nodeHeader;
        this.ctx.fill();
        this.ctx.restore();

        // Header separator line
        this.ctx.beginPath();
        this.ctx.moveTo(x, y + this.headerHeight);
        this.ctx.lineTo(x + w, y + this.headerHeight);
        this.ctx.strokeStyle = isDimmed ? '#333' : this.colors.nodeBorder;
        this.ctx.lineWidth = 1;
        this.ctx.stroke();

        // Table name
        this.ctx.font = 'bold 13px "JetBrains Mono", Consolas, monospace';
        this.ctx.fillStyle = isDimmed ? this.colors.textDim :
            (isHighlighted ? this.colors.textBright : this.colors.text);
        this.ctx.textBaseline = 'middle';

        const tableName = this.truncateText(node.data.label, w - 20);
        this.ctx.fillText(tableName, x + 10, y + this.headerHeight / 2);

        // Columns
        const highlightedFields = this.highlightedFields.get(node.id) || new Set();

        cols.forEach((col, i) => {
            const rowY = y + this.headerHeight + i * this.rowHeight;
            const isFieldHighlighted = highlightedFields.has(col.name);

            // Row highlight
            if (isFieldHighlighted) {
                this.ctx.fillStyle = 'rgba(88, 157, 246, 0.15)';
                this.ctx.fillRect(x + 1, rowY, w - 2, this.rowHeight);
            }

            // Column icon/indicator
            let iconX = x + 10;
            this.ctx.font = '11px "JetBrains Mono", Consolas, monospace';

            if (col.isPrimary) {
                this.ctx.fillStyle = this.colors.primary;
                this.ctx.fillText('PK', iconX, rowY + this.rowHeight / 2);
                iconX += 22;
            } else if (col.isForeign) {
                this.ctx.fillStyle = this.colors.foreign;
                this.ctx.fillText('FK', iconX, rowY + this.rowHeight / 2);
                iconX += 22;
            } else {
                iconX += 22;
            }

            // Column name
            this.ctx.fillStyle = isDimmed ? this.colors.textDim :
                (isFieldHighlighted ? this.colors.foreign : this.colors.text);
            const colName = this.truncateText(col.name, w * 0.45);
            this.ctx.fillText(colName, iconX, rowY + this.rowHeight / 2);

            // Type
            this.ctx.fillStyle = isDimmed ? '#444' : '#707070';
            this.ctx.textAlign = 'right';
            const colType = this.truncateText(col.type, w * 0.35);
            this.ctx.fillText(colType, x + w - 10, rowY + this.rowHeight / 2);
            this.ctx.textAlign = 'left';
        });
    }

    roundRect(x, y, w, h, r) {
        this.ctx.beginPath();
        this.ctx.moveTo(x + r, y);
        this.ctx.lineTo(x + w - r, y);
        this.ctx.quadraticCurveTo(x + w, y, x + w, y + r);
        this.ctx.lineTo(x + w, y + h - r);
        this.ctx.quadraticCurveTo(x + w, y + h, x + w - r, y + h);
        this.ctx.lineTo(x + r, y + h);
        this.ctx.quadraticCurveTo(x, y + h, x, y + h - r);
        this.ctx.lineTo(x, y + r);
        this.ctx.quadraticCurveTo(x, y, x + r, y);
        this.ctx.closePath();
    }

    drawEdge(edge) {
        const src = this.nodes.find(n => n.id === edge.source);
        const tgt = this.nodes.find(n => n.id === edge.target);
        if (!src || !tgt) return;

        const isHighlighted = this.hoveredEdge === edge ||
            (this.hoveredNode && (this.hoveredNode.id === edge.source || this.hoveredNode.id === edge.target));
        const isDimmed = this.hoveredNode && !isHighlighted;

        // Find field Y positions
        let srcFieldY = src.position.y + this.headerHeight / 2;
        let tgtFieldY = tgt.position.y + this.headerHeight / 2;

        if (edge.sourceHandle) {
            const idx = src.data.columns?.findIndex(c => c.name === edge.sourceHandle);
            if (idx >= 0) srcFieldY = src.position.y + this.headerHeight + idx * this.rowHeight + this.rowHeight / 2;
        }
        if (edge.targetHandle) {
            const idx = tgt.data.columns?.findIndex(c => c.name === edge.targetHandle);
            if (idx >= 0) tgtFieldY = tgt.position.y + this.headerHeight + idx * this.rowHeight + this.rowHeight / 2;
        }

        // Determine connection sides
        const srcCenterX = src.position.x + this.nodeWidth / 2;
        const tgtCenterX = tgt.position.x + this.nodeWidth / 2;

        let srcX, tgtX;
        if (srcCenterX < tgtCenterX) {
            srcX = src.position.x + this.nodeWidth;
            tgtX = tgt.position.x;
        } else {
            srcX = src.position.x;
            tgtX = tgt.position.x + this.nodeWidth;
        }

        // Style
        this.ctx.strokeStyle = isDimmed ? this.colors.edgeDim :
            (isHighlighted ? this.colors.nodeHoverBorder : this.colors.edge);
        this.ctx.lineWidth = isHighlighted ? 2 : 1.5;
        this.ctx.globalAlpha = isDimmed ? 0.3 : (isHighlighted ? 1 : 0.7);

        // Draw orthogonal path
        const midX = (srcX + tgtX) / 2;

        this.ctx.beginPath();
        this.ctx.moveTo(srcX, srcFieldY);
        this.ctx.lineTo(midX, srcFieldY);
        this.ctx.lineTo(midX, tgtFieldY);
        this.ctx.lineTo(tgtX, tgtFieldY);
        this.ctx.stroke();

        // Arrow at target
        const arrowSize = 6;
        const arrowDir = tgtX > midX ? 1 : -1;

        this.ctx.beginPath();
        this.ctx.moveTo(tgtX, tgtFieldY);
        this.ctx.lineTo(tgtX - arrowDir * arrowSize, tgtFieldY - arrowSize / 2);
        this.ctx.lineTo(tgtX - arrowDir * arrowSize, tgtFieldY + arrowSize / 2);
        this.ctx.closePath();
        this.ctx.fillStyle = this.ctx.strokeStyle;
        this.ctx.fill();

        // Cardinality indicator (small circle at source = many side)
        this.ctx.beginPath();
        this.ctx.arc(srcX + (srcX < tgtX ? 8 : -8), srcFieldY, 3, 0, Math.PI * 2);
        this.ctx.fill();

        this.ctx.globalAlpha = 1;
    }

    truncateText(text, maxWidth) {
        if (!text) return '';
        const measured = this.ctx.measureText(text).width;
        if (measured <= maxWidth) return text;

        let truncated = text;
        while (truncated.length > 1 && this.ctx.measureText(truncated + '…').width > maxWidth) {
            truncated = truncated.slice(0, -1);
        }
        return truncated + '…';
    }

    setupEvents() {
        // Search
        const searchInput = document.getElementById('table-search');
        if (searchInput) {
            searchInput.addEventListener('input', e => {
                const query = e.target.value.toLowerCase();
                if (!query) return;
                const node = this.nodes.find(n => n.data.label.toLowerCase().includes(query));
                if (node) this.focusNode(node);
            });
        }

        // Context menu
        this.canvas.addEventListener('contextmenu', e => {
            e.preventDefault();
            const menu = document.getElementById('context-menu');
            if (menu) {
                menu.style.display = 'block';
                menu.style.left = e.clientX + 'px';
                menu.style.top = e.clientY + 'px';
            }
        });

        document.addEventListener('click', () => {
            const menu = document.getElementById('context-menu');
            if (menu) menu.style.display = 'none';
        });

        // Mouse move
        this.canvas.addEventListener('mousemove', e => {
            const rect = this.canvas.getBoundingClientRect();
            const x = (e.clientX - rect.left) / this.scale - this.offset.x;
            const y = (e.clientY - rect.top) / this.scale - this.offset.y;

            if (this.draggedNode) {
                this.draggedNode.position.x = x - this.dragOffset.x;
                this.draggedNode.position.y = y - this.dragOffset.y;
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
                const h = this.getNodeHeight(n);
                return x >= n.position.x && x <= n.position.x + this.nodeWidth &&
                    y >= n.position.y && y <= n.position.y + h;
            });

            if (!this.hoveredNode) {
                this.hoveredEdge = this.findHoveredEdge(x, y);
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
                const h = this.getNodeHeight(n);
                return x >= n.position.x && x <= n.position.x + this.nodeWidth &&
                    y >= n.position.y && y <= n.position.y + h;
            });

            if (node) {
                this.draggedNode = node;
                this.dragStartPos = { x: node.position.x, y: node.position.y };
                this.hasDragged = false;
                this.dragOffset = { x: x - node.position.x, y: y - node.position.y };
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
            const mouseX = e.clientX - rect.left;
            const mouseY = e.clientY - rect.top;
            const worldX = mouseX / this.scale - this.offset.x;
            const worldY = mouseY / this.scale - this.offset.y;

            const delta = e.deltaY < 0 ? 1.08 : 0.92;
            this.scale = Math.max(0.1, Math.min(3, this.scale * delta));

            this.offset.x = mouseX / this.scale - worldX;
            this.offset.y = mouseY / this.scale - worldY;
            this.render();
        });

        // Click
        this.canvas.addEventListener('click', () => {
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

        // Mouse leave
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

    findHoveredEdge(x, y) {
        return this.edges.find(e => {
            const src = this.nodes.find(n => n.id === e.source);
            const tgt = this.nodes.find(n => n.id === e.target);
            if (!src || !tgt) return false;

            let srcFieldY = src.position.y + this.headerHeight / 2;
            let tgtFieldY = tgt.position.y + this.headerHeight / 2;

            if (e.sourceHandle) {
                const idx = src.data.columns?.findIndex(c => c.name === e.sourceHandle);
                if (idx >= 0) srcFieldY = src.position.y + this.headerHeight + idx * this.rowHeight + this.rowHeight / 2;
            }
            if (e.targetHandle) {
                const idx = tgt.data.columns?.findIndex(c => c.name === e.targetHandle);
                if (idx >= 0) tgtFieldY = tgt.position.y + this.headerHeight + idx * this.rowHeight + this.rowHeight / 2;
            }

            const srcCenterX = src.position.x + this.nodeWidth / 2;
            const tgtCenterX = tgt.position.x + this.nodeWidth / 2;

            let srcX = srcCenterX < tgtCenterX ? src.position.x + this.nodeWidth : src.position.x;
            let tgtX = srcCenterX < tgtCenterX ? tgt.position.x : tgt.position.x + this.nodeWidth;
            const midX = (srcX + tgtX) / 2;

            // Check proximity to edge segments
            const tolerance = 8;

            // Horizontal segment from source
            if (y >= srcFieldY - tolerance && y <= srcFieldY + tolerance &&
                x >= Math.min(srcX, midX) - tolerance && x <= Math.max(srcX, midX) + tolerance) {
                return true;
            }
            // Vertical segment
            if (x >= midX - tolerance && x <= midX + tolerance &&
                y >= Math.min(srcFieldY, tgtFieldY) - tolerance && y <= Math.max(srcFieldY, tgtFieldY) + tolerance) {
                return true;
            }
            // Horizontal segment to target
            if (y >= tgtFieldY - tolerance && y <= tgtFieldY + tolerance &&
                x >= Math.min(midX, tgtX) - tolerance && x <= Math.max(midX, tgtX) + tolerance) {
                return true;
            }
            return false;
        });
    }

    updateHighlights() {
        this.highlightedNodes.clear();
        this.highlightedFields.clear();

        if (this.hoveredEdge) {
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

        if (!this.hoveredNode) return;

        const connectedEdges = this.edges.filter(e =>
            e.source === this.hoveredNode.id || e.target === this.hoveredNode.id);

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
        let overlay = this.container.querySelector('.schema-overlay');
        if (!overlay) {
            overlay = document.createElement('div');
            overlay.className = 'schema-overlay';
            this.container.appendChild(overlay);
        }
        overlay.style.cssText = 'position:absolute;top:0;left:0;right:0;bottom:0;display:flex;align-items:center;justify-content:center;background:rgba(30,30,30,0.95);z-index:100;';
        overlay.innerHTML = `<div style="color:#bababa;font-size:14px;">${msg}</div>`;
    }

    hideMessage() {
        // Remove all overlays and loading elements completely
        const overlays = this.container.querySelectorAll('.schema-overlay');
        overlays.forEach(el => el.remove());

        const loadings = this.container.querySelectorAll('.loading');
        loadings.forEach(el => el.remove());

        this.container.classList.remove('loading');
    }

    // Public API
    zoomIn() {
        this.scale = Math.min(3, this.scale * 1.2);
        this.render();
    }

    zoomOut() {
        this.scale = Math.max(0.1, this.scale / 1.2);
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
        this.offset.x = w / 2 / this.scale - node.position.x - this.nodeWidth / 2;
        this.offset.y = h / 2 / this.scale - node.position.y - this.getNodeHeight(node) / 2;
        this.render();
    }

    organizeLayout(type) {
        const rawNodes = this.nodes.map(n => ({
            id: n.id,
            data: n.data,
            position: { x: 0, y: 0 }
        }));

        switch (type) {
            case 'grid':
                this.layoutGrid(rawNodes);
                break;
            case 'circular':
                this.layoutCircular(rawNodes);
                break;
            case 'force':
                this.layoutForceDirected(rawNodes);
                break;
            case 'horizontal':
                this.layoutHorizontal(rawNodes);
                break;
            case 'compact':
                this.layoutCompact(rawNodes);
                break;
            default:
                this.nodes = this.layoutNodes(rawNodes, this.edges);
        }

        // Resolve any remaining overlaps
        this.resolveOverlaps();

        this.fitView();
        this.render();
    }

    // Grid layout - tables arranged in a uniform grid
    layoutGrid(rawNodes) {
        const cols = Math.ceil(Math.sqrt(rawNodes.length));
        const rowHeights = [];

        // Calculate max height for each row
        rawNodes.forEach((n, i) => {
            const row = Math.floor(i / cols);
            const h = this.getNodeHeight(n);
            if (!rowHeights[row] || rowHeights[row] < h) {
                rowHeights[row] = h;
            }
        });

        // Position nodes with proper row spacing
        let currentY = 50;
        rawNodes.forEach((n, i) => {
            const col = i % cols;
            const row = Math.floor(i / cols);

            if (col === 0 && row > 0) {
                currentY += rowHeights[row - 1] + 40;
            }

            n.position.x = 50 + col * (this.nodeWidth + 60);
            n.position.y = row === 0 ? 50 : currentY;
        });

        this.nodes = rawNodes.map(n => ({
            ...n,
            hasRelations: this.edges.some(e => e.source === n.id || e.target === n.id)
        }));
    }

    // Circular layout - tables arranged in a circle
    layoutCircular(rawNodes) {
        const count = rawNodes.length;
        if (count === 0) return;

        // Calculate radius based on node count to avoid overlaps
        const minRadius = Math.max(400, count * 60);
        const centerX = minRadius + 100;
        const centerY = minRadius + 100;

        rawNodes.forEach((n, i) => {
            const angle = (2 * Math.PI * i) / count - Math.PI / 2;
            n.position.x = centerX + minRadius * Math.cos(angle) - this.nodeWidth / 2;
            n.position.y = centerY + minRadius * Math.sin(angle) - this.getNodeHeight(n) / 2;
        });

        this.nodes = rawNodes.map(n => ({
            ...n,
            hasRelations: this.edges.some(e => e.source === n.id || e.target === n.id)
        }));
    }

    // Force-directed layout - simulates physical forces
    layoutForceDirected(rawNodes) {
        const count = rawNodes.length;
        if (count === 0) return;

        // Initialize positions in a grid to start
        const cols = Math.ceil(Math.sqrt(count));
        rawNodes.forEach((n, i) => {
            n.position.x = 100 + (i % cols) * (this.nodeWidth + 100);
            n.position.y = 100 + Math.floor(i / cols) * 250;
            n.vx = 0;
            n.vy = 0;
        });

        // Build adjacency for attraction
        const connected = new Map();
        rawNodes.forEach(n => connected.set(n.id, new Set()));
        this.edges.forEach(e => {
            if (connected.has(e.source)) connected.get(e.source).add(e.target);
            if (connected.has(e.target)) connected.get(e.target).add(e.source);
        });

        // Simulation parameters
        const iterations = 100;
        const repulsionStrength = 50000;
        const attractionStrength = 0.01;
        const damping = 0.9;
        const minDistance = this.nodeWidth + 80;

        for (let iter = 0; iter < iterations; iter++) {
            // Apply repulsion between all nodes
            for (let i = 0; i < count; i++) {
                for (let j = i + 1; j < count; j++) {
                    const n1 = rawNodes[i];
                    const n2 = rawNodes[j];

                    const dx = n2.position.x - n1.position.x;
                    const dy = n2.position.y - n1.position.y;
                    const dist = Math.sqrt(dx * dx + dy * dy) || 1;

                    if (dist < minDistance * 3) {
                        const force = repulsionStrength / (dist * dist);
                        const fx = (dx / dist) * force;
                        const fy = (dy / dist) * force;

                        n1.vx -= fx;
                        n1.vy -= fy;
                        n2.vx += fx;
                        n2.vy += fy;
                    }
                }
            }

            // Apply attraction for connected nodes
            this.edges.forEach(e => {
                const n1 = rawNodes.find(n => n.id === e.source);
                const n2 = rawNodes.find(n => n.id === e.target);
                if (!n1 || !n2) return;

                const dx = n2.position.x - n1.position.x;
                const dy = n2.position.y - n1.position.y;
                const dist = Math.sqrt(dx * dx + dy * dy) || 1;

                const idealDist = this.nodeWidth + 150;
                const force = (dist - idealDist) * attractionStrength;
                const fx = (dx / dist) * force;
                const fy = (dy / dist) * force;

                n1.vx += fx;
                n1.vy += fy;
                n2.vx -= fx;
                n2.vy -= fy;
            });

            // Update positions with damping
            rawNodes.forEach(n => {
                n.position.x += n.vx * damping;
                n.position.y += n.vy * damping;
                n.vx *= damping;
                n.vy *= damping;
            });
        }

        // Normalize positions to start from (50, 50)
        let minX = Infinity, minY = Infinity;
        rawNodes.forEach(n => {
            minX = Math.min(minX, n.position.x);
            minY = Math.min(minY, n.position.y);
        });
        rawNodes.forEach(n => {
            n.position.x = n.position.x - minX + 50;
            n.position.y = n.position.y - minY + 50;
        });

        this.nodes = rawNodes.map(n => ({
            ...n,
            hasRelations: this.edges.some(e => e.source === n.id || e.target === n.id)
        }));
    }

    // Horizontal layout - tables arranged in rows
    layoutHorizontal(rawNodes) {
        const maxRowWidth = 1800;
        let currentX = 50;
        let currentY = 50;
        let rowMaxHeight = 0;

        // Sort by connection count - most connected first
        rawNodes.sort((a, b) => {
            const aConns = this.edges.filter(e => e.source === a.id || e.target === a.id).length;
            const bConns = this.edges.filter(e => e.source === b.id || e.target === b.id).length;
            return bConns - aConns;
        });

        rawNodes.forEach(n => {
            const h = this.getNodeHeight(n);

            if (currentX + this.nodeWidth > maxRowWidth && currentX > 50) {
                currentX = 50;
                currentY += rowMaxHeight + 50;
                rowMaxHeight = 0;
            }

            n.position.x = currentX;
            n.position.y = currentY;

            currentX += this.nodeWidth + 60;
            rowMaxHeight = Math.max(rowMaxHeight, h);
        });

        this.nodes = rawNodes.map(n => ({
            ...n,
            hasRelations: this.edges.some(e => e.source === n.id || e.target === n.id)
        }));
    }

    // Compact layout - minimizes space while avoiding overlaps
    layoutCompact(rawNodes) {
        // Sort by connection count and then by column count
        rawNodes.sort((a, b) => {
            const aConns = this.edges.filter(e => e.source === a.id || e.target === a.id).length;
            const bConns = this.edges.filter(e => e.source === b.id || e.target === b.id).length;
            if (bConns !== aConns) return bConns - aConns;
            return (b.data.columns?.length || 0) - (a.data.columns?.length || 0);
        });

        const placed = [];
        const padding = 30;

        rawNodes.forEach(n => {
            const nodeHeight = this.getNodeHeight(n);
            let bestX = 50;
            let bestY = 50;
            let bestScore = Infinity;

            // Try to find the best position that minimizes space usage
            const candidatePositions = [{ x: 50, y: 50 }];

            // Add positions next to and below existing nodes
            placed.forEach(p => {
                const ph = this.getNodeHeight(p);
                candidatePositions.push({ x: p.position.x + this.nodeWidth + padding, y: p.position.y });
                candidatePositions.push({ x: p.position.x, y: p.position.y + ph + padding });
                candidatePositions.push({ x: p.position.x + this.nodeWidth + padding, y: p.position.y + ph + padding });
            });

            for (const pos of candidatePositions) {
                // Check for overlaps
                let hasOverlap = false;
                for (const p of placed) {
                    if (this.checkOverlap(pos.x, pos.y, this.nodeWidth, nodeHeight,
                        p.position.x, p.position.y, this.nodeWidth, this.getNodeHeight(p), padding)) {
                        hasOverlap = true;
                        break;
                    }
                }

                if (!hasOverlap) {
                    // Score based on distance from origin and compactness
                    const score = pos.x + pos.y * 1.5;
                    if (score < bestScore) {
                        bestScore = score;
                        bestX = pos.x;
                        bestY = pos.y;
                    }
                }
            }

            n.position.x = bestX;
            n.position.y = bestY;
            placed.push(n);
        });

        this.nodes = rawNodes.map(n => ({
            ...n,
            hasRelations: this.edges.some(e => e.source === n.id || e.target === n.id)
        }));
    }

    // Check if two rectangles overlap with padding
    checkOverlap(x1, y1, w1, h1, x2, y2, w2, h2, padding = 0) {
        return !(x1 + w1 + padding <= x2 ||
                 x2 + w2 + padding <= x1 ||
                 y1 + h1 + padding <= y2 ||
                 y2 + h2 + padding <= y1);
    }

    // Resolve any overlapping nodes
    resolveOverlaps() {
        const padding = 25;
        const maxIterations = 50;

        for (let iter = 0; iter < maxIterations; iter++) {
            let hasOverlap = false;

            for (let i = 0; i < this.nodes.length; i++) {
                for (let j = i + 1; j < this.nodes.length; j++) {
                    const n1 = this.nodes[i];
                    const n2 = this.nodes[j];
                    const h1 = this.getNodeHeight(n1);
                    const h2 = this.getNodeHeight(n2);

                    if (this.checkOverlap(n1.position.x, n1.position.y, this.nodeWidth, h1,
                        n2.position.x, n2.position.y, this.nodeWidth, h2, padding)) {

                        hasOverlap = true;

                        // Calculate overlap amounts
                        const overlapX = Math.min(
                            n1.position.x + this.nodeWidth + padding - n2.position.x,
                            n2.position.x + this.nodeWidth + padding - n1.position.x
                        );
                        const overlapY = Math.min(
                            n1.position.y + h1 + padding - n2.position.y,
                            n2.position.y + h2 + padding - n1.position.y
                        );

                        // Push apart in the direction with less overlap
                        if (overlapX < overlapY) {
                            const pushX = overlapX / 2 + 5;
                            if (n1.position.x < n2.position.x) {
                                n1.position.x -= pushX;
                                n2.position.x += pushX;
                            } else {
                                n1.position.x += pushX;
                                n2.position.x -= pushX;
                            }
                        } else {
                            const pushY = overlapY / 2 + 5;
                            if (n1.position.y < n2.position.y) {
                                n1.position.y -= pushY;
                                n2.position.y += pushY;
                            } else {
                                n1.position.y += pushY;
                                n2.position.y -= pushY;
                            }
                        }
                    }
                }
            }

            if (!hasOverlap) break;
        }

        // Ensure all nodes have positive coordinates
        let minX = Infinity, minY = Infinity;
        this.nodes.forEach(n => {
            minX = Math.min(minX, n.position.x);
            minY = Math.min(minY, n.position.y);
        });

        if (minX < 50 || minY < 50) {
            const offsetX = minX < 50 ? 50 - minX : 0;
            const offsetY = minY < 50 ? 50 - minY : 0;
            this.nodes.forEach(n => {
                n.position.x += offsetX;
                n.position.y += offsetY;
            });
        }
    }
}

// Initialize
let schemaCanvas;
document.addEventListener('DOMContentLoaded', () => {
    schemaCanvas = new SchemaCanvas('root');
});
