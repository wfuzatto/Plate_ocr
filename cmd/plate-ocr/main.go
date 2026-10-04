package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/wfuzatto/Plate_ocr/internal/app"
	"github.com/wfuzatto/Plate_ocr/internal/config"
	"github.com/wfuzatto/Plate_ocr/internal/domain"
	"github.com/wfuzatto/Plate_ocr/internal/normalize"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "normalize":
		normalizeCmd(os.Args[2:])
	case "replay":
		replayCmd(os.Args[2:])
	case "version":
		fmt.Println("plate-ocr 0.2.0")
	default:
		usage()
		os.Exit(2)
	}
}

func usage() { fmt.Fprintln(os.Stderr, "usage: plate-ocr <normalize|replay|version>") }

func normalizeCmd(args []string) {
	fs := flag.NewFlagSet("normalize",flag.ExitOnError)
	text := fs.String("text","","raw OCR text")
	conf := fs.Float64("confidence",1,"OCR confidence 0..1")
	_ = fs.Parse(args)
	_ = json.NewEncoder(os.Stdout).Encode(normalize.Candidates(*text,*conf,8))
}

func replayCmd(args []string) {
	fs := flag.NewFlagSet("replay",flag.ExitOnError)
	cfgPath := fs.String("config","","config JSON path")
	_ = fs.Parse(args)
	cfg,err := config.Load(*cfgPath)
	if err != nil { fatal(err) }
	eng := app.New(cfg)
	scan := bufio.NewScanner(os.Stdin)
	scan.Buffer(make([]byte,64*1024),4*1024*1024)
	enc := json.NewEncoder(os.Stdout)
	for scan.Scan() {
		var o domain.Observation
		if err:=json.Unmarshal(scan.Bytes(),&o); err!=nil { fatal(err) }
		events,err:=eng.Process(o)
		if err!=nil { fatal(err) }
		for _,ev:=range events { _=enc.Encode(ev) }
	}
	if err:=scan.Err(); err!=nil { fatal(err) }
	for _,ev:=range eng.FlushAll() { _=enc.Encode(ev) }
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr,"plate-ocr:",err)
	os.Exit(1)
}
