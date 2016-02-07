package model

import (
	"database/sql"

	"github.com/coopernurse/gorp"
	_ "github.com/mattn/go-sqlite3"
)

type Persona struct {
	Id         int64  `db:"id"`
	Cedula     string `db:"cedula"`
	Expiracion int64  `db:"expiracion"`
	Nombre     string `db:"nombre"`
	Apellido1  string `db:"apellido_1"`
	Apellido2  string `db:"apellido_2"`
	Genero     int    `db:"genero"`
}

type Provincia struct {
	Id     int64  `db:"id"`
	Nombre string `db:"nombre"`
}

type Canton struct {
	Id          int64  `db:"id"`
	Nombre      string `db:"nombre"`
	ProvinciaId int64  `db:"provincia_id"`
}

type Distrito struct {
	Id       int64  `db:"id"`
	Nombre   string `db:"nombre"`
	CantonId int64  `db:"canton_id"`
}

type DistritoElectoral struct {
	Id         int64  `db:"id"`
	Nombre     string `db:"nombre"`
	DistritoId int64  `db:"distrito_id"`
}

type Centro struct {
	Id                  int64  `db:"id"`
	Tipo                string `db:"tipo"`
	Nombre              string `db:"nombre"`
	Direccion           string `db:"direccion"`
	Url                 string `db:"url"`
	DistritoElectoralId int64  `db:"distrito_electoral_id"`
}

type Junta struct {
	Id       int64 `db:"id"`
	CentroId int64 `db:"centro_id"`
}

type ItemPadron struct {
	PersonaId int64 `db:"persona_id"`
	JuntaId   int64 `db:"junta_id"`
}

func InitDb() (*gorp.DbMap, error) {
	db, err := sql.Open("sqlite3", "padron.db")
	if err != nil {
		return nil, err
	}
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

	dbmap.AddTableWithName(Persona{}, "personas").SetKeys(true, "Id")
	dbmap.AddTableWithName(Junta{}, "juntas").SetKeys(false, "Id")
	dbmap.AddTableWithName(Centro{}, "centros").SetKeys(true, "Id")
	dbmap.AddTableWithName(Distrito{}, "distritos").SetKeys(false, "Id")
	dbmap.AddTableWithName(DistritoElectoral{}, "distritos_electorales").
		SetKeys(false, "Id")
	dbmap.AddTableWithName(Canton{}, "cantones").SetKeys(false, "Id")
	dbmap.AddTableWithName(Provincia{}, "provincias").SetKeys(false, "Id")
	dbmap.AddTableWithName(ItemPadron{}, "padron").
		SetKeys(false, "PersonaId", "JuntaId")

	err = dbmap.CreateTablesIfNotExists()
	if err != nil {
		dbmap.Db.Close()
		return nil, err
	}

	return dbmap, err
}
