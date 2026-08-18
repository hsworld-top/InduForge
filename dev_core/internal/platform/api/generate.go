package api

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.6.0 -generate types,chi-server,spec -package api -o generated.go ../../../api/openapi.yaml
//go:generate go run strip_custom_routes.go -- generated.go
