# Contributing

感谢参与 `pgparser` 的开发。本文档说明提交代码前需要完成的检查。

## 开发环境

- Go 1.26 或更高版本
- `golangci-lint`（可选，用于本地 lint）

## 流程

1. Fork 并克隆仓库：`git clone https://github.com/mia0x75/pg-parser.git`
2. 从 `main` 创建分支：`git checkout -b my-feature`
3. 修改代码并补充测试
4. 运行 `make`（vet + test + lint）
5. 提交 Pull Request

## 生成解析器

解析器由 `parser/grammar.y` 通过 `golang.org/x/tools/cmd/goyacc` 生成：

```bash
go run golang.org/x/tools/cmd/goyacc@v0.52.0 -o parser/parser_generated.go -l -p yy parser/grammar.y
```

- 不要手工修改 `parser/parser_generated.go`。
- 新增产生式后必须重新生成，并确认冲突数没有异常增长。
- 需要查看状态机与冲突时，可加 `-v y.output`；`y.output` 已被 `.gitignore` 忽略。

## 测试

```bash
make test
# 等价于
go test -race -count=1 ./...
```

PostgreSQL 回归语料的解析诊断默认跳过：

```bash
PGPARSER_TESTDATA=1 go test ./tests -run TestParseTestdataSQL -v
```

## 代码约定

- 遵循标准 Go 风格（`gofmt`、`go vet`）。
- 新增语法时同步补充 `tests/` 中的解析器黑盒单测；包内测试应与 parser 源码同目录。PostgreSQL 官方语料门禁也位于 `tests/`。
- PostgreSQL 独有的 AST 字段或节点请放在 `ast/` 中职责对应的文件；生成节点位于 `ast/nodes_generated.go`，不要删除或改写既有字段。
- 新增依赖前请先开 issue 讨论。

## 许可证

提交代码即表示同意以 [MIT](LICENSE) 或 [Anti-996](LICENSE-ANTI-996) 双协议授权你的贡献。
