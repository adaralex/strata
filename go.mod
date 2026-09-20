module github.com/adaralex/strata

go 1.24.7

require (
	github.com/paulmach/orb v0.13.0
	github.com/paulmach/osm v0.9.0
	github.com/uber/h3-go/v4 v4.5.0
)

require (
	github.com/DataDog/czlib v0.0.0-20240814115052-86a9592b3985 // indirect
	github.com/paulmach/protoscan v0.2.1 // indirect
	go.mongodb.org/mongo-driver/v2 v2.5.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace github.com/DataDog/czlib => ./third_party/czlib
