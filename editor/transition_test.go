package editor

import (
 "encoding/json"
 "math"
 "testing"
)
func transitionTestClip(id string) Clip { return Clip{ID:id,Path:"/tmp/"+id+".mp4",Name:id,SourceEnd:2,SourceDuration:2,HasAudio:true} }
func closeTime(t *testing.T,got,want float64){t.Helper();if math.Abs(got-want)>0.0001{t.Fatalf("got %.4f want %.4f",got,want)}}
func TestTransitionsTimelineSaveAndEdits(t *testing.T) {
 p:=NewProject()
 for _,id:=range []string{"a","b","c"} {if err:=p.Insert(len(p.Clips),transitionTestClip(id));err!=nil {t.Fatal(err)}}
 if err:=p.SetTransition(0,"dissolve",0.8);err!=nil {t.Fatal(err)}
 if err:=p.SetTransition(1,"black",0.6);err!=nil {t.Fatal(err)}
 closeTime(t,p.Duration(),4.6);closeTime(t,p.Start(1),1.2);closeTime(t,p.Start(2),2.6)
 closeTime(t,p.Cut(0),1.6);closeTime(t,p.Cut(1),2.9)
 if err:=p.SetTransition(1,"black",1.3);err==nil {t.Fatal("overlong overlapping transitions accepted")}
 if err:=p.Validate();err!=nil {t.Fatal(err)}
 encoded,err:=json.Marshal(p);if err!=nil {t.Fatal(err)}
 var loaded Project;if err:=json.Unmarshal(encoded,&loaded);err!=nil {t.Fatal(err)}
 closeTime(t,loaded.Duration(),4.6)
 if _,err:=p.SplitAt(0.9,"d");err!=nil {t.Fatal(err)}
 if err:=p.Validate();err!=nil {t.Fatal(err)}
 if err:=p.TrimLeft(1,0.8);err!=nil {t.Fatal(err)}
 if err:=p.Validate();err!=nil {t.Fatal(err)}
 if err:=p.Move(0,2);err!=nil {t.Fatal(err)}
 if p.HasTransitions(){t.Fatal("reorder should clear stale crossfades")}
 if err:=p.Validate();err!=nil {t.Fatal(err)}
}
func TestNoTransitionsBackwardCompatible(t *testing.T) {
 p:=NewProject();_ =p.Insert(0,transitionTestClip("a"));_ =p.Insert(1,transitionTestClip("b"))
 closeTime(t,p.Duration(),4);closeTime(t,p.Start(1),2)
 if err:=p.Validate();err!=nil {t.Fatal(err)}
}
