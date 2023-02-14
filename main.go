package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"personal-web/connection"
	"personal-web/middlewares"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var Data = map[string]interface{}{
	"Title":   "Fauzan | Personal Web",
	"IsLogin": false,
}

type User struct {
	Id       int
	Name     string
	Email    string
	Password string
}

type Project struct {
	Id           int
	ProjectName  string
	StartDate    time.Time
	EndDate      time.Time
	DurationText string
	Description  string
	Technologies []string
	Image        string
	Author       string
}

func main() {
	route := mux.NewRouter()

	connection.DatabaseConnect()

	route.PathPrefix("/public/").Handler(http.StripPrefix("/public/", http.FileServer(http.Dir("./public/"))))
	route.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads/"))))

	route.HandleFunc("/", Home).Methods("GET")
	route.HandleFunc("/contact-me", ContactMe).Methods("GET")
	route.HandleFunc("/add-project-page", AddProjectPage).Methods("GET")
	route.HandleFunc("/projects/{id}", ProjectDetails).Methods("GET")
	route.HandleFunc("/add-new-project", middlewares.UploadFile(AddNewProject)).Methods("POST")
	route.HandleFunc("/delete-project/{id}", DeleteProject).Methods("GET")
	route.HandleFunc("/update-project-page/{id}", UpdateProjectPage).Methods("GET")
	route.HandleFunc("/update-project/{id}", middlewares.UploadFile(UpdateProject)).Methods("POST")
	route.HandleFunc("/register", RegisterPage).Methods("GET")
	route.HandleFunc("/login", LoginPage).Methods("GET")
	route.HandleFunc("/register", Register).Methods("POST")
	route.HandleFunc("/login", Login).Methods("POST")
	route.HandleFunc("/logout", Logout).Methods("GET")

	fmt.Println("Server running on port 5000")
	http.ListenAndServe("localhost:5000", route)
}

func Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var tmpl, err = template.ParseFiles("views/index.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	if session.Values["IsLogin"] != true {
		Data["IsLogin"] = false
	} else {
		Data["IsLogin"] = session.Values["IsLogin"].(bool)
		Data["Username"] = session.Values["Name"].(string)
	}

	fm := session.Flashes("message")

	var flashes []string

	if len(fm) > 0 {
		session.Save(r, w)
		for _, fl := range fm {
			flashes = append(flashes, fl.(string))
		}
	}

	Data["FlashData"] = strings.Join(flashes, "")

	var rows pgx.Rows
	if Data["IsLogin"] == true {
		rows, _ = connection.Conn.Query(context.Background(), "SELECT tb_projects.id, project_name, start_date, end_date, description, technologies, image, tb_user.name as author FROM tb_projects LEFT JOIN tb_user ON tb_projects.author_id = tb_user.id WHERE tb_user.name=$1 ORDER BY end_date DESC", Data["Username"])
	} else {
		rows, _ = connection.Conn.Query(context.Background(), "SELECT tb_projects.id, project_name, start_date, end_date, description, technologies, image, tb_user.name as author FROM tb_projects LEFT JOIN tb_user ON tb_projects.author_id = tb_user.id  ORDER BY end_date DESC")
	}

	var Projects []Project
	for rows.Next() {
		var Project = Project{}

		var err = rows.Scan(&Project.Id, &Project.ProjectName, &Project.StartDate, &Project.EndDate, &Project.Description, &Project.Technologies, &Project.Image, &Project.Author)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		Project.DurationText = CalculateDuration(Project.StartDate, Project.EndDate)

		Projects = append(Projects, Project)
	}

	respData := map[string]interface{}{
		"Data":     Data,
		"Projects": Projects,
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, respData)
}

func ContactMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var tmpl, err = template.ParseFiles("views/contact-me.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, Data)
}

func RegisterPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var tmpl, err = template.ParseFiles("views/register.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, Data)
}

func LoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var tmpl, err = template.ParseFiles("views/login.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, Data)
}

func Register(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	name := r.PostForm.Get("name")
	email := r.PostForm.Get("email")

	password := r.PostForm.Get("password")
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)

	_, err = connection.Conn.Exec(context.Background(), "INSERT INTO tb_user(name, email, password) VALUES ($1,$2,$3)", name, email, passwordHash)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	http.Redirect(w, r, "/login", http.StatusMovedPermanently)
}

func Login(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	email := r.PostForm.Get("email")
	password := r.PostForm.Get("password")

	user := User{}

	err = connection.Conn.QueryRow(context.Background(), "SELECT * FROM tb_user WHERE email=$1", email).Scan(
		&user.Id, &user.Name, &user.Email, &user.Password,
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	session.Values["IsLogin"] = true
	session.Values["Name"] = user.Name
	session.Values["Id"] = user.Id
	session.Options.MaxAge = 10800

	session.AddFlash("Login success", "message")
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-chace, no-store, must-revalidate")

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	session.Options.MaxAge = -1
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func AddProjectPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var tmpl, err = template.ParseFiles("views/add-project.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, Data)
}

func ProjectDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	var tmpl, err = template.ParseFiles("views/projects/project-details.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	ProjectDetails := Project{}
	err = connection.Conn.QueryRow(context.Background(), "SELECT * FROM tb_projects WHERE id=$1", id).Scan(
		&ProjectDetails.Id, &ProjectDetails.ProjectName, &ProjectDetails.StartDate, &ProjectDetails.EndDate, &ProjectDetails.Description, &ProjectDetails.Technologies, &ProjectDetails.Image, &ProjectDetails.Author)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	DurationText := CalculateDuration(ProjectDetails.StartDate, ProjectDetails.EndDate)
	StartDateString := ProjectDetails.StartDate.Format("Jan 2, 2006")
	EndDateString := ProjectDetails.EndDate.Format("Jan 2, 2006")

	resp := map[string]interface{}{
		"Data":            Data,
		"ProjectDetails":  ProjectDetails,
		"DurationText":    DurationText,
		"StartDateString": StartDateString,
		"EndDateString":   EndDateString,
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, resp)
}

func AddNewProject(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	ProjectName := r.PostForm.Get("project-name")
	StartDate := r.PostForm.Get("start-date")
	EndDate := r.PostForm.Get("end-date")
	Description := r.PostForm.Get("description")
	Technologies := r.Form["technology"]

	dataContex := r.Context().Value("dataFile")
	Image := dataContex.(string)

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	AuthorId := session.Values["Id"].(int)

	_, err = connection.Conn.Exec(context.Background(), "INSERT INTO tb_projects(project_name, start_date, end_date, description, technologies, image, author_id) VALUES ($1, $2, $3, $4, $5, $6, $7)", ProjectName, StartDate, EndDate, Description, Technologies, Image, AuthorId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func DeleteProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-chace, no-store, must-revalidate")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	_, err := connection.Conn.Exec(context.Background(), "DELETE FROM tb_projects WHERE id=$1", id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func UpdateProjectPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	var tmpl, err = template.ParseFiles("views/update-project.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	ProjectDetails := Project{}
	err = connection.Conn.QueryRow(context.Background(), "SELECT * FROM tb_projects WHERE id=$1", id).Scan(
		&ProjectDetails.Id, &ProjectDetails.ProjectName, &ProjectDetails.StartDate, &ProjectDetails.EndDate, &ProjectDetails.Description, &ProjectDetails.Technologies, &ProjectDetails.Image, &ProjectDetails.Author)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	var Node, React, Angular, Vuejs bool
	for _, technology := range ProjectDetails.Technologies {
		if technology == "node" {
			Node = true
		}
		if technology == "react" {
			React = true
		}
		if technology == "angular" {
			Angular = true
		}
		if technology == "vuejs" {
			Vuejs = true
		}
	}

	StartDateString := ProjectDetails.StartDate.Format("2006-01-02")
	EndDateString := ProjectDetails.EndDate.Format("2006-01-02")

	respData := map[string]interface{}{
		"Data":            Data,
		"Id":              id,
		"ProjectDetails":  ProjectDetails,
		"StartDateString": StartDateString,
		"EndDateString":   EndDateString,
		"Node":            Node,
		"React":           React,
		"Angular":         Angular,
		"Vuejs":           Vuejs,
	}

	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, respData)
}

func UpdateProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	ProjectName := r.PostForm.Get("project-name")
	StartDateString := r.PostForm.Get("start-date")
	EndDateString := r.PostForm.Get("end-date")
	Description := r.PostForm.Get("description")
	Technologies := r.Form["technology"]

	StartDate, _ := time.Parse("2006-01-02", StartDateString)
	EndDate, _ := time.Parse("2006-01-02", EndDateString)

	dataContex := r.Context().Value("dataFile")
	Image := dataContex.(string)

	var store = sessions.NewCookieStore([]byte("SESSION_ID"))
	session, _ := store.Get(r, "SESSION_ID")

	AuthorId := session.Values["Id"].(int)

	_, err = connection.Conn.Exec(context.Background(), "UPDATE tb_projects SET project_name=$1, start_date=$2, end_date=$3, description=$4, technologies=$5, image=$6, author_id=$7 WHERE id=$8;", ProjectName, StartDate, EndDate, Description, Technologies, Image, AuthorId, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("message : " + err.Error()))
		return
	}

	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}

func CalculateDuration(StartDate time.Time, EndDate time.Time) string {
	Duration := EndDate.Sub(StartDate)
	DurationHours := Duration.Hours()
	DurationDays := math.Floor(DurationHours / 24)
	DurationWeeks := math.Floor(DurationDays / 7)
	DurationMonths := math.Floor(DurationDays / 30)
	var DurationText string
	if DurationMonths > 1 {
		DurationText = strconv.FormatFloat(DurationMonths, 'f', 0, 64) + " months"
	} else if DurationMonths > 0 {
		DurationText = strconv.FormatFloat(DurationMonths, 'f', 0, 64) + " month"
	} else {
		if DurationWeeks > 1 {
			DurationText = strconv.FormatFloat(DurationWeeks, 'f', 0, 64) + " weeks"
		} else if DurationWeeks > 0 {
			DurationText = strconv.FormatFloat(DurationWeeks, 'f', 0, 64) + " week"
		} else {
			if DurationDays > 1 {
				DurationText = strconv.FormatFloat(DurationDays, 'f', 0, 64) + " days"
			} else if DurationDays > 0 {
				DurationText = strconv.FormatFloat(DurationDays, 'f', 0, 64) + " day"
			} else {
				DurationText = "less than a day"
			}
		}
	}
	return DurationText
}
