//go:build embedding

package plugin

import (
	"murphysec-cli-simple/plugin/gradle"
	"murphysec-cli-simple/plugin/mvn2"
	"murphysec-cli-simple/plugin/plugin_base"
)

var Plugins = []plugin_base.Plugin{
	gradle.Instance,
	mvn2.Instance,
}