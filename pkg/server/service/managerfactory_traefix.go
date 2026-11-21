package service

// AddServiceBuilder adds a builder of services that have no configuration of their own,
// like the services of the routers learned from the mesh (Traefix).
func (f *ManagerFactory) AddServiceBuilder(builder ServiceBuilder) {
	f.traefixBuilders = append(f.traefixBuilders, builder)
}
