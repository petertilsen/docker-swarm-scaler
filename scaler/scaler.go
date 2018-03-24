package scaler

import (
	"bytes"
	"encoding/json"
	"fmt"
	slack "github.com/leprosus/golang-slack-notifier"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	firing   = "firing"
	resolved = "resolved"
)

type alerts struct {
	Status string  `json:"status"`
	Alerts []alert `json:"alerts"`
}

type alert struct {
	Labels      KV
	Annotations KV
	StartsAt    time.Time
	EndsAt      time.Time
}

type services struct {
	Service []service
	Status  string
}

type service struct {
	Name  string
	Scale int
}

type KV map[string]string

var replicas = getReplicas
var command = executeCmd

func handler(w http.ResponseWriter, r *http.Request) {
	var a alerts
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&a)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ss := a.handleAlerts()
	ss.Status = "OK"

	js, err := json.Marshal(ss)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (a *alerts) handleAlerts() services {
	var s service
	var ss services
	for _, element := range a.Alerts {
		s.Name = element.Annotations["service"]
		if s.Name != "" {
			switch a.Status {
			case firing:
				s.scaleUp()
			case resolved:
				s.scaleDown()
			}
		}
		ss.Service = append(ss.Service, s)
	}

	return ss
}

func (s *service) scaleUp() bool {
	c, t := replicas(s)
	if c >= 50 {
		fmt.Fprintf(os.Stderr, "Can't scale above 50. Current is %d", c)
		return true
	}

	if c < t {
		fmt.Fprintf(os.Stderr, "Can't scale up. current %d lower than target %d", c, t)
		return true
	}

	s.Scale = c + 1

	s.scaleCmd()
	s.update()

	return true
}

func (s *service) scaleDown() bool {
	c, t := replicas(s)
	if t <= 1 {
		fmt.Fprintf(os.Stderr, "Can't scale below 1")
		return true
	}
	if c <= t-1 {
		fmt.Fprintf(os.Stderr, "Current <= target")
		return true
	}

	s.Scale = t - 1
	s.scaleCmd()

	return true
}

func getReplicas(s *service) (int, int) {
	replicas := s.replicasCmd()
	r := strings.TrimSuffix(string(replicas[:]), "\n")
	sl := strings.Split(r, "/")

	c, _ := strconv.Atoi(sl[0])
	t, _ := strconv.Atoi(sl[len(sl)-1])

	return c, t
}

func (s *service) update() []byte {
	s.loginCmd()
	i := s.imageCmd()
	return s.updateCmd(i)
}

func (s *service) replicasCmd() []byte {
	cmdStr := fmt.Sprintf("docker service ls --filter name=%s --format '{{.Replicas}}'", s.Name)
	return command(cmdStr, false)
}

func (s *service) updateCmd(i string) []byte {
	cmdStr := fmt.Sprintf("docker pull %s "+
		"&& docker service update --force --update-parallelism 1 --update-delay 10s "+
		"--with-registry-auth --image %s %s", i, i, s.Name)
	return command(cmdStr, true)
}

func (s *service) loginCmd() []byte {
	cmdStr := fmt.Sprintf("export AWS_ACCESS_KEY_ID=%s && export AWS_SECRET_ACCESS_KEY=%s "+
		"&& login=`aws ecr get-login --no-include-email --region eu-central-1` && echo `$login`",
		os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"))
	return command(cmdStr, false)
}

func (s *service) imageCmd() string {
	cmdStr := fmt.Sprintf("docker service ls --filter name=%s --format '{{.Image}}'", s.Name)
	image := command(cmdStr, false)
	return strings.TrimSuffix(string(image[:]), "\n")
}

func (s *service) scaleCmd() []byte {
	cmdStr := fmt.Sprintf("docker service scale %s=%d", s.Name, s.Scale)
	notify(fmt.Sprintf("docker service %s scaled to %d", s.Name, s.Scale))
	return command(cmdStr, true)
}

func executeCmd(cmd string, dryrun bool) []byte {
	var outb, errb bytes.Buffer

	fmt.Fprintf(os.Stdout, cmd)
	if dryrun == true && os.Getenv("DRYRUN") != "" {
		fmt.Fprintf(os.Stdout, "Dryrun")
		return []byte{}
	}

	r := exec.Command("/bin/sh", "-c", cmd)

	r.Stdout = &outb
	r.Stderr = &errb
	err := r.Run()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Cmd: %s failed with %s", cmd, errb.String())
		panic(err.Error())
	}

	fmt.Fprintf(os.Stdout, "Cmd: %s succeeded with %s", cmd, outb.String())

	return outb.Bytes()
}

func notify(message string) {
	slackConn := slack.New(os.Getenv("SLACK_HOOK"))
	slackConn.Notify("scaler", message)
}

func Scaler() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8083", nil))
}
