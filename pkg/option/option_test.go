package option

import (
	"fmt"
	"go/token"
	"testing"

	"github.com/go-mosaic/gomosaic/pkg/gomosaic"
)

type OpenAPI struct {
	Test    string          `option:"name" valid:"required"`
	Headers []OpenapiHeader `option:"header,inline"`
	Tags    []string        `option:"tags"`
	Ints    []int           `option:"ints"`
}

type OpenapiHeader struct {
	Name     string `option:",fromValue" valid:"required"`
	Title    string `option:"title,fromParam"`
	Required bool   `option:",fromOption"`
}

type ErrorWrapper struct {
	Path          string `option:",fromValue" valid:"required"`
	InterfaceName string `option:"iface,fromParam" valid:"required"`
}

type testOption struct {
	Name         string       `option:"" valid:"in,params:'complex value'"`
	ApiDocEnable bool         `option:"api-doc"`
	OpenAPI      OpenAPI      `option:"openapi"`
	ErrorWrapper ErrorWrapper `option:"error-wrapper,inline"`
}

func TestUnmarshal(t *testing.T) {
	type args struct {
		prefix   string
		comments []*gomosaic.CommentInfo
		v        any
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "sucess",
			args: args{
				prefix: "http",
				comments: []*gomosaic.CommentInfo{
					{
						Value:        "@http-api-doc",
						IsAnnotation: true,
						Position:     token.Position{},
					},
					{
						Value:        "@http-name complex",
						IsAnnotation: true,
						Position:     token.Position{},
					},
					{
						Value:        "@http-openapi-name name",
						IsAnnotation: true,
						Position:     token.Position{},
					},
					{
						Value:        "@http-openapi-tags tag1,tag2,tag3",
						IsAnnotation: true,
						Position:     token.Position{},
					},
					{
						Value:        "@http-openapi-ints 1,2,3,4,5",
						IsAnnotation: true,
						Position:     token.Position{},
					},
					{
						Value:        "@http-openapi-header name,required,title=\"oh no\"",
						IsAnnotation: true,
						Position:     token.Position{},
					},
					{
						Value:        "@http-openapi-header name2,name=test,title=\"oh no 2\"",
						IsAnnotation: true,
						Position:     token.Position{},
					},
					{
						Value:        "@http-error-wrapper test,iface=test",
						IsAnnotation: true,
						Position:     token.Position{},
					},
				},
				v: &testOption{
					Name:         "",
					ApiDocEnable: false,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags, err := gomosaic.ParseAnnotations(tt.args.comments)
			if err != nil {
				t.Errorf("ParseTags() error = %v", err)
			}
			if err := Unmarshal(tt.args.prefix, tags, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			fmt.Println(tt.args.v)
		})
	}
}
