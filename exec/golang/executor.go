package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chaosblade-io/chaosblade-exec-golang/chaos/action"
	"github.com/chaosblade-io/chaosblade-exec-golang/chaos/transport"
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/sirupsen/logrus"
)

var Host = spec.ExpFlag{
	Name:                  "host",
	Desc:                  "Golang application host, default value is localhost",
	NoArgs:                false,
	Required:              false,
	RequiredWhenDestroyed: false,
	Default:               "localhost",
}

var Port = spec.ExpFlag{
	Name:                  "port",
	Desc:                  "Port of injection for Golang application, default value is 9526",
	NoArgs:                false,
	Required:              false,
	RequiredWhenDestroyed: false,
	Default:               "9526",
}

var Func = spec.ExpFlag{
	Name:                  "func",
	Desc:                  "Golang application function",
	NoArgs:                false,
	Required:              true,
	RequiredWhenDestroyed: false,
	Default:               "",
}

type Executor struct {
}

func (e *Executor) Name() string {
	return "golang"
}

func (e *Executor) Exec(uid string, ctx context.Context, model *spec.ExpModel) *spec.Response {
	// transfer model to sdk
	host := model.ActionFlags[Host.Name]
	port := model.ActionFlags[Port.Name]
	_, isDestroy := spec.IsDestroy(ctx)
	var url = fmt.Sprintf("http://%s:%s%s", host, port, transport.InjectUrl)
	if isDestroy {
		url = fmt.Sprintf("http://%s:%s%s", host, port, transport.RecoverUrl)
	} else {
		if model.ActionFlags[Func.Name] == "" {
			return spec.ResponseFailWithFlags(spec.ParameterLess, Func.Name)
		}
	}
	body, err := buildRequestBody(model)
	if err != nil {
		logrus.Warnf("build request body failed, %v", err)
		return spec.ResponseFailWithFlags(spec.HttpExecFailed, url, err)
	}
	result, err, code := util.PostCurl(url, body, "")
	if err != nil {
		logrus.Warnf("post request body failed, %v", err)
		return spec.ResponseFailWithFlags(spec.HttpExecFailed, url, err)
	}
	if code == 200 {
		var resp spec.Response
		err := json.Unmarshal([]byte(result), &resp)
		if err != nil {
			log.Errorf(ctx, spec.ResultUnmarshalFailed.Sprintf(result, err))
			return spec.ResponseFailWithFlags(spec.ResultUnmarshalFailed, result, err)
		}
		return &resp
	}
	log.Errorf(ctx, spec.HttpExecFailed.Sprintf(url, result))
	return spec.ResponseFailWithFlags(spec.HttpExecFailed, url, result)
}

func (e *Executor) SetChannel(channel spec.Channel) {
}

func GetExpModel() *spec.Models {
	return &spec.Models{
		Version: "v1",
		Kind:    "plugin",
		Models: []spec.ExpCommandModel{
			{
				ExpName:         "go",
				ExpShortDesc:    "Chaos engineering experiments for golang application",
				ExpLongDesc:     "Chaos engineering experiments for golang application",
				ExpActions:      getActionModels(),
				ExpExecutor:     &Executor{},
				ExpFlags:        []spec.ExpFlag{Host, Port, Func},
				ExpScope:        "",
				ExpPrepareModel: spec.ExpPrepareModel{},
				ExpSubTargets:   nil,
			},
		},
	}
}

func getActionModels() []spec.ActionModel {
	actionModels := make([]spec.ActionModel, 0)
	actions := action.GetAllActions()
	for _, a := range actions {
		model := spec.ActionModel{
			ActionName:        a.Name(),
			ActionAliases:     []string{a.Name()},
			ActionShortDesc:   a.Name(),
			ActionLongDesc:    a.Name(),
			ActionMatchers:    getActionMatchers(),
			ActionFlags:       getActionFlags(a),
			ActionCategories:  []string{"golang"},
			ActionProcessHang: false,
		}
		actionModels = append(actionModels, model)
	}
	return actionModels
}

func getActionFlags(a action.Action) []spec.ExpFlag {
	if a.Flags() == nil {
		return []spec.ExpFlag{}
	}
	flags := make([]spec.ExpFlag, 0)
	for key, f := range a.Flags() {
		flags = append(flags, spec.ExpFlag{
			Name:     key,
			Desc:     f.Desc,
			NoArgs:   f.NoArgs,
			Required: f.Required,
		})
	}
	return flags
}

// common flags
func getActionMatchers() []spec.ExpFlag {
	flags := action.CommonActionFlagMap
	matchers := make([]spec.ExpFlag, 0)
	for key, f := range flags {
		matchers = append(matchers, spec.ExpFlag{
			Name:     key,
			Desc:     f.Desc,
			NoArgs:   f.NoArgs,
			Required: f.Required,
		})
	}
	return matchers
}

func buildRequestBody(model *spec.ExpModel) ([]byte, error) {
	// 	Body: {"target": "main.(*Business).Execute","action":"modify","Flags":{"userId":"1.3.0.1","value":"Hanmeimei","effect-count":"5"}}
	bodyMap := make(map[string]interface{}, 0)
	bodyMap["action"] = model.ActionName
	flagMap := make(map[string]string, 0)
	for k, v := range model.ActionFlags {
		if v == "" || v == "false" || k == "timeout" {
			continue
		}
		if k == Func.Name {
			bodyMap["target"] = v
			continue
		}
		if k == Host.Name || k == Port.Name {
			continue
		}
		flagMap[k] = v
	}
	bodyMap["flags"] = flagMap
	return json.Marshal(bodyMap)
}
