package main

// This is the start of a possible siot command line utility. This may eventually
// be folded back into the main siot exe -- placeholder for now.

/*

import (
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/simpleiot/simpleiot/api"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

func main() {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	//flagSendPoint := flags.String("sendPoint", "", "Send point to 'portal': 'devId:sensId:value:type'")
	flagDumpDb := flags.Bool("dumpDb", false, "dump database to data.json file")
	flagImportDb := flags.Bool("importDb", false, "import database from data.json")

	flagSendPointNats := flags.String("sendPointNats", "", "Send point to 'portal' via NATS: 'devId:sensId:value:type'")
	flagSendPointText := flags.String("sendPointText", "", "Send text point to 'portal' via NATS: 'devId:sensId:text:type'")


	// =============================================
	// Dump Database
	// =============================================

	if *flagDumpDb {
		log.Fatal("not supported")

		/* FIXME
		dbInst, err := store.NewSqliteDb(*flagStore, dataDir)
		if err != nil {
			log.Println("Error opening db: ", err)
			os.Exit(-1)
		}
		defer dbInst.Close()

		f, err := os.Create("data.json")
		if err != nil {
			log.Println("Error opening data.json: ", err)
			os.Exit(-1)
		}
		err = store.DumpDb(dbInst, f)

		if err != nil {
			log.Println("Error dumping database: ", err)
			os.Exit(-1)
		}

		f.Close()
		log.Println("Database written to data.json")

		os.Exit(0)
	}

	if *flagImportDb {
		log.Fatal("not supported")
		/* FIXME
		dbInst, err := store.NewDb(store.Type(*flagStore), dataDir)
		if err != nil {
			log.Println("Error opening db: ", err)
			os.Exit(-1)
		}
		defer dbInst.Close()

		f, err := os.Open("data.json")
		if err != nil {
			log.Println("Error opening data.json: ", err)
			os.Exit(-1)
		}
		err = store.ImportDb(dbInst, f)

		if err != nil {
			log.Println("Error importing database: ", err)
			os.Exit(-1)
		}

		f.Close()
		log.Println("Database imported from data.json")

		os.Exit(0)
	}

	if *flagSendPointNats != "" {
		nodeID, point, err := parsePoint(*flagSendPointNats)
		if err != nil {
			log.Println("Error parsing point: ", err)
			os.Exit(-1)
		}

		err = client.SendNodePoint(nc, nodeID, point, *flagNatsAck)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
	}

	if *flagSendPointText != "" {
		nodeID, point, err := parsePointText(*flagSendPointText)
		if err != nil {
			log.Println("Error parsing point: ", err)
			os.Exit(-1)
		}

		err = client.SendNodePoint(nc, nodeID, point, *flagNatsAck)
		if err != nil {
			log.Println(err)
			os.Exit(-1)
		}
	}

}

func parsePointText(s string) (string, data.Point, error) {
	frags := strings.Split(s, ":")
	if len(frags) != 4 {
		return "", data.Point{},
			errors.New("format for point is: 'devId:sensId:value:type'")
	}

	nodeID := frags[0]
	pointKey := frags[1]
	text := frags[2]
	pointType := frags[3]

	return nodeID, data.Point{
		Key:  pointKey,
		Type: pointType,
		Text: text,
		Time: time.Now(),
	}, nil

}

func parsePoint(s string) (string, data.Point, error) {
	frags := strings.Split(s, ":")
	if len(frags) != 4 {
		return "", data.Point{},
			errors.New("format for point is: 'devId:sensId:value:type'")
	}

	nodeID := frags[0]
	pointKey := frags[1]
	value, err := strconv.ParseFloat(frags[2], 64)
	if err != nil {
		return "", data.Point{}, err
	}

	pointType := frags[3]

	return nodeID, data.Point{
		Key:   pointKey,
		Type:  pointType,
		Value: value,
		Time:  time.Now(),
	}, nil

}

func sendPoint(portal, authToken, s string) error {
	nodeID, point, err := parsePoint(s)

	if err != nil {
		return err
	}

	sendPoints := api.NewSendPoints(portal, nodeID, authToken, time.Second*10, false)

	err = sendPoints(data.Points{point})

	return err
}

*/
