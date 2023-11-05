package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"path"
)

//uploaderHandler uses the FormValue method in http.Request to get the
//user ID that we placed in the hidden input in our HTML form. Then, it gets an io.Reader
//type capable of reading the uploaded bytes by calling req.FormFile, which returns three
//arguments. The first argument represents the file itself with the multipart.File interface
//type, which is also io.Reader. The second is a multipart.FileHeader object that
//contains the metadata about the file, such as the filename. And finally, the third argument is
//an error that we hope will have a nil value
func uploaderHandler(w http.ResponseWriter, req *http.Request) {
	userId := req.FormValue("userid")
	file, header, err := req.FormFile("avatarFile")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filename := path.Join("avatars", userId+path.Ext(header.Filename))
	err = ioutil.WriteFile(filename, data, 0777)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, "Successful")
}
