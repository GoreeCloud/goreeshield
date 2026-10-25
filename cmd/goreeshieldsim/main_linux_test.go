//go:build linux

package main

import (
 "bytes"
 "path/filepath"
 "strings"
 "testing"
 "time"

 "github.com/GoreeCloud/goreeshield/internal/choices"
)

func TestSimulationCLI(t *testing.T) {
 stateFile:=filepath.Join(t.TempDir(),"private","choices.json")
 args:=[]string{"--purpose","demo-local-metrics","--state",stateFile}
 var output bytes.Buffer
 if err:=run(args,&output);err!=nil{t.Fatal(err)}
 if !strings.Contains(output.String(),"DENIED"){t.Fatal("default should deny",output.String())}
 store:=choices.Store{Path:stateFile}
 if err:=store.Update(func(s *choices.State)error{return s.Grant("demo-local-metrics","1",true,time.Now())});err!=nil{t.Fatal(err)}
 output.Reset()
 if err:=run(args,&output);err!=nil{t.Fatal(err)}
 if !strings.Contains(output.String(),"SYNTHETIC ACTION"){t.Fatal("grant not reflected",output.String())}
 if err:=store.Update(func(s *choices.State)error{return s.Withdraw("demo-local-metrics",time.Now())});err!=nil{t.Fatal(err)}
 output.Reset()
 if err:=run(args,&output);err!=nil{t.Fatal(err)}
 if !strings.Contains(output.String(),"DENIED"){t.Fatal("withdrawn choice did not block",output.String())}
}
