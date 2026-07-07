export const ROOT_PATH = "/";
export const HOME_PATH = "~";
export const WORK_MODE_START_PATHS = { frontend: "/var", backend: "/opt", ops: ROOT_PATH };
export const CLIPBOARD_KEY = "zshell.remote-file.clipboard.v1";
export const DEFAULT_FILE_OPEN_ACTION = "textEdit";
export const CLIPBOARD_ACTIONS = new Set(["copy", "move"]);
export const FILE_COLUMNS = [
  { key: "name", label: "\u540d\u79f0", width: 280 },
  { key: "type", label: "\u7c7b\u578b", width: 82 },
  { key: "size", label: "\u5927\u5c0f", width: 110 },
  { key: "modTime", label: "\u4fee\u6539\u65f6\u95f4", width: 175 },
  { key: "mode", label: "\u6743\u9650", width: 125 },
  { key: "owner", label: "\u6240\u5c5e\u7528\u6237", width: 110 },
];
export const FILE_OPEN_ACTIONS = [{ key: "textEdit", label: "\u5728\u7ebf\u7f16\u8f91" }];
export const DIRECTORY_CACHE_LIMIT = 2400;
export const MAX_PRELOAD_CONCURRENCY = 2;
export const PRELOAD_TARGET_LIMIT = 10;
export const PRELOAD_BATCH_DELAY_MS = 120;

export function normalizeWorkMode(value) {
  return ["frontend", "backend", "ops"].includes(value) ? value : "ops";
}

export function initialPathForMode(value) {
  return WORK_MODE_START_PATHS[normalizeWorkMode(value)] || ROOT_PATH;
}

export function normalizePath(value) {
  const text = String(value || "").trim();
  if (!text) return "";
  if (text === ROOT_PATH || text === HOME_PATH) return text;
  if (text.startsWith("~/")) {
    const parts = text.slice(2).split("/").filter(Boolean);
    return parts.length ? `${HOME_PATH}/${parts.join("/")}` : HOME_PATH;
  }
  if (text.startsWith(ROOT_PATH)) {
    const parts = text.split("/").filter(Boolean);
    return parts.length ? `${ROOT_PATH}${parts.join("/")}` : ROOT_PATH;
  }
  return text;
}

export function parentPath(value) {
  const path = normalizePath(value);
  if (!path || path === ROOT_PATH || path === HOME_PATH) return "";
  if (path.startsWith("~/")) {
    const parts = path.slice(2).split("/").filter(Boolean);
    return parts.length <= 1 ? HOME_PATH : `${HOME_PATH}/${parts.slice(0, -1).join("/")}`;
  }
  if (path.startsWith(ROOT_PATH)) {
    const parts = path.split("/").filter(Boolean);
    return parts.length <= 1 ? ROOT_PATH : `${ROOT_PATH}${parts.slice(0, -1).join("/")}`;
  }
  return "";
}

export function pathDepth(value) {
  return value === ROOT_PATH || value === HOME_PATH
    ? 0
    : value.startsWith("~/")
      ? value.slice(2).split("/").filter(Boolean).length
      : value.split("/").filter(Boolean).length;
}

export function isSameOrChildPath(value, parent) {
  const path = normalizePath(value);
  const root = normalizePath(parent);
  return Boolean(path && root && (path === root || (root === ROOT_PATH ? path.startsWith(ROOT_PATH) : path.startsWith(`${root}/`))));
}

export function comparePaths(left, right) {
  if (left === right) return 0;
  if (left === ROOT_PATH) return -1;
  if (right === ROOT_PATH) return 1;
  const leftHome = left === HOME_PATH || left.startsWith("~/");
  return leftHome !== (right === HOME_PATH || right.startsWith("~/"))
    ? leftHome
      ? 1
      : -1
    : left.localeCompare(right, void 0, { sensitivity: "base" });
}

export function buildBreadcrumbs(value) {
  const path = normalizePath(value);
  if (!path) return [];
  if (path === HOME_PATH) return [{ label: HOME_PATH, path: HOME_PATH }];
  if (path.startsWith("~/")) {
    const parts = path.slice(2).split("/").filter(Boolean);
    const crumbs = [{ label: HOME_PATH, path: HOME_PATH }];
    for (let index = 0; index < parts.length; index += 1) {
      crumbs.push({ label: parts[index], path: `${HOME_PATH}/${parts.slice(0, index + 1).join("/")}` });
    }
    return crumbs;
  }
  if (path.startsWith(ROOT_PATH)) {
    const parts = path.split("/").filter(Boolean);
    const crumbs = [{ label: ROOT_PATH, path: ROOT_PATH }];
    for (let index = 0; index < parts.length; index += 1) {
      crumbs.push({ label: parts[index], path: `${ROOT_PATH}${parts.slice(0, index + 1).join("/")}` });
    }
    return crumbs;
  }
  return [{ label: path, path }];
}

export function displayPath(value) {
  const path = normalizePath(value);
  if (path === ROOT_PATH || path === HOME_PATH) return path;
  if (path.startsWith("~/")) {
    return path.slice(2).split("/").filter(Boolean).at(-1) || HOME_PATH;
  }
  return path.split("/").filter(Boolean).at(-1) || path;
}

export function downloadArchiveName(entries) {
  return entries.length === 1 ? `${entries[0].name}.zip` : "zshell-selected.zip";
}

export function delay(milliseconds) {
  return new Promise((resolve) => {
    window.setTimeout(resolve, milliseconds);
  });
}

export function formatSize(value) {
  return typeof value !== "number"
    ? "-"
    : value < 1024
      ? `${Math.max(0, Math.round(value))} B`
      : value < 1048576
        ? `${(value / 1024).toFixed(1)} KB`
        : value < 1073741824
          ? `${(value / 1048576).toFixed(1)} MB`
          : `${(value / 1073741824).toFixed(1)} GB`;
}

export function formatTime(value) {
  if (!value) return "-";
  const time = new Date(value);
  return Number.isNaN(time.getTime()) ? value : time.toLocaleString();
}
