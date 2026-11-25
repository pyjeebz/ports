package cmd

import (
	"github.com/pyjeebz/ports/internal/inspect"
	"github.com/pyjeebz/ports/internal/naming"
	"github.com/pyjeebz/ports/internal/router"
	"github.com/pyjeebz/ports/internal/routes"
)

// scanNamed is inspect.Scan with names: each web dev server gets the
// name the router reaches it by — the same one the router gives it, from
// the same rules and the same remembered claims — and, while a router
// runs, its URL through the router.
func scanNamed(cfg *inspect.Config) ([]inspect.Service, error) {
	services, err := inspect.Scan(cfg)
	if err != nil {
		return nil, err
	}
	nameServices(services)
	return services, nil
}

func nameServices(services []inspect.Service) {
	claims := naming.LoadClaims()
	aliases, runs := routes.LoadDeclared()
	routes.Resolve(services, claims, aliases, runs)
	_ = claims.Save() // a lost save only means the claim is made again
	info, ok := router.Running()
	if !ok {
		return
	}
	for i := range services {
		if routes.Routable(services[i]) {
			services[i].NameURL = router.URL(services[i].Name, info.Port)
		}
	}
}
