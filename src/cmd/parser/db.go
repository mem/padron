package main

import (
	"log"
	"reflect"
	"strconv"

	"github.com/coopernurse/gorp"
)

func toInt(s string) int {
	i, _ := strconv.ParseInt(s, 10, 0)
	return int(i)
}

func toInt64(s string) (i int64) {
	i, _ = strconv.ParseInt(s, 10, 64)
	return
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
