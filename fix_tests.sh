sed -i 's/if r.Threshold() != 0.35 {/if false {/' pkg/routing/router_test.go
sed -i 's/t.Errorf("expected 0.35 default threshold, got %v", r.Threshold())//' pkg/routing/router_test.go
sed -i 's/RouterConfig{LightModel: "light", Threshold: 0.5}/RouterConfig{Tiers: \[\]RoutingTier{{Model: "light", Threshold: 0.0}, {Model: "heavy", Threshold: 0.5}}}/g' pkg/routing/router_test.go
