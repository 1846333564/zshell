import { reactive, ref } from 'vue';
import { getUIPreferences, saveUIPreferences } from '../../services/apiClient';
import {
  CUSTOM_THEME_KEY,
  DEFAULT_THEME_KEY,
  THEME_COLOR_FIELDS,
  THEME_OPTIONS,
  applyThemeToDocument,
  createDefaultCustomTheme,
  normalizeCustomTheme,
  normalizeThemeKey,
  resolveTheme,
} from '../../theme';

export function useUiPreferences() {
  const uiScale = ref(1);
  const terminalFontSize = ref(14);
  const themeKey = ref(DEFAULT_THEME_KEY);
  const customTheme = ref(createDefaultCustomTheme());
  const activeTheme = ref(resolveTheme(themeKey.value, customTheme.value));
  const gpuAccelerationEnabled = ref(true);
  const gpuPreferenceSaving = ref(false);
  const gpuRestartRequired = ref(false);
  const themeDialog = reactive({
    visible: false,
    draftKey: DEFAULT_THEME_KEY,
    draftCustomTheme: createDefaultCustomTheme(),
    saving: false,
    error: '',
  });
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

  function applyCurrentTheme() {
    activeTheme.value = resolveTheme(themeKey.value, customTheme.value);
    applyThemeToDocument(activeTheme.value);
  }

  async function loadUIPreferences() {
    try {
      const result = await getUIPreferences();
      const preferences = result?.preferences || {};
      setUiScale(preferences.uiScale || 1, false);
      terminalFontSize.value = clampTerminalFontSize(preferences.terminalFontSize || 14);
      themeKey.value = normalizeThemeKey(preferences.themeKey || DEFAULT_THEME_KEY);
      customTheme.value = normalizeCustomTheme(preferences.customTheme);
      gpuAccelerationEnabled.value = preferences.gpuAccelerationEnabled !== false;
      applyCurrentTheme();
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

  async function toggleGpuAcceleration() {
    if (gpuPreferenceSaving.value) {
      return;
    }
    const previous = gpuAccelerationEnabled.value;
    gpuAccelerationEnabled.value = !previous;
    gpuPreferenceSaving.value = true;
    try {
      await savePreferencesNow();
      gpuRestartRequired.value = true;
      window.alert(`GPU 渲染已${gpuAccelerationEnabled.value ? '开启' : '关闭'}，重启 wiShell 后生效。`);
    } catch (error) {
      gpuAccelerationEnabled.value = previous;
      window.alert(error instanceof Error ? error.message : 'GPU 渲染设置保存失败');
    } finally {
      gpuPreferenceSaving.value = false;
    }
  }

  function showThemeDialog() {
    themeDialog.visible = true;
    themeDialog.draftKey = themeKey.value;
    themeDialog.draftCustomTheme = { ...customTheme.value };
    themeDialog.error = '';
  }

  function cancelThemeDialog() {
    themeDialog.visible = false;
    themeDialog.error = '';
    applyCurrentTheme();
  }

  function selectThemeOption(value) {
    themeDialog.draftKey = normalizeThemeKey(value);
    previewThemeDraft();
  }

  function setCustomThemeColor(key, value) {
    if (!THEME_COLOR_FIELDS.some((field) => field.key === key)) {
      return;
    }
    themeDialog.draftKey = CUSTOM_THEME_KEY;
    themeDialog.draftCustomTheme = {
      ...themeDialog.draftCustomTheme,
      [key]: value,
    };
    previewThemeDraft();
  }

  function resetCustomTheme() {
    themeDialog.draftKey = CUSTOM_THEME_KEY;
    themeDialog.draftCustomTheme = createDefaultCustomTheme();
    previewThemeDraft();
  }

  async function saveThemeDialog() {
    themeDialog.saving = true;
    themeDialog.error = '';
    const nextKey = normalizeThemeKey(themeDialog.draftKey);
    const nextCustomTheme = normalizeCustomTheme(themeDialog.draftCustomTheme);
    const prevKey = themeKey.value;
    const prevCustomTheme = { ...customTheme.value };

    themeKey.value = nextKey;
    customTheme.value = nextCustomTheme;
    applyCurrentTheme();

    try {
      await savePreferencesNow();
      themeDialog.visible = false;
    } catch (error) {
      themeKey.value = prevKey;
      customTheme.value = prevCustomTheme;
      applyCurrentTheme();
      themeDialog.error = error instanceof Error ? error.message : '主题保存失败';
    } finally {
      themeDialog.saving = false;
    }
  }

  function previewThemeDraft() {
    applyThemeToDocument(resolveTheme(themeDialog.draftKey, themeDialog.draftCustomTheme));
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
        await savePreferencesNow();
      } catch (error) {
        console.warn('save ui preferences failed', error);
      }
    }, 260);
  }

  function savePreferencesNow() {
    return saveUIPreferences({
      uiScale: uiScale.value,
      terminalFontSize: terminalFontSize.value,
      themeKey: themeKey.value,
      customTheme: customTheme.value,
      gpuAccelerationEnabled: gpuAccelerationEnabled.value,
    });
  }

  function cleanupUiPreferences() {
    if (saveUiScaleTimer) {
      window.clearTimeout(saveUiScaleTimer);
    }
  }

  return {
    uiScale,
    terminalFontSize,
    themeKey,
    customTheme,
    activeTheme,
    gpuAccelerationEnabled,
    gpuPreferenceSaving,
    gpuRestartRequired,
    themeDialog,
    themeOptions: THEME_OPTIONS,
    themeColorFields: THEME_COLOR_FIELDS,
    handleGlobalKeydown,
    resetUiScale,
    applyUiScale,
    applyCurrentTheme,
    loadUIPreferences,
    handleTerminalFontSizeChange,
    toggleGpuAcceleration,
    showThemeDialog,
    cancelThemeDialog,
    selectThemeOption,
    setCustomThemeColor,
    resetCustomTheme,
    saveThemeDialog,
    cleanupUiPreferences,
  };
}
