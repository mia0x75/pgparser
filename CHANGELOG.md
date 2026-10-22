# Changelog

本项目的重要变更记录在此文件中，格式参考 [Keep a Changelog](https://keepachangelog.com/)。

## Unreleased

### Changed

- 解析器重写为基于 goyacc 的单一 `parser.y`，移除 ANTLR 运行时与生成代码。
- AST 以 PostgreSQL 语义为准，独有节点与字段集中在 `ast/pg_ext.go`、`ast/pg_nodes_generated.go`。
- 许可证调整为 MIT 与 Anti-996 双协议。

### Added

- 完整支持 `GRANT`/`REVOKE` 的 PostgreSQL 目标类型。
- 支持 `WITH ORDINALITY`、`ROWS FROM`、表函数列定义。
- 支持 `JOIN ... USING (...) AS alias`。
- 支持 `PARTITION OF` 的分区元素、`ON COMMIT` 与访问方法。
- 支持 `JSON_OBJECTAGG`、`JSON_ARRAYAGG` 等 JSON 聚合函数。
- 支持 `TABLESAMPLE`、`REINDEX ... SYSTEM`、`MOVE FORWARD ALL`、`CREATE OR REPLACE RECURSIVE VIEW`。
- 支持 `WHEN NOT MATCHED BY SOURCE THEN UPDATE`。
- 新增基于 PostgreSQL 官方回归语料的解析诊断测试（默认跳过）。
