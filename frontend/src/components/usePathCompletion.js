import { computed, ref } from 'vue';
import {
  PATH_COMPLETION_VISIBLE_LIMIT,
  findDirectoryCompletion,
  joinDirectoryPath,
  resolveDirectoryCompletionQuery,
  withTrailingDirectorySlash,
} from './pathCompletion.js';

export function usePathCompletion(options) {
  const indexRevision = ref(0);
  const open = ref(false);
  let lookupTimer = null;
  let lookupController = null;
  let lookupRevision = 0;
  let navigationDepth = 0;

  const query = computed(() => {
    indexRevision.value;
    return resolveDirectoryCompletionQuery(options.pathDraft.value, options.currentPath.value);
  });
  const completion = computed(() => completionForQuery(query.value));
  const visible = computed(
    () => open.value && Boolean(query.value?.fragment) && completion.value.total > 0,
  );
  const summary = computed(() => {
    const currentQuery = query.value;
    const currentCompletion = completion.value;
    if (!currentQuery || currentCompletion.total === 0) return '';
    if (currentCompletion.total === 1) {
      return `Tab 进入 ${withTrailingDirectorySlash(currentCompletion.items[0].path)}`;
    }
    if (currentCompletion.commonPrefix.length > currentQuery.fragment.length) {
      return `Tab 补全到 ${joinDirectoryPath(currentQuery.parent, currentCompletion.commonPrefix)}`;
    }
    if (currentCompletion.exact && currentCompletion.soleLonger) {
      return `Enter 确认 ${currentCompletion.exact.path} · Tab 进入 ${withTrailingDirectorySlash(currentCompletion.soleLonger.path)}`;
    }
    return `${currentCompletion.total} 个匹配 · 继续输入以缩小范围`;
  });

  function completionForQuery(currentQuery) {
    if (!currentQuery?.fragment || !options.connectionId()) {
      return findDirectoryCompletion([], '', PATH_COMPLETION_VISIBLE_LIMIT);
    }
    return findDirectoryCompletion(
      options.indexForParent(currentQuery.parent),
      currentQuery.fragment,
      PATH_COMPLETION_VISIBLE_LIMIT,
    );
  }

  async function ensureCompletionCache(value) {
    clearLookup();
    const currentQuery = resolveDirectoryCompletionQuery(value, options.currentPath.value);
    if (
      !currentQuery?.parent ||
      !options.connectionId() ||
      options.hasCache(currentQuery.parent)
    ) {
      return;
    }
    try {
      await options.loadDirectory(currentQuery.parent, { rememberTree: true });
    } catch {}
  }

  function scheduleLookup(value) {
    clearLookup();
    if (!options.connectionId()) return;
    const currentQuery = resolveDirectoryCompletionQuery(value, options.currentPath.value);
    if (!currentQuery?.parent || options.hasCache(currentQuery.parent)) return;
    const revision = (lookupRevision += 1);
    lookupTimer = window.setTimeout(async () => {
      lookupTimer = null;
      const controller = new AbortController();
      lookupController = controller;
      try {
        await options.loadDirectory(currentQuery.parent, {
          signal: controller.signal,
          rememberTree: true,
        });
      } catch {}
      if (lookupRevision === revision) lookupController = null;
    }, 45);
  }

  function clearLookup() {
    lookupRevision += 1;
    if (lookupTimer) {
      window.clearTimeout(lookupTimer);
      lookupTimer = null;
    }
    if (lookupController) {
      lookupController.abort();
      lookupController = null;
    }
  }

  async function navigate(path, trailingSlash) {
    if (!path) return;
    open.value = false;
    navigationDepth += 1;
    try {
      const resolvedPath = await options.navigate(path);
      const finalPath = resolvedPath || path;
      options.pathDraft.value = trailingSlash
        ? withTrailingDirectorySlash(finalPath)
        : finalPath;
    } finally {
      navigationDepth -= 1;
    }
  }

  async function commit(event) {
    if (event?.type === 'change' && navigationDepth > 0) return;
    if (!options.connectionId()) return;
    const draft = options.pathDraft.value.trim();
    if (!draft) {
      options.pathDraft.value = options.currentPath.value;
      open.value = false;
      return;
    }
    await ensureCompletionCache(draft);
    const currentQuery = resolveDirectoryCompletionQuery(draft, options.currentPath.value);
    const currentCompletion = completionForQuery(currentQuery);
    const target = currentCompletion.exact?.path || currentQuery?.target;
    if (target && target === options.currentPath.value) {
      open.value = false;
      if (event?.type === 'keydown') {
        options.pathDraft.value = withTrailingDirectorySlash(target);
      }
      return;
    }
    if (target) await navigate(target, true);
  }

  async function complete() {
    if (!options.connectionId()) return;
    const draft = options.pathDraft.value;
    await ensureCompletionCache(draft);
    const currentQuery = resolveDirectoryCompletionQuery(draft, options.currentPath.value);
    const currentCompletion = completionForQuery(currentQuery);
    if (!currentQuery?.fragment || currentCompletion.total === 0) return;

    if (currentCompletion.total === 1) {
      await navigate(currentCompletion.items[0].path, true);
      return;
    }

    if (currentCompletion.commonPrefix.length > currentQuery.fragment.length) {
      const completedPath = joinDirectoryPath(
        currentQuery.parent,
        currentCompletion.commonPrefix,
      );
      options.pathDraft.value = completedPath;
      const commonCompletion = findDirectoryCompletion(
        options.indexForParent(currentQuery.parent),
        currentCompletion.commonPrefix,
        1,
      );
      if (commonCompletion.exact?.path === completedPath) {
        await navigate(completedPath, false);
      } else {
        open.value = true;
      }
      return;
    }

    if (currentCompletion.exact && currentCompletion.soleLonger) {
      await navigate(currentCompletion.soleLonger.path, true);
    }
  }

  function onInput(event) {
    options.pathDraft.value = event.target.value || '';
    options.onInput?.();
    open.value = Boolean(options.pathDraft.value.trim());
    scheduleLookup(options.pathDraft.value);
  }

  async function openItem(item) {
    if (item?.path) await navigate(item.path, true);
  }

  function dismiss() {
    open.value = false;
  }

  function reset() {
    clearLookup();
    open.value = false;
    notifyIndexChanged();
  }

  function notifyIndexChanged() {
    indexRevision.value += 1;
  }

  return {
    completion,
    open,
    visible,
    summary,
    ensureCompletionCache,
    complete,
    commit,
    onInput,
    openItem,
    dismiss,
    reset,
    dispose: clearLookup,
    notifyIndexChanged,
    itemLabel: (item) => (item?.name ? `/${item.name}` : ''),
    matches: (value) => {
      const currentQuery = resolveDirectoryCompletionQuery(value, options.currentPath.value);
      return completionForQuery(currentQuery).items.map((item) => item.path);
    },
    candidates: () =>
      new Set(options.cachedDirectories().flatMap((directory) => [
        directory.path,
        ...(directory.directoryIndex || []).map((item) => item.path),
      ])),
    target: (value) =>
      resolveDirectoryCompletionQuery(value, options.currentPath.value)?.target || '',
    parent: (value) =>
      resolveDirectoryCompletionQuery(value, options.currentPath.value)?.parent || '',
    commonPrefix: (value) =>
      completionForQuery(
        resolveDirectoryCompletionQuery(value, options.currentPath.value),
      ).commonPrefix,
  };
}
