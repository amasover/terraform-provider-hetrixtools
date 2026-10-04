package provider

import (
	"context"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nullUnknowns sets every attribute of model (a pointer to a resource model) that is
// still unknown to null. Create and Update save the plan as the new state without
// reading the resource back, so an optional, computed attribute the configuration leaves
// unset is still unknown there, and Terraform refuses an unknown value after apply
// ("Provider returned invalid result object after apply"). Null is always accepted, and
// the next refresh fills the attribute from the API through Read.
func nullUnknowns(model any) {
	nullUnknownFields(reflect.ValueOf(model).Elem())
}

func nullUnknownFields(v reflect.Value) {
	for i := range v.NumField() {
		f := v.Field(i)
		if !f.CanInterface() {
			// An embedded model of an unexported type: its exported fields can still be set.
			if f.Kind() == reflect.Struct {
				nullUnknownFields(f)
			}
			continue
		}
		switch x := f.Interface().(type) {
		case types.String:
			if x.IsUnknown() {
				f.Set(reflect.ValueOf(types.StringNull()))
			}
		case types.Int64:
			if x.IsUnknown() {
				f.Set(reflect.ValueOf(types.Int64Null()))
			}
		case types.Bool:
			if x.IsUnknown() {
				f.Set(reflect.ValueOf(types.BoolNull()))
			}
		case types.List:
			if x.IsUnknown() {
				f.Set(reflect.ValueOf(types.ListNull(x.ElementType(context.Background()))))
			}
		case types.Set:
			if x.IsUnknown() {
				f.Set(reflect.ValueOf(types.SetNull(x.ElementType(context.Background()))))
			}
		default:
			if f.Kind() == reflect.Struct {
				nullUnknownFields(f)
			}
		}
	}
}
