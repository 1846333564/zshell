<template>
  <div class="remote-monaco-shell" @mousedown.stop @click.stop @keydown.stop>
    <div v-show="!loadError" ref="editorMount" class="remote-monaco-mount"></div>
    <textarea
      v-show="loading || loadError"
      class="remote-editor-textarea remote-editor-fallback"
      :value="fallbackContent"
      :disabled="disabled"
      :spellcheck="false"
      @input="handleFallbackInput"
      @keydown.ctrl.s.prevent="emit('save')"
      @keydown.meta.s.prevent="emit('save')"
      @focus="emit('focus')"
    ></textarea>
    <div v-if="loading && !loadError" class="remote-editor-loading">{{ loadMessage }}</div>
  </div>
</template>

<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';

const props = defineProps({
  modelValue: {
    type: String,
    default: '',
  },
  path: {
    type: String,
    default: '',
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  active: {
    type: Boolean,
    default: false,
  },
  appendChunks: {
    type: Array,
    default: () => [],
  },
  appendVersion: {
    type: Number,
    default: 0,
  },
});

const emit = defineEmits(['update:modelValue', 'focus', 'save', 'state']);

const EDITOR_APPEND_FRAME_CHARS = 32768;
const EDITOR_APPEND_FRAME_CHUNKS = 1;

const editorMount = ref(null);
const loading = ref(true);
const loadError = ref('');
const loadMessage = ref('准备加载编辑器...');
const fallbackContent = ref('');

let monacoApi = null;
let monacoLoader = null;
let editor = null;
let model = null;
let contentSubscription = null;
let focusSubscription = null;
let resizeObserver = null;
let themeChangeHandler = null;
let disposed = false;
let applyingExternalValue = false;
let appliedModelValue = '';
let appliedModelLength = 0;
let appendChunkIndex = 0;
let appendFrame = null;

onMounted(() => {
  initializeEditor();
});

onBeforeUnmount(() => {
  disposed = true;
  cancelAppendFrame();
  resizeObserver?.disconnect();
  contentSubscription?.dispose();
  focusSubscription?.dispose();
  if (themeChangeHandler) {
    window.removeEventListener('wiShell-theme-change', themeChangeHandler);
  }
  editor?.dispose();
  model?.dispose();
});

watch(
  () => props.modelValue,
  (value) => {
    const nextValue = value || '';
    if (!model) {
      if (nextValue || !(props.appendChunks || []).length) {
        fallbackContent.value = nextValue;
        appliedModelValue = nextValue;
        appliedModelLength = nextValue.length;
        if (nextValue) {
          appendChunkIndex = (props.appendChunks || []).length;
        }
      }
      return;
    }
    applyingExternalValue = true;
    try {
      if (nextValue.length >= appliedModelLength && appendChunkIndex < (props.appendChunks || []).length) {
        scheduleAppendChunks();
        return;
      }
      const currentValue = appliedModelValue || model.getValue();
      if (currentValue === nextValue) {
        appliedModelValue = nextValue;
        appliedModelLength = nextValue.length;
        return;
      }
      if (nextValue.startsWith(currentValue)) {
        appendModelText(nextValue.slice(currentValue.length));
      } else {
        model.setValue(nextValue);
      }
      appliedModelValue = nextValue;
      appliedModelLength = nextValue.length;
    } finally {
      applyingExternalValue = false;
    }
  },
);

watch(
  () => props.appendVersion,
  () => {
    scheduleAppendChunks();
  },
);

watch(
  () => props.disabled,
  (value) => {
    editor?.updateOptions({
      readOnly: value,
      domReadOnly: value,
    });
  },
);

watch(
  () => props.active,
  async (value) => {
    if (!value || !editor) {
      return;
    }
    await nextTick();
    editor.layout();
    editor.focus();
  },
);

watch(
  () => props.path,
  () => {
    updateModelLanguage();
  },
);

async function initializeEditor() {
  loading.value = true;
  loadError.value = '';
  updateLoadingState('准备加载编辑器...', 0.1, 1);

  try {
    updateLoadingState('加载编辑器模块 1/4', 0.25, 1);
    monacoLoader = await import('../utils/monacoLoader');
    updateLoadingState('加载 Monaco Worker 2/4', 0.5, 2);
    monacoApi = await monacoLoader.loadMonaco();
    if (disposed) {
      return;
    }

    updateLoadingState('创建编辑器实例 3/4', 0.78, 3);
    await nextTick();
    if (!editorMount.value) {
      return;
    }

    const language = monacoLoader.detectMonacoLanguage(monacoApi, props.path);
    const hasStreamChunks = (props.appendChunks || []).length > 0;
    appendChunkIndex = hasStreamChunks ? 0 : (props.appendChunks || []).length;
    appliedModelValue = hasStreamChunks ? '' : props.modelValue || '';
    appliedModelLength = appliedModelValue.length;
    fallbackContent.value = appliedModelValue;
    model = monacoApi.editor.createModel(appliedModelValue, language);
    editor = monacoApi.editor.create(editorMount.value, {
      model,
      theme: monacoLoader.MONACO_THEME,
      automaticLayout: true,
      readOnly: props.disabled,
      domReadOnly: props.disabled,
      fontFamily: "'JetBrains Mono', Consolas, 'Courier New', monospace",
      fontSize: 13,
      lineHeight: 20,
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      smoothScrolling: true,
      tabSize: 2,
      insertSpaces: true,
      detectIndentation: true,
      wordWrap: 'off',
      renderWhitespace: 'selection',
      bracketPairColorization: { enabled: true },
      guides: {
        indentation: true,
        bracketPairs: true,
      },
      fixedOverflowWidgets: true,
      contextmenu: true,
      find: {
        addExtraSpaceOnTop: false,
        autoFindInSelection: 'never',
      },
    });

    contentSubscription = editor.onDidChangeModelContent(() => {
      if (applyingExternalValue) {
        return;
      }
      const nextValue = model.getValue();
      appliedModelValue = nextValue;
      appliedModelLength = nextValue.length;
      fallbackContent.value = nextValue;
      if (nextValue !== props.modelValue) {
        emit('update:modelValue', nextValue);
      }
    });
    focusSubscription = editor.onDidFocusEditorWidget(() => emit('focus'));
    editor.addCommand(monacoApi.KeyMod.CtrlCmd | monacoApi.KeyCode.KeyS, () => emit('save'));
    themeChangeHandler = () => monacoLoader?.applyMonacoTheme(monacoApi);
    window.addEventListener('wiShell-theme-change', themeChangeHandler);

    resizeObserver = new ResizeObserver(() => editor?.layout());
    resizeObserver.observe(editorMount.value);

    loading.value = false;
    updateLoadingState('编辑器布局完成 4/4', 0.95, 4);
    if (!scheduleAppendChunks()) {
      emit('state', { status: 'ready', message: 'Monaco 已就绪' });
    }
    if (props.active) {
      editor.focus();
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Monaco 加载失败';
    loading.value = false;
    loadError.value = message;
    emit('state', { status: 'error', message });
  }
}

function updateLoadingState(message, progress, step) {
  loadMessage.value = message;
  emit('state', {
    status: 'loading',
    message,
    progress,
    step,
    totalSteps: 4,
  });
}

function updateModelLanguage() {
  if (!monacoApi || !monacoLoader || !model) {
    return;
  }
  const language = monacoLoader.detectMonacoLanguage(monacoApi, props.path);
  monacoApi.editor.setModelLanguage(model, language);
}

function scheduleAppendChunks() {
  if (appendFrame) {
    return true;
  }
  if (appendChunkIndex >= (props.appendChunks || []).length) {
    return false;
  }
  emitAppendState();
  const run = () => {
    appendFrame = null;
    applyPendingAppendChunks();
    if (appendChunkIndex < (props.appendChunks || []).length) {
      scheduleAppendChunks();
    } else if (!loading.value && !loadError.value) {
      emit('state', { status: 'ready', message: 'Monaco 已就绪' });
    }
  };
  if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
    appendFrame = window.requestAnimationFrame(run);
    return true;
  }
  appendFrame = window.setTimeout(run, 16);
  return true;
}

function cancelAppendFrame() {
  if (!appendFrame) {
    return;
  }
  if (typeof window !== 'undefined' && typeof window.cancelAnimationFrame === 'function') {
    window.cancelAnimationFrame(appendFrame);
  } else {
    window.clearTimeout(appendFrame);
  }
  appendFrame = null;
}

function applyPendingAppendChunks() {
  const chunks = props.appendChunks || [];
  if (appendChunkIndex >= chunks.length) {
    return;
  }
  let text = '';
  let chunkCount = 0;
  while (
    appendChunkIndex < chunks.length &&
    chunkCount < EDITOR_APPEND_FRAME_CHUNKS &&
    (text.length < EDITOR_APPEND_FRAME_CHARS || chunkCount === 0)
  ) {
    text += String(chunks[appendChunkIndex] || '');
    appendChunkIndex += 1;
    chunkCount += 1;
  }
  if (!text) {
    return;
  }
  if (!model || loadError.value) {
    fallbackContent.value += text;
  }
  applyingExternalValue = true;
  try {
    if (model) {
      appendModelText(text);
    }
    appliedModelValue = '';
    appliedModelLength += text.length;
  } finally {
    applyingExternalValue = false;
  }
  emitAppendState();
}

function emitAppendState() {
  const total = (props.appendChunks || []).length;
  if (loading.value || loadError.value || total <= 0 || appendChunkIndex >= total) {
    return;
  }
  emit('state', {
    status: 'rendering',
    message: '渲染内容中',
    progress: appendChunkIndex / total,
  });
}

function handleFallbackInput(event) {
  const value = event.target.value;
  fallbackContent.value = value;
  appliedModelValue = value;
  appliedModelLength = value.length;
  emit('update:modelValue', value);
}

function appendModelText(text) {
  if (!text || !monacoApi || !model) {
    return;
  }
  const line = model.getLineCount();
  const column = model.getLineMaxColumn(line);
  model.applyEdits([
    {
      range: new monacoApi.Range(line, column, line, column),
      text,
      forceMoveMarkers: true,
    },
  ]);
}
</script>
