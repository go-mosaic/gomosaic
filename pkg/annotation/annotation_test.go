package annotation

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    *Annotation
		wantErr bool
	}{
		{
			name: "успешный парсинг",
			args: args{
				s: `@http-error "result Result" "dvsv" '234324 dsv' type="int dsvsdv sdvsdv" description="Some \"error\" description"`,
			},
			want: &Annotation{
				Key: "http-error",
				Options: []string{
					"result Result",
					"dvsv",
					"234324 dsv",
				},
				Params: map[string]string{
					"type":        "int dsvsdv sdvsdv",
					"description": "Some \"error\" description",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Key != tt.want.Key {
				t.Errorf("Parse() Key = %v, want %v", got, tt.want)
			}

			if got.Value() != tt.want.Value() {
				t.Errorf("Parse() Value = %v, want %v", got, tt.want)
			}

			if !reflect.DeepEqual(got.Options, tt.want.Options) {
				t.Errorf("Parse() Options = %v, want %v", got, tt.want)
			}

			if !reflect.DeepEqual(got.Params, tt.want.Params) {
				t.Errorf("Parse() Params = %v, want %v", got, tt.want)
			}

		})
	}
}
