package marathon

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/CiscoCloud/marathon-consul/apps"
	"github.com/CiscoCloud/marathon-consul/tasks"
	log "github.com/Sirupsen/logrus"
	"github.com/sethgrid/pester"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Marathoner interface {
	Apps() ([]*apps.App, error)
	Tasks(string) ([]*tasks.Task, error)
}

type Marathon struct {
	Location    string
	Protocol    string
	Auth        *url.Userinfo
	NoVerifySsl bool
}

func NewMarathon(location, protocol string, auth *url.Userinfo) (Marathon, error) {
	return Marathon{location, protocol, auth, false}, nil
}

func (m Marathon) Url(path string) string {
	marathon := url.URL{
		Scheme: m.Protocol,
		User:   m.Auth,
		Host:   m.Location,
		Path:   path,
	}

	return marathon.String()
}

func (m Marathon) getClient() *pester.Client {
	client := pester.New()
	client.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: m.NoVerifySsl,
		},
	}

	return client
}

func (m Marathon) Apps() ([]*apps.App, error) {
	log.WithField("location", m.Location).Debug("asking Marathon for apps")
	client := m.getClient()

	appsResponse, err := client.Get(m.Url("/v2/apps"))
	if err != nil || (appsResponse.StatusCode != 200) {
		m.logHTTPError(appsResponse, err)
		return nil, err
	}

	body, err := ioutil.ReadAll(appsResponse.Body)
	if err != nil {
		m.logHTTPError(appsResponse, err)
		return nil, err
	}

	appList, err := m.ParseApps(body)
	if err != nil {
		m.logHTTPError(appsResponse, err)
	}

	return appList, err
}

type AppResponse struct {
	Apps []*apps.App `json:"apps"`
}

func (m Marathon) ParseApps(jsonBlob []byte) ([]*apps.App, error) {
	apps := &AppResponse{}
	err := json.Unmarshal(jsonBlob, apps)

	return apps.Apps, err
}

func (m Marathon) Tasks(app string) ([]*tasks.Task, error) {
	log.WithFields(log.Fields{
		"location": m.Location,
		"app":      app,
	}).Debug("asking Marathon for tasks")
	client := m.getClient()

	if app[0] == '/' {
		app = app[1:]
	}

	tasksResponse, err := client.Get(m.Url(fmt.Sprintf("/v2/apps/%s/tasks", app)))
	if err != nil || (tasksResponse.StatusCode != 200) {
		m.logHTTPError(tasksResponse, err)
		return nil, err
	}

	body, err := ioutil.ReadAll(tasksResponse.Body)
	if err != nil {
		m.logHTTPError(tasksResponse, err)
		return nil, err
	}

	taskList, err := m.ParseTasks(body)
	if err != nil {
		m.logHTTPError(tasksResponse, err)
	}

	return taskList, err
}

type TasksResponse struct {
	Tasks []*tasks.Task `json:"tasks"`
}

func (m Marathon) ParseTasks(jsonBlob []byte) ([]*tasks.Task, error) {
	tasks := &TasksResponse{}
	err := json.Unmarshal(jsonBlob, tasks)

	return tasks.Tasks, err
}

func (m Marathon) logHTTPError(rep *http.Response, err error) {
	log.WithFields(log.Fields{
		"location":   m.Location,
		"statusCode": rep.StatusCode,
	}).Error(err)
}
