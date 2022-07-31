package tpl

import _ "embed"

//go:embed route.tpl
var RouteTpl string

//go:embed biz.tpl
var BizTpl string

//go:embed repo.tpl
var RepoTpl string
