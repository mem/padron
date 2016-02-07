package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"log"
	"model"
	"os"
	"strings"

	"github.com/coopernurse/gorp"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

func process_zip(fn string, trans *gorp.Transaction) {
	r, err := zip.OpenReader(fn)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	for _, f := range r.File {
		if !padron_re.MatchString(f.Name) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			log.Fatal(err)
		}

		s := bufio.NewScanner(transform.NewReader(rc, charmap.ISO8859_15.NewDecoder()))

		s.Split(bufio.ScanLines)

		for s.Scan() {
			l := s.Text()
			// 0: id
			// 1: distrito electoral
			// 2: sexo
			// 3: vencimiento
			// 4: junta
			// 5: nombre
			// 6: apellido 1
			// 7: apellido 2
			fields := strings.Split(l, ",")
			for i := range fields {
				fields[i] = strings.TrimSpace(fields[i])
			}

			p := model.Persona{
				Id:         toInt64(fields[0]),
				Cedula:     fields[0],
				Expiracion: toInt64(fields[3]),
				Nombre:     fields[5],
				Apellido1:  fields[6],
				Apellido2:  fields[7],
				Genero:     toInt(fields[2]),
			}

			i := model.ItemPadron{
				PersonaId: p.Id,
				JuntaId:   toInt64(fields[4]),
			}

			_, err := getOrInsert(trans, &p)
			if err != nil {
				log.Println("Can't get or persona %d: %s", p.Id, err)
			}

			obj, err := trans.Get(&i, i.PersonaId, i.JuntaId)
			if err != nil {
				log.Println("Failed to get item padron %d: %s", i.PersonaId, err)
				continue
			}

			if obj != nil {
				continue
			}

			if err = trans.Insert(&i); err != nil {
				log.Println("Can't get or item padron %d: %s", i.PersonaId, err)
			}
		}

		rc.Close()
		fmt.Println()
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Need at least two arguments: centros.xlsx juntas.xlsx")
	}

	padron := processInput(os.Args[1], os.Args[2])

	dbmap, err := model.InitDb()
	if err != nil {
		log.Fatalf(`E: Can't initialize database: %s. Abort.`, err)
	}
	defer dbmap.Db.Close()

	dbmap.Exec("PRAGMA synchronous=OFF")
	dbmap.Exec("PRAGMA journal_model=OFF")

	trans, err := dbmap.Begin()
	if err != nil {
		log.Fatalf(`E: Can't initialize transaction: %s. Abort.`, err)
	}

	for id, nombre := range padron.Provincias {
		provincia := model.Provincia{
			Id:     toInt64(id),
			Nombre: nombre,
		}
		_, err := getOrInsert(trans, &provincia)
		if err != nil {
			log.Println("Can't get or insert provincia %d: %s",
				provincia.Id, err)
		}
	}

	for id, nombre := range padron.Cantones {
		canton := model.Canton{
			Id:          toInt64(id),
			Nombre:      nombre,
			ProvinciaId: toInt64(id[0:1]),
		}
		_, err := getOrInsert(trans, &canton)
		if err != nil {
			log.Println("Can't get or insert canton %d: %s",
				canton.Id, err)
		}
	}

	for id, nombre := range padron.Distritos {
		distrito := model.Distrito{
			Id:       toInt64(id),
			Nombre:   nombre,
			CantonId: toInt64(id[0:3]),
		}
		_, err := getOrInsert(trans, &distrito)
		if err != nil {
			log.Println("Can't get or insert distrito %d: %s",
				distrito.Id, err)
		}
	}

	for id, nombre := range padron.DEs {
		distrito := model.DistritoElectoral{
			Id:         toInt64(id),
			Nombre:     nombre,
			DistritoId: toInt64(id[0:6]),
		}
		_, err := getOrInsert(trans, &distrito)
		if err != nil {
			log.Println("Can't get or insert distrito electoral %d: %s",
				distrito.Id, err)
		}
	}

	for id, centro := range padron.Centros {
		deId := padron.Juntas[centro.JuntaStartId].DEId
		c := model.Centro{
			Id:                  int64(id),
			Tipo:                centro.Tipo,
			Nombre:              centro.Nombre,
			DistritoElectoralId: toInt64(deId),
		}
		_, err := getOrInsert(trans, &c)
		if err != nil {
			log.Println("Can't get or insert centro %d: %s", c.Id, err)
		}

		for jid := centro.JuntaStartId; jid <= centro.JuntaEndId; jid++ {
			j := model.Junta{
				Id:       int64(jid),
				CentroId: c.Id,
			}
			_, err := getOrInsert(trans, &j)
			if err != nil {
				log.Println("Can't get or insert centro %d: %s", c.Id, err)
			}

		}
	}

	for _, z := range os.Args[3:] {
		process_zip(z, trans)
	}

	trans.Commit()

}
