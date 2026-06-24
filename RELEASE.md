# zShell 发布说明

## 版本号规则

当前版本号写在根目录 `VERSION` 文件中。初始版本为 `0.0.1`。

默认情况下，后续修改只递增最后一位，例如 `0.0.1` 之后是 `0.0.2`。只有在明确需要大版本或中版本跳跃时，才改成类似 `1.0.1` 或 `0.1.1`。

## 本地打包

在项目根目录运行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\build-windows.ps1
```

脚本会执行：

- `npm ci`
- `npm run build`
- 复制前端资源到后端嵌入目录
- `go test ./...`
- `wails build`
- 输出 `release\zshell.<版本号>.exe`

每次构建后，`release` 文件夹只保留一个 `.exe`。

## GitHub Release

仓库包含 `.github/workflows/release.yml`。

发布方式：

- 推送 tag，例如 `v0.0.1`。
- 或在 GitHub Actions 中手动触发“发布 zShell”，填写版本号。

工作流会在 Windows runner 上构建 exe，生成 SHA256 校验文件，并创建或更新 GitHub Release。Release 资产命名为：

```text
zshell.<版本号>.exe
zshell.<版本号>.exe.sha256
```

应用内更新功能会读取最新 GitHub Release，查找同名 exe 资产，下载并替换当前运行的程序。
