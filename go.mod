module murmapp.hook

go 1.20

require (
	github.com/go-chi/chi/v5 v5.0.8
	github.com/streadway/amqp v1.0.0
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace murmapp.hook/proto => ./proto
