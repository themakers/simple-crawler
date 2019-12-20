package main

import (
	"context"
	"github.com/rs/xid"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func DummyPageHandler() func(w http.ResponseWriter, q *http.Request) {
	loremIpsum := []string{
		`Lorem ipsum dolor sit amet, vel ex vero paulo vocent, illud consetetur te his. Possit postulant consetetur sit eu, in eos insolens voluptatum. Ne vim labore disputando. Ut eos etiam albucius neglegentur. Nisl utinam aeterno usu ad, ea nisl velit dissentias eos, diam tation mel ne. Usu an detracto appareat, cum laoreet gubergren id.`,
		`Noster eirmod quaeque sed in. Dolore forensibus adipiscing id est. Affert tation reprehendunt at vel. Ei vis quaestio dignissim, in delicata facilisis vulputate nec, nec te inermis quaestio cotidieque.`,
		`Mea an numquam incorrupte, sed postea cetero voluptatibus eu. Brute labitur in per, accumsan appareat pro ea, mei ex habemus voluptatum honestatis. Mel eius percipit id. Affert graeci ut his, per cu ferri dicta.`,
		`An duis mnesarchum nam, eu equidem fastidii necessitatibus pri, libris fabulas comprehensam eu vel. An vis iracundia appellantur, vel ne prima augue bonorum. Et liber insolens ius. Autem aliquip ex mei, agam fugit has ea, vis elit tincidunt eu. Labore nominavi singulis vix in.`,
		`Usu omnes dictas prodesset ad, velit verear ut cum. Cu mei nihil noster vulputate, partem hendrerit vix an. Cum harum impetus omittantur id, nominavi volutpat et usu, natum temporibus at quo. Ad est ceteros electram. Pri nullam praesent te, eu adhuc veniam honestatis duo.`,
		`Modo partiendo his no, labores epicuri quaestio usu eu, cu quo atqui tantas quaestio. Semper iisque ad vel, nec eligendi officiis iracundia ut. Ea duo malis tibique. Eum repudiandae disputationi no, odio sale assueverit et sed, facer consequuntur definitionem et vim. Tibique democritum voluptaria mea ut, mei cu magna alienum, ad hinc partem tritani sea. Sit quis agam cu, enim nibh consetetur cum an, vel ad etiam semper inimicus.`,
		`Et his summo populo impetus. Sit cu wisi choro viderer, no explicari definiebas mei, elitr exerci est ad. Dolore saperet oportere no has, in usu vide cetero. Quo sale diceret ponderum te. Vel ne prompta legimus, legendos deterruisset vel et.`,
		`Consulatu cotidieque vis ea. Utroque suavitate per eu, has oratio probatus invenire ne, sea enim zril populo cu. Cu hinc vivendum nam, postea insolens mei ex. Duo malis facilis at. An eum commodo facilisi disputando, sumo imperdiet honestatis usu te, audiam aperiri accusam an vis.`,
		`In verear feugait omnesque ius. Mel ex idque dolorem, ei assum scribentur eum, diam suscipit luptatum ex pro. Vix ex percipit urbanitas argumentum, unum invenire philosophia ut nam. Minimum repudiandae id est. Docendi qualisque democritum ut sit.`,
		`Diceret docendi honestatis in pro, mea quodsi sanctus albucius ne. Movet diceret placerat at pri, an per magna aliquip epicuri. Ad esse mundi prompta sed, cu sea noster detracto principes, mei cu alii nominati interesset. Ex vis aperiri convenire. Iriure praesent te nam, odio eripuit corrumpit duo ex. Mea exerci legere ne.`,
	}

	tpl := template.New("").
		Funcs(map[string]interface{}{
			"pageLink": func() string {
				return xid.New().String()
			},
			"placeholder": func() string {
				return loremIpsum[rand.Intn(len(loremIpsum))]
			},
		})

	tpl, err := tpl.Parse(`
		<!DOCTYPE html>
		<html>
		<head>
		<meta charset="UTF-8">
		<title>generated page</title>
		</head>
		<body>
			{{range $idx, $e := .repeat}} <p>{{placeholder}}</p>
			{{end}}
			{{range $idx, $e := .repeat}} <p><a href="/{{pageLink}}">{{placeholder}}</a></p>
			{{end}}
			{{range $idx, $e := .repeat}} <p>{{placeholder}}</p>
			{{end}}
			{{range $idx, $e := .repeat}} <p><a href="/{{pageLink}}">{{placeholder}}</a></p>
			{{end}}
			{{range $idx, $e := .repeat}} <p>{{placeholder}}</p>
			{{end}}
			{{range $idx, $e := .repeat}} <p><a href="/{{pageLink}}">{{placeholder}}</a></p>
			{{end}}
			{{range $idx, $e := .repeat}} <p>{{placeholder}}</p>
			{{end}}
		</body>
		</html>
	`)
	if err != nil {
		panic(err)
	}

	var repeat [10]interface{}

	return func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		if err := tpl.Execute(w, map[string]interface{}{
			"repeat": repeat,
		}); err != nil {
			panic(err)
		}
	}
}

// Constant-performance web server to measure crawler speed in synthetic test
func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go signalHandler(ctx, cancel)

	h := DummyPageHandler()

	server := &http.Server{
		Addr: ":7532",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
			h(w, q)
		}),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	<-ctx.Done()
}

func signalHandler(ctx context.Context, cancel context.CancelFunc) {
	//> usually systemd sends SIGTERM, waits 90 secs, then sends SIGKILL

	c := make(chan os.Signal, 1)

	signal.Notify(c)

	sigintsCount := 0

	for {
		select {
		case s := <-c:
			switch s {
			case os.Interrupt, syscall.SIGTERM, syscall.SIGHUP:
				log.Println("interrupt signal caught:", s.String())
				cancel()
				sigintsCount++
				if sigintsCount >= 3 {
					log.Println("too much interrupts, exiting with error:", s.String())
					os.Exit(1)
				}
			case os.Kill:
				log.Println("kill signal caught")
				cancel() //> In case Kill was the first signal

				go func() {
					time.Sleep(2 * time.Second)
					os.Exit(1)
				}()
			default:
			}

			//> dont need this, going to handle more signals after SIGTERM/SIGINT
			//case <-ctx.Done():
			//	return
		}
	}
}
