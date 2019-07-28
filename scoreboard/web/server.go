package web

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/websocket"
	"golang.org/x/xerrors"
)

// Server is a struct that supports web file browsing as well as adding
// specific paths to functions.
type Server struct {
	fs   http.Handler
	box  *packr.Box
	dir  http.FileSystem
	bind string
}

// Stream repersents a WebSocket connection, returned by the WebSocket upgrader.
type Stream struct {
	*websocket.Conn
}
type websockServer struct {
	u  *websocket.Upgrader
	cb func(*Stream)
}
type handleFunc func(http.ResponseWriter, *http.Request)

// IP returns the IP of the client comnected to this Steam.
func (s *Stream) IP() string {
	return s.RemoteAddr().String()
}

// Start starts the Server listening loop and returns an error if the server could not be started.
// Only returns an error if any IO issues occur during operation.
func (s *Server) Start() error {
	return http.ListenAndServe(s.bind, nil)
}

// Open satisifies the http.FileSystem interface.
func (s *Server) Open(n string) (http.File, error) {
	f, err := s.dir.Open(n)
	if err != nil && s.box != nil {
		if r, err := s.box.Open(path.Join("public", n)); err == nil {
			return r, nil
		}
	}
	return f, err
}

// AddHandlerFunc adds the following function to be triggered for the provded path.
func (s *Server) AddHandlerFunc(path string, f handleFunc) {
	http.HandleFunc(path, f)
}

// AddHandler adds the following handler to be triggered for the provded path.
func (s *Server) AddHandler(path string, handler http.Handler) {
	http.Handle(path, handler)
}

// ServeHTTP satisifies the http.Handler requirement for the interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.fs.ServeHTTP(w, r)
}

// NewWebSocket returns a WebSocket HTTP handler that can upgrade standard HTTP connections into
// websockets. The passed callback function will be called when a websocket is created.
func NewWebSocket(bufsize int, callback func(*Stream)) http.Handler {
	return &websockServer{
		cb: callback,
		u: &websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  bufsize,
			WriteBufferSize: bufsize,
		},
	}
}

// NewServer creates a Server struct from the provded listen address and directory path.
// This function will return an error if the provded directory path is not valid.
func NewServer(listen, dir string, box *packr.Box) (*Server, error) {
	if len(dir) > 0 {
		z, err := os.Stat(dir)
		if err != nil {
			return nil, xerrors.Errorf("cannot get directory \"%s\": %w", dir, err)
		}
		if !z.IsDir() {
			return nil, xerrors.Errorf("path \"%s\" is not a directory", dir)
		}
	}
	s := &Server{
		box:  box,
		dir:  http.Dir(dir),
		bind: listen,
	}
	s.fs = http.FileServer(s)
	return s, nil
}
func (s *websockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := s.u.Upgrade(w, r, nil)
	if err == nil {
		s.cb(&Stream{c})
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, http.StatusText(http.StatusInternalServerError))
	}
}
