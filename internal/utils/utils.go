package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"reflect"
	"strings"
)

func PrintStruct(s interface{}) {

	sJson, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("%s\n", string(sJson))

}

func MustParseJson[T any](path string, validator ...func(T) bool) T {

	var result T

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("file %s does not exists or is not acessible in the current path\n", path)
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Fatalf("error while parsing json %s\n", err.Error())
	}

	switch len(validator) {

	case 1: // one validator function is passed
		if !validator[0](result) {
			log.Fatalf("json object is not valid for type %v\n", reflect.TypeOf(result))
		}
		break

	case 0: // no validator function is passed
		break

	default: // more than one validator function is passed
		log.Fatalf("only one validator function is accepted, but got %d\n", len(validator))

	}

	return result
}

func CloseConnection(conn net.Conn, addr string) {
	err := conn.Close()
	Check(err)
	log.Printf("(%v) closed connection\n", addr)
}

func Check(err error) {
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func ReplacePortFromAddressString(address string, port string) string {
	split := strings.Split(address, ":")
	return split[0] + ":" + port
}
