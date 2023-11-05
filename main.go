package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/facebook"
	"github.com/stretchr/gomniauth/providers/github"
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/stretchr/objx"

	"github.com/law-lee/chat_server/trace"
)

// set the active Avatar implementation
var avatars Avatar = TryAvatars{
	UseFileSystemAvatar,
	UseAuthAvatar,
	UseGravatar}

// templateHandler represents a single template
type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

// ServeHTTP handles the HTTP request.
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(
			template.ParseFiles(
				filepath.Join("templates", t.filename)))
	})
	//Instead of just passing the entire http.Request object to our template as data, we are
	//creating a new map[string]interface{} definition for a data object that potentially has
	//two fields: Host and UserData
	data := map[string]interface{}{
		"Host": r.Host,
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
	t.templ.Execute(w, data)
}

func main() {
	var addr = flag.String("addr", ":8080", "The addr of the application.")
	flag.Parse() // parse the flags
	// replace your own google client auth
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSec := os.Getenv("GOOGLE_CLIENT_SEC")
	// setup gomniauth
	gomniauth.SetSecurityKey("AIzaSyA1p5cwIbIOk1Dnn9IrRRdTGjxwaqvw")
	gomniauth.WithProviders(
		facebook.New("key", "secret",
			"http://localhost:8080/auth/callback/facebook"),
		github.New("key", "secret",
			"http://localhost:8080/auth/callback/github"),
		google.New(clientID, clientSec, "http://localhost:8080/auth/callback/google"),
	)
	// options UseAuthAvatar/UseGravatarAvatar/UseFileSystemAvatar
	r := newRoom()
	r.tracer = trace.New(os.Stdout)
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.Handle("/room", r)
	//If we build and run our application having logged in with a previous version, you will find
	//that the auth cookie that doesn't contain the avatar URL is still there. We are not asked to
	//authenticate again (since we are already logged in), and the code that adds the avatar_url
	//field never gets a chance to run. We could delete our cookie and refresh the page, but we
	//would have to keep doing this whenever we make changes during development. Let's solve
	//this problem properly by adding a logout feature
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "auth",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		w.Header().Set("Location", "/chat")
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.Handle("/upload", &templateHandler{filename: "upload.html"})
	http.HandleFunc("/uploader", uploaderHandler)
	//If we didn't strip the /avatars/ prefix from the requests with
	//http.StripPrefix, the file server would look for another folder called
	//avatars inside the actual avatars folder, that is,
	///avatars/avatars/filename instead of /avatars/filename.
	http.Handle("/avatars/",
		http.StripPrefix("/avatars/",
			http.FileServer(http.Dir("./avatars"))))
	// get the room going
	go r.run()
	// start the web server
	log.Println("Starting web server on", *addr)

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
