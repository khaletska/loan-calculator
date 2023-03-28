package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

type errRes struct {
	ErrorCode int
	Message   string
}

type res struct {
	Desicion   bool
	Loan       int
	Period     int
	ErrReason  error
	InputError errRes
}

var modifiers = map[string]int{
	"49002010965": 0,
	"49002010976": 100,
	"49002010987": 300,
	"49002010998": 1000,
}

var ServerResp res

func main() {
	// Attaching css
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("static"))))

	// Handlers
	http.HandleFunc("/", renderMainPage)
	http.HandleFunc("/calculate-loan", calculateLoan)

	// Start server
	fmt.Println("Server started on the http://localhost:8080/")
	fmt.Println("Press Ctrl+C to stop the server")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// renderMainPage is a function that handles GET requests to the root URL ("/") and renders an HTML template.
func renderMainPage(w http.ResponseWriter, r *http.Request) {
	// check the URL
	if r.URL.Path != "/" {
		renderErrorPage(w, http.StatusNotFound)
		return
	}

	// check the method
	if r.Method != "GET" {
		renderErrorPage(w, http.StatusMethodNotAllowed)
		return
	}

	// Parse the HTML template file and handle any errors
	template, err := template.ParseFiles("templates/index.html")
	if err != nil {
		renderErrorPage(w, http.StatusInternalServerError)
		return
	}

	// Execute the template with a response and handle any errors
	var temp bytes.Buffer
	err = template.Execute(&temp, ServerResp)
	if err != nil {
		renderErrorPage(w, http.StatusInternalServerError)
		return
	}

	// Write the rendered HTML to the HTTP response
	w.Write(temp.Bytes())
}

func calculateLoan(w http.ResponseWriter, r *http.Request) {
	ServerResp = res{}

	if r.URL.Path != "/calculate-loan" {
		renderErrorPage(w, http.StatusNotFound)
		return
	}

	if r.Method != "POST" {
		renderErrorPage(w, http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	personalCode := r.FormValue("personal-code")
	creditModifier, ok := modifiers[personalCode]
	// check if the personal code is valid
	if !ok {
		ServerResp.InputError.ErrorCode = 400
		ServerResp.InputError.Message = "invalid personal code"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	loanAmount, err := strconv.Atoi(r.FormValue("loan-amount"))
	// check if the loan amount corresponds to the constraint
	if err != nil || !isAmountValid(loanAmount) {
		ServerResp.InputError.ErrorCode = 400
		ServerResp.InputError.Message = "please insert loan sum between €2000 and €10000"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// check if the loan period corresponds to the constraint
	loanPeriod, err := strconv.Atoi(r.FormValue("loan-period"))
	if err != nil || !isPeriodValid(loanPeriod) {
		ServerResp.InputError.ErrorCode = 400
		ServerResp.InputError.Message = "please insert loan period between 12 and 60 months"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	ServerResp.Desicion, ServerResp.Loan, ServerResp.Period, ServerResp.ErrReason = calculateLoanValue(creditModifier, loanAmount, loanPeriod)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func calculateLoanValue(creditModifier int, loanAmount int, loanPeriod int) (bool, int, int, error) {
	if creditModifier == 0 {
		return false, -1, 0, errors.New("you have debt")
	}

	maxLoanAmount := creditModifier * loanPeriod
	// IF we can give the client more than 10000 euro for desired period
	// THEN return 10000 euro and desired period
	if maxLoanAmount > 10000 {
		return true, 10000, loanPeriod, nil
	}

	// IF we can give the client more than he wants
	// OR less than ne wants, but more than 2000 euro
	// for the desired period
	// THEN return maximum that we can give for the desired period
	if maxLoanAmount >= loanAmount || maxLoanAmount >= 2000 {
		return true, maxLoanAmount, loanPeriod, nil
	} else {
		// ELSE we are trying to find suitable period for desired loan sum
		changedPeriod := loanPeriod
		for changedPeriod < 60 {
			changedPeriod++
			// recalculating maximum loan that we can give for the new period
			maxLoanAmount = creditModifier * changedPeriod
			// IF maximum loan amount for the increased period is more or equal to the desired sum
			// THEN return new maximum loan amount and increased period
			if maxLoanAmount >= loanAmount {
				return true, maxLoanAmount, changedPeriod, nil
			}
		}
		// IF we didn't find a suitable period for the desired sum, we are trying
		// to find a shortest period for which we can give 2000 euros
		changedPeriod = loanPeriod
		for changedPeriod < 60 {
			changedPeriod++
			maxLoanAmount = creditModifier * changedPeriod
			if maxLoanAmount >= 2000 {
				return true, maxLoanAmount, changedPeriod, nil
			}
		}
		return false, -1, 0, errors.New("your credit score is too low")
	}
}

// function for error page rendering
func renderErrorPage(w http.ResponseWriter, statusCode int) {
	template, err := template.ParseFiles("templates/error.html")
	if err != nil {
		http.Error(w, fmt.Sprint("Error parsing:", err), statusCode)
		return
	}

	res := errRes{
		ErrorCode: statusCode,
		Message:   http.StatusText(statusCode),
	}

	w.WriteHeader(statusCode)
	template.Execute(w, res)
}

// function to check if the loan amount corresponds to the constraint
func isAmountValid(loanAmount int) bool {
	if loanAmount < 2000 || loanAmount > 10000 {
		return false
	}
	return true
}

// function to check if the loan period corresponds to the constraint
func isPeriodValid(loanPeriod int) bool {
	if loanPeriod < 12 || loanPeriod > 60 {
		return false
	}
	return true
}
