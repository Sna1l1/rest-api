package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	apiserver "github.com/Sna1l1/rest-api/internal/app/apiserver/utils"
	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/rs/xid"

	"github.com/gorilla/csrf"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type contextKey int

const (
	ctxKeyRequestID = contextKey(iota)
)

type Server struct {
	Router  *mux.Router
	Logger  *logrus.Logger
	Mongo   *mongo.Client
	Session sessions.Store
	ctx     context.Context
}

func NewServer(mongo mongo.Client, session sessions.Store, ctx context.Context) *Server {
	server := &Server{
		Router:  mux.NewRouter(),
		Logger:  logrus.New(),
		Mongo:   &mongo,
		Session: session,
		ctx:     ctx,
	}
	server.Configure()

	return server
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}

func (s *Server) Configure() {
	s.Router.Use(handlers.CORS(handlers.AllowedOrigins([]string{"*"})))
	s.Router.Use(s.SetUUID)
	s.Router.Use(s.logRequest)

	s.Router.HandleFunc("/", s.HandleMain()).Methods("GET")
	s.Router.HandleFunc("/signin", s.HandleSignin()).Methods("GET")
	s.Router.HandleFunc("/signup", s.HandleSignup()).Methods("GET")
	s.Router.HandleFunc("/register", s.HandleRegister()).Methods("POST")
	s.Router.HandleFunc("/login", s.HandleLogin()).Methods("POST")
	s.Router.HandleFunc("/logout", s.HandleLogout()).Methods("GET")
}
func (s *Server) error(w http.ResponseWriter, r *http.Request, code int, err error) {
	s.respond(w, r, code, map[string]string{"error": err.Error()})
}
func (s *Server) respond(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.WriteHeader(code)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}
func (s *Server) SetUUID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyRequestID, id)))
	})
}
func (s *Server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := s.Logger.WithFields(logrus.Fields{
			"remote_addr": r.RemoteAddr,
			"request_id":  r.Context().Value(ctxKeyRequestID),
		})
		logger.Infof("Started %s , %s", r.Method, r.RequestURI)

		start := time.Now()
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		var level logrus.Level

		switch {
		case rw.code >= 500:
			level = logrus.ErrorLevel
		case rw.code >= 400:
			level = logrus.WarnLevel
		default:
			level = logrus.InfoLevel
		}
		logger.Logf(
			level,
			"completed with %d %s in %v",
			rw.code,
			http.StatusText(rw.code),
			time.Since(start),
		)
	})
}

func (s *Server) HandleMain() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := s.Session.Get(r, "session")
		if err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}
		templ, err := template.ParseFiles("template/index.html")

		if err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}

		if auth, ok := sessions.Values["auth"].(bool); !auth || !ok {
			templ.Execute(w, true)
			return
		}
		templ.Execute(w, false)
	}
}
func (s *Server) HandleSignin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := s.Session.Get(r, "session")
		if err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}
		if auth, ok := sessions.Values["auth"].(bool); !auth || !ok {
			templ, err := template.ParseFiles("template/main.html")

			if err != nil {
				s.error(w, r, http.StatusInternalServerError, err)
				return
			}

			templ.Execute(w, map[string]interface{}{
				csrf.TemplateTag: csrf.TemplateField(r),
				"csrfToken":      csrf.Token(r),
			})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func (s *Server) HandleSignup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := s.Session.Get(r, "session")
		if err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}
		if auth, ok := sessions.Values["auth"].(bool); !auth || !ok {
			templ, err := template.ParseFiles("template/signup.html")

			if err != nil {
				s.error(w, r, http.StatusInternalServerError, err)
				return
			}

			templ.Execute(w, map[string]interface{}{
				csrf.TemplateTag: csrf.TemplateField(r),
				"csrfToken":      csrf.Token(r),
			})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func (s *Server) HandleRegister() http.HandlerFunc {
	type User struct {
		Username string
		HashPass string
		Email    string
		User_id  string
	}
	return func(w http.ResponseWriter, r *http.Request) {
		pass, err := apiserver.HashPsw(r.FormValue("pass"), r.FormValue("rePass"))
		if err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}
		if err = apiserver.ValidateUsername(s.Mongo, s.ctx, r.FormValue("username")); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}
		if err = apiserver.ValidateEmail(s.Mongo, s.ctx, r.FormValue("email")); err != nil {
			s.error(w, r, http.StatusBadRequest, err)
			return
		}
		u := User{r.FormValue("username"), pass, r.FormValue("email"), xid.New().String()}
		if err = InsertOne(s.Mongo, "test", s.ctx, u); err != nil {
			s.error(w, r, http.StatusInternalServerError, err)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)

	}
}
func (s *Server) HandleLogin() http.HandlerFunc {
	type User struct {
		Username string
		HashPass string
		Email    string
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("email") == "" || r.FormValue("pass") == "" {
			s.error(w, r, http.StatusBadRequest, fmt.Errorf("empty email or password"))
			return
		}
		var user User

		collection := s.Mongo.Database("sample_analytics").Collection("test")

		if err := collection.FindOne(s.ctx, bson.M{"email": r.FormValue("email")}).Decode(&user); err != nil {
			s.error(w, r, http.StatusUnauthorized, fmt.Errorf("invalid email or password"))
			return
		}
		match, err := argon2id.ComparePasswordAndHash(r.FormValue("pass"), user.HashPass)
		if err != nil || !match {
			s.error(w, r, http.StatusUnauthorized, fmt.Errorf("invalid email or password"))
			return
		}

		session, _ := s.Session.Get(r, "session")
		session.Values["auth"] = true
		session.Save(r, w)
		http.Redirect(w, r, "/", http.StatusSeeOther)

	}
}
func (s *Server) HandleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.Session.Get(r, "session")

		session.Values["auth"] = false
		session.Save(r, w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
