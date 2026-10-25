package media

import (
 "context"
 "math"
 "os/exec"
 "path/filepath"
 "testing"
 "time"
 "github.com/SeraphinaDX/VelvetCut/editor"
)
func TestRealTransitionsDurationAndAudio(t *testing.T) {
 for _,name:=range []string{"ffmpeg","ffprobe"} {if _,err:=exec.LookPath(name);err!=nil{t.Skip(name+" unavailable")}}
 ctx,cancel:=context.WithTimeout(context.Background(),60*time.Second);defer cancel()
 clips:=[]editor.Clip{}
 for i,color:=range []string{"red","blue","yellow"} {
  path:=filepath.Join(t.TempDir(),color+".mp4")
  freq:=[]string{"330","550","770"}[i]
  cmd:=exec.CommandContext(ctx,"ffmpeg","-hide_banner","-loglevel","error","-y","-f","lavfi","-i","color=c="+color+":s=160x90:r=30:d=2","-f","lavfi","-i","sine=frequency="+freq+":duration=2","-shortest","-c:v","mpeg4","-c:a","aac",path)
  if out,err:=cmd.CombinedOutput();err!=nil{t.Fatalf("setup: %v %s",err,out)}
  clips=append(clips,editor.Clip{ID:color,Path:path,Name:color,SourceEnd:2,SourceDuration:2,HasAudio:true})
 }
 p:=editor.NewProject();p.Clips=clips
 if err:=p.SetTransition(0,"dissolve",0.6);err!=nil{t.Fatal(err)}
 if err:=p.SetTransition(1,"black",0.5);err!=nil{t.Fatal(err)}
 output:=filepath.Join(t.TempDir(),"render.mp4")
 if err:=ExportPreview(ctx,p,output);err!=nil{t.Fatal(err)}
 meta,err:=Probe(ctx,output);if err!=nil{t.Fatal(err)}
 if !meta.HasAudio {t.Fatal("missing exported audio")}
 if math.Abs(meta.Duration-p.Duration())>0.2{t.Fatalf("output %.3f expected %.3f",meta.Duration,p.Duration())}
}
