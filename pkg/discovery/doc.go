// ABOUTME: mDNS service discovery package
// ABOUTME: Discover and advertise Resonate servers on local network
// Package discovery provides mDNS service discovery for Resonate servers.
//
// Allows discovering servers on the local network and advertising
// server availability.
//
// Example:
//
//	services, err := discovery.Discover(5 * time.Second)
//	for _, svc := range services {
//	    fmt.Printf("Found: %s at %s:%d\n", svc.Name, svc.Address, svc.Port)
//	}
package discovery
