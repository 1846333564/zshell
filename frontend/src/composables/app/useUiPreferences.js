import { ref } from 'vue';
import { getUIPreferences, saveUIPreferences } from '../../services/apiClient';

export function useUiPreferences() {
  const uiScale = ref(1);
  const terminalFontSize = ref(14);
  let saveUiScaleTimer = null;

  function handleGlobalKeydown(event) {
    if (!event.ctrlKey || isTerminalEvent(event)) {
      return;
    }

    const key = event.key.toLowerCase();
    if (key === '+' || key === '=') {
      adjustUiScale(0.05);
      event.preventDefault();
      return;
    }
    if (key === '-' || key === '_') {
      adjustUiScale(-0.05);
      event.preventDefault();
      return;
    }
    if (key === '0') {
      setUiScale(1, true);
      event.preventDefault();
    }
  }

  function isTerminalEvent(event) {
    const target = event.target;
    if (!(target instanceof Element)) {
      return false;
    }
    return Boolean(target.closest('.terminal-wrap'));
  }

  function adjustUiScale(delta) {
    setUiScale(uiScale.value + delta, true);
  }

  function resetUiScale() {
    setUiScale(1, true);
  }

  function setUiScale(value, persist = false) {
    uiScale.value = clampUiScale(value);
    applyUiScale();
    if (persist) {
      scheduleSaveUiScale();
    }
  }

  function clampUiScale(value) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return 1;
    }
    return Math.min(1.35, Math.max(0.82, Number(parsed.toFixed(2))));
  }

  function applyUiScale() {
    const scale = uiScale.value;
    document.documentElement.style.setProperty('--ui-scale', scale.toFixed(2));
    document.documentElement.style.setProperty('--ui-scale-inverse', (1 / scale).toFixed(6));
  }

  async function loadUIPreferences() {
    try {
      const result = await getUIPreferences();
      setUiScale(result?.preferences?.uiScale || 1, false);
      terminalFontSize.value = clampTerminalFontSize(result?.preferences?.terminalFontSize || 14);
    } catch (error) {
      console.warn('load ui preferences failed', error);
    }
  }

  function scheduleSaveUiScale() {
    scheduleSavePreferences();
  }

  function handleTerminalFontSizeChange(value) {
    terminalFontSize.value = clampTerminalFontSize(value);
    scheduleSavePreferences();
  }

  function clampTerminalFontSize(value) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return 14;
    }
    return Math.min(28, Math.max(10, Math.round(parsed)));
  }

  function scheduleSavePreferences() {
    if (saveUiScaleTimer) {
      window.clearTimeout(saveUiScaleTimer);
    }
    saveUiScaleTimer = window.setTimeout(async () => {
      saveUiScaleTimer = null;
      try {
        await saveUIPreferences({
          uiScale: uiScale.value,
          terminalFontSize: terminalFontSize.value,
        });
      } catch (error) {
        console.warn('save ui preferences failed', error);
      }
    }, 260);
  }

  function cleanupUiPreferences() {
    if (saveUiScaleTimer) {
      window.clearTimeout(saveUiScaleTimer);
    }
  }

  return {
    uiScale,
    terminalFontSize,
    handleGlobalKeydown,
    resetUiScale,
    applyUiScale,
    loadUIPreferences,
    handleTerminalFontSizeChange,
    cleanupUiPreferences,
  };
}
