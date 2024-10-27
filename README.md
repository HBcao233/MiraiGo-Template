# MiraiGo-Template

A template for MiraiGo

Fork from https://github.com/Logiase/MiraiGo-Template (对原仓库做了较大改动, 暂不merge)

[![Go Report Card](https://goreportcard.com/badge/github.com/Logiase/MiraiGo-Template)](https://goreportcard.com/report/github.com/Logiase/MiraiGo-Template)

基于 [MiraiGo](https://github.com/Mrs4s/MiraiGo) 的多模块组合设计

包装了基础功能,同时设计了一个~~良好~~的项目结构

## 不了解go?

golang 极速入门

[点我看书](https://github.com/justjavac/free-programming-books-zh_CN#go)

## 使用方法

1. 将 [config.yml.example](./config.yml.example) 改为 `config.yml` 修改配置

2. 运行 `go run main.go` 

## 插件配置 Plugins Configure

插件参考 [ping/ping.go](./plugins/ping/ping.go) 编写自己的插件，然后在[main.go](./main.go) 中启用插件

```go
import (
   _ "plugins/ping"
)
```

# 示例

* [HBcao233/qbotGo](https://github.com/HBcao233/qbotGo)
