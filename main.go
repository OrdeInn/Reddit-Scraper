package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
	"os"

	"github.com/joho/godotenv"
)

// Reddit API URLs and headers
const (
    AuthURL         = "https://www.reddit.com/api/v1/access_token"
    SubredditURL    = "https://oauth.reddit.com/r/%s/hot?raw_json=1&limit=10&after=%s"
    CommentsURL     = "https://oauth.reddit.com/r/%s/comments/%s?raw_json=1&limit=10&after=%s"
)

// AccessToken represents the JSON structure of the OAuth response
type AccessToken struct {
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
    ExpiresIn   int    `json:"expires_in"`
    Scope       string `json:"scope"`
}

// Thread represents a Reddit thread (post)
type Thread struct {
    ID    string `json:"id"`
    Title string `json:"title"`
    URL   string `json:"url"`
}

// Comment represents a Reddit comment
type Comment struct {
    ID      string `json:"id"`
    Body    string `json:"body"`
    Author  string `json:"author"`
    Created int64  `json:"created_utc"`
}

// Global variable to store the access token
var accessToken string

// Function to load environment variables from the .env file
func loadEnv() {
    err := godotenv.Load()
    if err != nil {
        log.Fatalf("Error loading .env file")
    }
}

// Function to authenticate and get the access token
func getAccessToken() string {
	clientID := os.Getenv("CLIENT_ID")
    clientSecret := os.Getenv("CLIENT_SECRET")
    username := os.Getenv("USERNAME")
    password := os.Getenv("PASSWORD")
    userAgent := os.Getenv("USER_AGENT")

    data := []byte(fmt.Sprintf("grant_type=password&username=%s&password=%s", username, password))

    req, err := http.NewRequest("POST", AuthURL, bytes.NewBuffer(data))
    if err != nil {
        log.Fatalf("Error creating auth request: %v", err)
    }
    req.SetBasicAuth(clientID, clientSecret)
    req.Header.Set("User-Agent", userAgent)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Fatalf("Error sending auth request: %v", err)
    }
    defer resp.Body.Close()

    var token AccessToken
    body, _ := ioutil.ReadAll(resp.Body)
    json.Unmarshal(body, &token)

    return token.AccessToken
}

// Function to fetch threads from a subreddit with pagination
func getSubredditThreads(subreddit string) []Thread {
	userAgent := os.Getenv("USER_AGENT")

    var threads []Thread
    after := "" // Initially empty to fetch the first page

    for {
        url := fmt.Sprintf(SubredditURL, subreddit, after)

        req, err := http.NewRequest("GET", url, nil)
        if err != nil {
            log.Fatalf("Error creating request to get threads: %v", err)
        }
        req.Header.Set("Authorization", "Bearer "+accessToken)
        req.Header.Set("User-Agent", userAgent)

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            log.Fatalf("Error sending request to get threads: %v", err)
        }
        defer resp.Body.Close()

        var result map[string]interface{}
        body, _ := ioutil.ReadAll(resp.Body)
        json.Unmarshal(body, &result)

        // Parse threads and add them to the list
        for _, item := range result["data"].(map[string]interface{})["children"].([]interface{}) {
            data := item.(map[string]interface{})["data"].(map[string]interface{})
            threads = append(threads, Thread{
                ID:    data["id"].(string),
                Title: data["title"].(string),
                URL:   data["url"].(string),
            })
        }

        // Check if there's more data to fetch
        afterVal, ok := result["data"].(map[string]interface{})["after"].(string)
        if !ok || afterVal == "" {
            break // Exit loop if there's no more data
        }
        after = afterVal
    }

    return threads
}

// Function to fetch comments for a specific thread with pagination
func getComments(subreddit, threadID string) []Comment {
	userAgent := os.Getenv("USER_AGENT")

    var comments []Comment
    after := "" // Initially empty to fetch the first page

    for {
        url := fmt.Sprintf(CommentsURL, subreddit, threadID, after)

        req, err := http.NewRequest("GET", url, nil)
        if err != nil {
            log.Fatalf("Error creating request to get comments: %v", err)
        }
        req.Header.Set("Authorization", "Bearer "+accessToken)
        req.Header.Set("User-Agent", userAgent)

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            log.Fatalf("Error sending request to get comments: %v", err)
        }
        defer resp.Body.Close()

        var result []interface{}
        body, _ := ioutil.ReadAll(resp.Body)
        json.Unmarshal(body, &result)

        // Parse comments and add them to the list
        if len(result) > 1 {
            for _, item := range result[1].(map[string]interface{})["data"].(map[string]interface{})["children"].([]interface{}) {
                data := item.(map[string]interface{})["data"].(map[string]interface{})
                comments = append(comments, Comment{
                    ID:      data["id"].(string),
                    Body:    data["body"].(string),
                    Author:  data["author"].(string),
                    Created: int64(data["created_utc"].(float64)),
                })
            }
        }

        // Check if there's more data to fetch
        afterVal, ok := result[1].(map[string]interface{})["data"].(map[string]interface{})["after"].(string)
        if !ok || afterVal == "" {
            break // Exit loop if there's no more data
        }
        after = afterVal
    }

    return comments
}

func main() {
	// Load environment variables from .env file
	loadEnv()

    // Authenticate and get access token
    accessToken = getAccessToken()
    if accessToken == "" {
        log.Fatal("Failed to obtain access token")
    }

    // Get threads from a specific subreddit
    subreddit := "Home" // Replace with desired subreddit
    threads := getSubredditThreads(subreddit)

    // For each thread, get comments and display thread title and comments
    for _, thread := range threads {
        fmt.Printf("Thread: %s (ID: %s)\n", thread.Title, thread.ID)

        comments := getComments(subreddit, thread.ID)
        for _, comment := range comments {
            fmt.Printf("Comment by %s: %s\n", comment.Author, comment.Body)
        }
        fmt.Println("------")
    }
}
