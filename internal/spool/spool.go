package spool

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/wfuzatto/Plate_ocr/internal/domain"
)

type diskJob struct {
	Event domain.PlateEvent `json:"event"`
	EvidenceFile string      `json:"evidence_file,omitempty"`
}

type Job struct {
	Event domain.PlateEvent
	Evidence []byte
	Path string
	EvidencePath string
}

type Store struct{
	dir string
	mu sync.Mutex
}

func Open(dir string)(*Store,error){
	if strings.TrimSpace(dir)==""{return nil,errors.New("spool dir is required")}
	if err:=os.MkdirAll(dir,0o750);err!=nil{return nil,err}
	return &Store{dir:dir},nil
}

func (s *Store) Enqueue(ev domain.PlateEvent,evidence []byte)error{
	s.mu.Lock();defer s.mu.Unlock()
	base:=safe(ev.EventID)
	if base==""{return errors.New("event id is required")}
	evidenceName:=""
	if len(evidence)>0{
		evidenceName=base+".jpg"
		if err:=atomicWrite(filepath.Join(s.dir,evidenceName),evidence,0o600);err!=nil{return err}
	}
	clean:=ev
	clean.EvidenceJPEG=nil
	payload,err:=json.Marshal(diskJob{Event:clean,EvidenceFile:evidenceName})
	if err!=nil{return err}
	if err:=atomicWrite(filepath.Join(s.dir,base+".json"),append(payload,'
'),0o600);err!=nil{
		if evidenceName!=""{_ = os.Remove(filepath.Join(s.dir,evidenceName))}
		return err
	}
	return nil
}

func (s *Store) Pending(limit int)([]Job,error){
	s.mu.Lock();defer s.mu.Unlock()
	entries,err:=os.ReadDir(s.dir)
	if err!=nil{return nil,err}
	names:=make([]string,0)
	for _,e:=range entries{if !e.IsDir()&&strings.HasSuffix(e.Name(),".json"){names=append(names,e.Name())}}
	sort.Strings(names)
	if limit>0&&len(names)>limit{names=names[:limit]}
	out:=make([]Job,0,len(names))
	for _,name:=range names{
		path:=filepath.Join(s.dir,name)
		b,err:=os.ReadFile(path);if err!=nil{return nil,err}
		var d diskJob
		if err:=json.Unmarshal(b,&d);err!=nil{return nil,err}
		job:=Job{Event:d.Event,Path:path}
		if d.EvidenceFile!=""{
			job.EvidencePath=filepath.Join(s.dir,d.EvidenceFile)
			job.Evidence,err=os.ReadFile(job.EvidencePath)
			if err!=nil&&!errors.Is(err,os.ErrNotExist){return nil,err}
		}
		out=append(out,job)
	}
	return out,nil
}

func (s *Store) Ack(job Job)error{
	s.mu.Lock();defer s.mu.Unlock()
	if job.EvidencePath!=""{_ = os.Remove(job.EvidencePath)}
	if job.Path!=""{return os.Remove(job.Path)}
	return nil
}

func (s *Store) Count()(int,error){
	entries,err:=os.ReadDir(s.dir);if err!=nil{return 0,err}
	n:=0;for _,e:=range entries{if !e.IsDir()&&strings.HasSuffix(e.Name(),".json"){n++}}
	return n,nil
}

func atomicWrite(path string,b []byte,mode os.FileMode)error{
	tmp:=path+".tmp"
	f,err:=os.OpenFile(tmp,os.O_CREATE|os.O_TRUNC|os.O_WRONLY,mode);if err!=nil{return err}
	if _,err=f.Write(b);err!=nil{_ = f.Close();return err}
	if err=f.Sync();err!=nil{_ = f.Close();return err}
	if err=f.Close();err!=nil{return err}
	if err=os.Rename(tmp,path);err!=nil{return err}
	return os.Chmod(path,mode)
}
func safe(v string)string{
	var b strings.Builder
	for _,r:=range v{if (r>='a'&&r<='z')||(r>='A'&&r<='Z')||(r>='0'&&r<='9')||r=='-'||r=='_'{b.WriteRune(r)}}
	return b.String()
}
