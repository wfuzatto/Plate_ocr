package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wfuzatto/Plate_ocr/internal/app"
	"github.com/wfuzatto/Plate_ocr/internal/config"
	"github.com/wfuzatto/Plate_ocr/internal/domain"
	"github.com/wfuzatto/Plate_ocr/internal/inference/classic"
	"github.com/wfuzatto/Plate_ocr/internal/normalize"
	"github.com/wfuzatto/Plate_ocr/internal/pipeline"
	runtimepkg "github.com/wfuzatto/Plate_ocr/internal/runtime"
	"github.com/wfuzatto/Plate_ocr/internal/runtime/nvrclient"
	"github.com/wfuzatto/Plate_ocr/internal/spool"
)

const version="1.0.0"

func main() {
	if len(os.Args) < 2 { usage(); os.Exit(2) }
	switch os.Args[1] {
	case "serve","run": serveCmd(os.Args[2:])
	case "normalize": normalizeCmd(os.Args[2:])
	case "replay": replayCmd(os.Args[2:])
	case "version": fmt.Println("plate-ocr "+version)
	default: usage(); os.Exit(2)
	}
}

func usage() { fmt.Fprintln(os.Stderr, "usage: plate-ocr <serve|normalize|replay|version>") }

func serveCmd(args []string){
	fs:=flag.NewFlagSet("serve",flag.ExitOnError)
	cfgPath:=fs.String("config","configs/default.json","config JSON path; empty uses defaults")
	_ = fs.Parse(args)
	cfg,err:=config.Load(*cfgPath);if err!=nil{fatal(err)}
	token,err:=nvrclient.LoadToken(cfg.PluginTokenFile);if err!=nil{fatal(fmt.Errorf("plugin token: %w",err))}
	client,err:=nvrclient.New(cfg.NVRBaseURL,token,cfg.HTTPTimeout);if err!=nil{fatal(err)}
	sp,err:=spool.Open(cfg.SpoolDir);if err!=nil{fatal(err)}
	det:=classic.NewDetector(classic.DetectorConfig{
		Threshold:cfg.DetectorThreshold,MaxResults:cfg.MaxDetections,
		MinWidthPX:cfg.MinPlateWidthPX,MinHeightPX:cfg.MinPlateHeightPX,
	})
	rec:=classic.NewRecognizer(classic.RecognizerConfig{Threshold:cfg.RecognizerThreshold})
	eng:=app.New(cfg)
	pipe,err:=pipeline.New(det,rec,eng);if err!=nil{fatal(err)}
	svc,err:=runtimepkg.NewService(cfg,client,pipe,sp);if err!=nil{fatal(err)}

	ctx,cancel:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM)
	defer cancel()
	statusErr:=make(chan error,1)
	go func(){statusErr<-svc.ServeStatus(ctx)}()
	runErr:=make(chan error,1)
	go func(){runErr<-svc.Run(ctx)}()
	select{
	case err:=<-statusErr:
		if err!=nil{cancel();fatal(err)}
	case err:=<-runErr:
		if err!=nil{cancel();fatal(err)}
	case<-ctx.Done():
	}
}

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
	scan.Buffer(make([]byte,64*1024),16*1024*1024)
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
