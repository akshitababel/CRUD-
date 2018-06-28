package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	gohttp "net/http"
	"os"

	files "github.com/ipfs/go-ipfs-cmdkit/files"
)

var path = "C:/Users/Documents/temp/test.txt"

type RadioButton struct {
	Name       string
	Value      string
	IsDisabled bool
	IsChecked  bool
	Text       string
}

type PageVariables struct {
	PageTitle        string
	PageRadioButtons []RadioButton
	Answer           string
}

func main() {
	gohttp.HandleFunc("/index", DisplayRadioButtons)
	gohttp.HandleFunc("/selected", UserSelected)
	log.Fatal(gohttp.ListenAndServe(":3000", nil))
}

func DisplayRadioButtons(w gohttp.ResponseWriter, r *gohttp.Request) {
	// Display some radio buttons to the user

	Title := "Which do you prefer?"
	MyRadioButtons := []RadioButton{
		RadioButton{"option", "create", false, false, "Create"},
		RadioButton{"option", "insert", false, false, "Insert"},
		RadioButton{"option", "read", false, false, "Read"},
		RadioButton{"option", "delete", false, false, "Delete"},
	}

	MyPageVariables := PageVariables{
		PageTitle:        Title,
		PageRadioButtons: MyRadioButtons,
	}

	t, err := template.ParseFiles("select.html") //parse the html file homepage.html
	if err != nil {                              // if there is an error
		log.Print("template parsing error: ", err) // log it
	}

	err = t.Execute(w, MyPageVariables) //execute the template and pass it the HomePageVars struct to fill in the gaps
	if err != nil {                     // if there is an error
		log.Print("template executing error: ", err) //log it
	}

}

func UserSelected(w gohttp.ResponseWriter, r *gohttp.Request) {
	r.ParseForm()

	yourchoice := r.Form.Get("option")

	Title := "Your preferred choice"
	MyPageVariables := PageVariables{
		PageTitle: Title,
		Answer:    yourchoice,
	}

	t, err := template.ParseFiles("select.html")
	if err != nil {
		log.Print("template parsing error: ", err)
	}
	switch MyPageVariables.Answer {
	case "create":
		createFile()

	case "insert":
		writeFile()

	case "read":
		readFile()

	case "delete":
		deleteFile()
	}
	err = t.Execute(w, MyPageVariables)
	if err != nil {
		log.Print("template executing error: ", err)
	}

}
func createFile() {

	var _, err = os.Stat(path)

	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		if isError(err) {
			return
		}
		defer file.Close()
	}

	fmt.Println("==> done creating file", path)
	r := io.Reader(file)
	Add(r)
}

func writeFile() {

	var file, err = os.OpenFile(path, os.O_RDWR, 0644)
	if isError(err) {
		return
	}
	defer file.Close()

	_, err = file.WriteString("hello\n")
	if isError(err) {
		return
	}
	_, err = file.WriteString("golang is fun\n")
	if isError(err) {
		return
	}

	err = file.Sync()
	if isError(err) {
		return
	}

	fmt.Println("==> done writing to file")
	r := io.Reader(file)
	Add(r)
}

func readFile() {

	var file, err = os.OpenFile(path, os.O_RDWR, 0644)
	if isError(err) {
		return
	}
	defer file.Close()

	var text = make([]byte, 1024)
	for {
		_, err = file.Read(text)
		if err == io.EOF {
			break
		}

		if err != nil && err != io.EOF {
			isError(err)
			break
		}
	}

	fmt.Println("==> done reading from file")
	fmt.Println(string(text))
}

func deleteFile() {

	var err = os.Remove(path)
	if isError(err) {
		return
	}

	fmt.Println("==> done deleting file")
}

func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}

	return (err != nil)
}

type Shell struct {
	url     string
	httpcli *gohttp.Client
}

func (s *Shell) Add(r io.Reader) (string, error) {
	return s.AddWithOpts(r, true, false)
}

func (s *Shell) AddWithOpts(r io.Reader, pin bool, rawLeaves bool) (string, error) {
	var rc io.ReadCloser
	if rclose, ok := r.(io.ReadCloser); ok {
		rc = rclose
	} else {
		rc = ioutil.NopCloser(r)
	}

	// handler expects an array of files
	fr := files.NewReaderFile("", "", rc, nil)
	slf := files.NewSliceFile("", "", []files.File{fr})
	fileReader := files.NewMultiFileReader(slf, true)

	req := gohttp.NewRequest(context.Background(), s.url, "add")
	req.Body = fileReader
	req.Opts["progress"] = "false"
	if !pin {
		req.Opts["pin"] = "false"
	}

	if rawLeaves {
		req.Opts["raw-leaves"] = "true"
	}

	resp, err := req.Send(s.httpcli)
	if err != nil {
		return "", err
	}
	defer resp.Close()
	if resp.Error != nil {
		return "", resp.Error
	}

	var out object
	err = json.NewDecoder(resp.Output).Decode(&out)
	if err != nil {
		return "", err
	}

	return out.Hash, nil
}

type object struct {
	Hash string
}
