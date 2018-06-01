package main

import (
  "reflect"
  "bufio"
  "syscall"
  "fmt"
  "net/http"
  "flag"
  "io/ioutil"
  "os"
  "os/user"
  "strings"
  "strconv"
)

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

// THANK YOU Joe Harnish
func queryRT(method string,URL string) string {
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		fmt.Println("Error making request: ", err, " For URL ", URL)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error Executing request: ", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	outputbyte, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Read Error: ", err)
		os.Exit(1)
	}

	return string(outputbyte)
}

func queryHistory(settings map[string]string, tnumber string) []string {
	contents := queryRT("GET", "http://"+settings["endpoint"]+"/REST/1.0/ticket/"+tnumber+"/history?user="+settings["username"]+"&pass="+settings["password"])
	content := strings.Split(string(contents), "\n")
	content = append(content[:0], content[1:]...)
	return append(content[:0], content[1:]...)
}

func queryTicket(settings map[string]string, ticket string) map[string]string {
	contents := queryRT("GET", "http://"+settings["endpoint"]+"/REST/1.0/ticket/"+ticket+"/show?user="+settings["username"]+"&pass="+settings["password"])
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
	t, err := strconv.Atoi(tnumber)
	if err != nil {
		fmt.Println("Ticket Number (", tnumber, ") is not numeric!\nPlease include a valid ticket number. rt -t 1234567")
		os.Exit(0)
	}
	if t == -1 {
		showGetHelp()
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

func showTicket(settings map[string]string, tnumber string, comments bool, history bool, sum bool, links bool, attachments bool, verb bool) {
	if sum || verb {
		showSummary(settings, tnumber, verb)
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
	}
	if attachments {
		showAttachments(settings, tnumber)
	}
	if !attachments && !links && !history && !comments && !sum && !verb {
		showSummary(settings, tnumber, verb)
	}
	os.Exit(0)
}

func showSummary(settings map[string]string, tnumber string, verb bool) {
	ticket := getTicket(settings, tnumber)
    if verb {
        settings["summaryFields"] = ""
    }
	if settings["summaryFields"] == "" {
		if !verb {
            fmt.Println("summaryFields is not defined in config, -f was not set.\n  - Dumping Ticket:")
        } else {
            fmt.Println("Verbose Specified. Dumping Ticket:")
        }
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
			contents2 := queryRT("GET", "http://"+settings["endpoint"]+"/REST/1.0/ticket/"+tnumber+"/history/id/"+id+"?user="+settings["username"]+"&pass="+settings["password"])
			fmt.Println("-----------------------------------------------------")
			fmt.Println(string(contents2))
			fmt.Println("-----------------------------------------------------")
		}
	}
}

func showLinks(settings map[string]string, tnumber string) {
	content := queryRT("GET", "http://"+settings["endpoint"]+"/REST/1.0/ticket/"+tnumber+"/links/show?user="+settings["username"]+"&pass="+settings["password"])
	contents := strings.Split(string(content), "\n")
	contents = append(contents[:0], contents[3:]...)
	for _, l := range contents {
		if l != "" {
			//linktype := strings.Split(l, ":")[0]
			tktnumadj := strings.Split(l, "/")
			tktnum := strings.Replace(tktnumadj[len(tktnumadj)-1],",","",-1)
			tkt := getTicket(settings, tktnum)
			fmt.Print(" ", tktnum, " ", tkt["Subject"], " [", tkt["Status"], "] [", tkt["Owner"], "]\n")
		}
	}
}

func showAttachments(settings map[string]string, tnumber string) {
	content := queryRT("GET", "http://"+settings["endpoint"]+"/REST/1.0/ticket/"+tnumber+"/attachments?user="+settings["username"]+"&pass="+settings["password"])
	contents := strings.Split(string(content), "\n")
	contents = append(contents[:0], contents[4:]...)
	for _, l := range contents {
		fmt.Println(strings.Trim(strings.Replace(l, "Attachments:", "", 1), " "))
	}
}

func rtsearch(settings map[string]string, owners string, queues string, status string, titles string, showURL bool) {
	if owners == "" && queues == "" && status == "" && titles == "" {
		//No criteria? Send to help()
		showSearchHelp()
		os.Exit(0)
	}
	//We have some sort of search criteria, lets build the query
	query_o := ""
	query_q := ""
	query_s := ""
	query_t := ""
	query := "http://rt.llnw.com/REST/1.0/search/ticket?user=" + settings["username"] + "&pass=" + settings["password"] + "&query="
	o := strings.Split(owners, ",")
	q := strings.Split(queues, ",")
	s := strings.Split(status, ",")
	t := strings.Split(titles, ",")
	for _, owner := range o {
		query_o = query_o + "Owner = '" + owner + "' OR "
	}
	for _, queue := range q {
		if settings[queue] != "" {
			queue = settings[queue]
		}
		query_q = query_q + "Queue = '" + queue + "' OR "
	}
	for _, stat := range s {
		query_s = query_s + "Status = '" + stat + "' OR "
	}
	for _, title := range t {
		query_t = query_t + "Subject LIKE '" + title + "' OR "
	}
	query_o = "(" + strings.TrimRight(query_o, " OR ") + ")"
	query_q = "(" + strings.TrimRight(query_q, " OR ") + ")"
	query_s = "(" + strings.TrimRight(query_s, " OR ") + ")"
	query_t = "(" + strings.TrimRight(query_t, " OR ") + ")"
	if query_o != "(Owner = '')" {
		query = query + query_o + " AND "
	}
	if query_q != "(Queue = '')" {
		query = query + query_q + " AND "
	}
	if query_s != "(Status = '')" {
		query = query + query_s + " AND "
	}
	if query_t != "(Subject LIKE '')" {
		query = query + query_t + "' AND "
	}
	query = strings.TrimRight(query, " AND ")
	query = strings.Replace(strings.Replace(query, "'", "%27", -1), " ", "%20", -1)
	//START SEARCH
	if showURL {
		fmt.Println(query)
		os.Exit(0)
	}
    content := queryRT("GET",query);
	fmt.Println(string(content))
}

func showHelp() {
	//HELP MENU
	fmt.Println("Usage: rt <subcommand> [globaloptions] [options]")
	fmt.Println("\n  Sub-commands (rt <subcommand> -h)")
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
	fmt.Println("Usage: rt get [globaloptions] -t <ticketnum> [options]")
	fmt.Println("\n  Global Options")
	fmt.Println("    -config [string]: Alt config file (~/.rt.d/config)")
	fmt.Println("    -user [string]:   Specify username for Endpoint")
	fmt.Println("    -e [string]:      Endpoint (ie: my.rtserver.com)")
	fmt.Println("\n  Options")
	fmt.Println("    -f: Specify Custom Fields for Summary (csv,list)")
	fmt.Println("    -t: Ticket Number (required)")
	fmt.Println("    -p: Show Ticket History")
	fmt.Println("    -c: Show Ticket Comments")
	fmt.Println("    -a: List Ticket Attachments")
	fmt.Println("    -l: List Related Tickets")
	fmt.Println("    -s: Show Ticket Summary (default)")
    fmt.Println("    -v: implies -s, Dump All Summary Data")
	fmt.Println("")
	os.Exit(0)
}

func showUpdateHelp() {
	fmt.Println("Usage: rt update [globaloptions] [options]")
	fmt.Println("\n  Global Options")
	fmt.Println("    -config [string]: Alt config file (~/.rt.d/config)")
	fmt.Println("    -user [string]:   Specify username for Endpoint")
	fmt.Println("    -e [string]:      Endpoint (ie: my.rtserver.com)")
	fmt.Println("\n  Options")

	fmt.Println("")
	os.Exit(0)
}

func showCreateHelp() {
	fmt.Println("Usage: rt create [globaloptions] [options]")
	fmt.Println("\n  Global Options")
	fmt.Println("    -config [string]: Alt config file (~/.rt.d/config)")
	fmt.Println("    -user [string]:   Specify username for Endpoint")
	fmt.Println("    -e [string]:      Endpoint (ie: my.rtserver.com)")
	fmt.Println("")

	os.Exit(0)
}

func showSearchHelp() {
	fmt.Println("Usage: rt search [globaloptions] [options]")
	fmt.Println("\n  Global Options")
	fmt.Println("    -config [string]: Alt config file (~/.rt.d/config)")
	fmt.Println("    -user [string]:   Specify username for Endpoint")
	fmt.Println("    -e [string]:      Endpoint (ie: my.rtserver.com)")
	fmt.Println("\n  Options")
	fmt.Println("    -d:          Only show URL (dry run)")
	fmt.Println("    -o [string]: Query Tickets by Owner (csv,list)")
	fmt.Println("    -q [string]: Query Tickets by Queue (csv,list)")
	fmt.Println("    -s [string]: Query Tickets by Status (csv,list)")
	fmt.Println("    -t [string]: Query Tickets by Title (csv,list)")
	fmt.Println("      ie: rt search -o jsmith,jdoe -s new,open -q q1,q2")
	fmt.Println("      ie: rt search -s open -t 'text 1',txt,'text 2'")
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
    getVerbose := getCommand.Bool("v", false, "Show Verbose Summary")
	getAttach := getCommand.Bool("a", false, "Show Attachment List")
	getLinks := getCommand.Bool("l", false, "Show Linked Tickets")
	getHelp := getCommand.Bool("h", false, "Show Help Menu")
	gettNum := getCommand.String("t", "-1", "Ticket Number")
	getFields := getCommand.String("f", "", "List of custom fields to show in summary, sep ','")
	geteFunc := getCommand.String("e", "", "RT Server Endpoint")
	getcFunc := getCommand.String("config", user.HomeDir+"/.rt.d/config", "Specify Alternate Config File")
	getuFunc := getCommand.String("user", "", "Specify Username")

	updCommand := flag.NewFlagSet("update", flag.ExitOnError)
	updHelp := updCommand.Bool("h", false, "Show Help Menu")
	//	updtNum := updCommand.String("t", "-1", "Ticket Number")
	updeFunc := updCommand.String("e", "", "RT Server Endpoint")
	updcFunc := updCommand.String("config", user.HomeDir+"/.rt.d/config", "Specify Alternate Config File")
	upduFunc := updCommand.String("user", "", "Specify Username")

	searchCommand := flag.NewFlagSet("search", flag.ExitOnError)
	searchHelp := searchCommand.Bool("h", false, "Show Help Menu")
	searcheFunc := searchCommand.String("e", "", "RT Server Endpoint")
	searchcFunc := searchCommand.String("config", user.HomeDir+"/.rt.d/config", "Specify Alternate Config File")
	searchuFunc := searchCommand.String("user", "", "Specify Username")
	searchOwners := searchCommand.String("o", "", "Search for Tickets Maching Owner")
	searchQueues := searchCommand.String("q", "", "Search for Tickets in Specified Queues")
	searchStatus := searchCommand.String("s", "", "Search for Tickets Maching Status")
	searchTitle := searchCommand.String("t", "", "Search for Tickets Matching Title")
	searchShowURL := searchCommand.Bool("d", false, "Only Show Query URL (dry run)")

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
	action := os.Args[1]
	switch action {
	case "get":
		showTicket(settings, strings.Trim(string(*gettNum), " "), *getComments, *getHistory, *getSummary, *getLinks, *getAttach, *getVerbose)
	case "update":
		fmt.Println("Feature to come!")
	case "search":
		rtsearch(settings, *searchOwners, *searchQueues, *searchStatus, *searchTitle, *searchShowURL)
	case "create":
		fmt.Println("Feature to come!")
	default:
		showHelp()
	}
}
