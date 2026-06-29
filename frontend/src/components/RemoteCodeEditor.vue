<template>
  <div class="remote-monaco-shell" @mousedown.stop @click.stop @keydown.stop>
    <div v-show="!loadError" ref="editorMount" class="remote-monaco-mount"></div>
    <div v-if="loading && !loadError" class="remote-editor-loading">{{ loadMessage }}</div>
    <textarea
      v-if="loadError"
      class="remote-editor-textarea remote-editor-fallback"
      :value="modelValue"
      :disabled="disabled"
      :spellcheck="false"
      @input="emit('update:modelValue', $event.target.value)"
      @keydown.ctrl.s.prevent="emit('save')"
      @keydown.meta.s.prevent="emit('save')"
      @focus="emit('focus')"
    ></textarea>
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
});

const emit = defineEmits(['update:modelValue', 'focus', 'save', 'state']);

const editorMount = ref(null);
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

onMounted(() => {
  initializeEditor();
});

onBeforeUnmount(() => {
  disposed = true;
  resizeObserver?.disconnect();
  contentSubscription?.dispose();
  focusSubscription?.dispose();
  if (themeChangeHandler) {
    window.removeEventListener('zshell-theme-change', themeChangeHandler);
  }
  editor?.dispose();
  model?.dispose();
});

watch(
  () => props.modelValue,
  (value) => {
    if (!model || model.getValue() === value) {
      return;
    }
    model.setValue(value || '');
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
    model = monacoApi.editor.createModel(props.modelValue || '', language);
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
      const nextValue = model.getValue();
      if (nextValue !== props.modelValue) {
        emit('update:modelValue', nextValue);
      }
    });
    focusSubscription = editor.onDidFocusEditorWidget(() => emit('focus'));
    editor.addCommand(monacoApi.KeyMod.CtrlCmd | monacoApi.KeyCode.KeyS, () => emit('save'));
    themeChangeHandler = () => monacoLoader?.applyMonacoTheme(monacoApi);
    window.addEventListener('zshell-theme-change', themeChangeHandler);

    resizeObserver = new ResizeObserver(() => editor?.layout());
    resizeObserver.observe(editorMount.value);

    updateLoadingState('编辑器布局完成 4/4', 0.95, 4);
    loading.value = false;
    emit('state', { status: 'ready', message: 'Monaco 已就绪' });
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
</script>
