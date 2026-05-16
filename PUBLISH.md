# 发布 `omnivideo-sdk-go`（Go modules）

Go modules 没有「中心化的包管理平台」要上传 —— 它直接从公开 Git 仓库的 tag 拉代码，Go proxy（`proxy.golang.org`）会在首次有人 `go get` 时自动收录。

## 前置条件

1. 准备一个公开的 Git 托管账号（GitHub / GitLab / 自托管 Gitea 均可，推荐 GitHub）。
2. 仓库地址要和 `go.mod` 里的 `module` 路径**完全一致**。当前模块路径是：

   ```
   module github.com/omnivideo/omnivideo-sdk-go
   ```

   也就是说你需要：
   - 一个 GitHub 组织或个人账号叫 `omnivideo`（如果用别的名字，要同步改 `go.mod` 第一行 + README 安装命令）。
   - 在该账号下创建公开仓库 `omnivideo-sdk-go`。
3. 一个有 `repo` 权限的 GitHub Personal Access Token（用于推送代码 + 打 tag）。在 <https://github.com/settings/tokens/new> 生成，scope 勾 `public_repo` 即可。

## 发布步骤

```bash
cd golang
git init
git add .
git commit -m "Initial release v0.1.0"
git branch -M main
git remote add origin https://<GITHUB_TOKEN>@github.com/omnivideo/omnivideo-sdk-go.git
git push -u origin main
git tag v0.1.0
git push origin v0.1.0
```

打完 tag 后，任何人都可以：

```bash
go get github.com/omnivideo/omnivideo-sdk-go@v0.1.0
```

第一次 `go get` 时，Go proxy 会拉到自己的镜像，之后就稳定可用了。

## 后续版本升级

按 [SemVer](https://semver.org/) 打 tag：

```bash
git tag v0.2.0
git push origin v0.2.0
```

> 注意：发布 `v2.0.0` 及以上**主版本号**时，`go.mod` 里要改成 `module github.com/omnivideo/omnivideo-sdk-go/v2`，目录里再加 `/v2` 子目录或在主分支 stash major 升级。具体见 <https://go.dev/ref/mod#major-version-suffixes>。
