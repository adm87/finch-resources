package resources

import "github.com/adm87/finch-resources/manifest"

var loadedManifest manifest.Model

func Manifest() manifest.Model {
	return loadedManifest
}

func UseManifest(m manifest.Model) {
	loadedManifest = m
}
