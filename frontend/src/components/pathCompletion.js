import {
  HOME_PATH,
  ROOT_PATH,
  normalizePath,
  parentPath,
} from './fileManagerUtils.js';

export const PATH_COMPLETION_VISIBLE_LIMIT = 10;

function compareDirectoryNames(left, right) {
  if (left.name === right.name) {
    return left.path < right.path ? -1 : left.path > right.path ? 1 : 0;
  }
  return left.name < right.name ? -1 : 1;
}

export function buildDirectoryCompletionIndex(entries) {
  if (!Array.isArray(entries)) return [];
  return entries
    .filter((entry) => entry?.isDir)
    .map((entry) => ({
      name: String(entry.name || directoryName(entry.path)),
      path: normalizePath(entry.path),
    }))
    .filter((entry) => entry.name && entry.path)
    .sort(compareDirectoryNames);
}

export function resolveDirectoryCompletionQuery(draft, currentPath) {
  const raw = String(draft || '').trim();
  if (!raw) return null;

  const absolute = raw.startsWith(ROOT_PATH) || raw === HOME_PATH || raw.startsWith('~/');
  const basePath = normalizePath(currentPath) || ROOT_PATH;
  const target = normalizePath(absolute ? raw : joinDirectoryPath(basePath, raw));
  if (!target) return null;

  if (raw.endsWith('/') && raw !== ROOT_PATH) {
    return { raw, target, parent: target, fragment: '' };
  }
  if (raw === ROOT_PATH || raw === HOME_PATH) {
    return { raw, target, parent: target, fragment: '' };
  }

  const parent = parentPath(target) || (target.startsWith('~') ? HOME_PATH : ROOT_PATH);
  return {
    raw,
    target,
    parent,
    fragment: directoryName(target),
  };
}

export function findDirectoryCompletion(index, prefix, limit = PATH_COMPLETION_VISIBLE_LIMIT) {
  const items = Array.isArray(index) ? index : [];
  const fragment = String(prefix || '');
  if (!fragment || items.length === 0) return emptyCompletion();

  const start = prefixBoundary(items, fragment, false);
  const end = prefixBoundary(items, fragment, true);
  const total = Math.max(0, end - start);
  if (total === 0) return emptyCompletion();

  const first = items[start];
  const last = items[end - 1];
  const exact = first.name === fragment ? first : null;
  const longerCount = total - (exact ? 1 : 0);
  return {
    total,
    items: items.slice(start, Math.min(end, start + Math.max(1, limit))),
    commonPrefix: commonPrefix(first.name, last.name),
    exact,
    soleLonger: longerCount === 1 ? items[start + (exact ? 1 : 0)] : null,
  };
}

export function joinDirectoryPath(parent, name) {
  const normalizedParent = normalizePath(parent) || ROOT_PATH;
  const childName = String(name || '').replace(/^\/+|\/+$/g, '');
  if (!childName) return normalizedParent;
  if (normalizedParent === ROOT_PATH) return `${ROOT_PATH}${childName}`;
  return `${normalizedParent}/${childName}`;
}

export function withTrailingDirectorySlash(value) {
  const path = normalizePath(value);
  if (!path || path === ROOT_PATH) return path;
  return `${path}/`;
}

function directoryName(value) {
  const path = normalizePath(value);
  if (!path || path === ROOT_PATH || path === HOME_PATH) return path;
  return path.slice(path.lastIndexOf('/') + 1);
}

function prefixBoundary(items, prefix, upper) {
  let low = 0;
  let high = items.length;
  while (low < high) {
    const middle = (low + high) >> 1;
    const comparison = compareNameToPrefix(items[middle].name, prefix);
    if (comparison < 0 || (upper && comparison === 0)) low = middle + 1;
    else high = middle;
  }
  return low;
}

function compareNameToPrefix(name, prefix) {
  if (name.startsWith(prefix)) return 0;
  return name < prefix ? -1 : 1;
}

function commonPrefix(left, right) {
  const leftCharacters = Array.from(left);
  const rightCharacters = Array.from(right);
  const length = Math.min(leftCharacters.length, rightCharacters.length);
  let index = 0;
  while (index < length && leftCharacters[index] === rightCharacters[index]) index += 1;
  return leftCharacters.slice(0, index).join('');
}

function emptyCompletion() {
  return {
    total: 0,
    items: [],
    commonPrefix: '',
    exact: null,
    soleLonger: null,
  };
}
