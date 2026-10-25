//go:build linux

package main

import (
 "errors"
 "flag"
 "fmt"
 "io"
 "os"

 "github.com/GoreeCloud/goreeshield/internal/choices"
 "github.com/GoreeCloud/goreeshield/internal/synthetic"
)

// run models a single synthetic optional event, never a real integration.
func run(args []string, out io.Writer) error {
 flags:=flag.NewFlagSet("goreeshieldsim",flag.ContinueOnError)
 flags.SetOutput(io.Discard)
 purpose:=flags.String("purpose","","registered demonstration purpose")
 path:=flags.String("state","","absolute private state path")
 if err:=flags.Parse(args);err!=nil{return err}
 if flags.NArg()!=0||*purpose==""{return errors.New("usage: goreeshieldsim --purpose ID [--state ABSOLUTE_PATH]")}
 storePath:=*path
 if storePath==""{
  var err error
  storePath,err=choices.DefaultPath()
  if err!=nil{return err}
 }
 sink:=&synthetic.Sink{}
 outcome,err:=(synthetic.Adapter{Store:choices.Store{Path:storePath},Sink:sink}).Attempt(*purpose)
 if err!=nil{return err}
 if !outcome.Executed{
  fmt.Fprintf(out,"DENIED: %s (no synthetic action)\n",*purpose)
  return nil
 }
 fmt.Fprintf(out,"SYNTHETIC ACTION: %s (in-memory counter only; no data collected)\n",*purpose)
 return nil
}

func main(){
 if err:=run(os.Args[1:],os.Stdout);err!=nil{
  fmt.Fprintf(os.Stderr,"goreeshieldsim: %v\n",err)
  os.Exit(1)
 }
}
