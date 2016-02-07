package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"model"
	"net/http"
	"strings"
	"time"

	"github.com/coopernurse/gorp"
)

type ScrapeInfo struct {
	CentroId int64
	Cedula   string
}

const DONDE_VOTAR = "http://www.consulta.tse.go.cr/DondeVotarM/prRemoto.aspx/ObtenerDondeVotar"

func scraper(dbmap *gorp.DbMap, d ScrapeInfo) {
	log.Printf("Start processing %v\n", d)

	var result struct {
		D struct {
			Lista struct {
				CodElectoral         int
				DireccionEscuela     string
				NombreCentroVotacion string
				Url                  string
				DescripcionProvincia string
				DescripcionCanton    string
				DescripcionDistrito  string
			}
		}
	}

	query := fmt.Sprintf(`{"numeroCedula":"%s"}`, d.Cedula)
	buf := strings.NewReader(query)

	for retries := 5; retries > 0; retries-- {
		r, err := http.Post(DONDE_VOTAR, "application/json; charset=UTF-8",
			buf)
		if err != nil {
			log.Printf("W: Can't query data for %v: %s",
				d, err)
			return
		}

		resp, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("W: Can't read data for %v: %s\n", d, err)
			return
		}

		err = json.Unmarshal(resp, &result)
		if err != nil {
			log.Printf("W: Can't get data for %v: %s\n", d, err)
			return
		}

		if result.D.Lista.CodElectoral != 0 {
			break
		}

		// We are killing the server, and it won't admit
		// it.  Let's be gentle and let it take a breath
		time.Sleep(1 * time.Second)
	}

	l := &result.D.Lista

	if l.CodElectoral == 0 {
		log.Printf("W: Oops, we made the server cry...\n")
		return
	}

	var centro struct {
		Id        int64  `db:"id"`
		Tipo      string `db:"tipo"`
		Nombre    string `db:"nombre"`
		Direccion string `db:"direccion"`
		Url       string `db:"url"`
		Distrito  string `db:"distrito"`
		Canton    string `db:"canton"`
		Provincia string `db:"provincia"`
	}

	trans, err := dbmap.Begin()
	if err != nil {
		log.Printf("W: Can't start transaction: %s", err)
	}

	err = trans.SelectOne(&centro,
		`SELECT
			centros.id AS id,
			centros.tipo AS tipo,
			centros.nombre AS nombre,
			centros.direccion AS direccion,
			centros.url AS url,
			distritos_electorales.nombre AS distrito,
			cantones.nombre AS canton,
			provincias.nombre AS provincia
		FROM centros
		JOIN
			distritos_electorales ON distritos_electorales.id = centros.distrito_electoral_id,
			distritos ON distritos.id = distritos_electorales.distrito_id,
			cantones ON cantones.id = distritos.canton_id,
			provincias ON provincias.id = cantones.provincia_id
		WHERE centros.id=?`,
		d.CentroId)
	switch err {
	case nil:
		// ok
	case sql.ErrNoRows:
	default:
		log.Printf("W: Can't get data for %v: %s", centro, err)
		trans.Rollback()
		return
	}
	trans.Commit()

	if centro.Provincia != l.DescripcionProvincia {
		log.Println(centro.Id, centro.Provincia, l.DescripcionProvincia)
	}

	if centro.Canton != l.DescripcionCanton {
		log.Println(centro.Id, centro.Canton, l.DescripcionCanton)
	}

	if centro.Distrito != l.DescripcionDistrito {
		log.Println(centro.Id, centro.Distrito, l.DescripcionDistrito)
	}
}

func processCentros() {
	dbmap, err := model.InitDb()
	if err != nil {
		log.Fatalf(`E: Can't initialize database: %s. Abort.`, err)
	}
	defer dbmap.Db.Close()

	var data []ScrapeInfo

	_, err = dbmap.Select(
		&data,
		`SELECT
			centros.id AS CentroId,
			personas.cedula AS Cedula
		FROM
			padron
		JOIN
			personas ON personas.cedula = padron.persona_id,
			juntas ON juntas.id = padron.junta_id,
			centros ON centros.id = juntas.centro_id
		GROUP BY centros.id
		ORDER BY RANDOM()
		`)

	if err != nil {
		log.Fatalf(`E: Can't query padron: %s. Abort.`, err)
	}

	ch := make(chan int, 2)

	for i := 0; i < cap(ch); i++ {
		ch <- 1
	}

	pending := cap(ch)

	for _, d := range data {
		<-ch
		pending--
		go func(req ScrapeInfo) {
			scraper(dbmap, req)
			ch <- 1
		}(d)
		pending++
	}

	for ; pending > 0; pending-- {
		<-ch
	}
}

func main() {
	processCentros()
}
