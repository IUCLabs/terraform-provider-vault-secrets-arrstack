// Command terraform-provider-vault-secrets-arrstack is the Terraform/OpenTofu
// provider for the vault-plugin-secrets-arrstack secrets engine: it manages the
// engine's mount + connection config and its arrproxy API-key roles on
// HashiCorp Vault or OpenBao.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"git.unkin.net/unkin/terraform-provider-vault-secrets-arrstack/internal/provider"
)

var version = "0.0.1"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "enable debug mode")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "artifactapi.k8s.syd1.au.unkin.net/terraform-unkin/vault-secrets-arrstack",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err)
	}
}
