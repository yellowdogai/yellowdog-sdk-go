package model

import (
	"reflect"
	"testing"
)

func TestPolymorphicInterfacesDoNotAcceptOtherHierarchies(t *testing.T) {
	interfaces := map[string]reflect.Type{
		"NodeAction":          reflect.TypeFor[NodeAction](),
		"ComputeSource":       reflect.TypeFor[ComputeSource](),
		"AttributeValue":      reflect.TypeFor[AttributeValue](),
		"AttributePreference": reflect.TypeFor[AttributePreference](),
		"ClusterProvider":     reflect.TypeFor[ClusterProvider](),
	}
	subtypes := map[string]any{
		"NodeAction":          &NodeRunCommandAction{},
		"ComputeSource":       &AwsInstancesComputeSource{},
		"AttributeValue":      &NumericAttributeValue{},
		"AttributePreference": &NumericAttributePreference{},
		"ClusterProvider":     &EksClusterProvider{},
	}

	for owner, subtype := range subtypes {
		for name, iface := range interfaces {
			implements := reflect.TypeOf(subtype).Implements(iface)
			if name == owner && !implements {
				t.Errorf("%T does not implement its own interface %s", subtype, name)
			}
			if name != owner && implements {
				t.Errorf("%T implements %s, an interface of another hierarchy", subtype, name)
			}
		}
	}
}
