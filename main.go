package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type post struct {
	url       string
	blockID   string
	timestamp int64
	title     string
	tags      []string
}

var skippedTags []string

const MAXPAGES = 15

func main() {
	goodTags := []string{
		"2bwm",
		"4dwm",
		"9wm",
		"awesome",
		"awesomewm",
		"bspwm",
		"chunkwm",
		"cinnamon",
		"cwm",
		"dwm",
		"evilwm",
		"exwm",
		"gaps",
		"frankenwm",
		"fvwm",
		"hlwm",
		"herbstluftwm",
		"i3",
		"i3gaps",
		"i3-gaps",
		"i3wm",
		"i3-wm",
		"icewm",
		"lxqt",
		"manjaro",
		"mate",
		"pekwm",
		"qtile",
		"spectrwm",
		"sway",
		"eveningwm",
		"termux",
		"twm",
		"uwurawrxdwm",
		"vtwm",
		"xfce",
		"windowmaker",
	}
	firstURL := "https://old.reddit.com/r/unixporn/new/"

	sort.Strings(goodTags)
	lastTs, err := getLastTimestamp("timestamp.txt")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("last timestamp:", lastTs)

	// get page 1
	mustGoOn := true
	currentPage := 1
	postsToDownload := []post{}

	for mustGoOn {
		links, _ := getPage(firstURL)
		lastLinkID := links[len(links)-1].blockID
		lastLinkTimestamp := links[len(links)-1].timestamp

		filteredLinks := filterGoodLinks(links, lastTs, goodTags)

		fmt.Println("downloading page", currentPage, "(this page has", len(filteredLinks), "good images out of total", len(links), "images)")

		if lastLinkTimestamp <= lastTs || currentPage > MAXPAGES {
			fmt.Println("stopping fetching next pages:", lastLinkTimestamp <= lastTs, currentPage > 200)
			mustGoOn = false
		} else {
			time.Sleep(3 * time.Second)
		}

		for _, fl := range filteredLinks {
			postsToDownload = append(postsToDownload, fl)
		}

		firstURL = "https://old.reddit.com/r/unixporn/new/?count=25&after=" + lastLinkID
		currentPage++
	}

	var wg sync.WaitGroup
	wg.Add(len(postsToDownload))
	fmt.Println("----------------------------------------------------------------")

	for _, link := range postsToDownload {
		go downloadAndSaveImage(link.url, link.blockID, &wg, link.tags)
	}

	wg.Wait()

	newTs := time.Now().Unix() * 1000
	fmt.Println("--=[ DOWNLOAD FINISHED ]=--")
	fmt.Println("skipped tags:", skippedTags)
	fmt.Println("saving new last timestamp:", newTs)
	saveTimestamp(newTs, "timestamp.txt")
	fmt.Print("Press 'Enter' to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func getLastTimestamp(filename string) (int64, error) {

	tsbs, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	tss := strings.TrimSuffix(string(tsbs), "\n")

	ts, err := strconv.ParseInt(tss, 10, 64)
	if err != nil {
		return 0, err
	}

	return ts, nil
}

func getPage(link string) ([]post, error) {
	posts := []post{}

	client := &http.Client{}
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Close = true
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.109 Safari/537.36")

	res, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	docs := doc.Find("#siteTable .thing")

	docs.Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		blockID, found := s.Attr("data-fullname")
		if !found {
			fmt.Println("full-name was not found")
		}

		blockURL, found := s.Attr("data-url")
		if !found {
			fmt.Println("data-url was not found")
		}

		blockTimestamp, found := s.Attr("data-timestamp")
		if !found {
			fmt.Println("data-timestamp was not found")
		}

		blockTimestamp64, err := strconv.ParseInt(blockTimestamp, 10, 64)
		if err != nil {
			fmt.Println(err)
		}

		blockTitle := s.Find("a.title").Text()

		newPost := post{
			url:       blockURL,
			blockID:   blockID,
			timestamp: blockTimestamp64,
			title:     blockTitle,
			tags:      extractTags(blockTitle),
		}

		posts = append(posts, newPost)

		// fmt.Println("block_id", blockID, blockTimestamp, blockURL, blockTitle)
	})

	fmt.Println("finished")
	return posts, nil
}

func extractTags(tags string) []string {
	// title := "Screenshot[awesome] Triple monitor setup with single wallpaper"
	re := regexp.MustCompile(`\[([^]]+)\]`)
	matches := re.FindAllString(tags, -1)

	results := []string{}

	for _, match := range matches {
		// get rid of [ and ]
		match = strings.Replace(match, "[", "", -1)
		match = strings.Replace(match, "]", "", -1)

		// replace possible separators
		replacer := "---"
		match = strings.Replace(match, " ", replacer, -1)
		match = strings.Replace(match, "+", replacer, -1)
		match = strings.Replace(match, "|", replacer, -1)
		match = strings.Replace(match, "/", replacer, -1)

		// split by replacer
		splits := strings.Split(match, replacer)

		for _, tag := range splits {
			if len(tag) > 0 {
				results = append(results, strings.ToLower(strings.Replace(tag, " ", "", -1)))
			}
		}

	}

	sort.Strings(results)

	return results
}

func filterGoodLinks(links []post, lastTimestamp int64, goodTags []string) []post {

	goodPosts := []post{}

	for _, l := range links {
		if strings.Contains(l.url, ".png") && l.timestamp > lastTimestamp {

			// check tags
			isTagFound := false
			for _, tag := range l.tags {
				wasFound, _ := FindElementInArray(goodTags, tag)

				if wasFound && !isTagFound {
					isTagFound = true
				}

				if !wasFound {
					// append bad tag to tags skipped
					found, _ := FindElementInArray(skippedTags, tag)
					if !found {
						skippedTags = append(skippedTags, tag)
					}
				}
			}

			if isTagFound {
				goodPosts = append(goodPosts, l)
			}

		}
	}

	return goodPosts
}

func downloadAndSaveImage(link string, filename string, wg *sync.WaitGroup, tags []string) {
	defer wg.Done()

	resp, err := http.Get(link)
	if err != nil {
		fmt.Println("cannot get URL", link, err)
	}
	defer resp.Body.Close()

	out, err := os.Create("images/" + filename + " [" + strings.Join(tags, ",") + "].png")
	if err != nil {
		fmt.Println(err)
	}

	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(link, "has been downloaded as", filename)
}

func saveTimestamp(ts int64, filename string) {
	tsString := strconv.FormatInt(ts, 10)
	ioutil.WriteFile(filename, []byte(tsString), 0666)
}

// FindElementInArray finds given string in given string slice
func FindElementInArray(arr []string, text string) (bool, int) {
	// sort slice, just in case
	sort.Strings(arr)

	// locate text in arr
	location := sort.Search(len(arr), func(i int) bool { return arr[i] >= text })

	if location < len(arr) && arr[location] == text {
		return true, location
	}

	return false, -1
}
