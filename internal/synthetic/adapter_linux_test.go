//go:build linux

package synthetic

import (
 "os"
 "path/filepath"
 "testing"
 "time"

 "github.com/GoreeCloud/goreeshield/internal/choices"
)

func TestAdapterMustEnforceCurrentChoice(t *testing.T) {
 store:=choices.Store{Path:filepath.Join(t.TempDir(),"private","choices.json")}
 sink:=&Sink{}
 adapter:=Adapter{Store:store,Sink:sink}
 action:=func(id string)Outcome{
   out,err:=adapter.Attempt(id)
   if err!=nil{t.Fatal(err)}
   return out
 }
 if action("demo-local-metrics").Executed||sink.Count("demo-local-metrics")!=0{t.Fatal("optional action ran before opt-in")}
 if _,err:=adapter.Attempt("unknown-purpose");err==nil{t.Fatal("unknown purpose accepted")}
 if err:=store.Update(func(s *choices.State)error{return s.Grant("demo-local-metrics","1",true,time.Now())});err!=nil{t.Fatal(err)}
 if !action("demo-local-metrics").Executed||sink.Count("demo-local-metrics")!=1{t.Fatal("granted action was not simulated")}
 if action("demo-remote-assistance").Executed||sink.Count("demo-remote-assistance")!=0{t.Fatal("grant leaked between purposes")}
 if err:=store.Update(func(s *choices.State)error{return s.Withdraw("demo-local-metrics",time.Now())});err!=nil{t.Fatal(err)}
 if action("demo-local-metrics").Executed||sink.Count("demo-local-metrics")!=1{t.Fatal("withdrawal did not block the next action")}
 if err:=os.WriteFile(store.Path,[]byte("corrupt"),0600);err!=nil{t.Fatal(err)}
 if _,err:=adapter.Attempt("demo-local-metrics");err==nil{t.Fatal("corrupt choice state was treated as safe")}
 if sink.Count("demo-local-metrics")!=1{t.Fatal("corrupt state caused synthetic activity")}
}

func TestAdapterRequiresSink(t *testing.T) {
 adapter:=Adapter{Store:choices.Store{Path:filepath.Join(t.TempDir(),"private","choices.json")}}
 if _,err:=adapter.Attempt("demo-local-metrics");err==nil{t.Fatal("missing sink accepted")}
}
