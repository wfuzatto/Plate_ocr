package nvrclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/wfuzatto/Plate_ocr/internal/domain"
)

type Camera struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	City              string  `json:"city,omitempty"`
	Site              string  `json:"site,omitempty"`
	Enabled           bool    `json:"enabled"`
	SnapshotAvailable bool    `json:"snapshot_available"`
	Latitude          *float64 `json:"latitude,omitempty"`
	Longitude         *float64 `json:"longitude,omitempty"`
}

type Client struct {
	base string
	token string
	http *http.Client
}

func LoadToken(path string) (string,error){
	if direct:=strings.TrimSpace(os.Getenv("NVR_PLUGIN_TOKEN"));direct!=""{return direct,nil}
	b,err:=os.ReadFile(path)
	if err!=nil{return "",err}
	raw:=strings.TrimSpace(string(b))
	if raw==""{return "",errors.New("plugin token file is empty")}
	if decoded,err:=base64.RawStdEncoding.DecodeString(raw);err==nil&&len(decoded)>=24{
		return base64.RawURLEncoding.EncodeToString(decoded),nil
	}
	if decoded,err:=base64.StdEncoding.DecodeString(raw);err==nil&&len(decoded)>=24{
		return base64.RawURLEncoding.EncodeToString(decoded),nil
	}
	return raw,nil
}

func New(baseURL,token string,timeout time.Duration)(*Client,error){
	u,err:=url.Parse(strings.TrimRight(strings.TrimSpace(baseURL),"/"))
	if err!=nil||u.Scheme==""||u.Host==""{return nil,errors.New("invalid NVR base URL")}
	if u.Scheme!="http"&&u.Scheme!="https"{return nil,errors.New("NVR URL must use http or https")}
	if strings.TrimSpace(token)==""{return nil,errors.New("plugin token is required")}
	if timeout<=0{timeout=5*time.Second}
	return &Client{base:u.String(),token:token,http:&http.Client{Timeout:timeout}},nil
}

func (c *Client) ListCameras(ctx context.Context)([]Camera,error){
	req,err:=c.request(ctx,http.MethodGet,"/api/v1/plugin/v1/cameras",nil)
	if err!=nil{return nil,err}
	resp,err:=c.http.Do(req)
	if err!=nil{return nil,err}
	defer resp.Body.Close()
	if err:=expect(resp,http.StatusOK);err!=nil{return nil,err}
	var body struct{Items []Camera `json:"items"`}
	if err:=json.NewDecoder(io.LimitReader(resp.Body,2<<20)).Decode(&body);err!=nil{return nil,err}
	return body.Items,nil
}

func (c *Client) FetchFrame(ctx context.Context,cameraID string)([]byte,time.Time,error){
	req,err:=c.request(ctx,http.MethodGet,"/api/v1/plugin/v1/cameras/"+url.PathEscape(cameraID)+"/frame",nil)
	if err!=nil{return nil,time.Time{},err}
	req.Header.Set("Accept","image/jpeg,image/*")
	resp,err:=c.http.Do(req)
	if err!=nil{return nil,time.Time{},err}
	defer resp.Body.Close()
	if err:=expect(resp,http.StatusOK);err!=nil{return nil,time.Time{},err}
	const max=12<<20
	b,err:=io.ReadAll(io.LimitReader(resp.Body,max+1))
	if err!=nil{return nil,time.Time{},err}
	if len(b)>max{return nil,time.Time{},errors.New("frame exceeds 12 MiB")}
	observed:=time.Now().UTC()
	if raw:=resp.Header.Get("X-NVR-Observed-At");raw!=""{
		if t,e:=time.Parse(time.RFC3339Nano,raw);e==nil{observed=t.UTC()}
	}
	return b,observed,nil
}

func (c *Client) UploadEvidence(ctx context.Context,eventID string,jpeg []byte)(string,error){
	if len(jpeg)==0{return "",nil}
	req,err:=c.request(ctx,http.MethodPost,"/api/v1/plugin/v1/evidence",bytes.NewReader(jpeg))
	if err!=nil{return "",err}
	req.Header.Set("Content-Type","image/jpeg")
	req.Header.Set("X-Event-ID",eventID)
	resp,err:=c.http.Do(req)
	if err!=nil{return "",err}
	defer resp.Body.Close()
	if err:=expect(resp,http.StatusCreated);err!=nil{return "",err}
	var body struct{SnapshotRef string `json:"snapshot_ref"`}
	if err:=json.NewDecoder(io.LimitReader(resp.Body,1<<20)).Decode(&body);err!=nil{return "",err}
	if body.SnapshotRef==""{return "",errors.New("NVR returned empty snapshot_ref")}
	return body.SnapshotRef,nil
}

func (c *Client) PublishEvent(ctx context.Context,ev domain.PlateEvent)error{
	attrs:=map[string]any{
		"track_id":ev.TrackID,"raw_text":ev.RawText,"normalized_text":ev.NormalizedText,
		"format":ev.Format,"candidates":ev.Candidates,"detector_confidence":ev.DetectorConfidence,
		"bbox":ev.BBox,"lane":ev.Lane,"direction":ev.Direction,
	}
	body:=map[string]any{
		"event_id":ev.EventID,"event_type":ev.EventType,"schema_version":ev.SchemaVersion,
		"camera_id":ev.CameraID,"observed_at":ev.ObservedAt.UTC(),
		"plugin_id":ev.PluginID,"plugin_version":ev.PluginVersion,
		"confidence":ev.Confidence,"snapshot_ref":ev.SnapshotRef,"dedupe_key":ev.DedupeKey,
		"attributes":attrs,
	}
	payload,err:=json.Marshal(body)
	if err!=nil{return err}
	req,err:=c.request(ctx,http.MethodPost,"/api/v1/plugin/v1/events",bytes.NewReader(payload))
	if err!=nil{return err}
	req.Header.Set("Content-Type","application/json")
	resp,err:=c.http.Do(req)
	if err!=nil{return err}
	defer resp.Body.Close()
	if resp.StatusCode==http.StatusOK||resp.StatusCode==http.StatusCreated{return nil}
	return responseError(resp)
}

func (c *Client) request(ctx context.Context,method,path string,body io.Reader)(*http.Request,error){
	req,err:=http.NewRequestWithContext(ctx,method,c.base+path,body)
	if err!=nil{return nil,err}
	req.Header.Set("Authorization","Bearer "+c.token)
	req.Header.Set("User-Agent","plate-ocr/1.0")
	return req,nil
}
func expect(resp *http.Response,want int)error{if resp.StatusCode==want{return nil};return responseError(resp)}
func responseError(resp *http.Response)error{
	b,_:=io.ReadAll(io.LimitReader(resp.Body,64<<10))
	msg:=strings.TrimSpace(string(b))
	if msg==""{msg=http.StatusText(resp.StatusCode)}
	return fmt.Errorf("NVR HTTP %d: %s",resp.StatusCode,msg)
}
