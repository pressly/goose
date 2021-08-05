package goose

import (
	"database/sql/driver"
	"reflect"
	"testing"
)

func Test_booler_Scan(t *testing.T) {
	type args struct {
		src interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    booler
		wantErr bool
	}{
		{"bool_t", args{true}, booler(true), false},
		{"bool_f", args{false}, booler(false), false},
		{"byte_0", args{[]byte{0}}, booler(false), false},
		{"byte_1", args{[]byte{1}}, booler(true), false},
		{"byte_long", args{[]byte("too_long")}, booler(false), true},
		{"int_2", args{2}, booler(false), false},
		{"int_1", args{1}, booler(true), false},
		{"int_0", args{0}, booler(false), false},
		{"str_y", args{"y"}, booler(true), false},
		{"str_Y", args{"Y"}, booler(true), false},
		{"str_t", args{"t"}, booler(true), false},
		{"str_yes", args{"yes"}, booler(true), false},
		{"str_true", args{"true"}, booler(true), false},
		{"str_other", args{"x"}, booler(false), false},
		{"str_not_yes", args{"yas"}, booler(false), false},
		{"invalid type", args{float32(4.2)}, booler(false), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b booler
			if err := b.Scan(tt.args.src); (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if bool(b) != bool(tt.want) {
				t.Errorf("Scan() want=%v, got=%v", tt.want, b)
			}
		})
	}
}

func Test_booler_Value(t *testing.T) {
	tests := []struct {
		name    string
		b       booler
		want    driver.Value
		wantErr bool
	}{
		{"true", booler(true), driver.Value("Y"), false},
		{"false", booler(false), driver.Value("N"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.b.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Value() got = %v, want %v", got, tt.want)
			}
		})
	}
}
