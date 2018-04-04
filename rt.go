package main

import "reflect"
import "bufio"
import "syscall"
import "fmt"
import "net/http"
import "flag"
import "io/ioutil"
import "os"
import "os/user"
import "strings"
import "strconv"

// THANK YOU Joe Linoff @ STACKOVERFLOW
func getPassword(prompt string) string {
	fmt.Print(prompt)

	// Common settings and variables for both stty calls.
	attrs := syscall.ProcAttr{
		Dir:   "",
		Env:   []string{},
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
		Sys:   nil}
	var ws syscall.WaitStatus

	// Disable echoing.
	pid, err := syscall.ForkExec(
		"/bin/stty",
		[]string{"stty", "-echo"},
		&attrs)
	if err != nil {
		panic(err)
	}

	// Wait for the stty process to complete.
	_, err = syscall.Wait4(pid, &ws, 0, nil)
	if err != nil {
		panic(err)
	}

	// Echo is disabled, now grab the data.
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}

	// Re-enable echo.
	pid, err = syscall.ForkExec(
		"/bin/stty",
		[]string{"stty", "echo"},
		&attrs)
	if err != nil {
		panic(err)
	}

	// Wait for the stty process to complete.
	_, err = syscall.Wait4(pid, &ws, 0, nil)
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(text)
}
func queryHistory(settings map[string]string, tnumber string) []string {
	req, err := http.NewRequest("GET", "http://"+settings["endpoint"]+"/REST/1.0/ticket/"+tnumber+"/history?user="+settings["username"]+"&pass="+settings["password"], nil)
	if err != nil {
		fmt.Println("Error 1")
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	content := strings.Split(string(contents), "\n")
	content = append(content[:0], content[1:]...)
	return append(content[:0], content[1:]...)
}

func queryTicket(settings map[string]string, ticket string) map[string]string {
	req, err := http.NewRequest("GET", "http://"+settings["endpoint"]+"/REST/1.0/ticket/"+ticket+"/show?user="+settings["username"]+"&pass="+settings["password"], nil)
	if err != nil {
		fmt.Println("Error 1")
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	content := strings.Split(string(contents), "\n")
	cont := map[string]string{}
	for _, element := range content {
		nkey := strings.Trim(strings.Split(element, ":")[0], " ")
		ndata := strings.Split(element, ":")
		ndata = append(ndata[:0], ndata[1:]...)
		data := strings.Trim(string(strings.Join(ndata, ":")), " ")
		if nkey != "" {
			cont[nkey] = data
		}
	}
	return cont
}

func getTicket(settings map[string]string, tnumber string) map[string]string {
	// IS THE TICKET NUMBER NUMERIC?
	if _, err := strconv.Atoi(tnumber); err != nil {
		fmt.Println("Ticket Number is not numeric!\nPlease include a valid ticket number. rt -t 1234567")
		os.Exit(0)
	}
	if t, _ := strconv.Atoi(tnumber); t < 0 {
		fmt.Println(t, "is not valid. Please include a valid ticket number. rt -t 1234567")
		os.Exit(0)
	}

	// LETS GET THE CURRENT TICKET INFORMATION MAPPED
	ticket := queryTicket(settings, tnumber)

	// WAS THAT A VALID TICKET? (split the key for line 2 on phrase does not, if #ofElements >1 error)
	ec := string(reflect.ValueOf(ticket).MapKeys()[1].Interface().(string))
	if len(strings.SplitAfter(ec, " does not ")) > 1 {
		fmt.Println("Ticket number", tnumber, "does not exist.")
		os.Exit(0)
	}
	if len(strings.SplitAfter(ec, "username or password")) > 1 {
		fmt.Println("Username/Password Incorrect.")
		os.Exit(0)
	}
	return ticket
}

func showTicket(settings map[string]string, tnumber string, comments bool, history bool, sum bool, links bool, attachments bool) {
	if sum {
		showSummary(settings, tnumber)
		fmt.Println("")
	}
	if comments {
		showComments(settings, tnumber, queryHistory(settings, tnumber))
	}
	if history {
		for _, element := range queryHistory(settings, tnumber) {
			fmt.Println(element)
		}
	}
	if links {
		showLinks(settings, tnumber)
		fmt.Println("")
	}
	if attachments {
		showAttachments(settings, tnumber)
	}
	if !attachments && !links && !history && !comments && !sum {
		showSummary(settings, tnumber)
	}
	os.Exit(0)
}

func showSummary(settings map[string]string, tnumber string) {
	ticket := getTicket(settings, tnumber)
	if settings["summaryFields"] == "" {
		fmt.Println("summaryFields is not defined in config, -fields was not set.\n  - Dumping Ticket:")
		//dump all content
		for index, element := range ticket {
			if element != "" {
				fmt.Println(index, ": ", element)
			}
		}
	} else {
		fields := strings.Split(settings["summaryFields"], ",")
		for _, f := range fields {
			if ticket[f] != "" {
				fmt.Print(f, ": ", ticket[f], "\n")
			}
		}
	}

}

func showComments(settings map[string]string, tnumber string, history []string) {
	commentids := ""
	for _, e := range history {
		test := strings.SplitAfter(e, "Comments added")
		if len(test) > 1 {
			commentids += strings.SplitAfter(e, ":")[0]
		}
		test = strings.SplitAfter(e, "created by")
		if len(test) > 1 {
			commentids += strings.SplitAfter(e, ":")[0]
		}
	}
	cid := strings.Split(commentids, ":")
	fmt.Println("\nThere are", (len(cid) - 1), "comments:")
	for _, id := range cid {
		if id != "" {
			req2, err2 := http.NewRequest("GET", "http://"+settings["endpoint"]+"/REST/1.0/ticket/"+tnumber+"/history/id/"+id+"?user="+settings["username"]+"&pass="+settings["password"], nil)
			if err2 != nil {
				fmt.Printf("%s", err2)
				os.Exit(1)
			}
			resp2, err2 := http.DefaultClient.Do(req2)
			if err2 != nil {
				fmt.Printf("%s", err2)
				os.Exit(1)
			}
			defer resp2.Body.Close()
			contents2, err2 := ioutil.ReadAll(resp2.Body)
			if err2 != nil {
				fmt.Printf("%s", err2)
				os.Exit(1)
			}
			fmt.Println("-----------------------------------------------------")
			fmt.Println(string(contents2))
			fmt.Println("-----------------------------------------------------")
		}
	}
}

func showLinks(settings map[string]string, tnumber string) {
	req, err := http.NewRequest("GET", "http://"+settings["endpoint"]+"/REST/1.0/ticket/"+tnumber+"/links/show?user="+settings["username"]+"&pass="+settings["password"], nil)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	contents := strings.Split(string(content), "\n")
	contents = append(contents[:0], contents[3:]...)
	for _, l := range contents {
		if l != "" {
			linktype := strings.Split(l, ":")[0]
			tktnumadj := strings.Split(l, "/")
			tktnum := tktnumadj[len(tktnumadj)-1]
			tkt := getTicket(settings, tktnum)
			fmt.Print(linktype, ": ", tktnum, " ", tkt["Subject"], "\n")
		}
	}
}

func showAttachments(settings map[string]string, tnumber string) {
	req, err := http.NewRequest("GET", "http://"+settings["endpoint"]+"/REST/1.0/ticket/"+tnumber+"/attachments?user="+settings["username"]+"&pass="+settings["password"], nil)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	contents := strings.Split(string(content), "\n")
	contents = append(contents[:0], contents[4:]...)
	for _, l := range contents {
		fmt.Println(strings.Trim(strings.Replace(l, "Attachments:", "", 1), " "))
	}
}

func updateTicket(settings map[string]string, tnumber string) {
	fmt.Println("Will update ", tnumber)
}

func showHelp() {
	//HELP MENU
	fmt.Println("Usage: rt <subcommand> [globaloptions] [options]\n")
	fmt.Println("  Sub-commands (rt <subcommand> -h)")
	fmt.Println("    GET:    Pull Individual Ticket Data")
	fmt.Println("    UPDATE: Update Individual Ticket Data")
	fmt.Println("    CREATE: Create a Ticket")
	fmt.Println("    SEARCH: Search for tickets with given criteria")
	fmt.Println("\n  Global Options")
	fmt.Println("    -config [string]: Alt config file (~/.rt.d/config)")
	fmt.Println("    -user [string]:   Specify username for Endpoint")
	fmt.Println("    -e [string]:      Endpoint (ie: my.rtserver.com)")
	fmt.Println("")
	os.Exit(0)
}

func showGetHelp() {
	fmt.Println("Usage: rt get [globaloptions] -t <ticketnum> [options]\n")
	fmt.Println("  Global Options")
	fmt.Println("    -config [string]: Alt config file (~/.rt.d/config)")
	fmt.Println("    -user [string]:   Specify username for Endpoint")
	fmt.Println("    -e [string]:      Endpoint (ie: my.rtserver.com)")
	fmt.Println("\n  Options")
	fmt.Println("    -t: Ticket Number (required)")
	fmt.Println("    -p: Show Ticket History")
	fmt.Println("    -c: Show Ticket Comments")
	fmt.Println("    -a: List Ticket Attachments")
	fmt.Println("    -l: List Related Tickets")
	fmt.Println("    -s: Show Ticket Summary (default)")
	fmt.Println("")
	os.Exit(0)
}

func showUpdateHelp() {
	fmt.Println("Usage: rt update [globaloptions] [options]\n")
	fmt.Println("  Global Options")
	fmt.Println("    -config [string]: Alt config file (~/.rt.d/config)")
	fmt.Println("    -user [string]:   Specify username for Endpoint")
	fmt.Println("    -e [string]:      Endpoint (ie: my.rtserver.com)")
	fmt.Println("")

	os.Exit(0)
}

func showCreateHelp() {
	fmt.Println("Usage: rt create [globaloptions] [options]\n")
	fmt.Println("  Global Options")
	fmt.Println("    -config [string]: Alt config file (~/.rt.d/config)")
	fmt.Println("    -user [string]:   Specify username for Endpoint")
	fmt.Println("    -e [string]:      Endpoint (ie: my.rtserver.com)")
	fmt.Println("")

	os.Exit(0)
}

func showSearchHelp() {
	fmt.Println("Usage: rt search [globaloptions] [options]\n")
	fmt.Println("  Global Options")
	fmt.Println("    -config [string]: Alt config file (~/.rt.d/config)")
	fmt.Println("    -user [string]:   Specify username for Endpoint")
	fmt.Println("    -e [string]:      Endpoint (ie: my.rtserver.com)")
	fmt.Println("")

	os.Exit(0)
}

func main() {
	//CHECK FOR AT LEAST ONE ARGUMENT
	if len(os.Args) == 1 {
		showHelp()
	}

	//SETUP VARIABLES
	settings := map[string]string{}
	settings["username"] = ""
	settings["password"] = ""
	settings["endpoint"] = ""
	settings["summaryFields"] = ""
	user, _ := user.Current()
	eFunc := ""
	cFunc := ""
	uFunc := ""

	//SETUP FLAGS
	getCommand := flag.NewFlagSet("get", flag.ExitOnError)
	getComments := getCommand.Bool("c", false, "Show Comments")
	getHistory := getCommand.Bool("p", false, "Show History")
	getSummary := getCommand.Bool("s", false, "Show Summary")
	getAttach := getCommand.Bool("a", false, "Show Attachment List")
	getLinks := getCommand.Bool("l", false, "Show Linked Tickets")
	getHelp := getCommand.Bool("h", false, "Show Help Menu")
	gettNum := getCommand.String("t", "-1", "Ticket Number")
	getFields := getCommand.String("fields", "", "List of custom fields to show in summary, sep ','")
	geteFunc := getCommand.String("e", "", "RT Server Endpoint")
	getcFunc := getCommand.String("config", user.HomeDir+"/.rt.d/config", "Specify Alternate Config File")
	getuFunc := getCommand.String("user", "", "Specify Username")

	updCommand := flag.NewFlagSet("update", flag.ExitOnError)
	updHelp := updCommand.Bool("h", false, "Show Help Menu")
	updtNum := updCommand.String("t", "-1", "Ticket Number")
	updeFunc := updCommand.String("e", "", "RT Server Endpoint")
	updcFunc := updCommand.String("config", user.HomeDir+"/.rt.d/config", "Specify Alternate Config File")
	upduFunc := updCommand.String("user", "", "Specify Username")

	searchCommand := flag.NewFlagSet("search", flag.ExitOnError)
	searchHelp := searchCommand.Bool("h", false, "Show Help Menu")
	searcheFunc := searchCommand.String("e", "", "RT Server Endpoint")
	searchcFunc := searchCommand.String("config", user.HomeDir+"/.rt.d/config", "Specify Alternate Config File")
	searchuFunc := searchCommand.String("user", "", "Specify Username")

	createCommand := flag.NewFlagSet("create", flag.ExitOnError)
	createHelp := createCommand.Bool("h", false, "Show Help Menu")
	createeFunc := createCommand.String("e", "", "RT Server Endpoint")
	createcFunc := createCommand.String("config", user.HomeDir+"/.rt.d/config", "Specify Alternate Config File")
	createuFunc := createCommand.String("user", "", "Specify Username")

	//PARSE FLAGS BASED ON SUBCOMMAND
	switch os.Args[1] {
	case "update":
		updCommand.Parse(os.Args[2:])
		eFunc = *updeFunc
		cFunc = *updcFunc
		uFunc = *upduFunc
	case "get":
		getCommand.Parse(os.Args[2:])
		eFunc = *geteFunc
		cFunc = *getcFunc
		uFunc = *getuFunc
	case "search":
		searchCommand.Parse(os.Args[2:])
		eFunc = *searcheFunc
		cFunc = *searchcFunc
		uFunc = *searchuFunc
	case "create":
		createCommand.Parse(os.Args[2:])
		eFunc = *createeFunc
		cFunc = *createcFunc
		uFunc = *createuFunc
	default:
		showHelp()
	}
	//CHECK FOR INDEPENDENT HELP REQUESTS
	if *getHelp {
		showGetHelp()
	}
	if *updHelp {
		showUpdateHelp()
	}
	if *searchHelp {
		showSearchHelp()
	}
	if *createHelp {
		showCreateHelp()
	}

	//SETUP CONFIGS
	if _, err := os.Stat(cFunc); os.IsNotExist(err) {
		// THERE IS NO CONFIG FILE... CHECK IF WE HAVE AN ENDPOINT...
		if eFunc == "" {
			fmt.Println("Config file does not exist and no endpoint specified! Please, either:")
			fmt.Println("    add -e my.rt.server")
			fmt.Println("    add endpoint: my.rt.server to", cFunc)
			os.Exit(0)
		}
	} else {
		//PULL CONFIGS
		dat, err := ioutil.ReadFile(cFunc)
		if err != nil {
			fmt.Println("Unable to read from", eFunc, "- Check permissions")
			os.Exit(0)
		}
		content := strings.Split(string(dat), "\n")
		for _, line := range content {
			info := strings.Split(line, ":")
			key := info[0]
			data := strings.Join(append(info[:0], info[1:]...), ":")
			if len(info) == 2 {
				settings[strings.Trim(key, " ")] = strings.Trim(data, " ")
			}
		}
	}

	// ALLOW FLAGS TO OVERRIDE CONFIG FILE
	if eFunc != "" {
		settings["endpoint"] = strings.Trim(eFunc, " ")
	}
	if uFunc != "" {
		if uFunc != settings["username"] {
			settings["password"] = ""
		}
		settings["username"] = uFunc
	}
	if *getFields != "" {
		settings["summaryFields"] = *getFields
	}

	// DO WE HAVE LOGIN INFORMATION FOR THE ENDPOINT?
	if settings["username"] == "" || settings["password"] == "" {
		fmt.Println("Requesting login information for:", settings["endpoint"])
		if settings["username"] == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("  - Username: ")
			text, _ := reader.ReadString('\n')
			settings["username"] = strings.Trim(text, "\n")
		}
		if settings["password"] == "" {
			settings["password"] = getPassword("  - Password: ")
			fmt.Print("\n")
		}

	}

	// WE HAVE A VALID TICKET NUMBER AND DATA...
	//rt update
	//rt search
	//rt get
	//rt create
	action := os.Args[1]
	switch action {
	case "get":
		showTicket(settings, strings.Trim(string(*gettNum), " "), *getComments, *getHistory, *getSummary, *getLinks, *getAttach)
	case "update":
		updateTicket(settings, strings.Trim(string(*updtNum), " "))
	case "search":
		fmt.Println("Feature to come!")
	case "create":
		fmt.Println("Feature to come!")
	default:
		showHelp()
	}
}
