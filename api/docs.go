package api

import _ "embed"

//go:embed openapi.yml
var Spec []byte

//go:embed swagger.html
var SwaggerPage []byte
