module murmapp.hook

go 1.24.1

require (
	github.com/eugene-ruby/xconnect v0.3.3
	github.com/eugene-ruby/xencryptor v0.2.3
	github.com/go-chi/chi/v5 v5.0.8
	github.com/streadway/amqp v1.1.0
	github.com/stretchr/testify v1.10.0
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace murmapp.hook/proto => ./proto
