package routes

import (
	"goravel/app/facades"
	"goravel/app/http/controllers"
)

func Web() {
	paste := controllers.NewPasteController()

	facades.Route().Get("/", paste.New)
	facades.Route().Post("/documents", paste.Create)
	facades.Route().Put("/documents/{id}", paste.Update)
	facades.Route().Delete("/documents/{id}", paste.Destroy)
	facades.Route().Get("/raw/{id}", paste.Raw)
	facades.Route().Get("/download/{id}", paste.Download)
	facades.Route().Get("/{id}", paste.Show)

	facades.Route().Static("public", "./public")
}
