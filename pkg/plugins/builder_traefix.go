package plugins

// HasPlugin reports whether the static configuration declares a middleware plugin of that name (Traefix),
// so that plugin.coraza can stand for the native WAF when no plugin takes the name.
func (b Builder) HasPlugin(name string) bool {
	_, ok := b.middlewareBuilders[name]
	return ok
}
