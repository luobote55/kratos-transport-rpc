{{$svrType := .ServiceType}}
{{$svrName := .ServiceName}}

{{- range .MethodSets}}
//const Operation{{$svrType}}{{.OriginalName}} = "/{{$svrName}}/{{.OriginalName}}"
{{- end}}

type {{.ServiceType}}MQTTServer interface {
{{- range .MethodSets}}
	{{- if ne .Comment ""}}
	{{.Comment}}
	{{- end}}
	{{.Name}}(context.Context, *{{.Request}}) (*{{.Reply}}, error)
{{- end}}
}

func Register{{.ServiceType}}MQTTServer(s *mqtt.Server, srv {{.ServiceType}}MQTTServer) {
	r := s.Route("")
	{{- range .Methods}}
	r.{{.Method}}("{{.Path}}", _{{$svrType}}_{{.Name}}{{.Num}}_MQTT_Handler(srv))
	{{- end}}
}

{{range .Methods}}
func _{{$svrType}}_{{.Name}}{{.Num}}_MQTT_Handler(srv {{$svrType}}MQTTServer) func(ctx context.Context, req interface{}) (interface{}, error) {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		var in {{.Request}}
//		mqtt.SetOperation(ctx,Operation{{$svrType}}{{.OriginalName}})
		h := middleware.Handler(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.{{.Name}}(ctx, req.(*{{.Request}}))
		})
		return h(ctx, &in)
	}
}
{{end}}

type {{.ServiceType}}MQTTClient interface {
{{- range .MethodSets}}
	{{.Name}}(ctx context.Context, req *{{.Request}}) (rsp *{{.Reply}}, err error)
{{- end}}
}

type {{.ServiceType}}MQTTClientImpl struct{
	cc *mqtt.Client
}

func New{{.ServiceType}}MQTTClient (client *mqtt.Client) {{.ServiceType}}MQTTClient {
	return &{{.ServiceType}}MQTTClientImpl{client}
}

{{range .MethodSets}}
func (c *{{$svrType}}MQTTClientImpl) {{.Name}}(ctx context.Context, in *{{.Request}}) (*{{.Reply}}, error) {
	var out {{.Reply}}
	pattern := "{{.Path}}"
	err := c.cc.Invoke(ctx, "{{.Method}}", pattern, in, &out)
	if err != nil {
		return nil, err
	}
	return &out, err
}
{{end}}
