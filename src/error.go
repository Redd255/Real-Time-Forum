package myserver

import "net/http"

func errorPage(w http.ResponseWriter, message, templateName string) {
	data := map[string]string{
		"ErrorMessage": message,
	}
	err := templates.ExecuteTemplate(w, templateName, data)
	if err != nil {
		http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
	}
}
