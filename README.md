padron
======

Scraper and server for costarican electoral database

The Supreme Electoral Court (Tribunal Supremo de Elecciones, TSE)
publishes the whole database in a single file available from:

    http://www.tse.go.cr/zip/padron/padron_completo.zip

Information about this database is available from:

    http://www.tse.go.cr/descarga_padron.htm

(last accessed February 6th, 2016)

You also need the XLSX files from here:

    http://www.tse.go.cr/info_general.htm

This database DOES NOT contain the location of the voting sites.  It has
to be scraped from the website used to query one's voting site:

    http://www.consulta.tse.go.cr/DondeVotarM/prRemoto.aspx/ObtenerDondeVotar

How are you getting the voting locations?
-----------------------------------------

After repeated requests to the TSE for a machine-readable source for the
voting site information, I got nothing useful in return.  If you go to
the TSE's webpage, you'll find that you can input any citizen's id
number and you will get their voting site information.  Upon closer
inspection, you will find that this works:

    curl -i -X POST \
        -H "Content-Type: application/json" \
        -d '{"numeroCedula":"123456789"}' \
        http://www.consulta.tse.go.cr/DondeVotarM/prRemoto.aspx/ObtenerDondeVotar

that returns a JSON object with the information, including the address
and short description of the voting site.  Inspecting the electoral
database, you'll find that each person is assigned to one "junta" and
also to one "distrito".  You'll also notice that there are multiple
"juntas" per "distrito".  "juntas" are voting places within a school (or
location, in general), and there are several of them per school.

This means that you have to perform one query like the one above per
"junta".  In other words, pick one person per "junta", and use their id
to do the query.  You will get the information for a voting location
multiple times.  The scraper does exactly this, and for each query it
searches the dabase for a site matching the location provided in the
query it just made.  If it's not repeated, it considers the location to
be a new one and stores it and assigns the "junta" to it.  If it finds a
location matching the one it just got, it simply assigns that "junta" to
that location.

How to build
------------

Use "[gb](https://getgb.io/) build"

How to use
----------

After building, bin/parser is the data parser. You need to point it to
the XLSX file with the "juntas" and one or more ZIP files with the
actual database. This will create a SQLite database with all the
information.  bin/scraper is the scraping program described above.

Finally, bin/padron is the webserver that you can use to query the
database.
