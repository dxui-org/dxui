package theme

import "testing"

func TestResolveThreeTypedLayers(t *testing.T) {
	source := Source{
		PrimitiveColors:  map[string]Color{"blue": {B: 255, A: 255}},
		PrimitiveMetrics: map[string]float32{"space.2": 8},
		SemanticColors: map[string]ColorValue{
			"accent": {Token: "blue", IsToken: true, Set: true},
		},
		SemanticMetrics: map[string]MetricValue{
			"control.gap": {Token: "space.2", IsToken: true, Set: true},
		},
	}
	resolved, err := Resolve(source)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Metrics["control.gap"] != 8 || resolved.Colors["accent"].B != 255 {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveRejectsCyclesAndMissingTokensAtomically(t *testing.T) {
	for name, source := range map[string]Source{
		"cycle": {SemanticMetrics: map[string]MetricValue{
			"a": {Token: "b", IsToken: true, Set: true},
			"b": {Token: "a", IsToken: true, Set: true},
		}},
		"missing": {SemanticColors: map[string]ColorValue{
			"accent": {Token: "absent", IsToken: true, Set: true},
		}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Resolve(source); err == nil {
				t.Fatal("invalid theme accepted")
			}
		})
	}
}
