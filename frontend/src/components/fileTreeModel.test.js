import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildTreeRows,
  replacePathPrefix,
  treeContextTarget,
  virtualSlice,
} from './fileTreeModel.js';

function meta(paths) {
  return new Map(paths.map((path) => [path, { opened: true, collapsed: false }]));
}

test('tree renders every directory on its own row without compacting chains', () => {
  const paths = [
    '/',
    '/opt',
    '/src',
    '/src/main',
    '/src/main/java',
    '/src/main/java/com',
    '/src/main/java/com/chuangyi',
    '/src/main/java/com/chuangyi/test',
    '/src/main/java/com/chuangyi/test/module',
    '/src/main/resources',
  ];

  const rows = buildTreeRows(meta(paths));
  const javaRow = rows.find((row) => row.path === '/src/main/java');
  const comRow = rows.find((row) => row.path === '/src/main/java/com');
  const moduleRow = rows.find((row) => row.path === '/src/main/java/com/chuangyi/test/module');

  assert.equal(javaRow.label, 'java');
  assert.equal(comRow.label, 'com');
  assert.ok(moduleRow);
  assert.equal(moduleRow.depth, javaRow.depth + 4);
  assert.equal(rows.length, paths.length);
});

test('tree context targets selected node or the unselected node parent', () => {
  assert.equal(treeContextTarget('/opt/1', '/opt/1'), '/opt/1');
  assert.equal(treeContextTarget('/opt/2', '/opt/1'), '/opt');
  assert.equal(treeContextTarget('/', '/opt'), '/');
});

test('virtual slice keeps a bounded overscanned window', () => {
  assert.deepEqual(virtualSlice(1000, 2900, 290, 29, 5), { start: 95, end: 115 });
  assert.deepEqual(virtualSlice(2, 0, 290, 29, 5), { start: 0, end: 2 });
  assert.deepEqual(virtualSlice(2, 10000, 290, 29, 5), { start: 0, end: 2 });

  for (const total of [0, 1, 2, 1000]) {
    for (const scrollTop of [-1, 0, 29, 10000, Number.MAX_SAFE_INTEGER]) {
      const range = virtualSlice(total, scrollTop, 290, 29, 5);
      assert.ok(range.start >= 0);
      assert.ok(range.start <= range.end);
      assert.ok(range.end <= total);
    }
  }
});

test('replace path prefix only updates the renamed subtree', () => {
  assert.equal(replacePathPrefix('/opt/old/child', '/opt/old', '/opt/new'), '/opt/new/child');
  assert.equal(replacePathPrefix('/opt/older', '/opt/old', '/opt/new'), '/opt/older');
});

test('root and home anchors each keep their own row', () => {
  const rows = buildTreeRows(meta(['/', '/only', '~', '~/only']));

  assert.equal(rows.find((row) => row.path === '/').label, '/');
  assert.equal(rows.find((row) => row.path === '~').label, '~');
});

test('an unknown directory keeps an expand control until its listing is known', () => {
  const pathMeta = meta(['/']);
  pathMeta.set('/unknown', { opened: false, collapsed: true, listingKnown: false });
  const rows = buildTreeRows(pathMeta);

  assert.equal(rows.find((row) => row.path === '/unknown').hasChildren, true);
  assert.equal(rows.find((row) => row.path === '/unknown').collapsed, true);
});

test('a collapsed directory hides its descendants', () => {
  const pathMeta = meta(['/', '/parent', '/parent/child', '/parent/child/nested']);
  pathMeta.set('/parent/child', { opened: true, collapsed: true });
  const rows = buildTreeRows(pathMeta);
  const row = rows.find((item) => item.path === '/parent/child');

  assert.equal(row.collapsed, true);
  assert.equal(rows.some((item) => item.path === '/parent/child/nested'), false);
});
