package tpl

import _ "embed"

//go:embed route.tpl
var RouteTpl string

//go:embed biz.tpl
var BizTpl string

//go:embed biz_iface.tpl
var BizIfaceTpl string

//go:embed biz_params.tpl
var BizParamsTpl string

//go:embed biz_contract.tpl
var BizContractTpl string

//go:embed repo.tpl
var RepoTpl string

//go:embed cmd.tpl
var CmdTpl string

//go:embed cmd_app.tpl
var CmdAppTpl string

//go:embed biz_test.tpl
var BizTestTpl string
