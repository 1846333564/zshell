let scheduled = false;
let warmupTimer = null;
let idleHandle = null;

const WARMUP_DELAY_MS = 15000;
const IDLE_TIMEOUT_MS = 30000;

export function scheduleEditorWarmup() {
  if (scheduled || typeof window === 'undefined') {
    return;
  }
  scheduled = true;

  warmupTimer = window.setTimeout(() => {
    warmupTimer = null;
    const run = () => {
      import('../utils/monacoLoader')
        .then((module) => module.preloadMonaco())
        .catch((error) => console.warn('monaco warmup failed', error));
    };

    if (typeof window.requestIdleCallback === 'function') {
      idleHandle = window.requestIdleCallback(run, { timeout: IDLE_TIMEOUT_MS });
      return;
    }
    run();
  }, WARMUP_DELAY_MS);
}

export function cancelEditorWarmup() {
  if (warmupTimer) {
    window.clearTimeout(warmupTimer);
    warmupTimer = null;
  }
  if (idleHandle && typeof window.cancelIdleCallback === 'function') {
    window.cancelIdleCallback(idleHandle);
    idleHandle = null;
  }
}
