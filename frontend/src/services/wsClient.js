export function createTerminalClient({
  connectionId,
  onOpen,
  onClose,
  onOutput,
  onError,
}) {
  const backendBase = window.__WISHELL_BACKEND_BASE__ || window.location.origin;
  const backendUrl = new URL(backendBase);
  const protocol = backendUrl.protocol === 'https:' ? 'wss' : 'ws';
  const socketUrl = `${protocol}://${backendUrl.host}/ws/terminal?connectionId=${encodeURIComponent(connectionId)}`;

  const ws = new WebSocket(socketUrl);

  ws.onopen = () => {
    onOpen?.();
  };

  ws.onclose = () => {
    onClose?.();
  };

  ws.onerror = () => {
    onError?.('WS_ERROR', 'websocket transport error');
  };

  ws.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data);
      if (message.type === 'output') {
        const text = message?.data?.text || '';
        const isStderr = Boolean(message?.data?.stderr);
        onOutput?.(text, isStderr);
      } else if (message.type === 'error') {
        const code = message?.data?.code || 'UNKNOWN_ERROR';
        const msg = message?.data?.message || 'unknown error';
        onError?.(code, msg);
      }
    } catch {
      onError?.('WS_DECODE_FAILED', 'invalid websocket message payload');
    }
  };

  return {
    waitUntilOpen() {
      if (ws.readyState === WebSocket.OPEN) {
        return Promise.resolve();
      }

      return new Promise((resolve, reject) => {
        const timeout = window.setTimeout(() => {
          reject(new Error('websocket connect timeout'));
        }, 5000);

        const onReady = () => {
          clearTimeout(timeout);
          ws.removeEventListener('open', onReady);
          ws.removeEventListener('error', onFail);
          resolve();
        };

        const onFail = () => {
          clearTimeout(timeout);
          ws.removeEventListener('open', onReady);
          ws.removeEventListener('error', onFail);
          reject(new Error('websocket connect failed'));
        };

        ws.addEventListener('open', onReady);
        ws.addEventListener('error', onFail);
      });
    },
    sendInput(text) {
      if (ws.readyState !== WebSocket.OPEN) {
        return;
      }
      ws.send(
        JSON.stringify({
          type: 'input',
          data: { text },
        }),
      );
    },
    sendResize(cols, rows) {
      if (ws.readyState !== WebSocket.OPEN) {
        return;
      }
      ws.send(
        JSON.stringify({
          type: 'resize',
          data: { cols, rows },
        }),
      );
    },
    close() {
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close();
      }
    },
  };
}
