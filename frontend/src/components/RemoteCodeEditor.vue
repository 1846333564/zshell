<template>
  <div class="remote-monaco-shell" @mousedown.stop @click.stop @keydown.stop>
    <div v-show="!streaming && !loadError" ref="editorMount" class="remote-monaco-mount"></div>
    <pre
      v-show="streaming"
      ref="streamPreview"
      class="remote-editor-stream-preview"
      tabindex="0"
      aria-label="远程文件下载预览"
      aria-live="off"
      @focus="emit('focus')"
    ></pre>
    <textarea
      v-show="!streaming && (loading || loadError)"
      ref="fallbackTextarea"
      class="remote-editor-textarea remote-editor-fallback"
      :readonly="disabled"
      :spellcheck="false"
      wrap="off"
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
  streaming: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['update:modelValue', 'focus', 'save', 'state']);

const editorMount = ref(null);
const streamPreview = ref(null);
const fallbackTextarea = ref(null);
const loading = ref(true);
const loadError = ref('');
const loadMessage = ref('准备加载编辑器...');

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
let streamPreviewChunkIndex = 0;

onMounted(() => {
  syncFallbackValue(props.modelValue || '');
  drainStreamPreviewChunks();
  initializeEditor();
});

onBeforeUnmount(() => {
  disposed = true;
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
    syncFallbackValue(nextValue);
    syncModelValue(nextValue);
  },
);

watch(
  () => props.appendVersion,
  () => {
    drainStreamPreviewChunks();
  },
  { immediate: true, flush: 'sync' },
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
    if (!value || !editor || props.streaming) {
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

watch(
  () => props.streaming,
  async (streaming) => {
    if (streaming) {
      drainStreamPreviewChunks();
      if (monacoApi && model) {
        monacoApi.editor.setModelLanguage(model, 'plaintext');
      }
      return;
    }

    drainStreamPreviewChunks();
    syncFallbackValue(props.modelValue || '');
    syncModelValue(props.modelValue || '');
    updateModelLanguage();
    await nextTick();
    if (disposed || props.streaming || !editor) {
      return;
    }
    editor.layout();
    editor.render(true);
    if (props.active) {
      editor.focus();
    }
    clearHiddenFallback();
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
    if (disposed || !editorMount.value) {
      return;
    }

    const language = props.streaming
      ? 'plaintext'
      : monacoLoader.detectMonacoLanguage(monacoApi, props.path);
    const initialValue = props.streaming ? '' : props.modelValue || '';
    appliedModelValue = initialValue;
    model = monacoApi.editor.createModel(initialValue, language);
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
    emit('state', { status: 'ready', message: 'Monaco 已就绪' });
    if (!props.streaming) {
      syncModelValue(props.modelValue || '');
      await nextTick();
      editor.layout();
      editor.render(true);
      clearHiddenFallback();
    }
    if (props.active && !props.streaming) {
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
  if (props.streaming || !monacoApi || !monacoLoader || !model) {
    return;
  }
  const language = monacoLoader.detectMonacoLanguage(monacoApi, props.path);
  monacoApi.editor.setModelLanguage(model, language);
}

function drainStreamPreviewChunks() {
  const preview = streamPreview.value;
  const chunks = props.appendChunks || [];
  if (!preview || streamPreviewChunkIndex >= chunks.length) {
    return false;
  }
  const endIndex = chunks.length;
  const parts = [];
  for (let index = streamPreviewChunkIndex; index < endIndex; index += 1) {
    const part = String(chunks[index] || '');
    if (part) {
      parts.push(part);
    }
  }
  const text = parts.join('');
  if (!text) {
    streamPreviewChunkIndex = endIndex;
    return true;
  }

  const scrollTop = preview.scrollTop;
  const scrollLeft = preview.scrollLeft;
  try {
    preview.appendChild(document.createTextNode(text));
  } catch {
    return false;
  }
  preview.scrollTop = scrollTop;
  preview.scrollLeft = scrollLeft;
  streamPreviewChunkIndex = endIndex;
  return true;
}

function syncFallbackValue(value) {
  const textarea = fallbackTextarea.value;
  if (!textarea || props.streaming) {
    return;
  }
  if (loading.value || loadError.value) {
    textarea.value = value || '';
  }
}

function handleFallbackInput(event) {
  if (props.streaming || props.disabled) {
    return;
  }
  const value = event.target.value;
  appliedModelValue = value;
  emit('update:modelValue', value);
}

function syncModelValue(value) {
  if (!model) {
    return;
  }
  const nextValue = value || '';
  if (appliedModelValue === nextValue && model.getValueLength() === nextValue.length) {
    return;
  }
  applyingExternalValue = true;
  try {
    model.setValue(nextValue);
    appliedModelValue = nextValue;
    editor?.render(true);
  } finally {
    applyingExternalValue = false;
  }
}

function clearHiddenFallback() {
  if (props.streaming || loading.value || loadError.value) {
    return;
  }
  if (fallbackTextarea.value) {
    fallbackTextarea.value.value = '';
  }
  if (streamPreview.value) {
    streamPreview.value.replaceChildren();
  }
}
</script>
