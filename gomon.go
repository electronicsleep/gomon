// Author: https://github.com/electronicsleep
// Purpose: Golang application to monitor servers using prometheus metrics
// Released under the MIT License

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Verbose:
var verbose = false

// Single:
var single = false

// Threshold:
var threshold = 500

// Minutes to sleep between runs
const sleepInterval time.Duration = time.Minute * 1

// Webserver: Run webserver to show log output
var webserver = false

type configStruct struct {
	SlackURL string   `yaml:"slack_url"`
	SlackMsg string   `yaml:"slack_msg"`
	Email    string   `yaml:"email"`
	Servers  []string `yaml:"servers"`
}

type stateStruct struct {
	Hostname string
	ErrorNum int
	RunNum   int
}

func (config *configStruct) getConfig() *configStruct {
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Printf("ERROR: YAML file not found #%v ", err)
		os.Exit(1)
	}
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return config
}

func (state *stateStruct) getState() *stateStruct {
	state.ErrorNum = state.ErrorNum + 1
	fmt.Println("INFO: getState: state.ErrorNum ", state.ErrorNum)
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
	} else {
		state.Hostname = hostname
	}
	return state
}

func logOutput(cmd string, cmdOut string) {
	sCmd := string(cmdOut[:])
	linesCmd := strings.Split(sCmd, "\n")
	lineNum := 0
	for _, lineCmd := range linesCmd {
		lineNum++
		log.Println(cmd, lineCmd)
	}
}

func httpLogs(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("gomon.log")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > 20 {
			lines = lines[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	for _, line := range lines {
		w.Write([]byte(line + "\n"))
	}
}

func logMerics(msg string) {
	file, err := os.OpenFile("gomon_metrics.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString(msg)
	if err != nil {
		log.Fatal(err)
	}
}

func logMericsAppend(msg string) {
	file, err := os.OpenFile("gomon_metrics.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString(msg + "\n")
	if err != nil {
		log.Fatal(err)
	}
}

// Prometheus metrics
func httpMetrics(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("gomon_metrics.log")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > 20 {
			lines = lines[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	for _, line := range lines {
		w.Write([]byte(line + "\n"))
	}
}

func checkFatal(msg string, err error) {
	if err != nil {
		fmt.Println("FATAL: "+msg, err)
		log.Println("FATAL: "+msg, err)
		log.Fatal()
	}
}

func runMonitor() {
	loop := 0
	var state stateStruct

	if single {
		fmt.Println("INFO: Single run")
		logOutput("INFO:", "CHECK: gomon single run")
		checkSites(state)
		fmt.Println("INFO: exit")
		os.Exit(0)
	} else {
		fmt.Println("INFO: gomon loop run")
		fmt.Println("INFO: runtime:", loop, "minutes")
		logOutput("INFO:", "gomon loop run")
		for {
			loop++
			state = checkSites(state)
			fmt.Println("INFO: sleep check again:", sleepInterval)
			time.Sleep(sleepInterval)
		}
	}
}

func checkSites(state stateStruct) stateStruct {
	urls := return_url()
	if verbose {
		fmt.Println("INFO: verbose on")
		fmt.Println("INFO: urls:", urls)
	}

	fmt.Print("INFO: state.ErrorNum: ")
	fmt.Println(state.ErrorNum)
	fmt.Print("INFO: state.RunNum: ")
	fmt.Println(state.RunNum)

	errorNum := 0
	thresholdReachedNum := 0

	checkNum := 3
	errorNumAlert := 2
	fmt.Println("INFO: checkNum: " + strconv.Itoa(checkNum))
	for i := 0; i < checkNum; i++ {
		for n, requestURL := range urls {
			t := time.Now()
			tf := t.Format("2006.01.02-15.04.05:000")
			start := time.Now().UnixNano() / int64(time.Millisecond)

			fmt.Println("INFO: n:", n, "of", checkNum)
			fmt.Println("INFO: requestURL:", requestURL)
			res, err := http.Get(requestURL)
			// MOVE THIS BLOCK DOWN
			if err != nil {
				state.getState()
				errorNum += 1
				errStr := err.Error()
				logOutput("CHECK_ERROR:", requestURL+" error: "+errStr)
				fmt.Printf("ERROR: errorNum: %d errorNumAlert: %d err: %s\n", state.ErrorNum, errorNumAlert, err)
				if state.ErrorNum >= errorNumAlert {
					postMessage("ALERT: error with site over 2 errors: " + requestURL + ": " + strconv.Itoa(state.ErrorNum) + " Date: " + tf)
				}
				fmt.Println("CHECK: error with site: host: " + requestURL + " count: " + strconv.Itoa(state.ErrorNum) + " Date: " + tf)
			} else {
				logOutput("INFO: CHECK_OK", requestURL)
				fmt.Println("INFO: CHECK_OK", requestURL)
				fmt.Printf("INFO: client: status code: %d\n", res.StatusCode)
				defer res.Body.Close()
			}

			fmt.Print("INFO: errorNum: ")
			fmt.Println(errorNum)
			fmt.Print("INFO: state.ErrorNum: ")
			fmt.Println(state.ErrorNum)
			fmt.Print("INFO: state.RunNum: ")
			fmt.Println(state.RunNum)
			t_end := time.Now()
			tf_end := t_end.Format("2006.01.02-15.04.05:000")
			end := time.Now().UnixNano() / int64(time.Millisecond)
			duration := end - start
			logOutput("INFO: requestURL", requestURL+" Duration(ms) "+strconv.FormatInt(duration, 10)+" threshold "+strconv.Itoa(threshold))
			fmt.Println("INFO: requestURL", requestURL, "Duration(ms)", duration, "threshold", threshold)
			fmt.Println("DEBUG: Threshold:", threshold)
			durationDiffInt := int(duration)
			if durationDiffInt > threshold {
				thresholdReachedNum += 1
				fmt.Println("INFO: OVER Threshold Duration(ms):", duration, "thresholdReachedNum ", thresholdReachedNum)
				if thresholdReachedNum > 1 {
					postMessage("ALERT: error site over threshold: " + requestURL + ": Duration(ms): " + strconv.Itoa(durationDiffInt) + " Date: " + tf)
				}
			}
			newStr := strings.Replace(requestURL, ":", "_", -1)
			newStr1 := strings.Replace(newStr, "/", "_", -1)
			newStr2 := strings.Replace(newStr1, ".", "_", -1)
			var log_str = ""

			log_str = newStr2 + " " + strconv.Itoa(int(duration))
			if n == 0 {
				logMerics("")
				logMericsAppend(log_str)
			} else {
				logMericsAppend(log_str)
			}

			fmt.Println("INFO: time_start:", tf)
			fmt.Println("INFO: time_end:", tf_end)
		}
		fmt.Println("INFO: sleep check again:", sleepInterval)
		time.Sleep(sleepInterval)
		state.RunNum += 1
	}
	return state
}

func return_url() []string {
	var config configStruct
	config.getConfig()
	return config.Servers
}

func postMessage(message string) {
	var config configStruct
	config.getConfig()
	fmt.Println("INFO: config", config)
	if config.SlackURL == "" {
		fmt.Println("INFO: SlackMsg empty no messsages will be sent")
	} else {
		fmt.Println("INFO: SlackMsg found messages will be sent")
		postSlack(message)
	}
}

func postSlack(message string) {
	fmt.Println("postSlack message:" + message)

	var config configStruct
	config.getConfig()

	send_text := message + ": " + config.SlackMsg

	var jsonData = []byte(`{
                "text": "` + send_text + `",
        }`)

	if connected() {
		request, error := http.NewRequest("POST", config.SlackURL, bytes.NewBuffer(jsonData))
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		fmt.Println("INFO: Request:", request)
		if error != nil {
			fmt.Println("ERROR: postSlack: http.NewRequest:", error)
		}

		client := &http.Client{}
		response, error := client.Do(request)
		if error != nil {
			fmt.Println("ERROR: postSlack: client.Do", error)
			return
		}
		defer response.Body.Close()

		fmt.Println("response Status:", response.Status)
		fmt.Println("response Headers:", response.Header)
		body, _ := io.ReadAll(response.Body)
		fmt.Println("response Body:", string(body))
	} else {
		fmt.Println("ERROR: No connection to the net")
	}
}

func connected() (ok bool) {
	_, err := http.Get("http://clients3.google.com/generate_204")
	fmt.Println("ERROR: connected: err", err)
	return err == nil
}

func main() {
	verboseFlag := flag.Bool("v", false, "Verbose checks")
	singleFlag := flag.Bool("s", single, "Single checks")
	thresholdFlag := flag.Int("t", threshold, "Threshold checks")
	webserverFlag := flag.Bool("w", false, "Run webserver")

	flag.Parse()

	var state stateStruct
	state.getState()
	fmt.Println("INFO: state:", state)

	fmt.Println("INFO: Starting gomon hostname: " + state.Hostname)
	postMessage("INFO: Starting gomon hostname: " + state.Hostname)

	var config configStruct
	config.getConfig()
	fmt.Println("INFO: config:", config)

	verbose = *verboseFlag
	single = *singleFlag
	threshold = *thresholdFlag
	webserver = *webserverFlag

	if verbose {
		fmt.Println("INFO: Verbose:", verbose)
		fmt.Println("INFO: Single:", single)
		fmt.Println("INFO: Threshold:", threshold)
		fmt.Println("INFO: Webserver:", webserver)
	}

	// Start logging
	file, err := os.OpenFile("gomon.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	checkFatal("ERROR: opening file", err)

	defer file.Close()
	log.SetOutput(file)

	fmt.Println("INFO: Detect OS:", runtime.GOOS)
	fmt.Println("INFO: CPU Cores:", runtime.NumCPU())

	if webserver {
		go runMonitor()
		fmt.Println("INFO: Running webserver mode: http://localhost:8080/logs")
		fmt.Println("INFO: Running webserver mode: http://localhost:8080/metrics")
		http.Handle("/", http.FileServer(http.Dir("./src")))
		http.HandleFunc("/logs", httpLogs)
		http.HandleFunc("/metrics", httpMetrics)
		if err := http.ListenAndServe(":8080", nil); err != nil {
			checkFatal("ERROR: webserver: ", err)
		}
	} else {
		fmt.Println("INFO: Running console mode")
		runMonitor()
	}
}
