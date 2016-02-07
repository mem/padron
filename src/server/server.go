package server

import (
	"encoding/json"
	"fmt"
	"log"
	"model"
	"net/http"

	"github.com/gorilla/mux"
)

func RegisterHandlers() {
	r := mux.NewRouter()
	r.HandleFunc("/persona/{id}", errorHandler(GetPersona)).Methods("GET")
	http.Handle("/persona/", r)
}

type badRequest struct{ error }

type notFound struct{ error }

func errorHandler(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err == nil {
			return
		}
		switch err.(type) {
		case badRequest:
			http.Error(w, err.Error(), http.StatusBadRequest)
		case notFound:
			http.Error(w, "persona no encontrada: "+err.Error(),
				http.StatusNotFound)
		default:
			log.Println(err)
			http.Error(w, "oops", http.StatusInternalServerError)
		}
	}
}

func parseID(r *http.Request) (string, error) {
	txt, ok := mux.Vars(r)["id"]
	if !ok {
		return "", fmt.Errorf("id no está presente")
	}
	return txt, nil
}

func GetPersona(w http.ResponseWriter, r *http.Request) error {
	id, err := parseID(r)
	log.Println("Id para persona ", id)
	if err != nil {
		return badRequest{err}
	}

	dbmap, err := model.InitDb()
	if err != nil {
		return fmt.Errorf(`E: Can't initialize database: %s. Abort.`, err)
	}
	defer dbmap.Db.Close()

	var persona struct {
		Cedula    string
		Nombre    string
		Apellido1 string
		Apellido2 string
		Centro    string
		Direccion string
		Url       string
		Provincia string
		Canton    string
		Distrito  string
		Mesa      string
	}

	err = dbmap.SelectOne(&persona,
		`SELECT
			personas.cedula AS Cedula,
			personas.nombre AS Nombre,
			personas.apellido_1 AS Apellido1,
			personas.apellido_2 AS Apellido2,
			juntas.id AS mesa,
			centros.nombre AS centro,
			centros.direccion AS direccion,
			centros.url AS url,
			distritos.nombre AS distrito,
			cantones.nombre AS canton,
			provincias.nombre AS provincia
		FROM
			(SELECT * FROM personas WHERE cedula=?) AS personas
		JOIN
			padron ON padron.persona_id = personas.cedula,
			juntas ON juntas.id = padron.junta_id,
			centros ON centros.id = juntas.centro_id,
			distritos_electorales ON distritos_electorales.id = centros.distrito_electoral_id,
			distritos ON distritos.id = distritos_electorales.distrito_id,
			cantones ON cantones.id = distritos.canton_id,
			provincias ON provincias.id = cantones.provincia_id`,
		id)

	if err != nil {
		return notFound{err}
	}

	return json.NewEncoder(w).Encode(persona)
}
