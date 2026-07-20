import {
  HOME_PATH,
  ROOT_PATH,
  normalizePath,
  parentPath,
} from './fileManagerUtils.js';

function compareTreePaths(left, right) {
  if (left === right) return 0;
  if (left === ROOT_PATH) return -1;
  if (right === ROOT_PATH) return 1;
  const leftHome = left === HOME_PATH || left.startsWith('~/');
  const rightHome = right === HOME_PATH || right.startsWith('~/');
  if (leftHome !== rightHome) return leftHome ? 1 : -1;
  return left < right ? -1 : 1;
}

function treeParentPath(value) {
  if (!value || value === ROOT_PATH || value === HOME_PATH) return '';
  const separator = value.lastIndexOf('/');
  if (value.startsWith('~/')) return separator <= 1 ? HOME_PATH : value.slice(0, separator);
  if (value.startsWith(ROOT_PATH)) return separator <= 0 ? ROOT_PATH : value.slice(0, separator);
  return '';
}

function treePathLabel(value) {
  if (value === ROOT_PATH || value === HOME_PATH) return value;
  const separator = value.lastIndexOf('/');
  return separator >= 0 ? value.slice(separator + 1) : value;
}

function treeLabelWidth(value) {
  let width = 0;
  for (const character of String(value || '')) {
    width += character.codePointAt(0) > 255 ? 13 : 8;
  }
  return width;
}

function directChildrenByPath(paths) {
  const knownPaths = new Set(paths);
  const children = new Map();

  for (const childPath of paths) {
    const parent = treeParentPath(childPath);
    if (!parent || !knownPaths.has(parent)) {
      continue;
    }
    const siblings = children.get(parent) || [];
    siblings.push(childPath);
    children.set(parent, siblings);
  }

  return children;
}

export function buildTreeRows(pathMeta) {
  const paths = Array.from(pathMeta.keys()).filter(Boolean);
  const pathSet = new Set(paths);
  const childrenByPath = directChildrenByPath(paths);
  const roots = paths.filter((path) => {
    const parent = treeParentPath(path);
    return !parent || !pathSet.has(parent);
  });
  const rows = [];

  const visit = (path, visualDepth) => {
    const meta = pathMeta.get(path) || {};
    const children = childrenByPath.get(path) || [];
    const label = treePathLabel(path);
    const collapsed = Boolean(
      meta.collapsed || (meta.listingKnown !== true && children.length === 0),
    );
    const hasChildren =
      children.length > 0 ||
      (Array.isArray(meta.childDirPaths) && meta.childDirPaths.length > 0) ||
      meta.listingKnown !== true;
    const contentWidth =
      visualDepth * 14 +
      8 +
      22 +
      4 +
      treeLabelWidth(label) +
      24;
    rows.push({
      path,
      depth: visualDepth,
      label,
      opened: Boolean(meta.opened),
      collapsed,
      hasChildren,
      contentWidth,
    });

    if (!meta.collapsed) {
      for (const childPath of children) {
        visit(childPath, visualDepth + 1);
      }
    }
  };

  for (const rootPath of roots.sort(compareTreePaths)) {
    visit(rootPath, 0);
  }
  return rows;
}

export function treeContextTarget(nodePath, selectedTreePath) {
  const node = normalizePath(nodePath);
  if (!node) {
    return '';
  }
  return node === normalizePath(selectedTreePath) ? node : parentPath(node) || node;
}

export function virtualSlice(total, scrollTop, viewportHeight, rowHeight, overscan = 8) {
  const count = Math.max(0, Number(total) || 0);
  const height = Math.max(1, Number(rowHeight) || 1);
  const safeViewport = Math.max(height, Number(viewportHeight) || height);
  const safeOverscan = Math.max(0, Number(overscan) || 0);
  const visibleCount = Math.ceil(safeViewport / height);
  const maxFirstVisible = Math.max(0, count - visibleCount);
  const firstVisible = Math.min(
    maxFirstVisible,
    Math.floor(Math.max(0, Number(scrollTop) || 0) / height),
  );
  const start = Math.min(count, Math.max(0, firstVisible - safeOverscan));
  const end = Math.max(start, Math.min(count, firstVisible + visibleCount + safeOverscan));
  return { start, end };
}

export function replacePathPrefix(value, oldPrefix, newPrefix) {
  const path = normalizePath(value);
  const oldPath = normalizePath(oldPrefix);
  const newPath = normalizePath(newPrefix);
  if (!path || !oldPath || !newPath) {
    return path;
  }
  if (path === oldPath) {
    return newPath;
  }
  return path.startsWith(`${oldPath}/`) ? `${newPath}${path.slice(oldPath.length)}` : path;
}
