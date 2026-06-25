# 微型 VS Code 编辑器可行性调研

## 结论

可行，且 `0.3.1` 已选择 Monaco Editor 一步到位重构文件编辑器。当前实现保留原有远程文本读写、多编辑窗口、拖动、最小化、最大化、保存和脏关闭流程，把编辑区从 `textarea` 替换为 Monaco，并通过动态 import、Web Worker 懒加载和启动后 idle 预热降低首屏影响。

## 方案对比

| 方案 | 能力匹配 | 性能影响 | 集成成本 | 结论 |
| --- | --- | --- | --- | --- |
| 保留 `textarea` 并手写增强 | 可实现 Tab 和简单查找，但语法高亮与替换高亮成本高 | 最低 | 中高，长期维护成本高 | 不推荐继续堆功能 |
| CodeMirror 6 | 支持高亮、缩进、搜索、替换、快捷键和可扩展语言包 | 低到中，可动态 import | 中 | 保留为备选 |
| Monaco Editor | VS Code 同源编辑器，能力最完整 | 中到高，`monaco-editor` npm 包当前解包体积约 72 MB | 高，需要 worker/Vite 配置和懒加载 | 已采用 |

## 推荐落地方式

1. `RemoteCodeEditor.vue` 只在用户打开在线文本编辑器时动态加载 Monaco；`editorWarmup.js` 会在应用启动后空闲时后台预热同一个加载 Promise。
2. `monacoLoader.js` 配置 editor/json/css/html/typescript workers，并使用 Vite `?worker` 产出独立 worker 资产。
3. 编辑器窗口继续复用现有 `FileManager.vue` 的打开、保存、脏关闭、多窗口和最大化逻辑，避免改动后端远程文本读写契约。
4. Tab 键由 Monaco 内部处理，插入缩进或执行语言缩进，不再落到页面焦点导航，因此不会被顶部 `zShell`、`配置管理`、`UI管理` 菜单焦点链路抢走。
5. 文件大小继续沿用当前 10 MB 文本限制；超过阈值保留下载或只读提示，避免大文件高亮拖慢 WebView2。

## 性能控制

- 不在应用入口静态 import 编辑器依赖。
- 按文件后缀选择 Monaco language id，未知类型使用纯文本模式。
- 首次打开编辑器前显示当前浮窗 loading，加载完成后挂载 Monaco。
- 多编辑器窗口共享已加载 Monaco 模块和 worker 配置，不重复初始化加载流程。
- 搜索和替换仅在活动编辑器内启用，关闭窗口时销毁 view。

## 参考资料

- Monaco Editor 官方站点：https://microsoft.github.io/monaco-editor/
- Monaco Editor npm：`monaco-editor@0.55.1`，`dist.unpackedSize` 约 72 MB，MIT。
- CodeMirror 官方文档：https://codemirror.net/docs/
- CodeMirror npm：`codemirror@6.0.2`、`@codemirror/view@6.43.2`、`@codemirror/search@6.7.1`，MIT。
