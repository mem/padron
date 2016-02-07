package main

import (
	"errors"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/tealeg/xlsx"
)

type Padron struct {
	Provincias map[string]string
	Cantones   map[string]string
	Distritos  map[string]string
	DEs        map[string]string
	Centros    map[int]Centro
	Juntas     map[int]Junta
}

type Centro struct {
	Id           int
	JuntaStartId int
	JuntaEndId   int
	Tipo         string
	Nombre       string
	DE           string
}

type Junta struct {
	Id        int
	Electores int
	DEId      string
}

var (
	padron_re = regexp.MustCompile(`^[A-Z]{2}\.txt$`)
)

type Row xlsx.Row

func (r *Row) String() string {
	s := make([]string, 0, len(r.Cells))
	for _, c := range r.Cells {
		s = append(s, c.Value)
	}
	return strings.Join(s, ",")
}

func NewPadron() *Padron {
	return &Padron{
		Provincias: make(map[string]string),
		Cantones:   make(map[string]string),
		Distritos:  make(map[string]string),
		DEs:        make(map[string]string),
		Centros:    make(map[int]Centro),
		Juntas:     make(map[int]Junta),
	}
}

func (p *Padron) ProcessCentros(fn string) {
	x, err := xlsx.OpenFile(fn)
	if err != nil {
		log.Fatal(err)
	}

	id := 1

	for _, sheet := range x.Sheets {
		for _, r := range sheet.Rows {
			const (
				COL_CODIGO             = iota // 0: Codigo
				COL_PROVINCIA                 // 1: Provincia
				COL_CANTON                    // 2: Canton
				COL_DISTRITO_ELECTORAL        // 3: Distrito electoral
				COL_JRV_INICIAL               // 4: inicial
				COL_JRV_FINAL                 // 5: final
				COL_JRV_TOTAL                 // 6: total = 5-4+1
				COL_TIPO                      // 7: tipo
				COL_NOMBRE                    // 8: nombre
			)

			switch {
			case len(r.Cells) < COL_NOMBRE+1:
				continue
			case r.Cells[COL_JRV_INICIAL].Type() != xlsx.CellTypeNumeric:
				log.Println("Unexpected format in cell JRV_INICIAL:",
					r.Cells[COL_JRV_INICIAL].Type())
				continue
			case r.Cells[COL_JRV_FINAL].Type() != xlsx.CellTypeNumeric:
				log.Println("Unexpected format in cell JRV_FINAL:",
					r.Cells[COL_JRV_FINAL].Type())
				continue
			}

			j0, err := strconv.Atoi(r.Cells[COL_JRV_INICIAL].Value)
			if err != nil {
				log.Println("Failed to parse:", r.Cells[COL_JRV_INICIAL].Value, err)
				continue
			}

			j1, err := strconv.Atoi(r.Cells[COL_JRV_FINAL].Value)
			if err != nil {
				log.Println("Failed to parse:", r.Cells[COL_JRV_FINAL].Value, err)
				continue
			}

			t, err := strconv.Atoi(r.Cells[COL_JRV_TOTAL].Value)
			if err != nil {
				log.Println("Failed to parse:", r.Cells[COL_JRV_TOTAL].Value, err)
				continue
			}

			if t != j1-j0+1 {
				log.Println("Invalid data:", j0, j1, t)
				continue
			}

			de := r.Cells[COL_DISTRITO_ELECTORAL].Value

			if COL_NOMBRE == 8 {
				id := r.Cells[COL_CODIGO].Value

				if len(id) != 6 {
					log.Println("Unexpected id lenght:", id, len(id))
					continue
				}

				// provincia := r.Cells[COL_PROVINCIA].Value
				// canton := r.Cells[COL_CANTON].Value

				// provincia_id := id[0:1]
				// canton_id := id[0:3]

				// p.Provincias[provincia_id] = provincia
				// p.Cantones[canton_id] = canton
			}

			tipo := strings.TrimSpace(r.Cells[COL_TIPO].Value)
			nombre := strings.TrimSpace(strings.TrimPrefix(r.Cells[COL_NOMBRE].Value, tipo))

			p.Centros[id] = Centro{
				Id:           id,
				JuntaStartId: j0,
				JuntaEndId:   j1,
				Tipo:         tipo,
				Nombre:       nombre,
				DE:           de,
			}
			id++
		}
	}
}

func (p *Padron) ProcessJuntas(fn string) {
	x, err := xlsx.OpenFile(fn)
	if err != nil {
		log.Fatal(err)
	}

	split := func(s string) (id, name string, err error) {
		f := strings.SplitN(strings.TrimSpace(s), " ", 2)
		if len(f) != 2 {
			err = errors.New("Unexpected format")
			return "", "", err
		}
		return f[0], f[1], nil
	}

	check := func(m map[string]string, id, name string) bool {
		cur, ok := m[id]
		v := !ok || cur == name
		if !v {
			log.Println("Conflicting values:", id, cur, id, name)
		}
		return v
	}

	for _, sheet := range x.Sheets {
		for _, r := range sheet.Rows {
			// 0: provincia
			// 1: canton
			// 2: distrito
			// 3: distrito electoral
			// 4: id junta
			// 5: cantidad de electores
			switch {
			case len(r.Cells) < 6:
				continue
			case r.Cells[4].Type() != xlsx.CellTypeNumeric:
				continue
			case r.Cells[5].Type() != xlsx.CellTypeNumeric:
				continue
			}

			// Provincia
			tmp, provincia, err := split(r.Cells[0].Value)
			if err != nil {
				log.Println(err, r.Cells[0].Value)
				continue
			}
			provincia_id := tmp

			// Canton
			tmp, canton, err := split(r.Cells[1].Value)
			if err != nil {
				log.Println(err, r.Cells[1].Value)
				continue
			}
			canton_id := provincia_id + tmp

			// Distrito
			tmp, distrito, err := split(r.Cells[2].Value)
			if err != nil {
				log.Println(err, r.Cells[1].Value)
				continue
			}
			distrito_id := canton_id + tmp

			// Distrito electoral
			tmp, de, err := split(r.Cells[3].Value)
			if err != nil {
				log.Println(err, r.Cells[3].Value)
				continue
			}
			de_id := tmp

			// This checks that the ID format is correct:
			// PCCDDDNNN
			if de_id[0:6] != distrito_id {
				log.Println("Invalid data:", r.Cells[3].Value)
				continue
			}

			// ID junta
			id, err := strconv.Atoi(r.Cells[4].Value)
			if err != nil {
				log.Println("Failed to parse:", r.Cells[4].Value, err)
				continue
			}

			// Electores
			t, err := strconv.Atoi(r.Cells[5].Value)
			if err != nil {
				log.Println("Failed to parse:", r.Cells[5].Value, err)
				continue
			}

			// All the data seems good

			if false {
				// This is here to cross-validate with
				// the information from the other file
				bad := false

				if !check(p.Provincias, provincia_id, provincia) {
					bad = true
				}

				if !check(p.Cantones, canton_id, canton) {
					bad = true
				}

				if !check(p.Distritos, distrito_id, distrito) {
					bad = true
				}

				if !check(p.DEs, de_id, de) {
					bad = true
				}

				if bad {
					log.Println("Bad data")
					continue
				}
			}

			// Data is good
			p.Provincias[provincia_id] = provincia
			p.Cantones[canton_id] = canton
			p.Distritos[distrito_id] = distrito
			p.DEs[de_id] = de
			p.Juntas[id] = Junta{
				Id:        id,
				Electores: t,
				DEId:      de_id,
			}
		}
	}
}

func processInput(centros, juntas string) *Padron {
	padron := NewPadron()

	padron.ProcessCentros(centros)
	padron.ProcessJuntas(juntas)

	// Up to this point we have processed all the information we
	// have, now it's time to reconstruct data.

	// We tables with:
	//
	// provincia_id, name
	// canton_id, name
	// distrito_id, name
	// de_id, name
	//
	// Centros point to a set of juntas
	// Juntas point to DEs

	for _, centro := range padron.Centros {
		for jid := centro.JuntaStartId; jid <= centro.JuntaEndId; jid++ {
			deid := padron.Juntas[jid].DEId
			junta_de_name := padron.DEs[deid]
			centro_de_name := centro.DE
			if junta_de_name != centro_de_name {
				log.Printf("Data mismatch: %s != %s\n", junta_de_name, centro_de_name)
			}

			// This is how you obtain all the data starting
			// from DE id
			//
			// provincia := padron.Provincias[deid[0:1]]
			// canton := padron.Cantones[deid[0:3]]
			// distrito := padron.Distritos[deid[0:6]]
			// de := padron.DEs[deid[0:9]]
		}
	}

	return padron
}
