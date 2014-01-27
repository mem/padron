package main

import (
	"code.google.com/p/go-charset/charset"
	_ "code.google.com/p/go-charset/data"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/coopernurse/gorp"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"padron/model"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func toInt(s string) int {
	i, _ := strconv.ParseInt(s, 10, 0)
	return int(i)
}

func toInt64(s string) (i int64) {
	i, _ = strconv.ParseInt(s, 10, 64)
	return
}

func processFile(filename string,
	processData func([]string, *gorp.Transaction)) {

	r, err := os.Open(filename)
	if err != nil {
		log.Fatalf(`E: Can't open file "%s": %s. Abort.`,
			filename, err)
	}
	defer r.Close()

	r2, err := charset.NewReader("latin1", r)
	if err != nil {
		log.Fatalf(`E: Can't create charset reader: %s. Abort.`,
			err)
	}

	dbmap, err := model.InitDb()
	if err != nil {
		log.Fatalf(`E: Can't initialize database: %s. Abort.`, err)
	}
	defer dbmap.Db.Close()

	r3 := csv.NewReader(r2)

	// dbmap.TraceOn("", log.New(os.Stdout, "gorptest: ", log.Lmicroseconds))

	dbmap.Exec("PRAGMA synchronous=OFF")
	dbmap.Exec("PRAGMA journal_model=OFF")
	// dbmap.Exec("BEGIN TRANSACTION")

	trans, err := dbmap.Begin()
	if err != nil {
		log.Fatalf(`E: Can't initialize transaction: %s. Abort.`, err)
	}

	for i := 0; true; i++ {
		if i%1000 == 0 {
			log.Println(i)
		}

		data, err := r3.Read()
		if err != nil {
			break
		}

		processData(data, trans)
	}

	trans.Commit()
}

func getOrInsert(
	trans *gorp.Transaction,
	new_record interface{}) (interface{}, error) {

	id := reflect.Indirect(reflect.ValueOf(new_record)).FieldByName("Id")

	obj, err := trans.Get(new_record, id.Int())

	if err != nil {
		log.Printf("W: Can't get object: %s", err)
		return nil, err
	}

	if obj != nil {
		return obj, nil
	}

	if err = trans.Insert(new_record); err != nil {
		log.Printf("W: Can't insert %v: %s", new_record, err)
		return nil, err
	}

	return new_record, nil
}

func processDistritos(filename string) {
	provincias := make(map[int64]*model.Provincia)
	cantones := make(map[int64]*model.Canton)
	distritos := make(map[int64]*model.Distrito)

	processFile(filename, func(data []string, trans *gorp.Transaction) {
		provincia := &model.Provincia{
			Id: toInt64(data[0][0:1]),
		}

		if provincias[provincia.Id] == nil {
			provincia.Nombre = strings.TrimSpace(data[1])
			obj, err := getOrInsert(trans, provincia)
			if err != nil {
				log.Println("Can't get or insert provincia %d: %s",
					provincia.Id, err)
				return
			}
			provincia = obj.(*model.Provincia)
			provincias[provincia.Id] = provincia
		}

		canton := &model.Canton{
			Id:          toInt64(data[0][1:3]) + provincia.Id*100,
			ProvinciaId: provincia.Id,
		}

		if cantones[canton.Id] == nil {
			canton.Nombre = strings.TrimSpace(data[2])
			obj, err := getOrInsert(trans, canton)
			if err != nil {
				log.Println("Can't get or insert canton %d: %s",
					canton.Id, err)
				return
			}

			canton = obj.(*model.Canton)
			cantones[canton.Id] = canton
		}

		distrito := &model.Distrito{
			Id:       toInt64(data[0][3:6]) + canton.Id*1000,
			CantonId: canton.Id,
		}

		if distritos[distrito.Id] == nil {
			distrito.Nombre = strings.TrimSpace(data[3])
			obj, err := getOrInsert(trans, distrito)
			if err != nil {
				log.Println("Can't get or insert distrito %d: %s",
					distrito.Id, err)
				return
			}

			distrito = obj.(*model.Distrito)
			distritos[distrito.Id] = distrito
		}
	})
}

func processPadron(filename string) {
	loc, _ := time.LoadLocation("America/Costa_Rica")

	juntas := make(map[int64]*model.Junta)

	processFile(filename, func(data []string, trans *gorp.Transaction) {
		// Data format is:
		//	0: cedula
		//	1: codigo
		//	2: sexo
		//	3: fecha expiracion
		//	4: # junta
		//	5: nombre
		//	6: apellido
		//	7: apellido

		distrito_id := toInt64(data[1])

		/*
			centro := model.Centro{
				DistritoId: distrito_id,
			}
			err := trans.Insert(&centro)
			if err != nil {
				log.Printf("W: Can't insert %v: %s",
					centro, err)
				return
			}
		*/

		junta := &model.Junta{
			Id: toInt64(data[4]),
		}

		if juntas[junta.Id] == nil {
			obj, err := getOrInsert(trans, junta)
			if err != nil {
				log.Println("Can't get or insert junta %d: %s",
					junta.Id, err)
				return
			}

			junta = obj.(*model.Junta)
			juntas[junta.Id] = junta
		}

		exp, err := time.ParseInLocation("20060102", data[3], loc)
		if err != nil {
			return
		}

		persona := model.Persona{
			Cedula:     data[0],
			Expiracion: exp.Unix(),
			Nombre:     strings.TrimSpace(data[5]),
			Apellido1:  strings.TrimSpace(data[6]),
			Apellido2:  strings.TrimSpace(data[7]),
			Genero:     toInt(data[2]),
			DistritoId: distrito_id,
		}

		err = trans.Insert(&persona)
		if err != nil {
			log.Printf("W: Can't insert %v: %s", persona, err)
			return
		}

		item_padron := model.ItemPadron{
			PersonaId: persona.Id,
			JuntaId:   junta.Id,
		}

		err = trans.Insert(&item_padron)
		if err != nil {
			log.Printf("W: Can't insert %v: %s",
				item_padron, err)
			return
		}
	})
}

const DONDE_VOTAR = "http://www.tse.go.cr/aplicacionvisualizador/prRemoto.aspx/ObtenerDondeVotar"

func processCentros() {
	dbmap, err := model.InitDb()
	if err != nil {
		log.Fatalf(`E: Can't initialize database: %s. Abort.`, err)
	}
	defer dbmap.Db.Close()

	type ScrapeInfo struct {
		JuntaId int64
		Cedula  string
	}

	var data []ScrapeInfo

	_, err = dbmap.Select(
		&data,
		`SELECT
			juntas.id AS JuntaId,
			personas.cedula AS Cedula
		FROM
			padron
		JOIN
			personas ON personas.id = padron.persona_id,
			juntas ON juntas.id = padron.junta_id
		WHERE juntas.centro_id = 0
		GROUP BY juntas.id`)

	if err != nil {
		log.Fatalf(`E: Can't query padron: %s. Abort.`, err)
	}

	sem := make(chan int, 1)
	sem <- 1

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
			r, err := http.Post(DONDE_VOTAR, "application/json",
				buf)
			if err != nil {
				log.Printf("W: Can't query data for %v: %s",
					d, err)
				return
			}

			log.Printf("I: d=%v r=%v", d, r)

			resp, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Printf("W: Can't read data for %v: %s",
					d, err)
				return
			}

			err = json.Unmarshal(resp, &result)
			if err != nil {
				log.Printf("W: Can't get data for %v: %s",
					d, err)
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

		centro := model.Centro{
			Nombre:     l.NombreCentroVotacion,
			Direccion:  l.DireccionEscuela,
			Url:        l.Url,
			DistritoId: int64(l.CodElectoral),
		}

		if centro.DistritoId == 0 {
			log.Printf("W: Oops, we made the server cry...\n")
			return
		}

		<-sem
		defer func(){ sem <- 1 }()

		trans, err := dbmap.Begin()

		err = trans.SelectOne(&centro,
			`SELECT *
			FROM centros
			WHERE nombre=? AND direccion=? AND distrito_id=?`,
			centro.Nombre,
			centro.Direccion,
			centro.DistritoId)

		switch err {
		case nil:
			// ok
		case sql.ErrNoRows:
			err = trans.Insert(&centro)
		default:
			log.Printf("W: Can't get data for %v: %s",
				centro, err)
			trans.Rollback()
			return
		}

		junta := model.Junta{
			Id:       d.JuntaId,
			CentroId: centro.Id,
		}

		trans.Update(&junta)

		trans.Commit()

		log.Printf("End processing %v %v %v\n", d, centro, junta)
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
	if len(os.Args) < 3 {
		log.Fatalf(`E: Missing filename.  Abort.`)
	}
	padron, distritos := os.Args[1], os.Args[2]

	_ = padron
	_ = distritos

	processDistritos(distritos)
	processPadron(padron)
	processCentros()
}
