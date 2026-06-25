# 微型 VS Code 编辑器可行性调研

## 结论

可行，但不建议在 `0.2.2` 这个“修复终端选择异常”小版本里直接集成。当前在线编辑器是轻量 `textarea` 浮窗，启动成本极低；要补齐代码高亮、Tab 插入或缩进、`Ctrl+F` 搜索、当前命中高亮、替换等能力，推荐下一步使用 CodeMirror 6 按需加载替换编辑区。Monaco Editor 更接近 VS Code 体验，但包体和 worker 配置成本明显更高，适合后续需要 IntelliSense、诊断、符号跳转等强 IDE 能力时再引入。

## 方案对比

| 方案 | 能力匹配 | 性能影响 | 集成成本 | 结论 |
| --- | --- | --- | --- | --- |
| 保留 `textarea` 并手写增强 | 可实现 Tab 和简单查找，但语法高亮与替换高亮成本高 | 最低 | 中高，长期维护成本高 | 不推荐继续堆功能 |
| CodeMirror 6 | 支持高亮、缩进、搜索、替换、快捷键和可扩展语言包 | 低到中，可动态 import | 中 | 推荐 |
| Monaco Editor | VS Code 同源编辑器，能力最完整 | 中到高，`monaco-editor` npm 包当前解包体积约 72 MB | 高，需要 worker/Vite 配置和懒加载 | 暂不推荐作为默认路线 |

## 推荐落地方式

1. 新增 `RemoteCodeEditor.vue`，只在用户打开在线文本编辑器时动态加载 CodeMirror 相关包，避免影响 zShell 首页、终端和文件列表启动速度。
2. 初期只加载基础能力：`@codemirror/view`、`@codemirror/state`、`@codemirror/search`、`@codemirror/commands`、`@codemirror/language`、`@codemirror/lang-javascript`、`@codemirror/lang-html`、`@codemirror/lang-css`、`@codemirror/lang-json`。
3. 编辑器窗口继续复用现有 `FileManager.vue` 的打开、保存、脏关闭、多窗口和最大化逻辑，避免改动后端远程文本读写契约。
4. Tab 键由编辑器内部 keymap 处理，插入缩进或执行语言缩进，不再落到页面焦点导航，因此不会被顶部 `zShell`、`配置管理`、`UI管理` 菜单焦点链路抢走。
5. 文件大小继续沿用当前 10 MB 文本限制；超过阈值保留下载或只读提示，避免大文件高亮拖慢 WebView2。

## 性能控制

- 不在应用入口静态 import 编辑器依赖。
- 按文件后缀动态选择语言包，未知类型只加载纯文本模式。
- 首次打开编辑器前显示当前浮窗 loading，加载完成后挂载 CodeMirror。
- 多编辑器窗口共享已加载模块，不重复下载或初始化语言基础包。
- 搜索和替换仅在活动编辑器内启用，关闭窗口时销毁 view。

## 参考资料

- Monaco Editor 官方站点：https://microsoft.github.io/monaco-editor/
- Monaco Editor npm：`monaco-editor@0.55.1`，`dist.unpackedSize` 约 72 MB，MIT。
- CodeMirror 官方文档：https://codemirror.net/docs/
- CodeMirror npm：`codemirror@6.0.2`、`@codemirror/view@6.43.2`、`@codemirror/search@6.7.1`，MIT。
