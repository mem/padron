export GOPATH=$(CURDIR)
GOARCH := $(shell go env GOARCH)
GOOS := $(shell go env GOOS)

all : padron parser
	$(S) echo Done.

padron_SRC := $(addprefix src/padron/,local.go model/model.go server/server.go)
parser_SRC := $(addprefix src/padron/,model/model.go parser/main.go)

PKG := \
	code.google.com/p/go-charset/charset \
	code.google.com/p/go-charset/data \
	github.com/coopernurse/gorp \
	github.com/gorilla/context \
	github.com/gorilla/mux \
	github.com/mattn/go-sqlite3 \

LIBS := $(addprefix pkg/$(GOOS)_$(GOARCH)/,$(addsuffix .a,$(PKG)))

padron : $(padron_SRC) $(LIBS)
	$(T) GO '$@'
	$(Q) go build padron

parser : $(parser_SRC) $(LIBS)
	$(T) GO '$@'
	$(Q) go build padron/parser

$(LIBS) :
	$(T) LIB '$@'
	$(Q) go install $(patsubst pkg/$(GOOS)_$(GOARCH)/%.a,%,$@)

pkgs :

clean :
	$(Q) $(RM) padron parser

datos.tar.xz :
	mkdir -p datos
	test -e padron_completo.zip || \
	    wget -N http://www.tse.go.cr/zip/padron/padron_completo.zip
	cd datos && unzip ../padron_completo.zip
	tar cvJf $@ datos/

datos/PADRON_COMPLETO.txt datos/Distelec.txt : datos.tar.xz
	$(T) TXT '$@ <= $^'
	$(Q) tar xf $^ $@
	$(Q) touch $@

padron.db : parser schema.sql datos/PADRON_COMPLETO.txt datos/Distelec.txt
	$(T) DB '$@ <= $^'
	rm -f padron.db
	sqlite3 $@ < schema.sql
	./parser datos/PADRON_COMPLETO.txt datos/Distelec.txt

S := @
Q := @
T = $(S) printf '%8s    %s\n'
