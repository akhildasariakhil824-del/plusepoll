package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// ANSI color codes
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Clear   = "\033[H\033[2J"
)

type PollOption struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
	Votes int64  `json:"votes"`
}

type Poll struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Options     []PollOption `json:"options"`
	CreatorName string       `json:"creator_name"`
	IsActive    bool         `json:"is_active"`
	TotalVotes  int64        `json:"total_votes"`
	LiveViewers int64        `json:"live_viewers"`
	CreatedAt   time.Time    `json:"created_at"`
}

type WebSocketMessage struct {
	Type        string           `json:"type"`
	PollID      string           `json:"poll_id"`
	TotalVotes  int64            `json:"total_votes"`
	OptionVotes map[int]int64    `json:"option_votes"`
	Percentages map[int]float64  `json:"percentages"`
	ViewerCount int64            `json:"viewer_count"`
	IsActive    bool             `json:"is_active"`
	Timestamp   time.Time        `json:"timestamp"`
}

func getAPIBase() string {
	if u := os.Getenv("PULSEPOLL_API_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "http://localhost:8080"
}

func printBanner() {
	fmt.Printf("%s%s", Cyan, Bold)
	fmt.Print(`
  ____        _            ____       _ _ 
 |  _ \ _   _| |___  ___  |  _ \ ___ | | |
 | |_) | | | | / __|/ _ \ | |_) / _ \| | |
 |  __/| |_| | \__ \  __/ |  __/ (_) | | |
 |_|    \__,_|_|___/\___| |_|   \___/|_|_|
`)
	fmt.Printf("%s%s  ⚡ Real-Time Live Polling Platform (Go + React + Redis + MongoDB)%s\n\n", Reset, Yellow, Reset)
}

func main() {
	apiBase := flag.String("server", getAPIBase(), "Base URL of PulsePoll backend API")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		runInteractiveMenu(*apiBase)
		return
	}

	command := args[0]
	switch command {
	case "list":
		listPolls(*apiBase)
	case "create":
		createPollPrompt(*apiBase)
	case "vote":
		if len(args) < 2 {
			fmt.Printf("%sUsage: pulsepoll vote <poll_id> [option_index]%s\n", Yellow, Reset)
			os.Exit(1)
		}
		pollID := args[1]
		var optIdx *int
		if len(args) >= 3 {
			if idx, err := strconv.Atoi(args[2]); err == nil {
				optIdx = &idx
			}
		}
		votePollPrompt(*apiBase, pollID, optIdx)
	case "watch":
		if len(args) < 2 {
			fmt.Printf("%sUsage: pulsepoll watch <poll_id>%s\n", Yellow, Reset)
			os.Exit(1)
		}
		watchLivePoll(*apiBase, args[1])
	case "simulate":
		if len(args) < 2 {
			fmt.Printf("%sUsage: pulsepoll simulate <poll_id> [vote_count]%s\n", Yellow, Reset)
			os.Exit(1)
		}
		pollID := args[1]
		count := 50
		if len(args) >= 3 {
			if c, err := strconv.Atoi(args[2]); err == nil && c > 0 {
				count = c
			}
		}
		simulateVotes(*apiBase, pollID, count)
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Printf("%sUnknown command: %s%s\n\n", Red, command, Reset)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	printBanner()
	fmt.Printf("%sAvailable Terminal Commands:%s\n", Bold, Reset)
	fmt.Printf("  %slist%s                 List all active public polls\n", Green, Reset)
	fmt.Printf("  %screate%s               Interactively create a new poll\n", Green, Reset)
	fmt.Printf("  %svote <id> [opt]%s      Cast a vote on a poll from terminal\n", Green, Reset)
	fmt.Printf("  %swatch <id>%s           Watch real-time live ASCII bar chart over WebSockets/Redis\n", Green, Reset)
	fmt.Printf("  %ssimulate <id> [n]%s    Run concurrent vote traffic simulator to test Redis throughput\n", Green, Reset)
	fmt.Printf("  %shelp%s                 Show this help screen\n\n", Green, Reset)
	fmt.Printf("Example:\n  %spulsepoll watch 67014b2d29b48c12a819034f%s\n\n", Cyan, Reset)
}

func runInteractiveMenu(apiBase string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		printBanner()
		fmt.Printf("%sSelect a Terminal Action:%s\n", Bold, Reset)
		fmt.Printf("  [%s1%s] 📊 View All Active Polls\n", Green, Reset)
		fmt.Printf("  [%s2%s] ➕ Create a New Poll\n", Green, Reset)
		fmt.Printf("  [%s3%s] 🗳️  Cast a Vote in Terminal\n", Green, Reset)
		fmt.Printf("  [%s4%s] ⚡ Watch Real-Time Live Bar Chart (WebSocket Stream)\n", Green, Reset)
		fmt.Printf("  [%s5%s] 🚀 Run Concurrent Vote Simulator (Load Test Redis)\n", Green, Reset)
		fmt.Printf("  [%s6%s] 🌐 Health Check & System Status\n", Green, Reset)
		fmt.Printf("  [%s0%s] 🚪 Exit\n\n", Red, Reset)
		fmt.Printf("%sEnter option (0-6): %s", Bold, Reset)

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			listPolls(apiBase)
			pause(reader)
		case "2":
			createPollPrompt(apiBase)
			pause(reader)
		case "3":
			polls := fetchPolls(apiBase)
			if len(polls) == 0 {
				fmt.Printf("%sNo active polls found.%s\n", Yellow, Reset)
				pause(reader)
				continue
			}
			displayPollsTable(polls)
			fmt.Printf("\nEnter Poll ID or Index (1-%d): ", len(polls))
			pChoice, _ := reader.ReadString('\n')
			pChoice = strings.TrimSpace(pChoice)
			targetID := pChoice
			if idx, err := strconv.Atoi(pChoice); err == nil && idx >= 1 && idx <= len(polls) {
				targetID = polls[idx-1].ID
			}
			votePollPrompt(apiBase, targetID, nil)
			pause(reader)
		case "4":
			polls := fetchPolls(apiBase)
			if len(polls) == 0 {
				fmt.Printf("%sNo active polls found. Create one first!%s\n", Yellow, Reset)
				pause(reader)
				continue
			}
			displayPollsTable(polls)
			fmt.Printf("\nEnter Poll ID or Index to watch (1-%d): ", len(polls))
			pChoice, _ := reader.ReadString('\n')
			pChoice = strings.TrimSpace(pChoice)
			targetID := pChoice
			if idx, err := strconv.Atoi(pChoice); err == nil && idx >= 1 && idx <= len(polls) {
				targetID = polls[idx-1].ID
			}
			watchLivePoll(apiBase, targetID)
		case "5":
			polls := fetchPolls(apiBase)
			if len(polls) == 0 {
				fmt.Printf("%sNo active polls found. Create one first!%s\n", Yellow, Reset)
				pause(reader)
				continue
			}
			displayPollsTable(polls)
			fmt.Printf("\nEnter Poll ID or Index to simulate: ")
			pChoice, _ := reader.ReadString('\n')
			pChoice = strings.TrimSpace(pChoice)
			targetID := pChoice
			if idx, err := strconv.Atoi(pChoice); err == nil && idx >= 1 && idx <= len(polls) {
				targetID = polls[idx-1].ID
			}
			fmt.Printf("How many concurrent votes? (default 50): ")
			cStr, _ := reader.ReadString('\n')
			cStr = strings.TrimSpace(cStr)
			votesCount := 50
			if n, err := strconv.Atoi(cStr); err == nil && n > 0 {
				votesCount = n
			}
			simulateVotes(apiBase, targetID, votesCount)
			pause(reader)
		case "6":
			checkHealth(apiBase)
			pause(reader)
		case "0", "exit", "quit":
			fmt.Printf("\n%sGoodbye! Happy polling!%s\n", Green, Reset)
			return
		default:
			fmt.Printf("%sInvalid choice. Please select 0 to 6.%s\n", Red, Reset)
			time.Sleep(1 * time.Second)
		}
	}
}

func pause(r *bufio.Reader) {
	fmt.Printf("\n%sPress [Enter] to continue...%s", Dim, Reset)
	_, _ = r.ReadString('\n')
}

func fetchPolls(apiBase string) []Poll {
	resp, err := http.Get(apiBase + "/api/polls")
	if err != nil {
		fmt.Printf("%sFailed to connect to backend at %s: %v%s\n", Red, apiBase, err, Reset)
		return nil
	}
	defer resp.Body.Close()

	var rawList []map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &rawList); err != nil {
		return nil
	}

	var polls []Poll
	for _, item := range rawList {
		id, _ := item["id"].(string)
		title, _ := item["title"].(string)
		desc, _ := item["description"].(string)
		creator, _ := item["creator_name"].(string)
		isActive, _ := item["is_active"].(bool)
		totalVotes := int64(0)
		if tv, ok := item["total_votes"].(float64); ok {
			totalVotes = int64(tv)
		}
		viewers := int64(0)
		if lv, ok := item["live_viewers"].(float64); ok {
			viewers = int64(lv)
		}

		var options []PollOption
		if rawOpts, ok := item["options"].([]interface{}); ok {
			for idx, ro := range rawOpts {
				if optMap, ok := ro.(map[string]interface{}); ok {
					text, _ := optMap["text"].(string)
					v := int64(0)
					if votesVal, ok := optMap["votes"].(float64); ok {
						v = int64(votesVal)
					}
					options = append(options, PollOption{
						Index: idx,
						Text:  text,
						Votes: v,
					})
				}
			}
		}

		polls = append(polls, Poll{
			ID:          id,
			Title:       title,
			Description: desc,
			CreatorName: creator,
			IsActive:    isActive,
			TotalVotes:  totalVotes,
			LiveViewers: viewers,
			Options:     options,
		})
	}
	return polls
}

func displayPollsTable(polls []Poll) {
	fmt.Printf("\n%s%-4s %-26s %-40s %-8s %-9s %-8s%s\n",
		Bold, "#", "POLL ID", "TITLE", "VOTES", "VIEWERS", "STATUS", Reset)
	fmt.Println(strings.Repeat("─", 102))

	for i, p := range polls {
		status := fmt.Sprintf("%s🟢 LIVE%s", Green, Reset)
		if !p.IsActive {
			status = fmt.Sprintf("%s🔴 CLOSED%s", Red, Reset)
		}
		truncTitle := p.Title
		if len(truncTitle) > 38 {
			truncTitle = truncTitle[:35] + "..."
		}
		fmt.Printf("%-4d %-26s %-40s %-8d %-9d %-8s\n",
			i+1, p.ID, truncTitle, p.TotalVotes, p.LiveViewers, status)
	}
}

func listPolls(apiBase string) {
	fmt.Printf("\n%sFetching active polls from %s...%s\n", Cyan, apiBase, Reset)
	polls := fetchPolls(apiBase)
	if len(polls) == 0 {
		fmt.Printf("%sNo polls found. The backend might have just started or database is clean.%s\n", Yellow, Reset)
		return
	}
	displayPollsTable(polls)
}

func createPollPrompt(apiBase string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\n%s%s➕ Create a New Live Poll%s\n", Bold, Cyan, Reset)
	fmt.Println(strings.Repeat("─", 50))

	fmt.Print("Poll Title / Question: ")
	title, _ := reader.ReadString('\n')
	title = strings.TrimSpace(title)
	if len(title) < 3 {
		fmt.Printf("%sTitle must be at least 3 characters!%s\n", Red, Reset)
		return
	}

	fmt.Print("Description (optional): ")
	desc, _ := reader.ReadString('\n')
	desc = strings.TrimSpace(desc)

	var options []string
	fmt.Println("\nEnter options (minimum 2, press enter with blank line to finish):")
	for i := 1; ; i++ {
		fmt.Printf("Option %d: ", i)
		opt, _ := reader.ReadString('\n')
		opt = strings.TrimSpace(opt)
		if opt == "" {
			if len(options) >= 2 {
				break
			}
			fmt.Printf("%sAt least 2 options are required.%s\n", Yellow, Reset)
			continue
		}
		options = append(options, opt)
		if len(options) >= 10 {
			break
		}
	}

	// Login with demo presenter account to obtain JWT
	token := getAuthToken(apiBase)
	if token == "" {
		fmt.Printf("%sAuthentication failed. Could not authenticate demo presenter.%s\n", Red, Reset)
		return
	}

	payload := map[string]interface{}{
		"title":       title,
		"description": desc,
		"options":     options,
	}
	payloadBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", apiBase+"/api/polls", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%sFailed to create poll: %v%s\n", Red, err, Reset)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		fmt.Printf("%sServer error (%d): %s%s\n", Red, resp.StatusCode, string(b), Reset)
		return
	}

	var created Poll
	_ = json.NewDecoder(resp.Body).Decode(&created)

	fmt.Printf("\n%s%s✔ Poll created successfully!%s\n", Bold, Green, Reset)
	fmt.Printf("  Poll ID:          %s%s%s\n", Bold, created.ID, Reset)
	fmt.Printf("  Web URL:          %shttp://localhost:5173/poll/%s%s\n", Cyan, created.ID, Reset)
	fmt.Printf("  Terminal Stream:  %spulsepoll watch %s%s\n", Yellow, created.ID, Reset)
}

func getAuthToken(apiBase string) string {
	loginPayload := map[string]string{
		"email":    "demo@pulsepoll.io",
		"password": "demopass123",
	}
	b, _ := json.Marshal(loginPayload)
	resp, err := http.Post(apiBase+"/api/auth/login", "application/json", bytes.NewBuffer(b))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return ""
	}
	token, _ := res["token"].(string)
	return token
}

func votePollPrompt(apiBase, pollID string, preselectedOpt *int) {
	reader := bufio.NewReader(os.Stdin)

	resp, err := http.Get(fmt.Sprintf("%s/api/polls/%s", apiBase, pollID))
	if err != nil {
		fmt.Printf("%sFailed to fetch poll details: %v%s\n", Red, err, Reset)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("%sPoll not found with ID: %s%s\n", Red, pollID, Reset)
		return
	}

	var data map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&data)
	pollData, _ := data["poll"].(map[string]interface{})
	if pollData == nil {
		fmt.Printf("%sInvalid poll data returned%s\n", Red, Reset)
		return
	}

	title, _ := pollData["title"].(string)
	rawOptions, _ := pollData["options"].([]interface{})

	fmt.Printf("\n%s%sPoll: %s%s\n", Bold, Cyan, title, Reset)
	fmt.Println(strings.Repeat("─", 50))
	for idx, ro := range rawOptions {
		optMap := ro.(map[string]interface{})
		text, _ := optMap["text"].(string)
		votes, _ := optMap["votes"].(float64)
		fmt.Printf("  [%s%d%s] %s  (%s%.0f votes%s)\n", Bold, idx, Reset, text, Dim, votes, Reset)
	}

	optIdx := 0
	if preselectedOpt != nil {
		optIdx = *preselectedOpt
	} else {
		fmt.Printf("\nSelect an option to vote for (0-%d): ", len(rawOptions)-1)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		val, err := strconv.Atoi(input)
		if err != nil || val < 0 || val >= len(rawOptions) {
			fmt.Printf("%sInvalid option index.%s\n", Red, Reset)
			return
		}
		optIdx = val
	}

	voterID := fmt.Sprintf("terminal-%d-%d", time.Now().UnixNano(), rand.Intn(10000))
	voteReq := map[string]interface{}{
		"option_index": optIdx,
		"voter_id":     voterID,
	}
	reqBytes, _ := json.Marshal(voteReq)

	voteResp, err := http.Post(fmt.Sprintf("%s/api/polls/%s/vote", apiBase, pollID), "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		fmt.Printf("%sFailed to cast vote: %v%s\n", Red, err, Reset)
		return
	}
	defer voteResp.Body.Close()

	if voteResp.StatusCode == http.StatusOK {
		fmt.Printf("\n%s%s✔ Vote recorded successfully via Redis atomic counter!%s\n", Bold, Green, Reset)
	} else {
		b, _ := io.ReadAll(voteResp.Body)
		fmt.Printf("\n%sVote rejected (%d): %s%s\n", Red, voteResp.StatusCode, string(b), Reset)
	}
}

func watchLivePoll(apiBase, pollID string) {
	u, err := url.Parse(apiBase)
	if err != nil {
		fmt.Printf("%sInvalid API URL: %v%s\n", Red, err, Reset)
		return
	}

	wsScheme := "ws"
	if u.Scheme == "https" {
		wsScheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/api/ws/polls/%s", wsScheme, u.Host, pollID)

	// Fetch poll title and options first
	resp, err := http.Get(fmt.Sprintf("%s/api/polls/%s", apiBase, pollID))
	if err != nil {
		fmt.Printf("%sFailed to connect to backend: %v%s\n", Red, err, Reset)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("%sPoll not found with ID: %s%s\n", Red, pollID, Reset)
		return
	}

	var initialData map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&initialData)
	pollMap, _ := initialData["poll"].(map[string]interface{})
	if pollMap == nil {
		fmt.Printf("%sInvalid poll data%s\n", Red, Reset)
		return
	}

	title, _ := pollMap["title"].(string)
	optionsRaw, _ := pollMap["options"].([]interface{})
	optionLabels := make([]string, len(optionsRaw))
	for i, ro := range optionsRaw {
		om := ro.(map[string]interface{})
		optionLabels[i], _ = om["text"].(string)
	}

	fmt.Printf("%sConnecting WebSocket to %s...%s\n", Cyan, wsURL, Reset)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		fmt.Printf("%sWebSocket connection failed (%v). Falling back to HTTP polling mode.%s\n", Yellow, err, Reset)
		pollOverHTTP(apiBase, pollID, title, optionLabels)
		return
	}
	defer conn.Close()

	// Handle graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			var msg WebSocketMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				return
			}
			renderLiveDashboard(title, optionLabels, msg)
		}
	}()

	select {
	case <-interrupt:
		fmt.Printf("\n%sClosing live stream...%s\n", Yellow, Reset)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		time.Sleep(200 * time.Millisecond)
	case <-done:
		fmt.Printf("\n%sWebSocket stream ended.%s\n", Dim, Reset)
	}
}

func pollOverHTTP(apiBase, pollID, title string, optionLabels []string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-interrupt:
			return
		case <-ticker.C:
			resp, err := http.Get(fmt.Sprintf("%s/api/polls/%s", apiBase, pollID))
			if err != nil {
				continue
			}
			var data map[string]interface{}
			_ = json.NewDecoder(resp.Body).Decode(&data)
			resp.Body.Close()

			pollMap, _ := data["poll"].(map[string]interface{})
			if pollMap == nil {
				continue
			}
			totalVotes := int64(0)
			if tv, ok := pollMap["total_votes"].(float64); ok {
				totalVotes = int64(tv)
			}
			viewers := int64(0)
			if lv, ok := data["live_viewers"].(float64); ok {
				viewers = int64(lv)
			}
			optionsRaw, _ := pollMap["options"].([]interface{})
			optVotes := make(map[int]int64)
			percentages := make(map[int]float64)
			for i, ro := range optionsRaw {
				om := ro.(map[string]interface{})
				v := int64(0)
				if val, ok := om["votes"].(float64); ok {
					v = int64(val)
				}
				optVotes[i] = v
				if totalVotes > 0 {
					percentages[i] = float64(v) / float64(totalVotes) * 100.0
				}
			}

			msg := WebSocketMessage{
				TotalVotes:  totalVotes,
				OptionVotes: optVotes,
				Percentages: percentages,
				ViewerCount: viewers,
				IsActive:    true,
				Timestamp:   time.Now(),
			}
			renderLiveDashboard(title, optionLabels, msg)
		}
	}
}

func renderLiveDashboard(title string, options []string, msg WebSocketMessage) {
	fmt.Print(Clear)
	barWidth := 36

	statusBadge := fmt.Sprintf("%s%s🟢 LIVE%s", Green, Bold, Reset)
	if !msg.IsActive {
		statusBadge = fmt.Sprintf("%s%s🔴 CLOSED%s", Red, Bold, Reset)
	}

	fmt.Printf("%s╔══════════════════════════════════════════════════════════════════════════════╗%s\n", Cyan, Reset)
	fmt.Printf("%s║%s %s%s⚡ PULSEPOLL LIVE TERMINAL STREAM%s                                         %s║%s\n",
		Cyan, Reset, Yellow, Bold, Reset, Cyan, Reset)
	fmt.Printf("%s║%s  Title: %s%-66s%s %s║%s\n",
		Cyan, Reset, Bold, truncate(title, 66), Reset, Cyan, Reset)
	fmt.Printf("%s║%s  Status: %s  │  Viewers: %s👁️  %-4d%s  │  Total Votes: %s🗳️  %-8d%s       %s║%s\n",
		Cyan, Reset, statusBadge, Bold, msg.ViewerCount, Reset, Bold, msg.TotalVotes, Reset, Cyan, Reset)
	fmt.Printf("%s╚══════════════════════════════════════════════════════════════════════════════╝%s\n\n", Cyan, Reset)

	// Determine winner index
	maxVotes := int64(-1)
	winnerIdx := -1
	for idx, v := range msg.OptionVotes {
		if v > maxVotes && v > 0 {
			maxVotes = v
			winnerIdx = idx
		}
	}

	for idx, label := range options {
		votes := msg.OptionVotes[idx]
		pct := msg.Percentages[idx]

		filled := 0
		if msg.TotalVotes > 0 {
			filled = int(math.Round((pct / 100.0) * float64(barWidth)))
		}
		if filled > barWidth {
			filled = barWidth
		}
		unfilled := barWidth - filled

		barColor := Blue
		if idx == winnerIdx && msg.TotalVotes > 0 {
			barColor = Green
		}

		bar := fmt.Sprintf("%s%s%s%s%s",
			barColor, strings.Repeat("█", filled),
			Dim, strings.Repeat("░", unfilled), Reset)

		crown := " "
		if idx == winnerIdx && msg.TotalVotes > 0 {
			crown = "👑"
		}

		fmt.Printf("  [%s%d%s] %-48s %s\n", Bold, idx, Reset, truncate(label, 48), crown)
		fmt.Printf("      %s  %s%5.1f%%%s (%s%d votes%s)\n\n",
			bar, Bold, pct, Reset, Dim, votes, Reset)
	}

	fmt.Println(strings.Repeat("─", 78))
	fmt.Printf("  %s● Connected to Redis Pub/Sub stream via Go WebSocket.%s\n", Green, Reset)
	fmt.Printf("  %sReal-time live updates stream automatically on every cast vote.%s\n", Dim, Reset)
	fmt.Printf("  %sPress Ctrl+C to stop watching.%s\n", Dim, Reset)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func simulateVotes(apiBase, pollID string, totalCount int) {
	fmt.Printf("\n%s%s🚀 Starting Live Traffic Simulation (%d votes)...%s\n", Bold, Magenta, totalCount, Reset)

	// Fetch option count
	resp, err := http.Get(fmt.Sprintf("%s/api/polls/%s", apiBase, pollID))
	if err != nil {
		fmt.Printf("%sError fetching poll: %v%s\n", Red, err, Reset)
		return
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&data)
	pollMap, _ := data["poll"].(map[string]interface{})
	if pollMap == nil {
		fmt.Printf("%sInvalid poll ID%s\n", Red, Reset)
		return
	}

	options, _ := pollMap["options"].([]interface{})
	numOptions := len(options)
	if numOptions < 2 {
		fmt.Printf("%sPoll has less than 2 options.%s\n", Red, Reset)
		return
	}

	start := time.Now()
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20) // concurrency limit
	successCount := int64(0)
	var mu sync.Mutex

	fmt.Println("Simulating concurrent audience voting against Redis atomic counters...")

	for i := 0; i < totalCount; i++ {
		wg.Add(1)
		go func(voterNum int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			opt := rand.Intn(numOptions)
			voterID := fmt.Sprintf("sim-voter-%d-%d", voterNum, time.Now().UnixNano())

			body, _ := json.Marshal(map[string]interface{}{
				"option_index": opt,
				"voter_id":     voterID,
			})

			voteURL := fmt.Sprintf("%s/api/polls/%s/vote", apiBase, pollID)
			vResp, err := http.Post(voteURL, "application/json", bytes.NewBuffer(body))
			if err == nil {
				if vResp.StatusCode == http.StatusOK {
					mu.Lock()
					successCount++
					current := successCount
					mu.Unlock()
					if current%10 == 0 || current == int64(totalCount) {
						pct := float64(current) / float64(totalCount) * 100.0
						fmt.Printf("\r  Progress: [%-20s] %.0f%% (%d/%d votes cast)",
							strings.Repeat("█", int(pct/5)), pct, current, totalCount)
					}
				}
				vResp.Body.Close()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	rate := float64(successCount) / duration.Seconds()

	fmt.Printf("\n\n%s%s✔ Simulation Complete!%s\n", Bold, Green, Reset)
	fmt.Printf("  Successful Votes:   %s%d / %d%s\n", Bold, successCount, totalCount, Reset)
	fmt.Printf("  Total Elapsed Time: %s%v%s\n", Bold, duration.Round(time.Millisecond), Reset)
	fmt.Printf("  Throughput:         %s%.1f votes/second%s\n\n", Bold, rate, Reset)
	fmt.Println("Check your browser or terminal live monitor to see the results updated in real time!")
}

func checkHealth(apiBase string) {
	fmt.Printf("\n%sChecking PulsePoll Backend Health at %s/api/health...%s\n", Cyan, apiBase, Reset)
	resp, err := http.Get(apiBase + "/api/health")
	if err != nil {
		fmt.Printf("%s❌ Backend is offline or unreachable: %v%s\n", Red, err, Reset)
		return
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&health)

	fmt.Printf("%s%s✔ Backend Status: %s%s\n", Bold, Green, health["status"], Reset)
	if checks, ok := health["checks"].(map[string]interface{}); ok {
		redisOK, _ := checks["redis"].(bool)
		storageOK, _ := checks["storage"].(bool)
		rtMode, _ := checks["realtime_mode"].(string)

		if redisOK {
			fmt.Printf("  Redis Engine:    %s✔ Connected (%s)%s\n", Green, rtMode, Reset)
		} else {
			fmt.Printf("  Redis Engine:    %s⚠️ Offline (%s)%s\n", Yellow, rtMode, Reset)
		}

		if storageOK {
			fmt.Printf("  Database Store:  %s✔ Connected%s\n", Green, Reset)
		} else {
			fmt.Printf("  Database Store:  %s⚠️ In-Memory Fallback%s\n", Yellow, Reset)
		}
	}
}
