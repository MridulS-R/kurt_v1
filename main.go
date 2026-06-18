package main

import "kurt_v1/cmd"

// Version is the build version, set at build time via:
//   -ldflags "-X main.Version=<tag>"
// Defaults to "dev" for unversioned local builds.
var Version = "dev"

func main() {
	cmd.SetVersion(Version)
	cmd.Execute()
}
