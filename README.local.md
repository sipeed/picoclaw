# 开发流程与贡献规范
本仓库是上游开源项目的公司内部 Fork，用于支持公司业务开发，同时保持与上游项目同步，并将通用修复贡献回上游。

目标：
- 支持 业务快速修复 bug 和发布
- 保持 Fork 尽量接近上游仓库
- 可选：将通用 bug 修复 贡献回上游

上游项目：picoclaw

main 分支用于 同步 upstream 的 main 分支。 尽量保持与 upstream/main 一致
** 不要在 main 上直接开发，而是基于internal-main **。所有开发必须通过分支进行， 如feature 、bugfix。 

```
upstream/main      官方代码
        ↓
main               upstream镜像（只同步）
        ↓
internal-main      公司内部开发主线
        ↓
feature/*
bugfix/*
```


##  开发流程

1 同步 upstream （青柠操作）
```
git fetch upstream
git checkout main
git merge upstream/main
```
2 更新 internal-main（青柠操作）
```
git checkout internal-main
git merge main
```

3 开发 bugfix
```
git checkout -b bugfix/parser-crash internal-main
```

4 开发完成，提交 PR 到 internal-main
```
git push origin bugfix/parser-crash
```

## 提交 upstream PR
如果某个修复适合 upstream
从 main 开 PR 分支：
```
git checkout main
git checkout -b pr/fix-parser-crash
```
然后：
```
git cherry-pick <commit>
```

推送：

```
git push github-xxxx pr/fix-parser-crash
```

去 GitHub 提 PR。