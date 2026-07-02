const Charts = {
  // Renders a sleek line chart using SVG
  renderLineChart(containerId, dataPoints, labels) {
    const container = document.getElementById(containerId);
    if (!container) return;

    // Clear previous contents
    container.innerHTML = '';

    if (!dataPoints || dataPoints.length === 0) {
      container.innerHTML = '<div style="color: var(--text-muted); text-align: center; padding: 2rem;">No transaction data available yet</div>';
      return;
    }

    const width = container.clientWidth || 500;
    const height = 220;
    const padding = 30;

    const maxVal = Math.max(...dataPoints, 5); // Default min scale
    const minVal = 0;

    // Generate points
    const points = dataPoints.map((val, idx) => {
      const x = padding + (idx / (dataPoints.length - 1 || 1)) * (width - padding * 2);
      const y = height - padding - ((val - minVal) / (maxVal - minVal)) * (height - padding * 2);
      return { x, y, value: val, label: labels[idx] };
    });

    let pathD = `M ${points[0].x} ${points[0].y} `;
    for (let i = 1; i < points.length; i++) {
      // Smooth curve calculation (bezier control points)
      const cpX1 = points[i - 1].x + (points[i].x - points[i - 1].x) / 2;
      const cpY1 = points[i - 1].y;
      const cpX2 = points[i - 1].x + (points[i].x - points[i - 1].x) / 2;
      const cpY2 = points[i].y;
      pathD += `C ${cpX1} ${cpY1}, ${cpX2} ${cpY2}, ${points[i].x} ${points[i].y} `;
    }

    let areaD = `${pathD} L ${points[points.length - 1].x} ${height - padding} L ${points[0].x} ${height - padding} Z`;

    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.setAttribute('width', '100%');
    svg.setAttribute('height', '100%');
    svg.setAttribute('viewBox', `0 0 ${width} ${height}`);
    svg.style.overflow = 'visible';

    // Gradients
    svg.innerHTML = `
      <defs>
        <linearGradient id="chart-glow" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="var(--accent-cyan)" stop-opacity="0.3"/>
          <stop offset="100%" stop-color="var(--accent-cyan)" stop-opacity="0"/>
        </linearGradient>
      </defs>
    `;

    // Draw Grid Lines (horizontal)
    const gridCount = 4;
    for (let i = 0; i <= gridCount; i++) {
      const y = padding + (i / gridCount) * (height - padding * 2);
      const value = Math.round(maxVal - (i / gridCount) * (maxVal - minVal));
      
      // Line
      const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
      line.setAttribute('x1', padding);
      line.setAttribute('y1', y);
      line.setAttribute('x2', width - padding);
      line.setAttribute('y2', y);
      line.setAttribute('stroke', 'var(--border-color)');
      line.setAttribute('stroke-dasharray', '4 4');
      line.setAttribute('stroke-width', '0.5');
      svg.appendChild(line);

      // Label
      const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
      text.setAttribute('x', padding - 8);
      text.setAttribute('y', y + 4);
      text.setAttribute('fill', 'var(--text-muted)');
      text.setAttribute('font-size', '10');
      text.setAttribute('text-anchor', 'end');
      text.textContent = value;
      svg.appendChild(text);
    }

    // Fill Area under curve
    const areaPath = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    areaPath.setAttribute('d', areaD);
    areaPath.setAttribute('fill', 'url(#chart-glow)');
    svg.appendChild(areaPath);

    // Draw Line curve
    const linePath = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    linePath.setAttribute('d', pathD);
    linePath.setAttribute('fill', 'none');
    linePath.setAttribute('stroke', 'var(--accent-cyan)');
    linePath.setAttribute('stroke-width', '2');
    svg.appendChild(linePath);

    // Draw Interactive dots & X-axis labels
    points.forEach((pt, idx) => {
      // Circle dot
      const dot = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
      dot.setAttribute('cx', pt.x);
      dot.setAttribute('cy', pt.y);
      dot.setAttribute('r', '4');
      dot.setAttribute('fill', 'var(--bg-card)');
      dot.setAttribute('stroke', 'var(--accent-cyan)');
      dot.setAttribute('stroke-width', '2');
      
      // Interactive tooltip title
      const title = document.createElementNS('http://www.w3.org/2000/svg', 'title');
      title.textContent = `${pt.label}: ${pt.value} calls`;
      dot.appendChild(title);
      svg.appendChild(dot);

      // Label under X-axis
      if (idx === 0 || idx === points.length - 1 || points.length <= 7 || idx % Math.round(points.length / 5) === 0) {
        const xText = document.createElementNS('http://www.w3.org/2000/svg', 'text');
        xText.setAttribute('x', pt.x);
        xText.setAttribute('y', height - 8);
        xText.setAttribute('fill', 'var(--text-muted)');
        xText.setAttribute('font-size', '10');
        xText.setAttribute('text-anchor', 'middle');
        xText.textContent = pt.label;
        svg.appendChild(xText);
      }
    });

    container.appendChild(svg);
  }
};
