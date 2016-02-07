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
)

const DONDE_VOTAR = "http://www.consulta.tse.go.cr/DondeVotarM/prRemoto.aspx/ObtenerDondeVotar"

func processCentros() {
	dbmap, err := model.InitDb()
	if err != nil {
		log.Fatalf(`E: Can't initialize database: %s. Abort.`, err)
	}
	defer dbmap.Db.Close()

	type ScrapeInfo struct {
		CentroId int64
		Cedula   string
	}

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
		WHERE centros.url = ''
		GROUP BY centros.id`)

	if err != nil {
		log.Fatalf(`E: Can't query padron: %s. Abort.`, err)
	}

	scraper := func(d ScrapeInfo) {
		log.Printf("Start processing %v\n", d)

		var result struct {
			D struct {
				Lista struct {
					CodElectoral         int
					DireccionEscuela     string
					NombreCentroVotacion string
					Url                  string
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

			log.Printf("I: d=%v r=%v\n", d, r)

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

		var centro model.Centro

		if l.CodElectoral == 0 {
			log.Printf("W: Oops, we made the server cry...\n")
			return
		}

		trans, err := dbmap.Begin()

		err = trans.SelectOne(&centro, `SELECT * FROM centros WHERE id=?`, d.CentroId)

		switch err {
		case nil:
			// ok
		case sql.ErrNoRows:
			err = trans.Insert(&centro)
		default:
			log.Printf("W: Can't get data for %v: %s", centro, err)
			trans.Rollback()
			return
		}

		centro.Direccion = l.DireccionEscuela
		centro.Url = l.Url

		log.Printf("Centro: %#v\n", centro)

		trans.Update(&centro)

		trans.Commit()
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
			scraper(req)
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
