package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
	"time"

	"userapi/data"
	"userapi/db"
	uhealth "userapi/health"
	"userapi/pb"
	"userapi/validation"

	"github.com/bet365/jingo"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// log verbosity level
var (
	logVerbosity = 0
	HTTPPort     = 0
	GRPCPort     = 0

	userService *UserService

	// set our timeNow function, to allow us to stub it later
	timeNow = time.Now

	// set our newUUID function, to allow us to stub it later
	newUUID = uuid.NewString
)

func main() {
	flag.IntVar(&logVerbosity, "v", logVerbosity, "set the logging verbosity level")
	flag.IntVar(&HTTPPort, "httpport", 8080, "the main http server port to listen on")
	flag.IntVar(&GRPCPort, "grpcport", 9090, "the main grpc server port to listen on")

	flag.Parse()

	log.Printf("Starting version %v of userapi, httpport=%d, grpcport=%d", 1, HTTPPort, GRPCPort)

	err := db.Init()
	if err != nil {
		log.Fatal(err)
	}

	// start our server
	if err := start(); err != nil {
		log.Fatalf("error starting userapi service: %v", err)
	}

}

func start() error {

	mux := http.NewServeMux()

	// register http handlers
	mux.HandleFunc("/userapi/getall", getAllUsersHandler)
	mux.HandleFunc("/userapi/get", getUsersHandler)
	mux.HandleFunc("/userapi/add", addUserHandler)
	mux.HandleFunc("/userapi/update", updateUserHandler)
	mux.HandleFunc("/userapi/delete", deleteUserHandler)
	mux.HandleFunc("/userapi/deleteall", deleteAllUsersHandler)

	// Only returns OK when http & grpc is ready for serving connections
	mux.HandleFunc("/healthz", uhealth.CheckHandler)

	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", HTTPPort), Handler: mux}

	// Set up the gRPC server
	grpcServer := grpc.NewServer()
	userService = NewUserService()
	pb.RegisterUserServiceServer(grpcServer, userService)

	// Register health service
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	uhealth.GRPCAddress = fmt.Sprintf(":%d", GRPCPort)
	grpcLis, err := net.Listen("tcp", uhealth.GRPCAddress)
	if err != nil {
		err = fmt.Errorf("failed to listen: %v", err)
		return err
	}

	// WaitGroup to handle graceful shutdown of both servers
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Println("Starting HTTP server on port 8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Println("Starting gRPC server on port 9090")
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatalf("gRPC server Serve: %v", err)
		}
	}()

	// Listen for OS signals to gracefully shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Stop Signal received, shutting down")

	// Gracefully shutdown the HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server Shutdown: %v", err)
	}

	// Gracefully stop the gRPC server
	grpcServer.GracefulStop()

	// Wait for the servers to gracefully shutdown
	wg.Wait()
	log.Println("Servers gracefully stopped")

	return nil
}

//################################################################
// http rest Handlers
// why are all my handlers in main?
// 		Having all the main code entrypoints in main, makes it extremely easy to jump into a service and see whats going on
// 		It helps with visiblity within git
//		It helps keep a common structure amongst microservices
//		Helps with code navigation
//################################################################

var (
	userEncoder  = jingo.NewStructEncoder(data.User{})
	usersEncoder = jingo.NewSliceEncoder([]data.User{})
)

// getAllUsersHandler fetches all users from the DB
// this endpoint is designed to be performant. No queries used. And caching is utilised
func getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	var (
		err error
	)

	// The error defer pattern helps creates consistent logs for this handler.
	// And avoids important values being missed from logs.
	// See https://bet365techblog.com/better-error-handling-in-go
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%s\n%s", rec, debug.Stack())
		}

		if err != nil {
			log.Printf("getUserHandler >>> '%s', IP: %v, error: %v", r.URL.Path, r.RemoteAddr, err)
			// If this is a customer facing API, we dont really want to expose the errors.
			// This can lead to vulnerabilities, if the client knows what happened serverside.
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = fmt.Errorf("incorrect method %s", r.Method)
		return
	}

	users, err := db.GetUsers()
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")

	buf := jingo.NewBufferFromPool()
	defer buf.ReturnToPool()

	usersEncoder.Marshal(&users, buf)
	buf.WriteTo(w)
}

// getUsersHandler finds users with a given query from the database
// GET method is required
// parameters are to be supplied has url params.
// ?country=UK&nickname=meepo&createdAfter=2024-06-14T18:37:47.572Z&page=1&limit=50
// no params are required
func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	var (
		err error
	)

	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%s\n%s", rec, debug.Stack())
		}

		if err != nil {
			log.Printf("getUserHandler >>> '%s', IP: %v, error: %v", r.URL.Path, r.RemoteAddr, err)
			// If this is a customer facing API, we dont really want to expose the errors.
			// This can lead to vulnerabilities, if the client knows what happened serverside.
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = fmt.Errorf("incorrect method %s", r.Method)
		return
	}

	query := r.URL.Query()

	country := query.Get("country")
	nickname := query.Get("nickname")
	createdAfterStr := query.Get("createdAfter")
	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 || page > 1000 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 1
	}

	var createdAfter time.Time
	if createdAfterStr != "" {
		createdAfter, err = time.Parse(time.RFC3339, createdAfterStr)
		if err != nil {
			return
		}
	}

	users, err := db.GetUsersFiltered(country, nickname, createdAfter, page, limit)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")

	buf := jingo.NewBufferFromPool()
	defer buf.ReturnToPool()

	usersEncoder.Marshal(&users, buf)
	buf.WriteTo(w)
}

// addUserHandler creates a new user in the database, ensuring no username clashes
// POST method is required
// The user object must be on the post body in the standard user json format
func addUserHandler(w http.ResponseWriter, r *http.Request) {
	var (
		err error
	)

	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%s\n%s", rec, debug.Stack())
		}

		if err != nil {
			log.Printf("addUserHandler >>> '%s', IP: %v, error: %v", r.URL.Path, r.RemoteAddr, err)
			// If this is a customer facing API, we dont really want to expose the errors.
			// This can lead to vulnerabilities, if the client knows what happened serverside.
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		err = fmt.Errorf("incorrect method %s", r.Method)
		return
	}

	var user data.User
	if err = json.NewDecoder(r.Body).Decode(&user); err != nil {
		err = fmt.Errorf("error when decoding json body - err: %v", err)
		return
	}

	err = validation.User(user.FirstName, user.LastName, user.Nickname, user.Password, user.Country, user.Email)
	if err != nil {
		err = fmt.Errorf("user failed validation - err: %v, user:%+v", err, user)
		return
	}

	existingUser, err := db.GetUser(user.Nickname)
	if err != nil {
		err = fmt.Errorf("errored when attempting to lookup existing users - err: %v, user:%+v", err, user)
		return
	}

	if existingUser != nil && existingUser.Nickname != "" && existingUser.Nickname == user.Nickname {
		err = fmt.Errorf("a user with this username already exists - user:%+v", user)
		return
	}

	user.ID = newUUID()
	user.CreatedAt = timeNow().UTC()
	user.UpdatedAt = user.CreatedAt

	err = db.InsertUser(&user)
	if err != nil {
		return
	}

	// Spawn a go routine, so we dont impact the request
	go func() {
		userService.NotifyUpdate(user.ID, updateCREATED, convertToProtoUser(&user))
	}()

	w.Header().Set("Content-Type", "application/json")

	buf := jingo.NewBufferFromPool()
	defer buf.ReturnToPool()

	userEncoder.Marshal(&user, buf)
	buf.WriteTo(w)
}

// updateUserHandler updates the user from the database with a given id, ensuring no username clashes
// POST method is required
// The user object must be on the post body in the standard user json format
func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	var (
		err error
	)

	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%s\n%s", rec, debug.Stack())
		}

		if err != nil {
			log.Printf("updateUserHandler >>> '%s', IP: %v, error: %v", r.URL.Path, r.RemoteAddr, err)
			// If this is a customer facing API, we dont really want to expose the errors.
			// This can lead to vulnerabilities, if the client knows what happened serverside.
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		err = fmt.Errorf("incorrect method %s", r.Method)
		return
	}

	var user data.User
	if err = json.NewDecoder(r.Body).Decode(&user); err != nil {
		err = fmt.Errorf("error when decoding json body - err: %v", err)
		return
	}

	err = validation.User(user.FirstName, user.LastName, user.Nickname, user.Password, user.Country, user.Email)
	if err != nil {
		err = fmt.Errorf("user failed validation - err: %v, user:%+v", err, user)
		return
	}

	// ensure we have a correctly formatted uuid string
	err = uuid.Validate(user.ID)
	if err != nil {
		return
	}

	// Look up for users with this username, We want to prevent usernames being updated to usernames that already exist
	existingUser, err := db.GetUser(user.Nickname)
	if err != nil {
		err = fmt.Errorf("errored when attempting to lookup existing users - err: %v, user:%+v", err, user)
		return
	}

	// If the username matches an existing users, check we havn't matched with ourself
	if existingUser != nil && existingUser.ID != user.ID {
		err = fmt.Errorf("no users found with the given users, cannot update user, inputted user: %v", user)
		return
	}

	// Set the UpdatedAt field
	user.UpdatedAt = timeNow()

	updatedUser, err := db.UpdateUser(&user)
	if err != nil {
		return
	}

	// Spawn a go routine, so we dont impact the request
	go func() {
		userService.NotifyUpdate(user.ID, updateUPDATED, convertToProtoUser(updatedUser))
	}()

	w.Header().Set("Content-Type", "application/json")

	buf := jingo.NewBufferFromPool()
	defer buf.ReturnToPool()

	userEncoder.Marshal(updatedUser, buf)
	buf.WriteTo(w)
}

// deleteUserHandler deletes the user from the database with a given id
// POST method is required
// The ID must be provided on the post body in the standard user json format
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	var (
		err error
	)

	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%s\n%s", rec, debug.Stack())
		}

		if err != nil {
			log.Printf("deleteUserHandler >>> '%s', IP: %v, error: %v", r.URL.Path, r.RemoteAddr, err)
			// If this is a customer facing API, we dont really want to expose the errors.
			// This can lead to vulnerabilities, if the client knows what happened serverside.
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		err = fmt.Errorf("incorrect method %s", r.Method)
		return
	}

	var user pb.User
	if err = json.NewDecoder(r.Body).Decode(&user); err != nil {
		err = fmt.Errorf("error when decoding json body - err: %v", err)
		return
	}

	if user.ID == "" {
		err = fmt.Errorf("no userid provided to delete, user: %v", &user)
		return
	}

	// ensure we have a correctly formatted uuid string
	err = uuid.Validate(user.ID)
	if err != nil {
		return
	}

	err = db.DeleteUser(user.ID)
	if err != nil {
		return
	}

	// Spawn a go routine, so we dont impact the request
	go func() {
		userService.NotifyUpdate(user.ID, updateDELETED, &user)
	}()

	w.WriteHeader(http.StatusOK)
}

// deleteAllUsersHandler deletes the user from the database with a given id
// POST method is required
// The ID must be provided on the post body in the standard user json format
func deleteAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	var (
		err error
	)

	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("%s\n%s", rec, debug.Stack())
		}

		if err != nil {
			log.Printf("deleteUserHandler >>> '%s', IP: %v, error: %v", r.URL.Path, r.RemoteAddr, err)
			// If this is a customer facing API, we dont really want to expose the errors.
			// This can lead to vulnerabilities, if the client knows what happened serverside.
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		err = fmt.Errorf("incorrect method %s", r.Method)
		return
	}

	err = db.DeleteAllUsers()
	if err != nil {
		return
	}

	// Spawn a go routine, so we dont impact the request
	go func() {
		userService.NotifyUpdate("", updateALLDELETED, nil)
	}()

	w.WriteHeader(http.StatusOK)
}

//################################################################
// gRPC Handlers
//################################################################

// UserService contains our handlers for our gRPC service
type UserService struct {
	pb.UnimplementedUserServiceServer

	userUpdates chan *pb.UserUpdate
	mu          sync.RWMutex
	watchers    map[chan *pb.UserUpdate]struct{}
}

// NewUserService creates a new gRPC user server instance
func NewUserService() *UserService {
	return &UserService{
		userUpdates: make(chan *pb.UserUpdate),
		watchers:    make(map[chan *pb.UserUpdate]struct{}),
	}
}

// GetAllUsers fetches all users from the DB
// this endpoint is designed to be performant. No queries used. And caching is utilised
func (s *UserService) GetAllUsers(ctx context.Context, in *emptypb.Empty) (*pb.GetUsersResponse, error) {
	users, err := db.GetUsers()
	if err != nil {
		return nil, err
	}

	// Due to the way the users are stored in the database, we cant directly take protoUser out. (Dates....)
	protoUsers := make([]*pb.User, len(users))
	for i := range users {
		protoUsers[i] = convertToProtoUser(&users[i])
	}

	return &pb.GetUsersResponse{Users: protoUsers}, nil
}

// GetUsers finds users with a given query from the database
func (s *UserService) GetUsers(ctx context.Context, req *pb.GetUsersRequest) (*pb.GetUsersResponse, error) {

	if req.Page < 1 || req.Page > 1000 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 50 {
		req.Page = 1
	}

	users, err := db.GetUsersFiltered(req.Country, req.Nickname, req.CreatedAfter.AsTime(), int(req.Page), int(req.Limit))
	if err != nil {
		return nil, err
	}

	// Due to the way the users are stored in the database, we cant directly take protoUser out. (Dates....)
	protoUsers := make([]*pb.User, len(users))
	for i := range users {
		protoUsers[i] = convertToProtoUser(&users[i])
	}

	return &pb.GetUsersResponse{Users: protoUsers}, nil
}

// AddUser creates a new user in the database, ensuring no username clashes
func (s *UserService) AddUser(ctx context.Context, req *pb.AddUserRequest) (*pb.User, error) {
	err := validation.User(req.FirstName, req.LastName, req.Nickname, req.Password, req.Country, req.Email)
	if err != nil {
		err = fmt.Errorf("user failed validation - err: %v, user:%+v", err, req)
		return nil, err
	}

	existingUser, err := db.GetUser(req.Nickname)
	if err != nil {
		err = fmt.Errorf("errored when attempting to lookup existing users - err: %v, user:%+v", err, req)
		return nil, err
	}

	if existingUser != nil && existingUser.Nickname != "" && existingUser.Nickname == req.Nickname {
		err = fmt.Errorf("a user with this username already exists - user:%+v", req)
		return nil, err
	}

	user := data.User{
		ID:        newUUID(),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Nickname:  req.Nickname,
		Password:  req.Password,
		Country:   req.Country,
		Email:     req.Email,
		CreatedAt: timeNow().UTC(),
	}
	user.UpdatedAt = user.CreatedAt

	err = db.InsertUser(&user)
	if err != nil {
		return nil, err
	}

	// Spawn a go routine, so we dont impact the request
	go func() {
		s.NotifyUpdate(user.ID, updateCREATED, convertToProtoUser(&user))
	}()

	return convertToProtoUser(&user), nil
}

// UpdateUser updates the user from the database with a given id, ensuring no username clashes
func (s *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	err := validation.User(req.FirstName, req.LastName, req.Nickname, req.Password, req.Country, req.Email)
	if err != nil {
		err = fmt.Errorf("user failed validation - err: %v, user:%+v", err, req)
		return nil, err
	}

	// Look up for users with this username, We want to prevent usernames being updated to usernames that already exist
	existingUser, err := db.GetUser(req.Nickname)
	if err != nil {
		err = fmt.Errorf("errored when attempting to lookup existing users - err: %v, user:%+v", err, req)
		return nil, err
	}

	// If the username matches an existing users, check we havn't matched with ourself
	if existingUser != nil && existingUser.ID != req.ID {
		err = fmt.Errorf("no users found with the given users, cannot update user, inputted user: %v", req)
		return nil, err
	}

	// ensure we have a correctly formatted uuid string
	err = uuid.Validate(req.ID)
	if err != nil {
		return nil, err
	}

	user := data.User{
		ID:        req.ID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Nickname:  req.Nickname,
		Password:  req.Password,
		Country:   req.Country,
		Email:     req.Email,
	}

	// Set the UpdatedAt field
	user.UpdatedAt = timeNow()

	updatedUser, err := db.UpdateUser(&user)
	if err != nil {
		return nil, err
	}

	// Spawn a go routine, so we dont impact the request
	go func() {
		s.NotifyUpdate(user.ID, updateUPDATED, convertToProtoUser(updatedUser))
	}()

	return convertToProtoUser(updatedUser), nil
}

// DeleteUser deletes the user from the database with a given id
func (s *UserService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.Empty, error) {
	if req.ID == "" {
		err := fmt.Errorf("no userid provided to delete, user: %v", &req)
		return nil, err
	}

	// ensure we have a correctly formatted uuid string
	err := uuid.Validate(req.ID)
	if err != nil {
		return nil, err
	}

	err = db.DeleteUser(req.ID)
	if err != nil {
		return nil, err
	}

	// Spawn a go routine, so we dont impact the request
	go func() {
		s.NotifyUpdate(req.ID, updateDELETED, &pb.User{ID: req.ID})
	}()

	return nil, nil
}

// WatchUsers is the gRPC user update watcher, which notifies any watchers of updates to users
func (s *UserService) WatchUsers(req *pb.WatchRequest, stream pb.UserService_WatchUsersServer) error {
	// Create a personal chan for the connected watcher
	updateChan := make(chan *pb.UserUpdate)
	s.mu.Lock()

	// Add to our directory of watchers, so we can notify them all
	s.watchers[updateChan] = struct{}{}
	s.mu.Unlock()

	defer func() {
		// Clean up
		s.mu.Lock()
		delete(s.watchers, updateChan)
		s.mu.Unlock()
		close(updateChan)
	}()

	// Listen and distribute updates
	for update := range updateChan {
		if err := stream.Send(update); err != nil {
			return err
		}
	}

	return nil
}

const (
	updateCREATED    = "CREATED"
	updateDELETED    = "DELETED"
	updateUPDATED    = "UPDATED"
	updateALLDELETED = "ALL_DELETED"
)

// NotifyUpdate will spawn and notify all our watchers of an update
func (s *UserService) NotifyUpdate(userID, updateType string, user *pb.User) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.watchers {
		select {
		case ch <- &pb.UserUpdate{UserId: userID, UpdateType: updateType, User: user}:
		default:
			log.Printf("Dropping update for user %s: channel is full", userID)
		}
	}
}

// Convert a data.User to a protobuf User.
func convertToProtoUser(user *data.User) *pb.User {
	return &pb.User{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Nickname:  user.Nickname,
		Password:  user.Password,
		Email:     user.Email,
		Country:   user.Country,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}
