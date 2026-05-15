# Agent 指令

本项目使用 **bd** (beads) 进行任务追踪。运行 `bd prime` 获取完整工作流上下文。

> **架构概述：** 任务存储在本地 Dolt 数据库（`.beads/dolt/`）；跨机器同步使用 `bd dolt push/pull`（git 兼容协议），数据存放在 git remote 的 `refs/dolt/data` 下 —— 与 `refs/heads/*` 中的代码分离。`.beads/issues.jsonl` 是被动导出，不是传输协议。
>
> 详见 [SYNC_CONCEPTS.md](https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md)（一屏概览和反模式：不要把 JSONL 当作数据源；不要在正常运行时使用 `bd import`；不要在尝试默认方案前就去找第三方 Dolt 托管）。

## 快速参考

```bash
bd ready              # 查看可用任务
bd show <id>          # 查看任务详情
bd update <id> --claim  # 认领任务
bd close <id>         # 完成任务
bd dolt push          # 推送 beads 数据到远程
```

## 非交互式 Shell 命令

**始终使用非交互式参数**，避免在确认提示处挂起。

`cp`、`mv`、`rm` 等命令在某些系统上可能被别名为 `-i`（交互）模式，导致 agent 无限等待 y/n 输入。

**请使用以下形式：**
```bash
# 强制覆盖，不提示
cp -f source dest           # 不要用: cp source dest
mv -f source dest           # 不要用: mv source dest
rm -f file                  # 不要用: rm file

# 递归操作
rm -rf directory            # 不要用: rm -r directory
cp -rf source dest          # 不要用: cp -r source dest
```

**其他可能弹出提示的命令：**
- `scp` - 使用 `-o BatchMode=yes`
- `ssh` - 使用 `-o BatchMode=yes`，失败而非提示
- `apt-get` - 使用 `-y` 参数
- `brew` - 使用 `HOMEBREW_NO_AUTO_UPDATE=1` 环境变量

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:7510c1e2 -->
## Beads 任务追踪器

本项目使用 **bd (beads)** 进行任务追踪。运行 `bd prime` 获取完整工作流和命令参考。

### 快速参考

```bash
bd ready              # 查看可用任务
bd show <id>          # 查看任务详情
bd update <id> --claim  # 认领任务
bd close <id>         # 完成任务
```

### 规则

- 所有任务追踪使用 `bd` —— 禁止使用 TodoWrite、TaskCreate 或 markdown TODO 列表
- 运行 `bd prime` 获取详细命令参考和会话关闭协议
- 使用 `bd remember` 存储持久知识 —— 禁止使用 MEMORY.md 文件

**架构概述：** 任务存储在本地 Dolt 数据库；同步使用 git remote 的 `refs/dolt/data`；`.beads/issues.jsonl` 是被动导出。详见 https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md。

## 会话结束

**结束工作会话时**，必须完成以下所有步骤。工作在 `git push` 成功之前不算完成。

**必须执行的工作流：**

1. **为剩余工作创建任务** - 为需要后续跟进的内容创建 issue
2. **运行质量检查**（如果代码有变动）- 测试、lint、构建
3. **更新任务状态** - 关闭已完成的工作，更新进行中的任务
4. **推送到远程** - 这是必须的：
   ```bash
   git pull --rebase
   git push
   git status  # 必须显示 "up to date with origin"
   ```
5. **清理** - 清除 stash，清理远程分支
6. **验证** - 所有变更已提交且已推送
7. **交接** - 为下一次会话提供上下文

**关键规则：**
- `git push` 成功之前工作不算完成
- 绝不在推送前停止 —— 那会让工作滞留在本地
- 绝不说"准备好就可以推送" —— 你必须自己推送
- 如果推送失败，解决后重试直到成功
<!-- END BEADS INTEGRATION -->
