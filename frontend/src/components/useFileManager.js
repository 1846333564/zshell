import {
  computed as e,
  markRaw as markRawChunks,
  nextTick as t,
  onBeforeUnmount as n,
  onMounted as o,
  reactive as r,
  ref as a,
  watch as i,
} from "vue";
import {
  archiveRemoteItemsUrl as s,
  backendDownloadUrl as l,
  deleteRemoteItems as c,
  downloadRemoteFile as u,
  downloadRemoteItems as d,
  listRemoteFiles as f,
  readRemoteTextFileWithProgress as m,
  renameRemoteItem,
  saveRemoteTextFile as h,
  transferRemoteItems as p,
  uploadRemoteItems as v,
} from "../services/apiClient";
import { viewportContextMenuPosition as y } from "../utils/contextMenuPosition";
import {
  CLIPBOARD_ACTIONS as D,
  CLIPBOARD_KEY as M,
  DEFAULT_FILE_OPEN_ACTION as b,
  DIRECTORY_CACHE_LIMIT,
  FILE_COLUMNS as S,
  FILE_OPEN_ACTIONS as I,
  HOME_PATH as w,
  MAX_PRELOAD_CONCURRENCY,
  PRELOAD_BATCH_DELAY_MS,
  PRELOAD_TARGET_LIMIT,
  ROOT_PATH as g,
  WORK_MODE_START_PATHS as P,
  buildBreadcrumbs as Vn,
  comparePaths as Yn,
  delay as qn,
  displayPath as Xn,
  downloadArchiveName as jn,
  formatSize as Gn,
  formatTime as Zn,
  initialPathForMode as Xe,
  isSameOrChildPath as Un,
  normalizePath as On,
  normalizeWorkMode as qe,
  parentPath as Wn,
  pathDepth as Hn,
} from "./fileManagerUtils";
import {
  buildTreeRows,
  replacePathPrefix,
  treeContextTarget,
  virtualSlice,
} from "./fileTreeModel";
import {
  buildDirectoryCompletionIndex,
} from "./pathCompletion";
import { usePathCompletion } from "./usePathCompletion";

const FILE_ROW_HEIGHT = 29;
const FILE_HEADER_HEIGHT = 33;
const TREE_ROW_HEIGHT = 27;
const VIRTUAL_OVERSCAN = 8;
const EDITOR_PREVIEW_PUBLISH_INTERVAL_MS = 1000;
const x = new Map();
function E(E) {
  const C = a([]),
    T = a(Xe(E.workMode)),
    B = a(T.value),
    A = a(!1),
    $ = a(!1),
    k = a(!1),
    R = a(!1),
    z = a(""),
    N = a(!1),
    _ = a(new Set()),
    L = a(-1),
    F = a(Bn()),
    O = a(null),
    W = a(null),
    H = a(null),
    U = a(null),
    K = a(""),
    Y = a(new Map()),
    V = a(!1),
    X = r(Ln()),
    j = r({ key: "name", direction: "asc" }),
    q = a(""),
    G = r({
      visible: !1,
      expanded: !1,
      status: "idle",
      targetPath: "",
      files: [],
      directoryCount: 0,
      totalBytes: 0,
      loadedBytes: 0,
      speed: 0,
      startedAt: 0,
      message: "",
    }),
    Z = a([]),
    J = a(""),
    Q = a(Ge(T.value)),
    treeSelectedPath = a(T.value),
    fileListViewport = a(null),
    pathTreeViewport = a(null),
    contextMenuElement = a(null),
    fileScrollTop = a(0),
    treeScrollTop = a(0),
    fileViewportHeight = a(520),
    treeViewportHeight = a(520),
    renameState = r({
      path: "",
      originalName: "",
      name: "",
      isDir: !1,
      surface: "",
      saving: !1,
      error: "",
    });
  let ee = null,
    te = null,
    ne = null,
    oe = 900,
    re = 0,
    navigationRevision = 0,
    ae = 0,
    ie = null,
    se = null,
    le = [],
    ce = null,
    ue = null,
    preloadFailures = new Set(),
    directoryLoadController = null,
    viewportResizeObserver = null,
    activeRenameInput = null,
    contextMenuPositionRevision = 0;
  const editorReadControllers = new Map();
  const de = r({
      visible: !1,
      x: 0,
      y: 0,
      entry: null,
      targetPath: "",
      targetKind: "blank",
      source: "blank",
    }),
    fe = r({ x: 0, y: 0, width: 320, height: 260 });
  const pathCompletionModel = usePathCompletion({
      connectionId: () => E.connectionId,
      currentPath: T,
      pathDraft: B,
      hasCache: (path) => hasCachedDirectory(E.connectionId, path),
      indexForParent: (path) => rawCachedDirectoryIndex(E.connectionId, path),
      cachedDirectories: () => St(E.connectionId),
      loadDirectory: (path, options) => We(path, options),
      navigate: async (path) => {
        V.value = !1;
        on();
        await yt(path);
        return On(T.value) || On(path);
      },
      onInput: () => {
        V.value = !1;
      },
    }),
    pathCompletion = pathCompletionModel.completion,
    pathCompletionOpen = pathCompletionModel.open,
    pathCompletionVisible = pathCompletionModel.visible,
    pathCompletionSummary = pathCompletionModel.summary;
  i(
    () => [E.connectionId, E.workMode],
    async ([e, t]) => {
      on(), ot();
      const n = Xe(t);
      if ((Ve(n), !e))
        return (C.value = []), (T.value = n), void (z.value = "");
      await Oe(n);
    },
    { immediate: !0 },
  ),
    i(T, (e) => {
      B.value = e;
      pathCompletionOpen.value = !1;
    }),
    i(fileListViewport, (element, previous) => observeViewport(element, previous)),
    i(pathTreeViewport, (element, previous) => observeViewport(element, previous));
  const me = e(() => [...C.value].sort(ke)),
    entryByPath = e(() => new Map(C.value.map((entry) => [entry.path, entry]))),
    he = e(() =>
      Array.from(_.value)
        .map((path) => entryByPath.value.get(path))
        .filter(Boolean),
    ),
    pe = e(() => he.value.length > 0),
    ve = e(() => Boolean(E.connectionId && F.value?.items?.length)),
    ye = e(() => de.entry),
    ge = e(() => "directory" === de.targetKind),
    we = e(() => de.entry?.path || de.targetPath || je()),
    Pe = e(() => ("directory" === de.targetKind ? we.value : de.targetPath || je())),
    Me = e(() => {
      if (de.entry) {
        return { ...de.entry, path: On(de.entry.path), isDir: Boolean(de.entry.isDir) };
      }
      const path = On(de.targetPath);
      return "directory" === de.targetKind && path
        ? { path, name: Xn(path), isDir: !0 }
        : null;
    }),
    contextCanMutate = e(() => Boolean(Me.value && $t(Me.value.path))),
    be = e(() => contextCanMutate.value),
    De = e(() => S.map((e) => `${X[e.key]}px`).join(" ")),
    xe = e(() => Z.value.find((e) => e.id === J.value) || null),
    Se = e(() =>
      Math.min(
        MAX_PRELOAD_CONCURRENCY,
        Math.max(1, Math.round(Number(E.hardware?.cpuThreads) || 1)),
      ),
    ),
    Ie = e(() =>
      E.connectionId
        ? k.value
          ? `\u4e0a\u4f20 ${Te.value}% \xb7 ${Tn(G.speed)}`
          : $.value
            ? `\u7f13\u5b58 ${me.value.length} \u9879 \xb7 \u6b63\u5728\u66f4\u65b0`
            : A.value
              ? "\u8bfb\u53d6\u4e2d..."
              : F.value?.items?.length
                ? `${"move" === F.value.action ? "\u526a\u5207" : "\u590d\u5236"}\u4e86 ${F.value.items.length} \u9879`
                : `${me.value.length} \u9879`
        : "\u672a\u8fde\u63a5",
    ),
    Ee = e(() => Vn(T.value)),
    Ce = e(() => ({
      left: `${fe.x}px`,
      top: `${fe.y}px`,
      width: `${fe.width}px`,
      height: `${fe.height}px`,
    })),
    Te = e(() =>
      "done" === G.status
        ? 100
        : G.totalBytes <= 0
          ? 0
          : Math.min(
              100,
              Math.max(0, Math.round((G.loadedBytes / G.totalBytes) * 100)),
            ),
    ),
    Be = e(() =>
      "done" === G.status
        ? "\u4e0a\u4f20\u5b8c\u6210"
        : "error" === G.status
          ? "\u4e0a\u4f20\u5931\u8d25"
          : `\u4e0a\u4f20 ${G.files.length + G.directoryCount} \u9879`,
    ),
    Ae = e(() =>
      "done" === G.status
        ? "\u5df2\u5b8c\u6210\uff0c\u7a0d\u540e\u81ea\u52a8\u6298\u53e0"
        : "error" === G.status
          ? "\u8bf7\u68c0\u67e5\u8fde\u63a5\u6216\u8fdc\u7a0b\u76ee\u5f55\u6743\u9650"
          : `\u603b\u8fdb\u5ea6 ${Te.value}%`,
    ),
    $e = e(() =>
      Array.from(Y.value.entries())
        .map(([e, t]) => ({
          path: e,
          count: Number(t?.count) || 0,
          lastVisited: Number(t?.lastVisited) || 0,
        }))
        .sort((e, t) =>
          e.count !== t.count
            ? e.count - t.count
            : e.lastVisited !== t.lastVisited
              ? e.lastVisited - t.lastVisited
              : Yn(e.path, t.path),
        )
        .map((e) => e.path),
    );
  function ke(e, t) {
    const n = "desc" === j.direction ? -1 : 1;
    if ("type" !== j.key && e.isDir !== t.isDir) return e.isDir ? -1 : 1;
    const o = Re(e, t, j.key);
    return 0 !== o ? o * n : Re(e, t, "name");
  }
  function Re(e, t, n) {
    switch (n) {
      case "type":
        return e.isDir !== t.isDir ? (e.isDir ? -1 : 1) : ze(e.name, t.name);
      case "size":
        return Ne(
          e.isDir ? -1 : Number(e.size) || 0,
          t.isDir ? -1 : Number(t.size) || 0,
        );
      case "modTime":
        return Ne(Date.parse(e.modTime) || 0, Date.parse(t.modTime) || 0);
      case "mode":
        return ze(e.mode, t.mode);
      case "owner":
        return ze(e.owner, t.owner);
      default:
        return ze(e.name, t.name);
    }
  }
  function ze(e, t) {
    return String(e || "").localeCompare(String(t || ""), void 0, {
      sensitivity: "base",
      numeric: !0,
    });
  }
  function Ne(e, t) {
    return e === t ? 0 : e < t ? -1 : 1;
  }
  function _e(e) {
    Le(),
      (q.value = e),
      (ue = window.setTimeout(() => {
        (q.value = ""), (ue = null);
      }, 220));
  }
  function Le() {
    ue && (window.clearTimeout(ue), (ue = null));
  }
  const Fe = e(() => buildTreeRows(Q.value)),
    treeLayout = e(() => {
      const index = new Map();
      let contentWidth = 280;
      Fe.value.forEach((row, rowIndex) => {
        contentWidth = Math.max(contentWidth, Number(row.contentWidth) || 0);
        index.set(row.path, rowIndex);
      });
      return { index, contentWidth };
    }),
    treeRowIndex = e(() => treeLayout.value.index),
    fileRange = e(() =>
      virtualSlice(
        me.value.length,
        fileScrollTop.value,
        fileViewportHeight.value,
        FILE_ROW_HEIGHT,
        VIRTUAL_OVERSCAN,
      ),
    ),
    visibleEntries = e(() =>
      me.value.slice(fileRange.value.start, fileRange.value.end).map((entry, offset) => ({
        entry,
        index: fileRange.value.start + offset,
      })),
    ),
    treeRange = e(() =>
      virtualSlice(
        Fe.value.length,
        treeScrollTop.value,
        treeViewportHeight.value,
        TREE_ROW_HEIGHT,
        VIRTUAL_OVERSCAN,
      ),
    ),
    visibleTreeNodes = e(() =>
      Fe.value.slice(treeRange.value.start, treeRange.value.end).map((node, offset) => ({
        ...node,
        virtualIndex: treeRange.value.start + offset,
      })),
    ),
    fileVirtualHeight = e(() => me.value.length * FILE_ROW_HEIGHT),
    treeVirtualHeight = e(() => Fe.value.length * TREE_ROW_HEIGHT),
    treeContentWidth = e(() => treeLayout.value.contentWidth),
    fileContentWidth = e(() =>
      S.reduce((total, column) => total + Number(X[column.key] || column.width), 0) +
      8 * Math.max(0, S.length - 1) +
      16,
    );
  async function Oe(e = T.value || Xe(E.workMode), t = {}) {
    if (!E.connectionId) return;
    !1 !== t.cancelPreload && ot();
    directoryLoadController?.abort();
    const controller = new AbortController();
    directoryLoadController = controller;
    const n = e || Xe(E.workMode),
      o = (re += 1),
      r = !1 === t.useCache ? null : Ze(E.connectionId, n);
    preloadFailures.delete(Qe(E.connectionId, n));
    r
      ? (Ye(r.path, r.entries, r.requestedPath || n),
        nt(),
        (A.value = !1),
        ($.value = !0))
      : ((A.value = !0), ($.value = !1)),
      (z.value = "");
    try {
      const e = await We(n, { rememberTree: !0, signal: controller.signal });
      if (o !== re) return;
      Ye(e.resolvedPath, e.entries, n), nt();
    } catch (e) {
      if ("AbortError" === e?.name || controller.signal.aborted) return;
      o === re &&
        (z.value =
          e instanceof Error
            ? e.message
            : "\u8bfb\u53d6\u76ee\u5f55\u5931\u8d25");
    } finally {
      directoryLoadController === controller && (directoryLoadController = null);
      o === re && ((A.value = !1), ($.value = !1));
    }
  }
  async function We(e, t = {}) {
    const n = await f(E.connectionId, e, { signal: t.signal }),
      o = n.path || e,
      r = Array.isArray(n.entries) ? n.entries : [];
    return (
      Je(E.connectionId, e, o, r, { rememberTree: t.rememberTree }),
      o !== e && Je(E.connectionId, o, o, r, { rememberTree: !1 }),
      { requestedPath: e, resolvedPath: o, entries: r }
    );
  }
  async function He(e = T.value || Xe(E.workMode)) {
    if (!E.connectionId) return;
    ot();
    directoryLoadController?.abort();
    directoryLoadController = null;
    const t = (re += 1),
      n = e || T.value || Xe(E.workMode),
      o = On(n),
      r = Ue(n);
    let a = null,
      i = null;
    const s = [];
    (A.value = !0), ($.value = !1), (z.value = "");
    try {
      for (let e = 0; e < r.length; e += 8) {
        const n = r.slice(e, e + 8),
          l = await Promise.all(
            n.map(async (e) => {
              try {
                return await We(e, { rememberTree: !0 });
              } catch (t) {
                return { requestedPath: e, error: t };
              }
            }),
          );
        if (t !== re) return;
        for (const e of l)
          e.error
            ? (s.push(e), On(e.requestedPath) === o && (i = e.error))
            : Ke(e, o) && (a = e);
        e + 8 < r.length && (await qn(80));
      }
      if (i) throw i;
      if (a) Ye(a.resolvedPath, a.entries, a.requestedPath);
      else {
        const e = Ze(E.connectionId, n);
        e && Ye(e.path, e.entries, e.requestedPath || n);
      }
      s.length > 0 &&
        (z.value = `${s.length} \u4e2a\u5df2\u6253\u5f00\u76ee\u5f55\u5237\u65b0\u5931\u8d25`),
        nt();
    } catch (e) {
      t === re &&
        (z.value =
          e instanceof Error
            ? e.message
            : "\u5237\u65b0\u76ee\u5f55\u5931\u8d25");
    } finally {
      t === re && ((A.value = !1), ($.value = !1));
    }
  }
  function Ue(e) {
    const t = new Set(),
      n = [],
      o = (e) => {
        const o = On(e);
        o && !t.has(o) && (t.add(o), n.push(o));
      };
    o(e), o(T.value);
    for (const [e, t] of Q.value.entries()) t?.opened && o(e);
    return n;
  }
  function Ke(e, t) {
    return (
      !(!t || !e) && (On(e.requestedPath) === t || On(e.resolvedPath) === t)
    );
  }
  function Ye(e, entries, n = e) {
    const pathChanged = On(T.value) !== On(e);
    T.value = e;
    treeSelectedPath.value = e;
    C.value = et(entries);
    mt(e);
    ut(g);
    ut(w);
    (n === w || n.startsWith("~/")) && ut(w, { opened: !0 });
    tt(e, C.value);
    rn();
    if (pathChanged) {
      fileScrollTop.value = 0;
      t(() => {
        if (fileListViewport.value) fileListViewport.value.scrollTop = 0;
      });
    }
  }
  function Ve(e = Xe(E.workMode)) {
    ct(),
      pathCompletionModel.reset(),
      preloadFailures.clear(),
      (Q.value = Ge(e)),
      (treeSelectedPath.value = On(e)),
      (Y.value = new Map()),
      (V.value = !1),
      (pathCompletionOpen.value = !1);
  }
  function je() {
    return T.value || Xe(E.workMode);
  }
  function Ge(e) {
    const t = new Map([
        [g, { opened: !1, collapsed: !1 }],
        [w, { opened: !1, collapsed: !1 }],
      ]),
      n = On(e);
    return n && !t.has(n) && t.set(n, { opened: !1, collapsed: !1 }), t;
  }
  function Ze(e, t) {
    const n = x.get(Qe(e, t));
    return n ? { ...n, entries: et(n.entries) } : null;
  }
  function rawCachedDirectoryEntries(e, t) {
    return x.get(Qe(e, t))?.entries || null;
  }
  function rawCachedDirectoryIndex(e, t) {
    return x.get(Qe(e, t))?.directoryIndex || [];
  }
  function hasCachedDirectory(e, t) {
    return x.has(Qe(e, t));
  }
  function Je(e, t, n, o, r = {}) {
    const a = Qe(e, t);
    if (!a) return;
    const i = On(n || t),
      s = et(o);
    for (
      x.has(a) && x.delete(a),
        x.set(a, {
          requestedPath: t,
          path: i,
          entries: s,
          directoryIndex: buildDirectoryCompletionIndex(s),
          cachedAt: Date.now(),
        }),
        pathCompletionModel.notifyIndexChanged(),
        !1 !== r.rememberTree && tt(i, s);
      x.size > DIRECTORY_CACHE_LIMIT;

    ) {
      const e = x.keys().next().value;
      x.delete(e);
    }
  }
  function Qe(e, t) {
    const n = On(t || Xe(E.workMode));
    return e && n ? `${e}\0${n}` : "";
  }
  function et(e) {
    return Array.isArray(e) ? e.map((e) => ({ ...e })) : [];
  }
  function tt(e, t) {
    const parent = On(e),
      childDirectories = t
        .filter((entry) => entry.isDir)
        .map((entry) => On(entry.path))
        .filter(Boolean),
      paths = new Map(Q.value);
    dt(paths, parent);
    const collapsed = Boolean(paths.get(parent)?.collapsed);
    removeStaleTreeChildren(paths, parent, new Set(childDirectories));
    ft(paths, parent, {
      opened: !0,
      collapsed,
      listingKnown: !0,
      totalEntryCount: t.length,
      childDirPaths: childDirectories,
    });
    Q.value = paths;
    it(childDirectories);
  }
  function removeStaleTreeChildren(paths, parent, validChildren) {
    const staleRoots = [];
    for (const path of paths.keys()) {
      if (Wn(path) === parent && !validChildren.has(path)) staleRoots.push(path);
    }
    if (staleRoots.length === 0) return;
    for (const path of Array.from(paths.keys())) {
      if (staleRoots.some((root) => path === root || path.startsWith(`${root}/`))) paths.delete(path);
    }
  }
  function nt() {
    ot();
    if (0 === at().length) return;
    const e = (ae += 1),
      t = new AbortController();
    ie = t;
    const n = async () => {
      try {
        for (; ae === e && !t.signal.aborted; ) {
          const n = at();
          if (0 === n.length) break;
          const o = [...n],
            r = Math.min(Se.value, o.length),
            a = Array.from({ length: r }, async () => {
              for (; o.length > 0 && ae === e && !t.signal.aborted; ) {
                const n = o.shift();
                if (!n || hasCachedDirectory(E.connectionId, n)) continue;
                try {
                  await We(n, { signal: t.signal, rememberTree: !1 });
                  preloadFailures.delete(Qe(E.connectionId, n));
                } catch (o) {
                  if ("AbortError" === o?.name || t.signal.aborted) return;
                  preloadFailures.add(Qe(E.connectionId, n));
                }
              }
            });
          await Promise.allSettled(a);
          if (ae !== e || t.signal.aborted) return;
          await qn(PRELOAD_BATCH_DELAY_MS);
        }
      } finally {
        ae === e && (ie = null);
      }
    };
    n();
  }
  function ot() {
    (ae += 1), rt(), ie && (ie.abort(), (ie = null));
  }
  function rt() {
    se && (window.clearTimeout(se), (se = null));
  }
  function at() {
    const e = [],
      t = new Set(),
      n = [],
      o = On(T.value);
    if (!E.connectionId || !o) return e;
    n.push(o), t.add(o);
    for (; n.length > 0 && e.length < PRELOAD_TARGET_LIMIT; ) {
      const o = n.shift(),
        r = Ze(E.connectionId, o);
      if (!r) continue;
      for (const a of r.entries) {
        if (!a.isDir) continue;
        const i = On(a.path),
          s = Qe(E.connectionId, i);
        if (!i || t.has(i)) continue;
        t.add(i);
        if (hasCachedDirectory(E.connectionId, i)) {
          n.push(i);
          continue;
        }
        preloadFailures.has(s) ||
          (e.push(i), e.length >= PRELOAD_TARGET_LIMIT && (n.length = 0));
        if (e.length >= PRELOAD_TARGET_LIMIT) break;
      }
    }
    return e;
  }
  function it(e) {
    const t = e.map((e) => On(e)).filter((e) => e && !Q.value.has(e));
    0 !== t.length && (le.push(...t), st());
  }
  function st(e = 16) {
    ce || 0 === le.length || (ce = window.setTimeout(lt, e));
  }
  function lt() {
    if (((ce = null), 0 === le.length)) return;
    const e = le.splice(0, 60),
      t = new Map(Q.value);
    for (const n of e) dt(t, n), ft(t, n, { opened: !1 });
    (Q.value = t), le.length > 0 && st();
  }
  function ct() {
    (le = []), ce && (window.clearTimeout(ce), (ce = null));
  }
  function ut(e, t = {}) {
    const n = On(e);
    if (!n) return;
    const o = new Map(Q.value);
    ft(o, n, t), (Q.value = o);
  }
  function dt(e, t) {
    const n = On(t);
    if (!n) return;
    let o = n;
    const r = [];
    for (; o; ) r.unshift(o), (o = Wn(o));
    for (const t of r) ft(e, t);
  }
  function ft(e, t, n = {}) {
    const o = On(t);
    if (!o) return;
    const r = e.get(o) || { opened: !1, collapsed: !1 };
    e.set(o, { ...r, ...n });
  }
  function mt(e) {
    const t = On(e);
    if (!t) return;
    const n = new Map(Y.value),
      o = n.get(t) || { count: 0, lastVisited: 0 };
    n.set(t, { count: Number(o.count) + 1, lastVisited: Date.now() }),
      pt(n),
      (Y.value = n);
  }
  function ht() {
    const e = H.value;
    if (!e) return;
    const t = e.getBoundingClientRect(),
      n = Math.min(560, Math.max(340, window.innerWidth - 24)),
      o = Math.max(120, window.innerHeight - t.bottom - 12),
      r = Math.max(120, t.top - 12),
      a = Math.min(
        420,
        Math.max(180, Math.min(Math.max(o, r), 38 * $e.value.length + 42)),
      );
    (fe.x = Math.min(
      Math.max(8, t.left),
      Math.max(8, window.innerWidth - n - 8),
    )),
      (fe.y = o >= a ? t.bottom + 8 : Math.max(8, t.top - a - 8)),
      (fe.width = n),
      (fe.height = a);
  }
  function pt(e) {
    if (e.size <= 80) return;
    const t = Array.from(e.entries()).sort((e, t) => {
      const n = e[1] || {},
        o = t[1] || {},
        r = (Number(o.count) || 0) - (Number(n.count) || 0);
      return 0 !== r
        ? r
        : (Number(o.lastVisited) || 0) - (Number(n.lastVisited) || 0);
    });
    for (const [n] of t.slice(80)) e.delete(n);
  }
  function vt() {
    t(() => {
      const e = U.value;
      e && (e.scrollTop = e.scrollHeight);
    });
  }
  function yt(e) {
    const path = On(e);
    if (!path) return;
    navigationRevision += 1;
    revealTreePath(path);
    V.value = !1;
    on();
    return Oe(path);
  }
  async function activateTreeNode(node) {
    const path = On(node?.path || node);
    if (!path) return;
    const isCurrent = On(T.value) === path;
    const canToggle = Boolean(node?.hasChildren);

    if (canToggle && !node.collapsed) {
      const paths = new Map(Q.value);
      ft(paths, path, { opened: !0, collapsed: !0 });
      Q.value = paths;
      if (!isCurrent) {
        await yt(path);
        return;
      }
      revealTreePath(path);
      V.value = !1;
      on();
      t(() => scrollTreePathIntoView(path));
      return;
    }

    if (canToggle && node.collapsed) {
      const paths = new Map(Q.value);
      ft(paths, path, { opened: !0, collapsed: !1 });
      Q.value = paths;
      if (isCurrent) {
        V.value = !1;
        on();
        await expandDirectory(path);
        return;
      }
    }

    if (!canToggle && isCurrent) {
      revealTreePath(path);
      V.value = !1;
      on();
      return;
    }
    await yt(path);
  }
  function revealTreePath(value) {
    const path = On(value);
    if (!path) return;
    treeSelectedPath.value = path;
    let needsMetaUpdate = !Q.value.has(path);
    let ancestor = Wn(path);
    while (ancestor) {
      const meta = Q.value.get(ancestor);
      if (!meta || meta.collapsed) {
        needsMetaUpdate = !0;
        break;
      }
      ancestor = Wn(ancestor);
    }
    if (needsMetaUpdate) {
      const paths = new Map(Q.value);
      dt(paths, path);
      ancestor = Wn(path);
      while (ancestor) {
        ft(paths, ancestor, { collapsed: !1 });
        ancestor = Wn(ancestor);
      }
      Q.value = paths;
    }
    t(() => scrollTreePathIntoView(path));
  }
  function scrollTreePathIntoView(value) {
    const path = On(value);
    const viewport = pathTreeViewport.value;
    const index = treeRowIndex.value.get(path);
    if (!viewport || !Number.isInteger(index)) return;
    const top = index * TREE_ROW_HEIGHT;
    const bottom = top + TREE_ROW_HEIGHT;
    const viewportTop = viewport.scrollTop;
    const viewportBottom = viewportTop + viewport.clientHeight;
    if (top < viewportTop) viewport.scrollTop = top;
    else if (bottom > viewportBottom) viewport.scrollTop = Math.max(0, bottom - viewport.clientHeight);
    treeScrollTop.value = viewport.scrollTop;
  }
  function syncFileViewport(element = fileListViewport.value) {
    if (!element) return;
    fileScrollTop.value = Math.max(0, element.scrollTop - FILE_HEADER_HEIGHT);
    fileViewportHeight.value = Math.max(FILE_ROW_HEIGHT, element.clientHeight - FILE_HEADER_HEIGHT);
  }
  function syncTreeViewport(element = pathTreeViewport.value) {
    if (!element) return;
    treeScrollTop.value = Math.max(0, element.scrollTop);
    treeViewportHeight.value = Math.max(TREE_ROW_HEIGHT, element.clientHeight);
  }
  function observeViewport(element, previous) {
    previous && viewportResizeObserver?.unobserve(previous);
    element && viewportResizeObserver?.observe(element);
    element === fileListViewport.value && syncFileViewport(element);
    element === pathTreeViewport.value && syncTreeViewport(element);
  }
  function St(e) {
    if (!e) return [];
    const t = `${e}\0`,
      n = [];
    for (const [e, o] of x.entries())
      e.startsWith(t) && n.push({ ...o, entries: et(o.entries) });
    return n;
  }
  function It(e) {
    const t = e
      .map((e) => ({ path: On(e.path), isDir: Boolean(e.isDir) }))
      .filter((e) => e.path);
    if (0 === t.length) return;
    const n = `${E.connectionId}\0`;
    let cacheChanged = !1;
    for (const [e, o] of Array.from(x.entries())) {
      if (!e.startsWith(n)) continue;
      const r = On(o.path || o.requestedPath);
      if (t.some((e) => e.isDir && Un(r, e.path))) {
        x.delete(e);
        cacheChanged = !0;
        continue;
      }
      const a = et(o.entries).filter((e) => !t.some((t) => Ct(e, t)));
      a.length !== o.entries.length &&
        (x.set(e, {
          ...o,
          entries: a,
          directoryIndex: buildDirectoryCompletionIndex(a),
          cachedAt: Date.now(),
        }),
        (cacheChanged = !0));
    }
    cacheChanged && pathCompletionModel.notifyIndexChanged();
    (C.value = C.value.filter((e) => !t.some((t) => Ct(e, t)))), Tt(t);
  }
  function Et(e) {
    const t = On(e);
    if (!t || !E.connectionId) return;
    const n = [Qe(E.connectionId, t)];
    for (const e of n)
      e && x.delete(e) && pathCompletionModel.notifyIndexChanged();
  }
  function Ct(e, t) {
    const n = On(e.path);
    return !(!n || !t.path) && (t.isDir ? Un(n, t.path) : n === t.path);
  }
  function Tt(e) {
    const t = new Map(Q.value);
    for (const n of Array.from(t.keys()))
      e.some((e) => (e.isDir ? Un(n, e.path) : n === e.path)) && t.delete(n);
    Q.value = t;
  }
  function Bt(e) {
    const t = On(T.value) || Xe(E.workMode);
    for (const n of e) {
      const o = On(n.path);
      if (n.isDir && Un(t, o)) return At(o, e);
    }
    return t;
  }
  function At(e, t) {
    let n = Wn(e) || g;
    for (; n && t.some((e) => e.isDir && Un(n, On(e.path))); ) n = Wn(n);
    return n || g;
  }
  function $t(e) {
    const t = On(e);
    return Boolean(t && t !== g && t !== w);
  }
  function kt(e = Pe.value) {
    (K.value = e || je()), O.value?.click();
  }
  function Rt(e = Pe.value) {
    (K.value = e || je()), W.value?.click();
  }
  async function zt(e, t = [], n = je()) {
    const o = e.map((e) => ({
      file: e,
      relativePath: e.webkitRelativePath || e.name,
    }));
    await Nt(o, t, n);
  }
  async function Nt(e, t = [], n = je()) {
    if (!E.connectionId || (0 === e.length && 0 === t.length)) return;
    const targetPath = On(n || je());
    (k.value = !0), (z.value = ""), Mn(e, t, targetPath);
    let o = !1;
    try {
      await v(E.connectionId, targetPath, e, t, bn),
        (o = !0),
        xn(),
        Et(targetPath),
        await refreshAffectedDirectories([targetPath]);
    } catch (e) {
      const errorMessage = e instanceof Error ? e.message : "\u4e0a\u4f20\u5931\u8d25";
      Et(targetPath);
      await refreshAffectedDirectories([targetPath]);
      (z.value = errorMessage), Sn(errorMessage);
    } finally {
      (k.value = !1), o && In();
    }
  }
  async function _t(e, t) {
    if (E.connectionId) {
      z.value = "";
      try {
        await u(E.connectionId, e, t);
      } catch (e) {
        z.value = e instanceof Error ? e.message : "\u4e0b\u8f7d\u5931\u8d25";
      }
    }
  }
  async function downloadContextItem() {
    const item = Me.value ? { ...Me.value } : null;
    sn();
    if (!item || !E.connectionId) return;
    z.value = "";
    try {
      if (item.isDir) {
        const label = item.path === g ? "root" : item.path === w ? "home" : item.name || Xn(item.path) || "directory";
        await d(E.connectionId, [item.path], `${label}.zip`);
      } else {
        await _t(item.path, item.name);
      }
    } catch (error) {
      z.value = error instanceof Error ? error.message : "\u4e0b\u8f7d\u5931\u8d25";
    }
  }
  async function refreshTargetDirectory(value) {
    const path = On(value);
    if (!path || !E.connectionId) return;
    if (path === On(T.value)) {
      await Oe(path, { useCache: !1 });
      return;
    }
    A.value = !0;
    z.value = "";
    try {
      await We(path, { rememberTree: !0 });
    } catch (error) {
      z.value = error instanceof Error ? error.message : "\u5237\u65b0\u76ee\u5f55\u5931\u8d25";
    } finally {
      A.value = !1;
    }
  }
  async function refreshAffectedDirectories(values) {
    const paths = Array.from(
      new Set((Array.isArray(values) ? values : [values]).map((value) => On(value)).filter(Boolean)),
    );
    if (!E.connectionId || paths.length === 0) return { refreshedCurrent: !1, failures: [] };
    const currentPath = On(T.value);
    const currentNavigationRevision = navigationRevision;
    let currentListing = null;
    let refreshedCurrent = !1;
    const failures = [];
    const results = await Promise.allSettled(
      paths.map((path) => We(path, { rememberTree: !0 })),
    );
    results.forEach((result, index) => {
      if (result.status === "rejected") {
        failures.push({ path: paths[index], error: result.reason });
        return;
      }
      if (Ke(result.value, currentPath)) currentListing = result.value;
    });
    if (
      currentListing &&
      On(T.value) === currentPath &&
      navigationRevision === currentNavigationRevision
    ) {
      Ye(currentListing.resolvedPath, currentListing.entries, currentListing.requestedPath);
      nt();
      refreshedCurrent = !0;
    }
    if (failures.length > 0) {
      z.value = `操作已完成，但 ${failures.length} 个目录刷新失败`;
    }
    return { refreshedCurrent, failures };
  }
  async function expandDirectory(value) {
    const path = On(value);
    if (!path || !E.connectionId) return;
    revealTreePath(path);
    const paths = new Map(Q.value);
    ft(paths, path, { opened: !0, collapsed: !1 });
    Q.value = paths;
    const cached = Ze(E.connectionId, path);
    if (cached) {
      tt(cached.path || path, cached.entries);
      t(() => scrollTreePathIntoView(path));
      return;
    }
    A.value = !0;
    z.value = "";
    try {
      await We(path, { rememberTree: !0 });
      t(() => scrollTreePathIntoView(path));
    } catch (error) {
      z.value = error instanceof Error ? error.message : "\u5c55\u5f00\u76ee\u5f55\u5931\u8d25";
    } finally {
      A.value = !1;
    }
  }
  function startRename(item = Me.value, surface = "") {
    if (!item || !$t(item.path)) return;
    sn();
    renameState.path = On(item.path);
    renameState.originalName = item.name || Xn(item.path);
    renameState.name = renameState.originalName;
    renameState.isDir = Boolean(item.isDir);
    renameState.surface = surface || (entryByPath.value.has(item.path) ? "list" : "tree");
    renameState.saving = !1;
    renameState.error = "";
    item.isDir && revealTreePath(item.path);
  }
  function setRenameInputRef(element, path) {
    if (!element || On(path) !== renameState.path || element === activeRenameInput) return;
    activeRenameInput = element;
    t(() => {
      if (renameState.path === On(path) && document.contains(element)) {
        element.focus();
        element.select();
      }
    });
  }
  function updateRenameName(event) {
    renameState.name = event?.target?.value ?? "";
  }
  function cancelRename() {
    renameState.path = "";
    renameState.originalName = "";
    renameState.name = "";
    renameState.isDir = !1;
    renameState.surface = "";
    renameState.saving = !1;
    renameState.error = "";
    activeRenameInput = null;
  }
  async function commitRename() {
    if (!renameState.path || renameState.saving) return;
    const oldPath = renameState.path;
    const oldName = renameState.originalName;
    const isDir = renameState.isDir;
    const newName = String(renameState.name || "").trim();
    if (newName === oldName) return void cancelRename();
    if (!newName || newName === "." || newName === ".." || newName.includes("/") || newName.includes("\0")) {
      renameState.error = "\u540d\u79f0\u4e0d\u5408\u6cd5";
      z.value = renameState.error;
      return;
    }
    renameState.saving = !0;
    renameState.error = "";
    z.value = "";
    try {
      const response = await renameRemoteItem(E.connectionId, oldPath, newName);
      const result = response?.item || response?.rename || response || {};
      const parent = Wn(oldPath);
      const fallbackPath = parent === g ? `${g}${newName}` : `${parent}/${newName}`;
      const newPath = On(result.newPath || result.path || result.targetPath || fallbackPath);
      const originalCurrentPath = On(T.value);
      const currentPath = replacePathPrefix(originalCurrentPath, oldPath, newPath);
      const selectedTreePath = replacePathPrefix(treeSelectedPath.value, oldPath, newPath);

      for (const editor of Z.value) {
        editor.path = replacePathPrefix(editor.path, oldPath, newPath);
        editor.name = Xn(editor.path);
      }
      const clipboard = An(F.value);
      if (clipboard?.sourceConnectionId === E.connectionId) {
        clipboard.items = clipboard.items.map((item) => ({
          ...item,
          path: replacePathPrefix(item.path, oldPath, newPath),
        }));
        F.value = clipboard;
        localStorage.setItem(M, JSON.stringify(clipboard));
      }

      It([{ path: oldPath, isDir }]);
      parent && Et(parent);
      cancelRename();
      await refreshAffectedDirectories([parent]);
      const shouldApplyRenameSelection = On(T.value) === originalCurrentPath;
      if (shouldApplyRenameSelection && currentPath && currentPath !== originalCurrentPath) {
        await Oe(currentPath, { useCache: !1 });
      }
      if (shouldApplyRenameSelection) {
        _.value = new Set([newPath]);
        revealTreePath(selectedTreePath || (isDir ? newPath : Wn(newPath)) || T.value);
      }
    } catch (error) {
      renameState.saving = !1;
      renameState.error = error instanceof Error ? error.message : "\u91cd\u547d\u540d\u5931\u8d25";
      z.value = renameState.error;
      activeRenameInput = null;
    }
  }
  async function Lt() {
    if (!pe.value) return;
    const e = he.value,
      t = e.map((e) => e.path);
    z.value = "";
    try {
      if (1 === e.length && !e[0].isDir)
        return void (await u(E.connectionId, e[0].path, e[0].name));
      await d(E.connectionId, t, jn(e));
    } catch (e) {
      z.value = e instanceof Error ? e.message : "\u4e0b\u8f7d\u5931\u8d25";
    }
  }
  async function Ft(e, t) {
    "textEdit" === e && (await Ot(t));
  }
  async function Ot(e) {
    if (!E.connectionId || !e || e.isDir) return;
    const existingEditor = Z.value.find((t) => t.path === e.path);
    if (existingEditor)
      return (
        "minimized" === existingEditor.windowState && (existingEditor.windowState = "normal"),
        void Kt(existingEditor.id)
      );
    ot();
    const editor = Wt(e);
    const controller = new AbortController();
    editorReadControllers.set(editor.id, controller);
    const editorExists = () => Z.value.some((item) => item.id === editor.id);
    const isCurrentRead = () => editorExists() && editorReadControllers.get(editor.id) === controller;
    let pendingPreviewChunks = [];
    let pendingPreviewCharacters = 0;
    let previewPublishTimer = null;
    let lastPreviewPublishAt = performance.now();
    const cancelPreviewPublishTimer = () => {
      if (previewPublishTimer === null) return;
      window.clearTimeout(previewPublishTimer);
      previewPublishTimer = null;
    };
    const discardPendingPreview = () => {
      pendingPreviewChunks = [];
      pendingPreviewCharacters = 0;
    };
    const publishPreviewBatch = () => {
      cancelPreviewPublishTimer();
      if (pendingPreviewCharacters <= 0 || pendingPreviewChunks.length === 0) return;
      if (!isCurrentRead()) {
        discardPendingPreview();
        return;
      }
      const text = pendingPreviewChunks.join("");
      discardPendingPreview();
      if (!text) return;
      editor.contentChunks.push(text);
      editor.appendVersion += 1;
      lastPreviewPublishAt = performance.now();
    };
    const queuePreviewChunk = (text) => {
      pendingPreviewChunks.push(text);
      pendingPreviewCharacters += text.length;
      const elapsedMs = performance.now() - lastPreviewPublishAt;
      if (elapsedMs >= EDITOR_PREVIEW_PUBLISH_INTERVAL_MS) {
        publishPreviewBatch();
        return;
      }
      if (previewPublishTimer !== null) return;
      previewPublishTimer = window.setTimeout(() => {
        previewPublishTimer = null;
        publishPreviewBatch();
      }, Math.max(0, EDITOR_PREVIEW_PUBLISH_INTERVAL_MS - elapsedMs));
    };
    try {
      Gt(editor, {
        stage: "preparing",
        totalBytes: Number(e.size) || 0,
        message: "\u51c6\u5907\u6253\u5f00\u8fdc\u7a0b\u6587\u4ef6",
      });
      const response = await m(
        editor.connectionId,
        e.path,
        (progress) => {
          if (!isCurrentRead()) return;
          progress.path && (editor.path = String(progress.path));
          progress.fileName && (editor.name = String(progress.fileName));
          editor.loadedContentBytes = Number(progress.loadedBytes) || editor.loadedContentBytes || 0;
          editor.size = Number(progress.totalBytes) || Number(editor.size) || 0;
          Gt(editor, progress);
        },
        {
          signal: controller.signal,
          onChunk: (chunk) => {
            if (!isCurrentRead()) return;
            const text = String(chunk?.text || "");
            if (!text) return;
            queuePreviewChunk(text);
          },
        },
      );
      const remoteFile = response.file || {};
      const content = String(remoteFile.content ?? "");
      if (!isCurrentRead()) return;
      publishPreviewBatch();
      editor.path = String(remoteFile.path || e.path);
      editor.name = String(remoteFile.name || e.name || Xn(editor.path));
      editor.content = content;
      editor.originalContent = content;
      editor.size = Number(remoteFile.size) || new Blob([content]).size;
      editor.modTime = String(remoteFile.modTime || e.modTime || "");
      editor.contentLoading = false;
      editor.contentLoaded = true;
      editor.message = "\u5df2\u6253\u5f00";
      Gt(editor, {
        stage: "done",
        loadedBytes: editor.size,
        totalBytes: editor.size,
        message: "\u8fdc\u7a0b\u6587\u4ef6\u4e0b\u8f7d\u5b8c\u6210",
      });
      await t();
      if (isCurrentRead()) editor.contentChunks = markRawChunks([]);
    } catch (error) {
      cancelPreviewPublishTimer();
      discardPendingPreview();
      if (controller.signal.aborted || "AbortError" === error?.name || !isCurrentRead()) {
        return;
      }
      editor.error =
        error instanceof Error
          ? error.message
          : "\u6253\u5f00\u6587\u4ef6\u5931\u8d25";
      editor.contentChunks = markRawChunks([]);
      editor.contentLoading = false;
      editor.message = "\u6253\u5f00\u5931\u8d25";
      Gt(editor, {
        stage: "error",
        loadedBytes: Number(editor.openProgress?.loadedBytes) || 0,
        totalBytes: Number(editor.openProgress?.totalBytes) || Number(e.size) || 0,
        message: editor.error,
      });
    } finally {
      cancelPreviewPublishTimer();
      discardPendingPreview();
      editorReadControllers.get(editor.id) === controller && editorReadControllers.delete(editor.id);
      editorExists() && (editor.loading = false);
    }
  }
  function Wt(e) {
    const t = Ht(),
      n = r({
        id: crypto.randomUUID(),
        connectionId: E.connectionId,
        windowState: "normal",
        loading: !1,
        contentLoading: !0,
        contentLoaded: !1,
        contentChunks: markRawChunks([]),
        appendVersion: 0,
        loadedContentBytes: 0,
        saving: !1,
        path: e.path,
        name: e.name,
        content: "",
        originalContent: "",
        size: Number(e.size) || 0,
        modTime: e.modTime || "",
        error: "",
        message: "",
        editorRuntimeState: "loading",
        editorRuntimeMessage: "\u52a0\u8f7d Monaco...",
        editorRuntimeProgress: 0,
        editorRuntimeStep: 0,
        editorRuntimeTotalSteps: 0,
        openProgress: {
          stage: "preparing",
          loadedBytes: 0,
          totalBytes: Number(e.size) || 0,
          startedAt: Date.now(),
          message: "\u51c6\u5907\u6253\u5f00\u8fdc\u7a0b\u6587\u4ef6",
        },
        x: t.x,
        y: t.y,
        width: t.width,
        height: t.height,
        zIndex: Ut(),
        closePrompt: { visible: !1, afterClose: null },
      });
    Z.value.push(n);
    J.value = n.id;
    return n;
  }
  function Ht() {
    const e = window.innerWidth || 1200,
      t = window.innerHeight || 780,
      n = Math.min(980, Math.max(520, e - 64)),
      o = Math.min(660, Math.max(360, t - 126)),
      r = (Z.value.length % 7) * 26,
      a = Math.max(16, Math.round((e - n) / 2));
    return {
      x: Math.min(Math.max(16, a + r), Math.max(16, e - 120)),
      y: Math.min(48 + r, Math.max(48, t - 120)),
      width: n,
      height: o,
    };
  }
  function Ut() {
    return (oe += 1), oe;
  }
  function Kt(e) {
    const t = Z.value.find((t) => t.id === e);
    t && ((J.value = e), (t.zIndex = Ut()));
  }
  function Yt(e) {
    return Boolean(e && !e.contentLoading && e.content !== e.originalContent);
  }
  function Vt(e) {
    const t = e.openProgress || {},
      n = t.stage || "preparing",
      o = Number(t.loadedBytes) || 0,
      r = Number(t.totalBytes) || Number(e.size) || 0;
    if ("downloading" === n || o > 0) {
      return `\u4e0b\u8f7d\u4e2d ${r > 0 ? `${Math.min(100, Math.max(0, Math.round((o / r) * 100)))}%` : Gn(o)} \xb7 ${qt(o, r)} \xb7 ${Tn(jt(e))}`;
    }
    return "error" === n
      ? t.message || "\u6253\u5f00\u5931\u8d25"
      : "stat" === n
        ? "\u6253\u5f00\u4e2d \xb7 \u8bfb\u53d6\u8fdc\u7a0b\u6587\u4ef6\u4fe1\u606f"
        : t.message ||
          "\u6253\u5f00\u4e2d \xb7 \u5efa\u7acb\u8fdc\u7a0b\u8bfb\u53d6";
  }
  function Xt(e) {
    const t = e.editorRuntimeMessage || "\u52a0\u8f7d Monaco...",
      n = Number(e.editorRuntimeProgress) || 0;
    return n > 0
      ? `${t} \xb7 ${Math.round(100 * n)}%`
      : e.editorRuntimeStep && e.editorRuntimeTotalSteps
        ? `${t} \xb7 ${e.editorRuntimeStep}/${e.editorRuntimeTotalSteps}`
        : t;
  }
  function jt(e) {
    const t = e.openProgress || {};
    return (
      (Number(t.loadedBytes) || 0) /
      Math.max((Date.now() - (Number(t.startedAt) || Date.now())) / 1e3, 0.1)
    );
  }
  function qt(e, t) {
    return t > 0
      ? `${Gn(e)} / ${Gn(t)}`
      : e > 0
        ? `${Gn(e)} \u5df2\u4e0b\u8f7d`
        : "-";
  }
  function Gt(e, t = {}) {
    if (!e) return;
    const n = e.openProgress || {},
      o = Number(t.totalBytes) || Number(n.totalBytes) || Number(e.size) || 0;
    e.openProgress = {
      stage: t.stage || n.stage || "downloading",
      loadedBytes: Math.max(
        Number(n.loadedBytes) || 0,
        Number(t.loadedBytes) || 0,
      ),
      totalBytes: o,
      startedAt: n.startedAt || Date.now(),
      message: t.message || n.message || "",
    };
  }
  function Zt(e) {
    if (!te) return;
    const t = Z.value.find((e) => e.id === te.id);
    if (!t) return void Jt();
    const n = Math.max(16, (window.innerWidth || 1200) - 120),
      o = Math.max(48, (window.innerHeight || 780) - 60);
    (t.x = Math.min(n, Math.max(16, te.originX + e.clientX - te.startX))),
      (t.y = Math.min(o, Math.max(48, te.originY + e.clientY - te.startY)));
  }
  function Jt() {
    te &&
      ((te = null),
      (document.body.style.cursor = ""),
      (document.body.style.userSelect = ""),
      window.removeEventListener("mousemove", Zt),
      window.removeEventListener("mouseup", Jt));
  }
  async function Qt(e) {
    if (
      !(e?.connectionId || E.connectionId) ||
      !e ||
      !e.path ||
      e.contentLoading ||
      "rendering" === e.editorRuntimeState ||
      e.saving
    )
      return !1;
    (e.saving = !0), (e.error = ""), (e.message = "");
    const t = e.content;
    try {
      const n = (await h(e.connectionId || E.connectionId, e.path, t)).file || {};
      return (
        (e.path = String(n.path || e.path)),
        (e.name = String(n.name || e.name || Xn(e.path))),
        (e.originalContent = t),
        (e.size = Number(n.size) || new Blob([t]).size),
        (e.modTime = String(n.modTime || "")),
        (e.message = "\u5df2\u4fdd\u5b58"),
        Et(Wn(e.path) || T.value || Xe(E.workMode)),
        await Oe(T.value || Xe(E.workMode), { useCache: !1 }),
        !0
      );
    } catch (t) {
      return (
        (e.error = t instanceof Error ? t.message : "\u4fdd\u5b58\u5931\u8d25"),
        !1
      );
    } finally {
      e.saving = !1;
    }
  }
  function en(e) {
    if (!e) return;
    cancelEditorRead(e);
    const t = Z.value.findIndex((t) => t.id === e.id);
    if (-1 !== t && (Z.value.splice(t, 1), J.value === e.id)) {
      const e = [...Z.value].sort((e, t) => t.zIndex - e.zIndex)[0];
      J.value = e?.id || "";
    }
  }
  function cancelEditorRead(editor) {
    const controller = editorReadControllers.get(editor?.id);
    if (!controller) return;
    editorReadControllers.delete(editor.id);
    controller.abort();
  }
  function cancelAllEditorReads() {
    for (const controller of editorReadControllers.values()) controller.abort();
    editorReadControllers.clear();
  }
  async function tn(e) {
    "function" == typeof e && (await e());
  }
  function nn(e) {
    e && ((e.closePrompt.visible = !1), (e.closePrompt.afterClose = null));
  }
  function on() {
    (_.value = new Set()), (L.value = -1);
  }
  function rn() {
    const e = new Set(C.value.map((e) => e.path));
    _.value = new Set(Array.from(_.value).filter((t) => e.has(t)));
  }
  function an(e, t = {}) {
    const clientX = Number(e?.clientX) || 8;
    const clientY = Number(e?.clientY) || 8;
    const revision = ++contextMenuPositionRevision;
    const n = y({ clientX, clientY }, { width: 220, height: 320 });
    const entry = t.entry || null;
    const targetKind = t.targetKind || (entry?.isDir ? "directory" : entry ? "file" : "directory");
    (de.visible = !0),
      (de.entry = entry),
      (de.targetPath = On(t.targetPath || entry?.path || je())),
      (de.targetKind = targetKind),
      (de.source = t.source || "blank"),
      (de.x = n.x),
      (de.y = n.y);
    void positionContextMenuAfterRender(revision, clientX, clientY);
  }
  async function positionContextMenuAfterRender(revision, clientX, clientY) {
    await t();
    if (!de.visible || revision !== contextMenuPositionRevision) return;
    const element = contextMenuElement.value;
    if (!element) return;
    const bounds = element.getBoundingClientRect();
    const position = y(
      { clientX, clientY },
      {
        width: Math.ceil(bounds.width),
        height: Math.ceil(bounds.height),
      },
    );
    if (!de.visible || revision !== contextMenuPositionRevision) return;
    de.x = position.x;
    de.y = position.y;
  }
  function sn() {
    contextMenuPositionRevision += 1;
    de.visible = !1;
  }
  function ln(e) {
    const t = e.target;
    if (
      (pathCompletionOpen.value &&
        t instanceof Element &&
        !t.closest(".path-input-completion") &&
        (pathCompletionOpen.value = !1),
      V.value &&
        t instanceof Element &&
        (t.closest(".path-history-popover") ||
          t.closest(".path-history-button") ||
          (V.value = !1)),
      de.visible)
    ) {
      if (t instanceof Element && t.closest(".file-context-menu")) return;
      sn();
    }
  }
  function cn() {
    hn("copy");
  }
  function un() {
    hn("move");
  }
  async function dn() {
    pe.value && E.connectionId && (await fn(he.value));
  }
  async function fn(e) {
    if (!e.length || !E.connectionId) return;
    const t = e.map((e) => ({ path: e.path, isDir: Boolean(e.isDir) }));
    if (window.confirm(mn(t))) {
      (A.value = !0), (z.value = "");
      try {
        await c(
          E.connectionId,
          t.map((e) => ({ path: e.path, isDir: e.isDir })),
        );
        const e = Bt(t);
        It(t), on(), kn(t), await Oe(e, { useCache: !1 });
      } catch (e) {
        z.value = e instanceof Error ? e.message : "\u5220\u9664\u5931\u8d25";
      } finally {
        A.value = !1;
      }
    }
  }
  function mn(e) {
    const t = e
        .slice(0, 6)
        .map((e) => `- ${e.path}`)
        .join("\n"),
      n = e.length > 6 ? `\n... \u8fd8\u6709 ${e.length - 6} \u9879` : "";
    return `\u786e\u8ba4\u5f3a\u5236\u5220\u9664\u4ee5\u4e0b ${e.length} \u9879\uff1f\n\n${t}${n}\n\n\u6b64\u64cd\u4f5c\u4e0d\u53ef\u64a4\u9500\u3002`;
  }
  function hn(e, items = he.value) {
    if (!items.length) return;
    const t = $n(e),
      n = {
        sourceConnectionId: E.connectionId,
        action: t,
        items: items.map((e) => ({ path: e.path, isDir: Boolean(e.isDir) })),
        createdAt: Date.now(),
      };
    localStorage.setItem(M, JSON.stringify(n)), (F.value = n);
  }
  async function pn(e = Pe.value) {
    if (!ve.value) return;
    const t = An(F.value);
    if (!t) return (F.value = null), void localStorage.removeItem(M);
    const targetPath = On(e || je());
    const sameConnectionMove = "move" === t.action && t.sourceConnectionId === E.connectionId;
    const affectedDirectories = [targetPath];
    if (sameConnectionMove) {
      for (const item of t.items) affectedDirectories.push(Wn(item.path));
    }
    (A.value = !0), (z.value = "");
    try {
      await p({
        sourceConnectionId: t.sourceConnectionId,
        targetConnectionId: E.connectionId,
        targetPath,
        action: t.action,
        items: t.items,
      });
      const currentPathAfterTransfer = On(T.value);
      const navigationRevisionAfterTransfer = navigationRevision;
      let relocatedCurrentPath = currentPathAfterTransfer;
      if (sameConnectionMove) {
        for (const item of t.items) {
          const sourcePath = On(item.path);
          if (!item.isDir || !sourcePath || !Un(currentPathAfterTransfer, sourcePath)) continue;
          const destinationRoot = targetPath === g
            ? `${g}${Xn(sourcePath)}`
            : `${targetPath}/${Xn(sourcePath)}`;
          relocatedCurrentPath = replacePathPrefix(currentPathAfterTransfer, sourcePath, destinationRoot);
          break;
        }
      }
      Et(targetPath);
      if (sameConnectionMove) {
        It(t.items);
        for (const item of t.items) Et(Wn(item.path));
      }
      "move" === t.action && (localStorage.removeItem(M), (F.value = null)),
        await refreshAffectedDirectories(affectedDirectories);
      if (
        relocatedCurrentPath &&
        relocatedCurrentPath !== currentPathAfterTransfer &&
        On(T.value) === currentPathAfterTransfer &&
        navigationRevision === navigationRevisionAfterTransfer
      ) {
        await Oe(relocatedCurrentPath, { useCache: !1 });
      }
    } catch (e) {
      const errorMessage = e instanceof Error ? e.message : "\u7c98\u8d34\u5931\u8d25";
      const currentPathAtFailure = On(T.value);
      const navigationRevisionAtFailure = navigationRevision;
      let relocatedFailurePath = currentPathAtFailure;
      if (sameConnectionMove) {
        for (const item of t.items) {
          const sourcePath = On(item.path);
          if (!item.isDir || !sourcePath || !Un(currentPathAtFailure, sourcePath)) continue;
          const destinationRoot = targetPath === g
            ? `${g}${Xn(sourcePath)}`
            : `${targetPath}/${Xn(sourcePath)}`;
          relocatedFailurePath = replacePathPrefix(currentPathAtFailure, sourcePath, destinationRoot);
          break;
        }
        It(t.items);
      }
      for (const path of affectedDirectories) Et(path);
      await refreshAffectedDirectories(affectedDirectories);
      if (
        sameConnectionMove &&
        relocatedFailurePath !== currentPathAtFailure &&
        On(T.value) === currentPathAtFailure &&
        navigationRevision === navigationRevisionAtFailure
      ) {
        try {
          const currentListing = await We(currentPathAtFailure, { rememberTree: !0 });
          if (
            On(T.value) === currentPathAtFailure &&
            navigationRevision === navigationRevisionAtFailure
          ) {
            Ye(currentListing.resolvedPath, currentListing.entries, currentListing.requestedPath);
          }
        } catch {
          if (
            On(T.value) === currentPathAtFailure &&
            navigationRevision === navigationRevisionAtFailure
          ) {
            await Oe(relocatedFailurePath, { useCache: !1 });
          }
        }
      }
      z.value = errorMessage;
    } finally {
      A.value = !1;
    }
  }
  function vn(e) {
    const t = new Set();
    for (const n of e) {
      const e = (n.webkitRelativePath || "").split("/").filter(Boolean);
      e.pop();
      for (let n = 1; n <= e.length; n += 1) t.add(e.slice(0, n).join("/"));
    }
    return Array.from(t);
  }
  async function yn(e) {
    const t = Array.from(e.items || [])
      .map((e) =>
        "function" == typeof e.webkitGetAsEntry ? e.webkitGetAsEntry() : null,
      )
      .filter(Boolean);
    if (0 === t.length)
      return {
        files: Array.from(e.files || []).map((e) => ({
          file: e,
          relativePath: e.name,
        })),
        directories: [],
      };
    const n = [],
      o = new Set();
    for (const e of t) await gn(e, "", n, o);
    return { files: n, directories: Array.from(o) };
  }
  async function gn(e, t, n, o) {
    const r = t ? `${t}/${e.name}` : e.name;
    if (e.isFile) {
      const t = await wn(e);
      return void n.push({ file: t, relativePath: r });
    }
    if (!e.isDirectory) return;
    o.add(r);
    const a = await Pn(e.createReader());
    for (const e of a) await gn(e, r, n, o);
  }
  function wn(e) {
    return new Promise((t, n) => {
      e.file(t, n);
    });
  }
  function Pn(e) {
    return new Promise((t, n) => {
      const o = [],
        r = () => {
          e.readEntries(
            (e) => {
              0 !== e.length ? (o.push(...e), r()) : t(o);
            },
            (e) => n(e),
          );
        };
      r();
    });
  }
  function Mn(e, t, n) {
    En();
    const o = e.map((e, t) => ({
      id: `${Date.now()}-${t}-${e.relativePath || e.file?.name || "file"}`,
      relativePath:
        e.relativePath ||
        e.file?.webkitRelativePath ||
        e.file?.name ||
        `file-${t + 1}`,
      size: Number(e.file?.size) || 0,
      loaded: 0,
    }));
    (G.visible = !0),
      (G.expanded = !0),
      (G.status = "uploading"),
      (G.targetPath = n),
      (G.files = o),
      (G.directoryCount = t.length),
      (G.totalBytes = o.reduce((e, t) => e + t.size, 0)),
      (G.loadedBytes = 0),
      (G.speed = 0),
      (G.startedAt = Date.now()),
      (G.message = "");
  }
  function bn(e) {
    if (!G.visible || "uploading" !== G.status) return;
    const t = Number(e?.loadedBytes ?? e?.loaded) || 0,
      n = Number(e?.totalBytes ?? e?.total) || 0;
    n > G.totalBytes && (G.totalBytes = n);
    const o = G.totalBytes > 0 ? Math.min(G.totalBytes, t) : t;
    (G.loadedBytes = Math.max(G.loadedBytes, o)), Dn(G.loadedBytes);
    const r = Math.max((Date.now() - G.startedAt) / 1e3, 0.1);
    G.speed = G.loadedBytes / r;
  }
  function Dn(e) {
    let t = e;
    for (const e of G.files)
      (e.loaded = Math.min(e.size, Math.max(0, t))), (t -= e.size);
  }
  function xn() {
    (G.status = "done"),
      (G.loadedBytes = G.totalBytes),
      Dn(G.totalBytes),
      (G.speed = 0),
      (G.message = "\u4e0a\u4f20\u5df2\u5b8c\u6210");
  }
  function Sn(e) {
    (G.status = "error"),
      (G.expanded = !0),
      (G.message = e || "\u4e0a\u4f20\u5931\u8d25");
  }
  function In() {
    En(),
      (ne = window.setTimeout(() => {
        Cn(), (ne = null);
      }, 1200));
  }
  function En() {
    ne && (window.clearTimeout(ne), (ne = null));
  }
  function Cn() {
    G.visible && (G.expanded = !1);
  }
  function Tn(e) {
    return !e || e < 1 ? "-" : `${Gn(e)}/s`;
  }
  function Bn() {
    try {
      const e = localStorage.getItem(M);
      return e ? An(JSON.parse(e)) : null;
    } catch {
      return null;
    }
  }
  function An(e) {
    if (
      !e?.sourceConnectionId ||
      !Array.isArray(e.items) ||
      0 === e.items.length
    )
      return null;
    const t = e.items
      .filter((e) => e?.path)
      .map((e) => ({ path: e.path, isDir: Boolean(e.isDir) }));
    return 0 === t.length
      ? null
      : {
          sourceConnectionId: e.sourceConnectionId,
          action: $n(e.action),
          items: t,
          createdAt: Number(e.createdAt) || Date.now(),
        };
  }
  function $n(e) {
    return D.has(e) ? e : "copy";
  }
  function kn(e) {
    const t = An(F.value);
    if (!t) return;
    e.some((e) =>
      t.items.some((t) => t.path === e.path || (e.isDir && Un(t.path, e.path))),
    ) && (localStorage.removeItem(M), (F.value = null));
  }
  function Rn(e) {
    e.key === M && (F.value = Bn());
  }
  function zn(e) {
    const t = e.key.toLowerCase(),
      n = xe.value;
    if ((e.ctrlKey || e.metaKey) && "s" === t && n)
      return e.preventDefault(), void Qt(n);
    if ("Escape" === e.key) {
      if (n?.closePrompt.visible) return void nn(n);
      sn();
    }
    "F5" === e.key &&
      E.connectionId &&
      (e.preventDefault(), He(T.value || Xe(E.workMode)));
  }
  function Nn(e) {
    if (!ee) return;
    const t = Math.max(70, ee.startWidth + e.clientX - ee.startX);
    (X[ee.key] = t), Fn();
  }
  function _n() {
    ee &&
      ((ee = null),
      (document.body.style.cursor = ""),
      (document.body.style.userSelect = ""),
      window.removeEventListener("mousemove", Nn),
      window.removeEventListener("mouseup", _n));
  }
  function Ln() {
    const e = {};
    for (const t of S) {
      const n = document.documentElement.style.getPropertyValue(
          `--file-col-${t.key}`,
        ),
        o = Number.parseInt(n, 10);
      e[t.key] = Number.isFinite(o) ? o : t.width;
    }
    return e;
  }
  function Fn() {
    for (const e of S)
      document.documentElement.style.setProperty(
        `--file-col-${e.key}`,
        `${X[e.key]}px`,
      );
  }
  function Kn(e) {
    if (e === g || e === w) return !0;
    let t = Wn(e);
    for (; t; ) {
      if (Q.value.get(t)?.collapsed) return !1;
      t = Wn(t);
    }
    return !0;
  }
  return (
    o(() => {
      Fn(),
        (viewportResizeObserver = new ResizeObserver((entries) => {
          for (const entry of entries) {
            entry.target === fileListViewport.value && syncFileViewport(entry.target);
            entry.target === pathTreeViewport.value && syncTreeViewport(entry.target);
          }
        })),
        fileListViewport.value && viewportResizeObserver.observe(fileListViewport.value),
        pathTreeViewport.value && viewportResizeObserver.observe(pathTreeViewport.value),
        window.addEventListener("storage", Rn),
        window.addEventListener("keydown", zn),
        window.addEventListener("pointerdown", ln, !0);
    }),
    n(() => {
      window.removeEventListener("storage", Rn),
        window.removeEventListener("keydown", zn),
        window.removeEventListener("pointerdown", ln, !0),
        En(),
        ct(),
        Le(),
        ot(),
        pathCompletionModel.dispose(),
        cancelAllEditorReads(),
        directoryLoadController?.abort(),
        (directoryLoadController = null),
        viewportResizeObserver?.disconnect(),
        (viewportResizeObserver = null),
        rt(),
        _n(),
        Jt();
    }),
    {
      ROOT_PATH: g,
      HOME_PATH: w,
      WORK_MODE_START_PATHS: P,
      CLIPBOARD_KEY: M,
      DIRECTORY_CACHE_LIMIT,
      MAX_PRELOAD_CONCURRENCY,
      TREE_INDEX_BATCH_SIZE: 60,
      TREE_INDEX_DELAY_MS: 16,
      PRELOAD_START_DELAY_MS: 0,
      PRELOAD_TARGET_LIMIT,
      PRELOAD_BATCH_SIZE: PRELOAD_TARGET_LIMIT,
      PRELOAD_BATCH_DELAY_MS,
      REFRESH_BATCH_SIZE: 8,
      REFRESH_BATCH_DELAY_MS: 80,
      MIN_COLUMN_WIDTH: 70,
      UPLOAD_CLOSE_DELAY_MS: 1200,
      SORT_PULSE_MS: 220,
      DEFAULT_FILE_OPEN_ACTION: b,
      DEFAULT_EDITOR_WIDTH: 980,
      DEFAULT_EDITOR_HEIGHT: 660,
      MIN_EDITOR_TOP: 48,
      CLIPBOARD_ACTIONS: D,
      directoryCache: x,
      columns: S,
      fileOpenActions: I,
      entries: C,
      currentPath: T,
      pathDraft: B,
      loading: A,
      refreshingCached: $,
      uploading: k,
      dragOver: R,
      errorMessage: z,
      navCollapsed: N,
      selectedPaths: _,
      lastSelectedIndex: L,
      clipboard: F,
      fileInput: O,
      folderInput: W,
      pathHistoryButton: H,
      pathHistoryListRef: U,
      pendingUploadPath: K,
      pathHistoryStats: Y,
      pathHistoryOpen: V,
      pathCompletion,
      pathCompletionOpen,
      pathCompletionVisible,
      pathCompletionSummary,
      columnWidths: X,
      sortState: j,
      sortPulseKey: q,
      uploadProgress: G,
      editors: Z,
      activeEditorId: J,
      pathMeta: Q,
      treeSelectedPath,
      fileListViewport,
      pathTreeViewport,
      contextMenuElement,
      fileScrollTop,
      treeScrollTop,
      fileViewportHeight,
      treeViewportHeight,
      renameState,
      resizeState: ee,
      editorDragState: te,
      uploadCloseTimer: ne,
      editorZIndex: oe,
      refreshSerial: re,
      preloadSerial: ae,
      preloadController: ie,
      preloadTimer: se,
      treeIndexQueue: le,
      treeIndexTimer: ce,
      sortPulseTimer: ue,
      contextMenu: de,
      pathHistoryPanel: fe,
      orderedEntries: me,
      selectedEntries: he,
      hasSelection: pe,
      canPaste: ve,
      contextEntry: ye,
      contextCanOpenDirectory: ge,
      contextPath: we,
      contextTargetDir: Pe,
      contextTargetItem: Me,
      canDeleteFromMenu: be,
      contextCanMutate,
      gridTemplateColumns: De,
      activeEditor: xe,
      preloadConcurrency: Se,
      statusText: Ie,
      breadcrumbs: Ee,
      pathHistoryStyle: Ce,
      uploadPercent: Te,
      uploadTitle: Be,
      uploadDetail: Ae,
      pathHistory: $e,
      compareEntriesForSort: ke,
      compareEntryValue: Re,
      compareText: ze,
      compareNumber: Ne,
      changeSort: function (e) {
        j.key === e
          ? (j.direction = "asc" === j.direction ? "desc" : "asc")
          : ((j.key = e), (j.direction = "asc")),
          _e(e);
      },
      pulseSortColumn: _e,
      clearSortPulseTimer: Le,
      treeNodes: Fe,
      visibleTreeNodes,
      visibleEntries,
      fileVirtualHeight,
      treeVirtualHeight,
      treeContentWidth,
      fileContentWidth,
      fileRowHeight: FILE_ROW_HEIGHT,
      treeRowHeight: TREE_ROW_HEIGHT,
      onFileListScroll: (event) => syncFileViewport(event.currentTarget),
      onTreeScroll: (event) => syncTreeViewport(event.currentTarget),
      revealTreePath,
      scrollTreePathIntoView,
      refresh: Oe,
      loadDirectoryFromRemote: We,
      refreshOpenDirectories: He,
      openedDirectoryPaths: Ue,
      directoryListingMatchesFocus: Ke,
      applyDirectoryListing: Ye,
      resetPathState: Ve,
      initialPathForMode: Xe,
      defaultDirectoryPath: je,
      normalizeWorkMode: qe,
      initialPathMeta: Ge,
      getCachedDirectory: Ze,
      rawCachedDirectoryEntries,
      hasCachedDirectory,
      setCachedDirectory: Je,
      directoryCacheKey: Qe,
      cloneEntries: et,
      rememberDirectoryTree: tt,
      scheduleDirectoryPreload: nt,
      cancelDirectoryPreload: ot,
      clearPreloadTimer: rt,
      directoryPreloadTargets: at,
      enqueueTreeIndex: it,
      scheduleTreeIndexWork: st,
      processTreeIndexBatch: lt,
      clearTreeIndexQueue: ct,
      rememberPath: ut,
      rememberParentChain: function (e) {
        const t = On(e);
        if (!t) return;
        const n = [];
        let o = t;
        for (; o; ) n.unshift(o), (o = Wn(o));
        const r = new Map(Q.value);
        for (const e of n) ft(r, e);
        Q.value = r;
      },
      rememberParentChainInMap: dt,
      mergePathMeta: ft,
      rememberPathHistory: mt,
      togglePathHistory: function () {
        pathCompletionOpen.value = !1;
        E.connectionId && 0 !== $e.value.length
          ? V.value
            ? (V.value = !1)
            : (ht(), (V.value = !0), vt())
          : (V.value = !1);
      },
      positionPathHistoryPanel: ht,
      trimPathHistoryStats: pt,
      scrollPathHistoryToBottom: vt,
      openHistoryPath: function (e) {
        (V.value = !1), (pathCompletionOpen.value = !1), on(), yt(e);
      },
      activateTreeNode,
      openDir: yt,
      openEntry: function (e) {
        e.isDir ? yt(e.path) : Ft(b, e);
      },
      onPathInput: pathCompletionModel.onInput,
      commitPathDraft: pathCompletionModel.commit,
      completePathDraft: pathCompletionModel.complete,
      dismissPathCompletion: pathCompletionModel.dismiss,
      openPathCompletionItem: pathCompletionModel.openItem,
      pathCompletionItemLabel: pathCompletionModel.itemLabel,
      ensureCompletionCaches: pathCompletionModel.ensureCompletionCache,
      pathCompletionMatches: pathCompletionModel.matches,
      pathCompletionCandidates: pathCompletionModel.candidates,
      completionMatchesForPrefix: pathCompletionModel.matches,
      currentRelativeCompletionPrefix: pathCompletionModel.target,
      completionParentPath: pathCompletionModel.parent,
      completionTarget: pathCompletionModel.commonPrefix,
      cachedDirectoriesForConnection: St,
      syncDirectoryCacheAfterDelete: It,
      invalidateDirectoryCache: Et,
      entryMatchesDeletedItem: Ct,
      removeDeletedPathMeta: Tt,
      pathAfterDeletingItems: Bt,
      nearestExistingParentAfterDelete: At,
      isDeletableRemotePath: $t,
      chooseFiles: kt,
      chooseFolder: Rt,
      chooseFilesFromMenu: function () {
        const e = Pe.value;
        sn(), kt(e);
      },
      chooseFolderFromMenu: function () {
        const e = Pe.value;
        sn(), Rt(e);
      },
      onFilePickerUpload: async function (e) {
        const t = Array.from(e.target.files || []);
        await zt(t, [], K.value || je()), (e.target.value = "");
      },
      onFolderPickerUpload: async function (e) {
        const t = Array.from(e.target.files || []),
          n = vn(t);
        await zt(t, n, K.value || je()), (e.target.value = "");
      },
      uploadPickedFiles: zt,
      uploadItems: Nt,
      onDragOver: function (e) {
        E.connectionId &&
          ((e.dataTransfer.dropEffect = "copy"), (R.value = !0));
      },
      onDragLeave: function (e) {
        e.currentTarget.contains(e.relatedTarget) || (R.value = !1);
      },
      onDropUpload: async function (e) {
        if (((R.value = !1), !E.connectionId)) return;
        const { files: t, directories: n } = await yn(e.dataTransfer);
        await Nt(t, n, je());
      },
      download: _t,
      downloadSelection: Lt,
      downloadSelectionFromMenu: async function () {
        sn(), await Lt();
      },
      downloadContextEntry: downloadContextItem,
      downloadContextItem,
      openContextDirectory: function () {
        const e = we.value;
        sn(), e && yt(e);
      },
      expandContextDirectory: async function () {
        const e = we.value;
        sn(), e && (await expandDirectory(e));
      },
      refreshTargetDirectory,
      expandDirectory,
      startRenameFromMenu: function () {
        startRename(Me.value, de.source === "entry" ? "list" : "tree");
      },
      startRename,
      setRenameInputRef,
      updateRenameName,
      commitRename,
      cancelRename,
      runContextFileOpenAction: async function (e) {
        const t = de.entry;
        sn(), t && !t.isDir && (await Ft(e, t));
      },
      runFileOpenAction: Ft,
      openTextEditor: Ot,
      createEditorWindow: Wt,
      defaultEditorBounds: Ht,
      nextEditorZIndex: Ut,
      activateEditor: Kt,
      editorDirty: Yt,
      editorTitle: function (e) {
        return e?.name || Xn(e?.path) || "\u8fdc\u7a0b\u6587\u4ef6";
      },
      editorStatus: function (e) {
        return e.contentLoading
          ? Vt(e)
          : e.error && "error" === e.openProgress?.stage
            ? "\u6253\u5f00\u5931\u8d25"
            : e.saving
              ? "\u4fdd\u5b58\u4e2d..."
              : "loading" === e.editorRuntimeState ||
                  "rendering" === e.editorRuntimeState
                ? Xt(e)
                : "error" === e.editorRuntimeState
                  ? "\u7f16\u8f91\u5668\u964d\u7ea7"
                  : Yt(e)
                    ? "\u672a\u4fdd\u5b58"
                    : e.message || "\u5df2\u4fdd\u5b58";
      },
      editorOpenStatus: Vt,
      editorRuntimeStatus: Xt,
      editorOpenSpeed: jt,
      formatEditorBytes: qt,
      setEditorRuntimeState: function (e, t) {
        if (!e) return;
        const n = e.editorRuntimeState,
          o = e.editorRuntimeMessage;
        (e.editorRuntimeState = t?.status || ""),
          (e.editorRuntimeMessage = t?.message || ""),
          (e.editorRuntimeProgress = Math.max(
            0,
            Math.min(1, Number(t?.progress) || 0),
          )),
          (e.editorRuntimeStep = Number(t?.step) || 0),
          (e.editorRuntimeTotalSteps = Number(t?.totalSteps) || 0),
          "error" !== t?.status
            ? "error" === n && e.error === o && (e.error = "")
            : (e.error =
                t.message ||
                "Monaco \u52a0\u8f7d\u5931\u8d25\uff0c\u5df2\u5207\u6362\u5230\u57fa\u7840\u7f16\u8f91\u6a21\u5f0f");
      },
      updateEditorOpenProgress: Gt,
      editorMeta: function (e) {
        if (!e) return "";
        const n = Number(e.openProgress?.loadedBytes) || Number(e.loadedContentBytes) || 0,
          o = Number(e.openProgress?.totalBytes) || Number(e.size) || 0,
          t = [
            e.contentLoading
              ? qt(n, o)
              : Gn(Number(e.size) || new Blob([e.content]).size),
          ];
        return e.modTime && t.push(Zn(e.modTime)), t.join(" \xb7 ");
      },
      editorWindowStyle: function (e) {
        if ("minimized" === e.windowState) {
          const t = Z.value
            .filter((e) => "minimized" === e.windowState)
            .findIndex((t) => t.id === e.id);
          return {
            left: 16 + 252 * Math.max(0, t) + "px",
            bottom: "14px",
            width: "240px",
            height: "38px",
            zIndex: e.zIndex,
          };
        }
        return "maximized" === e.windowState
          ? {
              left: "12px",
              top: "50px",
              width: "calc(100vw - 24px)",
              height: "calc(100vh - 62px)",
              zIndex: e.zIndex,
            }
          : {
              left: `${e.x}px`,
              top: `${e.y}px`,
              width: `${e.width}px`,
              height: `${e.height}px`,
              zIndex: e.zIndex,
            };
      },
      minimizeEditor: function (e) {
        e && ((e.windowState = "minimized"), Kt(e.id));
      },
      toggleMaximizeEditor: function (e) {
        e &&
          ((e.windowState =
            "normal" === e.windowState ? "maximized" : "normal"),
          Kt(e.id));
      },
      startEditorDrag: function (e, t) {
        t &&
          "normal" === t.windowState &&
          0 === e.button &&
          (Kt(t.id),
          (te = {
            id: t.id,
            startX: e.clientX,
            startY: e.clientY,
            originX: t.x,
            originY: t.y,
          }),
          (document.body.style.cursor = "move"),
          (document.body.style.userSelect = "none"),
          window.addEventListener("mousemove", Zt),
          window.addEventListener("mouseup", Jt));
      },
      onEditorDrag: Zt,
      stopEditorDrag: Jt,
      saveEditor: Qt,
      requestEditorClose: function (e, t = {}) {
        const n = "function" == typeof t.afterClose ? t.afterClose : null;
        if (!e || !Yt(e)) return en(e), void tn(n);
        (e.closePrompt.visible = !0),
          (e.closePrompt.afterClose = n),
          "minimized" === e.windowState && (e.windowState = "normal"),
          Kt(e.id);
      },
      closeEditorImmediately: en,
      runAfterEditorClose: tn,
      saveEditorFromPrompt: async function (e) {
        (await Qt(e)) &&
          ((e.closePrompt.visible = !1), (e.closePrompt.afterClose = null));
      },
      saveAndCloseEditorFromPrompt: async function (e) {
        const t = e.closePrompt.afterClose;
        (await Qt(e)) && (en(e), await tn(t));
      },
      discardEditorFromPrompt: async function (e) {
        const t = e.closePrompt.afterClose;
        en(e), await tn(t);
      },
      cancelEditorClosePrompt: nn,
      selectEntry: function (e, t, n) {
        revealTreePath(Wn(e.path) || T.value);
        if (n.shiftKey && L.value >= 0) {
          const e = Math.min(L.value, t),
            n = Math.max(L.value, t),
            o = new Set(_.value);
          for (let t = e; t <= n; t += 1) o.add(me.value[t].path);
          _.value = o;
        } else {
          if (n.ctrlKey || n.metaKey) {
            const n = new Set(_.value);
            n.has(e.path) ? n.delete(e.path) : n.add(e.path),
              (_.value = n),
              (L.value = t);
            return;
          }
          (_.value = new Set([e.path])), (L.value = t);
        }
      },
      isSelected: function (e) {
        return _.value.has(e.path);
      },
      clearSelection: on,
      keepExistingSelection: rn,
      onPaneClick: function (e) {
        e.target.closest(".file-row") || (on(), revealTreePath(T.value)), sn();
      },
      onShellClick: function () {
        sn(), Cn(), (V.value = !1), (pathCompletionOpen.value = !1);
      },
      openEntryContextMenu: function (e, t, n) {
        (_.value = new Set([t.path])),
          (L.value = n),
          revealTreePath(Wn(t.path) || T.value),
          an(e, {
            entry: t,
            targetPath: t.path,
            targetKind: t.isDir ? "directory" : "file",
            source: "entry",
          });
      },
      openPathContextMenu: function (e, t) {
        const n = treeContextTarget(t, treeSelectedPath.value);
        an(e, { targetPath: n, targetKind: "directory", source: "tree" });
      },
      openBlankContextMenu: function (e) {
        on(), revealTreePath(je()), an(e, { targetPath: je(), targetKind: "directory", source: "blank" });
      },
      openContextMenu: an,
      hideContextMenu: sn,
      onGlobalPointerDown: ln,
      refreshFromMenu: async function () {
        const e = de.targetPath || T.value || Xe(E.workMode);
        sn(), await refreshTargetDirectory(e);
      },
      copySelection: cn,
      cutSelection: un,
      copySelectionFromMenu: function () {
        const e = Me.value;
        sn(), e && hn("copy", [e]);
      },
      cutSelectionFromMenu: function () {
        const e = Me.value;
        sn(), e && $t(e.path) && hn("move", [e]);
      },
      deleteFromMenu: async function () {
        const e = Me.value;
        sn(), e && $t(e.path) && (await fn([e]));
      },
      deleteSelection: dn,
      deleteItems: fn,
      deleteConfirmMessage: mn,
      writeClipboard: hn,
      pasteClipboard: pn,
      pasteClipboardFromMenu: async function () {
        const e = Pe.value;
        sn(), await pn(e);
      },
      copyPathFromMenu: async function () {
        const e = we.value;
        sn();
        if (!e) return;
        try {
          await navigator.clipboard.writeText(e);
        } catch {
          z.value = "\u590d\u5236\u8def\u5f84\u5931\u8d25\uff0c\u8bf7\u68c0\u67e5\u7cfb\u7edf\u526a\u8d34\u677f\u6743\u9650";
        }
      },
      onRemoteDragStart: function (e, t, n) {
        _.value.has(t.path) || ((_.value = new Set([t.path])), (L.value = n));
        const o = he.value.length ? he.value : [t],
          r = o.map((e) => e.path),
          a = 1 !== o.length || o[0].isDir ? jn(o) : o[0].name,
          i =
            1 !== o.length || o[0].isDir
              ? s(E.connectionId, r)
              : l(
                  `/api/sftp/download?connectionId=${encodeURIComponent(E.connectionId)}&path=${encodeURIComponent(o[0].path)}`,
                ),
          c = new URL(i, window.location.origin).toString();
        (e.dataTransfer.effectAllowed = "copy"),
          e.dataTransfer.setData(
            "DownloadURL",
            `application/octet-stream:${a}:${c}`,
          ),
          e.dataTransfer.setData("text/uri-list", c),
          e.dataTransfer.setData("text/plain", c);
      },
      deriveDirectoriesFromFiles: vn,
      collectDroppedItems: yn,
      walkDroppedEntry: gn,
      fileFromEntry: wn,
      readAllDirectoryEntries: Pn,
      startUploadProgress: Mn,
      onUploadProgress: bn,
      distributeUploadLoaded: Dn,
      markUploadComplete: xn,
      markUploadError: Sn,
      scheduleUploadPanelClose: In,
      clearUploadCloseTimer: En,
      collapseUploadPanel: Cn,
      expandUploadPanel: function () {
        G.visible && (En(), (G.expanded = !0));
      },
      uploadFilePercent: function (e) {
        return "done" === G.status
          ? 100
          : e.size
            ? Math.min(100, Math.max(0, Math.round((e.loaded / e.size) * 100)))
            : "uploading" === G.status
              ? 0
              : 100;
      },
      formatUploadSpeed: Tn,
      readClipboard: Bn,
      normalizeClipboardPayload: An,
      normalizeClipboardAction: $n,
      clearClipboardIfDeleted: kn,
      onStorage: Rn,
      onKeydown: zn,
      startColumnResize: function (e, t) {
        (ee = { key: t, startX: e.clientX, startWidth: X[t] }),
          (document.body.style.cursor = "col-resize"),
          (document.body.style.userSelect = "none"),
          window.addEventListener("mousemove", Nn),
          window.addEventListener("mouseup", _n);
      },
      onColumnResize: Nn,
      stopColumnResize: _n,
      loadColumnWidths: Ln,
      applyColumnWidths: Fn,
      normalizePath: On,
      parentPath: Wn,
      pathDepth: Hn,
      isSameOrChildPath: Un,
      isPathVisible: Kn,
      comparePaths: Yn,
      buildBreadcrumbs: Vn,
      displayPath: Xn,
      downloadArchiveName: jn,
      delay: qn,
      formatSize: Gn,
      formatTime: Zn,
    }
  );
}
export { E as useFileManager };
