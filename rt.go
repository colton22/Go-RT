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

func getTicket(ticket string, settings map[string]string) map[string]string {
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

func comment(ticket string, url string, msg string) {
}

func main() {

	//SETUP VARIABLES
	settings := map[string]string{}
	settings["username"] = ""
	settings["password"] = ""
	settings["endpoint"] = ""
	user, _ := user.Current()

	//SETUP FLAGS
	tNum := flag.String("t", "-1", "Ticket Number")
	//qPtr := flag.String("q", "", "Change the Queue of a ticket.")
	//sPtr := flag.String("s", "", "Change the Status of a ticket.")
	//oPtr := flag.String("o", "", "Change the Owner of a ticket.")
	//pPtr := flag.Int("p", -1, "Change the Priority of a ticket.")
	eFunc := flag.String("e", "", "RT Server Endpoint")
	cFunc := flag.String("config", user.HomeDir+"/.rt.d/config", "Specify Alternate Config File")
	uFunc := flag.String("user", "", "Specify Username")
	flag.Parse()

	//SETUP CONFIGS
	if _, err := os.Stat(*cFunc); os.IsNotExist(err) {
		// THERE IS NO CONFIG FILE... CHECK IF WE HAVE AN ENDPOINT...
		if *eFunc == "" {
			fmt.Println("Config file does not exist and no endpoint specified! Please, either:")
			fmt.Println("    add -e my.rt.server")
			fmt.Println("    add endpoint: my.rt.server to", *cFunc)
			os.Exit(0)
		}
	} else {
		//PULL CONFIGS
		dat, err := ioutil.ReadFile(*cFunc)
		if err != nil {
			fmt.Println("Unable to read from", *eFunc, "- Check permissions")
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
	if *eFunc != "" {
		settings["endpoint"] = strings.Trim(*eFunc, " ")
	}
	if *uFunc != "" {
		if *uFunc != settings["username"] {
			settings["password"] = ""
		}
		settings["username"] = *uFunc
	}

	// IS THE TICKET NUMBER NUMERIC?
	if _, err := strconv.Atoi(*tNum); err != nil {
		fmt.Println("Please include a valid ticket number. rt -t 1234567")
		os.Exit(0)
	}
	if t, _ := strconv.Atoi(*tNum); t < 0 {
		fmt.Println("Please include a valid ticket number.  rt -t 1234567")
		os.Exit(0)
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

	// LETS GET THE CURRENT TICKET INFORMATION MAPPED
	ticket := getTicket(*tNum, settings)

	// WAS THAT A VALID TICKET? (split the key for line 2 on phrase does not, if #ofElements >1 error)
	ec := string(reflect.ValueOf(ticket).MapKeys()[1].Interface().(string))
	if len(strings.SplitAfter(ec, " does not ")) > 1 {
		fmt.Println("Ticket number", *tNum, "does not exist.")
		os.Exit(0)
	}
	if len(strings.SplitAfter(ec, "username or password")) > 1 {
		fmt.Println("Username/Password Incorrect.")
		os.Exit(0)
	}

	// WE HAVE A VALID TICKET NUMBER AND DATA...
	fmt.Println("")
	fmt.Println("Settings:")
	for index, element := range settings {
		fmt.Print("settings[", index, "]='", element, "'\n")
	}
	fmt.Println("Ticket Data:")
	for index, element := range ticket {
		fmt.Print("ticket[", index, "]='", element, "'\n")
	}
	fmt.Println("|done|")
}
