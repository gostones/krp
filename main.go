package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gostones/krp/tunnel"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

func getEnv(key, def string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return def
}

// func getListenAddr() string {
// 	port := getEnv("PORT", "8000")
// 	return ":" + port
// }

// func httpProxy(remote string) http.Handler {
// 	url := toURL(remote)
// 	proxy := httputil.NewSingleHostReverseProxy(url)

// 	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
// 		req.URL.Host = url.Host
// 		req.URL.Scheme = url.Scheme
// 		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
// 		req.Host = url.Host

// 		proxy.ServeHTTP(res, req)
// 	})
// }

type Health struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

type K8sService struct {
	Items []struct {
		Kind     string
		Metadata struct {
			Name      string
			Namespace string
		}
		Spec struct {
			ClusterIP string
			Ports     []struct {
				Name       string
				Port       int
				Protocol   string
				targetPort int
			}
			Type     string
			Template struct {
				Spec struct {
					Containers []struct {
						Name  string
						Ports []struct {
							ContainerPort int
							Name          string
							Protocol      string
						}
					}
				}
			}
		}
	}
}

//TODO Angular or React
func formatSvc(j *K8sService) string {
	s := ""
	for _, item := range j.Items {
		fmt.Printf("%v\n", item)
		if item.Kind == "Service" {
			for _, port := range item.Spec.Ports {
				if port.Protocol != "TCP" {
					continue
				}
				// /port-forward/{ns}/{type}/{name}/{port}
				a := fmt.Sprintf("/port-forward/%v/svc/%v/%v", item.Metadata.Namespace, item.Metadata.Name, port.Port)
				s += fmt.Sprintf(`<a href="%v" target="_blank">%v %v</a><br /><br />`, a, item.Metadata.Namespace, item.Metadata.Name)
			}
		}
	}
	return s
}

//k8s API server
var (
	kserver string
)

func kubeGetSvc() *K8sService {
	args := []string{"kubectl", "--server", kserver, "get", "--all-namespaces", "pod,svc,deploy", "-o", "json"}

	s := kubectl(args)
	j := &K8sService{}
	if err := json.Unmarshal([]byte(s), j); err != nil {
		fmt.Printf("%v", err)
		return nil
	}
	return j
}

func kubePortForward(ns string, name string, port int) int {
	lport := FreePort()
	args := []string{"kubectl", "--server", kserver, "port-forward", "-n", ns, name, fmt.Sprintf("%v:%v", lport, port)}

	go func() {
		log.Printf("@@@ %v\n", args)
		s := kubectl(args)
		log.Printf("@@@ kubectl: %v\n", s)
	}()

	return lport
}

var help = `
	Usage: krp --help --tunnel <url> --port 8001 --proxy <url>

`

func usage() {
	fmt.Fprintf(os.Stderr, help)
	os.Exit(1)
}

var httpClient = &http.Client{
	Timeout: time.Second * 10,
}

func main() {
	h := flag.Bool("help", false, "")

	bind := flag.Int("bind", -1, "")
	port := flag.Int("port", 8001, "")
	tun := flag.String("tunnel", "", "")
	proxy := flag.String("proxy", "", "")

	flag.Parse()

	if *h {
		usage()
	}

	if *bind == -1 {
		*bind = parseInt(getEnv("PORT", ""), 8000)
	}

	if *proxy == "" {
		*proxy = os.Getenv("http_proxy")
	}

	if *tun == "" {
		kserver = fmt.Sprintf("localhost:%v", *port)
	} else {
		lport := FreePort()
		kserver = fmt.Sprintf("localhost:%v", lport)
		remote := fmt.Sprintf("localhost:%v:localhost:%v", lport, *port)
		go tunnel.TunClient(*proxy, *tun, remote)
	}
	log.Printf("k8s API server: %v\n", kserver)

	//
	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		m := &Health{
			Status:    "OK",
			Timestamp: ToTimestamp(time.Now()),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		b, _ := json.Marshal(m)
		fmt.Fprintf(w, string(b))
	})

	r.HandleFunc("/api/all", func(w http.ResponseWriter, r *http.Request) {
		m := kubeGetSvc()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		b, _ := json.Marshal(m)
		fmt.Fprintf(w, string(b))
	})

	r.HandleFunc("/port-forward/{ns}/{type}/{name}/{port}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ns := vars["ns"]
		name := fmt.Sprintf("%v/%v", vars["type"], vars["name"])
		port := parseInt(vars["port"], -1)

		lport := kubePortForward(ns, name, port)

		redirect := fmt.Sprintf("http://localhost:%v/", lport)

		//TODO handle error properly
		serviceReady(redirect)

		log.Printf("@@@ redirect: %s", redirect)

		http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
	})

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		j := kubeGetSvc()
		s := formatSvc(j)
		w.Write([]byte(s))
	})

	la := fmt.Sprintf(":%v", *bind)
	log.Printf("Server listening at: %v\n", la)

	log.Fatal(http.ListenAndServe(la, r))
}

func toURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func parseInt(s string, v int) int {
	if s == "" {
		return v
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		i = v
	}
	return i
}

func serviceReady(uri string) error {
	return Retry(func() error {
		res, err := httpClient.Get(uri)

		if err != nil {
			log.Printf("@@@ isServiceReady: %v", err)
			return err
		}
		defer res.Body.Close()

		log.Printf("@@@ isServiceReady: %v", res)

		if res.StatusCode == 200 {
			return nil
		}

		return fmt.Errorf("Error: %v", res.StatusCode)
	}, NewBackOff(12, 1*time.Second))
}

func FreePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func ToTimestamp(d time.Time) int64 {
	return d.UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}
