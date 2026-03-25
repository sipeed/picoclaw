sed -i '244,256d' pkg/routing/router_test.go
sed -i 's/r.LightModel()/"my-fast-model"/g' pkg/routing/router_test.go
