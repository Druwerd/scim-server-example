package main

import (
	"log"
	"net/http"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
)

func main() {
	config := scim.ServiceProviderConfig{
		DocumentationURI: optional.NewString("www.example.com/scim"),
	}

	resourceTypes := []scim.ResourceType{
		userType,
	}

	server := scim.Server{
		Config:        config,
		ResourceTypes: resourceTypes,
	}

	log.Fatal(http.ListenAndServe(":7643", server))
}
