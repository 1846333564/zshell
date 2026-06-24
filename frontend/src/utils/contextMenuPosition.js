export function viewportContextMenuPosition(event, options = {}) {
  const width = Number(options.width) || 220;
  const height = Number(options.height) || 320;
  const margin = Number(options.margin) || 8;
  const maxX = Math.max(margin, window.innerWidth - width - margin);
  const maxY = Math.max(margin, window.innerHeight - height - margin);

  return {
    x: Math.min(Math.max(margin, event.clientX), maxX),
    y: Math.min(Math.max(margin, event.clientY), maxY),
  };
}
